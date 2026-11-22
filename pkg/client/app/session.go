package app

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/waxb/kubernites/pkg/live/kubectltree"

	"github.com/waxb/kubernites/pkg/client"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/sim/store"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	pollSeconds = 0.25
	OmenLinger  = 1.5
)

type Session struct {
	forwards        []kubectltree.Forward
	forwardsSeen    []int
	captured        []kubectltree.Forward
	holdSays        bool
	Live            bool `json:"-"`
	steps           stepHold
	chests          treasureChests
	beats           storageBeats
	rounds          watchRounds
	Realm           client.Realm `json:"-"`
	Cache           *model.Cache `json:"-"`
	pending         []store.Event
	version         uint64
	State           view.GameState `json:"-"`
	metrics         view.MetricsSnapshot
	Site            view.Site       `json:"-"`
	Pods            []model.PodView `json:"-"`
	Tidings         []hud.Tiding    `json:"-"`
	Cam             render.Camera   `json:"-"`
	Renderer        *world.Renderer `json:"-"`
	Hud             *hud.HUD        `json:"-"`
	Selection       hud.Selection   `json:"-"`
	panel           string
	Placing         string `json:"-"`
	placeHome       *world.Vec
	Ghost           *world.Ghost `json:"-"`
	Paused          bool         `json:"-"`
	Typing          bool         `json:"-"`
	step            stepCue
	Record          func(kind string, fields map[string]any) `json:"-"`
	Clock           float64                                  `json:"-"`
	Clocks          Clocks                                   `json:"-"`
	Poll            float64                                  `json:"-"`
	Transients      []world.Transient                        `json:"-"`
	seen            map[string]bool
	ProcT           float64           `json:"-"`
	Built           map[string]string `json:"-"`
	Hearths         map[string]string `json:"-"`
	Omen            *world.Omen       `json:"-"`
	Reduce          bool              `json:"-"`
	TrueNames       bool              `json:"-"`
	volumes         hud.Volumes
	Fullscreen      bool    `json:"-"`
	Reweave         float64 `json:"-"`
	impTold         bool
	quietNews       map[string]bool
	Age             string         `json:"-"`
	Vocab           hud.Vocabulary `json:"-"`
	ServedAt        float64        `json:"-"`
	Lens            Lens           `json:"-"`
	Horns           []model.Horn   `json:"-"`
	replicas        map[string]int
	Coins           purse      `json:"-"`
	Rail            Rail       `json:"-"`
	Town            Town       `json:"-"`
	Guide           hud.Guide  `json:"-"`
	GuideFixture    *hud.Guide `json:"-"`
	notes           []hud.Intent
	Echoes          []hud.Tiding `json:"-"`
	nodes           map[string]model.NodeView
	Rolls           []model.GuildRoll `json:"-"`
	Autonomy        Autonomy          `json:"-"`
	autonomyToday   int
	AutonomyFixture func(float64) ([]model.Autonomy, int) `json:"-"`
	caesar          model.Caesar
	ExtraReveals    []string `json:"-"`
	debuts          []world.Debut
	Rollouts        model.Rollouts `json:"-"`
	Hover           string         `json:"-"`
	TipAt           [2]int         `json:"-"`
	Cursor          bool           `json:"-"`
	Land            landView       `json:"-"`
	Tier            int            `json:"-"`
	portLevels      map[string]int
	seasons         map[string]int
	Crews           []model.CrewChain `json:"-"`
	Writer          *liveWriter       `json:"-"`
	liveEnded       bool              `json:"-"`
	hints           bool
	cycle           PickCycle
	tipped          map[string]bool
	stood           map[string]float64
	lands           landCache
	Saved           bool      `json:"-"`
	SavedAt         float64   `json:"-"`
	SaveErr         string    `json:"-"`
	Auto            AutoPause `json:"-"`
	AutoPauseOn     bool      `json:"-"`
	pauseTap        func([]hud.PauseAsk)
	Away            awaySpan `json:"-"`
	awayDue         bool
	LastActAt       float64    `json:"-"`
	StepCam         stepCam    `json:"-"`
	Layout          liveLayout `json:"-"`
	herald          *world.Heraldry
	heraldAt        uint64
	Talk            chatter `json:"-"`
	worker          workerCard
	drawnAt         uint64
	cardAt          uint64
	cardBase        hud.LiveCard
}

