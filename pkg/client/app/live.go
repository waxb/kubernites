package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/client"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/write"
	"github.com/waxb/kubernites/pkg/sim/podutil"
	"github.com/waxb/kubernites/pkg/sim/store"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	liveOpenTimeout = 2 * time.Minute
	liveWhoTimeout  = 15 * time.Second
	liveHoverHead   = "live:"
)

func hudStep(step string) string {
	switch step {
	case live.StepMetrics:
		return hud.LiveReadingUsage
	case live.StepDrawing:
		return hud.LiveDrawing
	}
	return ""
}

func liveStage(p live.Progress) string {
	switch {
	case p.Step == live.StepDrawing:
		return "drawing the realm"
	case p.Step == live.StepMetrics:
		return "reading usage from metrics-server"
	case p.List != "":
		return "reading " + p.List
	}
	return "asking the cluster for its version"
}

type LiveSource struct {
	Files     []string
	List      func(files []string) ([]live.ContextInfo, error)
	Who       func(ctx context.Context, files []string, name string) (live.Identity, error)
	Open      func(ctx context.Context, files []string, name string, progress func(live.Progress)) (client.Realm, error)
	OpenGated func(ctx context.Context, files []string, name string, gate *write.Gate, progress func(live.Progress)) (client.Realm, error)
}

func (s LiveSource) withDefaults() LiveSource {
	if s.List == nil {
		s.List = live.ListContexts
	}
	if s.Who == nil {
		s.Who = live.WhoAmI
	}
	if s.Open == nil {
		s.Open = func(ctx context.Context, files []string, name string, progress func(live.Progress)) (client.Realm, error) {
			return live.Connect(ctx, files, name, time.Now(), progress)
		}
	}
	if s.OpenGated == nil {
		s.OpenGated = func(ctx context.Context, files []string, name string, gate *write.Gate, progress func(live.Progress)) (client.Realm, error) {
			return live.ConnectGated(ctx, files, name, gate, time.Now(), progress)
		}
	}
	return s
}

type liveResult struct {
	realm client.Realm
	err   error
}

type liveOpening struct {
	cancel  context.CancelFunc
	done    chan liveResult
	mu      sync.Mutex
	at      live.Progress
	started time.Time
	limit   time.Duration
}

func (o *liveOpening) tell(p live.Progress) {
	o.mu.Lock()
	o.at = p
	o.mu.Unlock()
}

func (o *liveOpening) seen() hud.LiveProgress {
	o.mu.Lock()
	defer o.mu.Unlock()
	p := o.at
	return hud.LiveProgress{Step: hudStep(p.Step), List: p.List, Count: p.Count, Done: p.Done, Total: p.Total, Nodes: p.Nodes, Namespaces: p.Namespaces, Pods: p.Pods}
}

func (o *liveOpening) stalled() (error, bool) {
	if time.Since(o.started) <= o.limit {
		return nil, false
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	return fmt.Errorf("the cluster did not open within %s: it stopped while %s. Nothing was written; try again, or open another context", o.limit.Round(time.Second), liveStage(o.at)), true
}

func (o *liveOpening) readable(err error) error {
	if !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	return fmt.Errorf("the cluster did not open within %s: it stopped while %s (%w)", o.limit.Round(time.Second), liveStage(o.at), err)
}

type liveAsking struct {
	context string
	cancel  context.CancelFunc
	done    chan whoResult
}

type whoResult struct {
	who live.Identity
	err error
}

func (g *Game) liveIntent(it hud.Intent) bool {
	src := g.Opts.Live.withDefaults()
	switch it.Kind {
	case hud.IntentLive:
		g.loadChoices()
		g.Picker.Open(src.Files)
		g.audio.PlaySFX(sound.SFXClick)
	case hud.IntentLiveForget:
		g.forgetChoice(it.Target)
	case hud.IntentLiveList:
		list, err := src.List(src.Files)
		g.Picker.ShowContexts(liveContexts(list), err)
	case hud.IntentLiveOpen:
		g.cancelLive()
		limit := g.LiveLimit
		if limit <= 0 {
			limit = liveOpenTimeout
		}
		ctx, cancel := context.WithTimeout(context.Background(), limit)
		done := make(chan liveResult, 1)
		opening := &liveOpening{cancel: cancel, done: done, started: time.Now(), limit: limit}
		g.Opening = opening
		files, name := src.Files, it.Target
		gate, err := g.gateFor(name, it.Command)
		if err != nil {
			cancel()
			g.Opening = nil
			g.Picker.Failed(err)
			return true
		}
		go func() {
			var r client.Realm
			var err error
			if gate == nil {
				r, err = src.Open(ctx, files, name, opening.tell)
			} else {
				r, err = src.OpenGated(ctx, files, name, gate, opening.tell)
			}
			done <- liveResult{realm: r, err: err}
		}()
	case hud.IntentLiveWho:
		g.cancelAsk()
		ctx, cancel := context.WithTimeout(context.Background(), liveWhoTimeout)
		done := make(chan whoResult, 1)
		g.asking = &liveAsking{context: it.Target, cancel: cancel, done: done}
		files, name := src.Files, it.Target
		go func() {
			who, err := src.Who(ctx, files, name)
			done <- whoResult{who: who, err: err}
		}()
	case hud.IntentLiveCancel:
		g.cancelLive()
	default:
		return false
	}
	return true
}

func (g *Game) cancelLive() {
	g.cancelAsk()
	if g.Opening != nil {
		g.Opening.cancel()
		g.Opening = nil
	}
}

func (g *Game) cancelAsk() {
	if g.asking != nil {
		g.asking.cancel()
		g.asking = nil
	}
}

func (g *Game) stepAsk() {
	if g.asking == nil {
		return
	}
	select {
	case res := <-g.asking.done:
		g.asking.cancel()
		g.Picker.Answered(g.asking.context, res.who.Username, res.who.Groups, res.err)
		g.asking = nil
	default:
	}
}

func (g *Game) StepLive() {
	g.stepAsk()
	if g.Opening == nil {
		return
	}
	g.Picker.Progress = g.Opening.seen()
	select {
	case res := <-g.Opening.done:
		opening := g.Opening
		opening.cancel()
		g.Opening = nil
		if res.err != nil {
			var changed *live.IdentityChanged
			if errors.As(res.err, &changed) {
				g.identityChanged(changed.Context, changed.Facets)
				return
			}
			g.Picker.Failed(opening.readable(res.err))
			return
		}
		g.EnterLive(res.realm)
	default:
		if err, late := g.Opening.stalled(); late {
			g.Opening.cancel()
			g.Opening = nil
			g.Picker.Failed(err)
		}
	}
}

func (g *Game) identityChanged(name, facets string) {
	was := ""
	if i := g.Picker.Chosen; i >= 0 && i < len(g.Picker.Contexts) && g.Picker.Contexts[i].Name == name {
		was = g.Picker.Contexts[i].Facets
	}
	g.forgetChoice(name)
	src := g.Opts.Live.withDefaults()
	list, err := src.List(src.Files)
	g.Picker.IdentityChanged(name, liveContexts(list), err, live.ChangedFacets(was, facets))
	g.audio.PlaySFX(sound.SFXError)
}

type pumper interface {
	Pump(realSeconds float64)
}

type watcher interface {
	StartWatching()
	Stop()
}

func (g *Game) EnterLive(r client.Realm) {
	g.Picker.Close()
	if g.Sess != nil && !g.Sess.Live {
		g.campaign = g.Sess
	}
	g.stopWatching()
	g.Sess = g.settledSession(r)
	g.Scene = SceneRealm
	if g.Opts.Capture == "" {
		ApplyNames(g.liveNames())
		g.Sess.TrueNames = g.liveNames()
	}
	g.pushChoices()
	if w, ok := r.(watcher); ok {
		w.StartWatching()
	}
}

func (g *Game) stopWatching() {
	g.Sess.endLive()
}

const liveEndWait = 2 * time.Second

func (s *Session) endLive() {
	if s == nil || !s.Live || s.liveEnded {
		return
	}
	s.liveEnded = true
	if w := s.Writer; w != nil {
		w.end(liveEndWait)
	}
	if w, ok := s.Realm.(watcher); ok {
		w.Stop()
	}
	for deadline := time.Now().Add(liveEndWait); s.Hud.EndStoneJobs() && time.Now().Before(deadline); {
		time.Sleep(5 * time.Millisecond)
	}
	s.stepForwards()
	if s.Writer != nil {
		s.stepWriter()
	}
}

func (g *Game) leaveLive() {
	if g.Sess != nil && g.Sess.Live {
		if g.Opts.Capture == "" {
			on := g.Settings.TrueNames
			if g.Opts.LiveOnly {
				on = g.liveNames()
			}
			ApplyNames(on)
		}
		g.stopWatching()
		g.Sess, g.campaign = g.campaign, nil
	}
}

func liveContexts(list []live.ContextInfo) []hud.LiveContext {
	out := make([]hud.LiveContext, len(list))
	for i, c := range list {
		out[i] = hud.LiveContext{Name: c.Name, Cluster: c.Cluster, Server: c.Server, User: c.User, Namespace: c.Namespace, Identity: c.Identity, Facets: c.Facets, ClusterID: c.ClusterID}
	}
	return out
}

type cardSource interface {
	Snapshot() live.Snapshot
	World() *live.World
}

type liveView interface {
	cardSource
	Drawn() uint64
}

type liveLayout struct {
	World   *live.World            `json:"-"`
	Pins    map[string]world.Point `json:"-"`
	Away    map[string]bool        `json:"-"`
	lots    []world.HeraldLot
	fiefs   []string
	plaques map[string]world.Plaque
}

func layoutOf(w *live.World) liveLayout {
	l := liveLayout{World: w, Pins: map[string]world.Point{}, Away: map[string]bool{}}
	for _, h := range w.Huts {
		switch {
		case h.Shrine:
			l.Away[string(h.UID)] = true
		case h.Placed:
			l.Pins[string(h.UID)] = world.Point{X: h.X, Y: h.Y}
		}
	}
	for _, lot := range w.Lots {
		tiles := make([]world.Point, len(lot.Tiles))
		for i, t := range lot.Tiles {
			tiles[i] = world.Point{X: t.X, Y: t.Y}
		}
		l.lots = append(l.lots, world.HeraldLot{Order: lot.Order, Tiles: tiles})
	}
	for _, d := range w.Districts {
		if d.Namespace != "" {
			l.fiefs = append(l.fiefs, d.Namespace)
		}
	}
	for _, st := range w.Site.Structures {
		if st.Kind == live.KindSeaChancery {
			l.plaques = map[string]world.Plaque{st.ID: {Title: st.Label, Line: w.Chancery}}
		}
	}
	return l
}

func (s *Session) goLive(r liveView) {
	s.Live = true
	s.AutoPauseOn, s.pauseTap = false, nil
	s.Layout = layoutOf(r.World())
	s.drawnAt = r.Drawn()
	s.Pods, s.Rolls = s.podViews(), s.guildRolls()
	s.Hud.SetLive(s.liveCard(r))
	s.Hud.SetStaging(false)
	s.Hud.Stone.Greet(hud.LiveGreeting(nodeNames(r.Snapshot())))
}

func nodeNames(snap live.Snapshot) []string {
	out := make([]string, 0, len(snap.Nodes))
	for _, n := range snap.Nodes {
		out = append(out, n.Name)
	}
	return out
}

func (s *Session) liveCard(r liveView) hud.LiveCard {
	if s.cardAt != r.Drawn() || s.cardBase.Context == "" {
		s.cardBase, s.cardAt = LiveCardOf(r), r.Drawn()
	}
	card := s.cardBase
	card.Reads = live.ReadOnlyCommand
	s.liveCardWrites(&card)
	card.Quiet = quietTrouble(s.State.Calamities)
	card.Listed = listedReads(r)
	card.Forwards = s.liveForwards()
	if f, ok := r.(freshness); ok {
		fresh := f.Freshness()
		card.Watching = fresh.Watching
		if !fresh.StaleSince.IsZero() {
			card.Stale, card.StaleWhy = fresh.StaleSince.UTC().Format("15:04:05 UTC"), fresh.Why
		}
	}
	return card
}

type freshness interface {
	Freshness() live.Freshness
}

func quietTrouble(list []view.CalamityView) []hud.QuietTrouble {
	var out []hud.QuietTrouble
	for _, v := range list {
		if v.Quiet && v.Mended == "" {
			out = append(out, hud.QuietTrouble{ID: v.ID, Kind: v.Kind, Cause: v.Cause})
		}
	}
	return out
}

func (s *Session) redrawLive() {
	if !s.Live {
		return
	}
	r, ok := s.Realm.(liveView)
	if !ok || r.Drawn() == s.drawnAt {
		return
	}
	s.drawnAt = r.Drawn()
	s.Layout = layoutOf(r.World())
	s.herald = nil
}

func (s *Session) refreshLive() {
	if r, ok := s.Realm.(liveView); ok && s.Live {
		s.Hud.SetLive(s.liveCard(r))
	}
}

func (s *Session) podViews() []model.PodView {
	views := s.Cache.PodViews()
	if s.Live {
		for i := range views {
			views[i].System = false
		}
	}
	return views
}

func (s *Session) guildRolls() []model.GuildRoll {
	if s.Live {
		return s.Cache.EveryGuildRoll()
	}
	return s.Cache.GuildRolls()
}

func (s *Session) scenePods() []model.PodView {
	if len(s.Layout.Away) == 0 {
		return s.Pods
	}
	out := make([]model.PodView, 0, len(s.Pods))
	for _, p := range s.Pods {
		if !s.Layout.Away[p.UID] {
			out = append(out, p)
		}
	}
	return out
}

func (s *Session) liveTip(id string) string {
	if s.Layout.World == nil {
		return ""
	}
	tip, ok := s.Layout.World.Tips[id]
	if !ok {
		return ""
	}
	return ui.Bridge(tip.Story, tip.Names)
}

func (s *Session) Heraldry() *world.Heraldry {
	if s.herald == nil || s.heraldAt != s.version {
		fiefs := s.Layout.fiefs
		if !s.Live {
			fiefs = s.Cache.Names(store.KindNamespace)
		}
		s.herald, s.heraldAt = world.NewHeraldry(fiefs, s.Pods, s.Layout.lots), s.version
	}
	return s.herald
}

func LiveCardOf(r cardSource) hud.LiveCard {
	snap, w := r.Snapshot(), r.World()
	trouble := 0
	for i := range snap.Pods {
		if podTroubled(&snap.Pods[i]) {
			trouble++
		}
	}
	card := hud.LiveCard{
		Context: snap.Context.Name, Server: snap.Context.Server, Version: snap.Version, Who: snap.Identity.Username,
		Keeps: len(snap.Nodes), Huts: len(snap.Pods), Fiefs: len(snap.Namespaces), Isles: len(w.Isles),
		Trouble: trouble, Seen: snap.TakenAt.UTC().Format("15:04 UTC"), Denied: snap.Denied, Idle: w.Idle,
	}
	for _, h := range w.Huts {
		if h.Shrine {
			card.Shrines++
		}
	}
	for _, r := range snap.Refused {
		if r.Namespace != "" {
			card.Refused, card.Namespace = append(card.Refused, r.List), r.Namespace
		}
	}
	return card
}

func podTroubled(p *corev1.Pod) bool {
	if p.Status.Phase == corev1.PodSucceeded {
		return false
	}
	if p.Status.Phase != corev1.PodRunning {
		return true
	}
	return !podutil.IsReady(p)
}