func NewSession(r client.Realm, lib *gfx.Library) *Session {
	s := &Session{
		Realm: r, Cache: model.NewCache(), Renderer: world.NewRenderer(lib),
		Cam: render.NewCamera(hud.Width/2, hud.TopH+30), seen: map[string]bool{}, Built: map[string]string{},
		ServedAt: -100, replicas: map[string]int{},
	}
	s.Hud = hud.New(lib, s.treeBackend(hud.Backend{Kubectl: s.recorded(r.Kubectl), Inspect: r.Inspect, Apply: r.Apply}))
	ui.WatchText(s.Hud.SeeText)
	ui.WatchTextAt(s.Hud.SeeTextAt)
	r.Watch(func(e store.Event) { s.pending = append(s.pending, e) })
	r.WithSnapshot(func(objs []store.Object) { s.Cache.Reset(objs) })
	s.pending = nil
	for _, e := range s.Cache.Events() {
		s.seen[string(e.UID)+"/"+e.ResourceVersion] = true
	}
	s.Refresh()
	for _, st := range s.Site.Structures {
		s.Built[st.ID] = st.State
	}
	s.Hearths = HearthStates(s.Pods)
	s.Age = s.State.Age
	s.ProcT = -1
	for _, p := range s.State.Proclamations {
		s.ProcT = max(s.ProcT, p.T)
	}
	s.CenterOn(float64(s.Site.Gate.X), float64(s.Site.Gate.Y))
	return s
}

func (s *Session) zoomClosest() {
	cx, cy := float64(hud.Width)/2, float64(hud.ConsoleY+hud.TopH)/2
	s.Cam = s.Cam.ZoomAt(render.ZoomLevels[len(render.ZoomLevels)-1], cx, cy).Clamp(s.Site.Width, hud.Width, hud.ConsoleY)
}

func (s *Session) CenterOn(gx, gy float64) {
	s.Cam = s.Cam.CenterOn(gx, gy, hud.Width, hud.ConsoleY+hud.TopH).Clamp(s.Site.Width, hud.Width, hud.ConsoleY)
}

func (s *Session) Advance(dt float64) {
	s.Clock += dt
	running := !s.Paused && !hud.Holds(s.Hud.Overlay()) && !s.ScrollOpen()
	s.Clocks = s.Clocks.Step(dt, s.State.Speed, running)
	s.StepLens()
	if p, ok := s.Realm.(pumper); ok {
		p.Pump(dt)
	} else if running {
		s.Realm.Advance(dt)
	}
	for _, e := range s.pending {
		s.Cache.Apply(e)
	}
	s.pending = s.pending[:0]
	s.redrawLive()
	s.stepWriter()
	s.stepForwards()
	if v := s.Cache.Version(); v != s.version {
		s.version = v
		s.Pods = s.podViews()
		s.Vocab = Vocab(s.Cache)
		s.nodes, s.Rolls = s.Cache.NodeViews(), s.guildRolls()
		s.Crews = s.Cache.CrewChains()
		s.trackStood()
	}
	s.Poll += dt
	if s.Poll >= pollSeconds {
		s.Poll = 0
		s.Refresh()
	}
	s.Transients = PruneTransients(s.Transients, s.Clocks.World)
}

func (s *Session) Refresh() {
	roll := s.State.Roll
	s.State = hud.WithoutIdleScale(WithTry(s.Realm.GameState()))
	s.State.Objectives = s.steps.steady(s.State.Objectives, s.Clock)
	s.State.Reveals = append(s.State.Reveals, s.ExtraReveals...)
	s.Guide = s.CurrentGuide()
	if s.GuideFixture != nil {
		s.Hud.SetStaging(false)
	}
	s.TreasuryCoins()
	s.metrics = s.Realm.Metrics()
	s.caesar = CaesarOf(s.State, s.metrics)
	s.Horns = s.Cache.Horns()
	s.Site = DecorateSite(s.Realm.Site(), s.Horns)
	s.debuts = s.debutsNow()
	s.Pods = s.podViews()
	epoch := float64(s.State.FoundedAt.UnixNano()) / 1e9
	s.Tidings = WithEchoes(Tidings(s.Cache.Events(), s.State.Proclamations, contains(s.State.Reveals, "tidings"), epoch), s.Echoes)
	s.nodes, s.Rolls = s.Cache.NodeViews(), s.guildRolls()
	s.Crews = s.Cache.CrewChains()
	s.trackStood()
	s.TrackRollouts()
	s.Selection = KeepChosen(RefreshSelection(s.Selection, s.Site, s.Pods), s.Crews)
	s.trackLand()
	s.Transients = append(s.Transients, EventTransients(s.Cache.Events(), s.seen, s.Renderer.Home, s.Clocks.World)...)
	s.trackTown()
	s.Vocab = Vocab(s.Cache)
	s.TrackOmen()
	s.trackAutonomy()
	s.trackBadgeAct(roll)
	s.TrackWorkers()
	s.StepRail()
	s.forecastOrder()
	s.refreshLive()
}

func (s *Session) StepRail() {
	s.Rail = s.Rail.Step(RailInput{Now: s.Clock, Frame: s.railFrame(), Quiet: s.quietNews})
}

type orderForecaster interface {
	OrderRoom(a view.Action) (view.RolloutRoomView, bool)
}

func (s *Session) forecastOrder() {
	a, ok := s.Hud.PendingOrder(hud.Frame{State: s.State, Selection: s.Selection, Crews: s.Crews, Guide: s.Guide})
	fc, can := s.Realm.(orderForecaster)
	if !ok || !can {
		s.Hud.SetOrderRoom(nil)
		return
	}
	if v, ok := fc.OrderRoom(a); ok {
		s.Hud.SetOrderRoom(&v)
		return
	}
	s.Hud.SetOrderRoom(nil)
}

func (s *Session) TrackOmen() {
	imp := s.State.Imp
	switch {
	case imp != nil && (s.Omen == nil || s.Omen.Pod != imp.Pod):
		s.Omen = &world.Omen{Pod: imp.Pod, Start: s.Clocks.World, Arrive: s.Clocks.World + imp.In}
	case imp == nil && s.Omen != nil && s.Clocks.World > s.Omen.Arrive+OmenLinger:
		s.Omen = nil
	}
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func (s *Session) ScrollOpen() bool {
	return s.Hud.ScrollOpen(hud.Frame{State: s.State, Guide: s.shownGuide(), Unguided: s.IsUnguided()})
}

func (s *Session) shownGuide() hud.Guide {
	return s.Hud.StageGuide(s.Guide, s.State)
}

func (s *Session) Frame(muted bool) hud.Frame {
	return s.Hud.Stage(hud.Frame{
		State: s.State, Metrics: s.metrics, Pods: s.Pods, Events: WithOmen(s.Tidings, s.State.Imp),
		Selection: s.Selection, Worker: s.workerView(), Placing: s.Placing, Paused: s.Paused || s.Hud.Overlay() == hud.OverlayBook, Muted: muted, Clock: s.Clock, EventClock: s.Clocks.Event,
		Site: s.Site, View: s.viewCorners(), Vocabulary: s.Vocab, Reduce: s.Reduce, TrueNames: s.TrueNames, Unguided: s.IsUnguided(), Version: VersionLabel(), Fullscreen: s.Fullscreen, Volumes: s.volumes,
		Horns: s.Horns, KeepsReady: KeepsReady(s.Cache), Joining: Joining(s.Cache), Alerts: s.Alerts(),
		Notices: s.Rail.Notices, Guide: s.Guide, Nodes: s.nodes, Rolls: s.Rolls, AutonomyToday: s.autonomyToday,
		GoalAt: s.goalAt(), GoalOfNext: s.goalOfNext(), Caesar: s.caesar, Beacons: s.beacons(), WorldBoxes: worldRects(s.Renderer.Drawn()),
		Survey: s.SurveyPanel(), Causes: s.CauseTips(), Places: s.Places(), Tip: s.HoverTip(), TipAt: s.TipAt, Crews: s.Crews,
		Land: s.selectionLand(), Panels: s.Hud.Panels(),
		Saved: s.Saved, SavedAgo: s.savedAgo(), SaveError: s.SaveErr,
	})
}

func (s *Session) BeaconScene() world.Scene {
	sc := world.Scene{Site: s.Site, Ghost: s.Ghost, Plot: s.FirstKeepPlot(), Age: gfx.AgeTier(s.State.Age), Beckon: s.Beckon(), Reduce: s.Reduce, Time: s.Clocks.World, Real: s.Clock}
	if v, ok := s.BeaconGoal(); ok {
		sc.Goal = &v
	}
	return sc
}

func (s *Session) beacons() []ui.Rect {
	return s.Renderer.BeaconRects(s.Cam, s.BeaconScene())
}

func (s *Session) CoveredBeacons(f hud.Frame) bool {
	return s.beaconsUnder(f, s.Hud.Covers)
}

func (s *Session) HiddenBeacons(f hud.Frame) bool {
	return s.beaconsUnder(f, s.Hud.Hides)
}

func (s *Session) beaconsUnder(f hud.Frame, under func(ui.Rect, hud.Frame) bool) bool {
	covered := false
	for _, r := range f.Beacons {
		if r.Y+r.H > hud.ConsoleY || r.Y+r.H <= hud.TopH {
			return false
		}
		covered = covered || under(r, f)
	}
	return covered
}

func (s *Session) BeaconsOnTop(f hud.Frame) bool {
	return s.Hud.Overlay() == "" && s.CoveredBeacons(f) && !s.HiddenBeacons(f)
}

func (s *Session) drawCoveredBeacons(dst *ebiten.Image, f hud.Frame) {
	if s.BeaconsOnTop(f) {
		s.Renderer.DrawBeacons(dst, s.Cam, s.BeaconScene())
	}
}

func (s *Session) viewCorners() [4][2]float64 {
	pts := [4][2]float64{{0, hud.TopH}, {hud.Width, hud.TopH}, {hud.Width, hud.ConsoleY}, {0, hud.ConsoleY}}
	var out [4][2]float64
	for i, p := range pts {
		gx, gy := s.Cam.ScreenToGrid(p[0], p[1])
		out[i] = [2]float64{gx, gy}
	}
	return out
}

func (s *Session) Scene() world.Scene {
	pg := s.State.Pilgrims
	return world.Scene{
		Site: s.Site, Pods: s.scenePods(), Pins: s.Layout.Pins, Plaques: s.Layout.plaques, Metrics: s.metrics, ArrivalRate: pg.ArrivalRate, Queue: pg.Queue,
		DayPhase: pg.DayPhase, Time: s.Clocks.World, Real: s.Clock, Selected: s.Selection.ID, Ghost: s.Ghost, Transients: s.Transients,
		Omen: s.Omen, Reduce: s.Reduce, GateOpen: true, ServedAt: s.ServedAt, Errands: s.Errands(),
		Beckon: s.Beckon(), Hint: ClickHint(s.State), Served: pg.TotalServed, Dropped: pg.TotalDropped, HandServed: pg.HandServed,
		RealmTime: s.State.RealmTime, Frictions: s.frictions(),
		Ledger: world.Ledger{Open: s.State.Ledger.Open, Share: LedgerShare(s.State.Ledger)}, Surge: s.State.Siege.Phase == "active",
		Caesar: s.caesar, Freight: s.State.Freight, Chapter: s.State.Tutorial, HoldSays: s.holdSays, Hover: s.Hover, Golden: Gilded(s.State.Golden), Berths: s.State.Berths.Open,
		Borders: &s.Land.borders, Tints: s.landTints(), Causes: s.Causes(), Incidents: s.incidentMarks(), Chests: s.chests.marks(s.Clocks.World), Watch: s.State.Watch, Preview: s.previewMarks(), Spots: s.SpotMarks(), Crews: s.Crews,
		Plot: s.FirstKeepPlot(), Age: gfx.AgeTier(s.State.Age), Debuts: s.debuts, Rollouts: s.Rollouts, Keeps: s.State.Keeps, Storage: s.State.StorageState,
		Stone: s.stoneStands(), Stood: s.stood, Fallen: fallenOnRoll(s.State.Roll), Idle: idleOnRoll(s.State.Roll),
		Focus: s.focusKey(), Speech: s.speechShown(),
	}
}

func (s *Session) Draw(dst *ebiten.Image, muted bool) {
	f := s.Frame(muted)
	sc := s.Scene()
	sc.Panels = s.Hud.PanelRects(f)
	sc.Avoid = append(s.Hud.AvoidRects(f), sc.Panels...)
	sc.Panels = append(sc.Panels, speechRects(s.Renderer.Drawn())...)
	if mode := s.Hud.Lens(); mode != "" {
		sc.Lens = LensOf(mode, LensSources{Site: s.Site, Metrics: s.metrics, Caesar: s.caesar})
	}
	if s.Hud.Lens() == hud.LensHeraldry {
		sc.Heraldry = s.Heraldry()
	}
	if !f.Introducing() {
		sc.Callouts = s.callouts()
	}
	if v, ok := s.BeaconGoal(); ok {
		sc.Goal = &v
	}
	s.drawInset(sc)
	s.Renderer.Draw(dst, s.Cam, sc)
	s.Renderer.DrawLamps(dst, s.Cam, sc)
	s.Renderer.DrawOmenMark(dst, s.Cam, sc)
	s.Renderer.DrawStructureBars(dst, s.Cam, sc)
	s.Renderer.DrawKeepPips(dst, s.Cam, sc)
	s.Renderer.DrawCodexAlarm(dst, s.Cam, sc)
	s.Renderer.DrawLensMarks(dst, s.Cam, sc)
	s.Renderer.DrawCauseTags(dst, s.Cam, sc)
	s.Renderer.DrawStorageMarks(dst, s.Cam, sc)
	s.Renderer.DrawSpeech(dst, s.Cam, sc)
	s.Renderer.DrawCallouts(dst, s.Cam, sc)
	if !f.Introducing() {
		s.Renderer.DrawForemanCallouts(dst, s.Cam, sc)
	}
	f.WorldBoxes = worldRects(s.Renderer.Drawn())
	s.Hud.Draw(dst, f)
	s.drawCoveredBeacons(dst, f)
}

func (s *Session) recorded(run hud.Runner) hud.Runner {
	return func(command string) view.CommandResult {
		out := run(command)
		s.Hud.NoteAct("kubectl", s.State.RealmTime)
		if s.Record != nil {
			s.Record("kubectl", map[string]any{"command": command, "ok": out.OK})
		}
		return out
	}
}

func (s *Session) savedAgo() float64 { return max(s.Clock-s.SavedAt, 0) }

func (s *Session) IsUnguided() bool {
	return s.State.Guidance == view.GuidanceStrategist || s.Hud.InChapter()
}
