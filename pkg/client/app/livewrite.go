package app

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/live/write"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	liveSendTimeout = 30 * time.Second
	ledgerShown     = 4
	stageInput      = "input"
	stageClass      = "class"
	stageAsking     = "asking"
	stagePreview    = "preview"
	stageSending    = "sending"
	stageDraining   = "draining"
)

type accessState struct {
	done bool
	v    write.Verdict
}

type orderRun struct {
	id     int
	spec   orderSpec
	stage  string
	values []string
	order  write.Order
	drain  *write.Drain
	need   write.Class
	note   string
	bad    bool
	fields []write.Field
	change []string
	after  string
}

type drainRun struct {
	node    string
	stop    context.CancelFunc
	stopped bool
	plan    write.Drain
	done    int
	blocked string
	budget  string
	failed  []drainFailure
	over    bool
	err     string
}

type liveWriter struct {
	gate    *write.Gate
	ledger  *write.Ledger
	level   write.Level
	reviews map[write.Access]*accessState
	mu      sync.Mutex
	inbox   []func(*Session)
	sync    bool
	run     *orderRun
	drain   *drainRun
	entries []ledgerLine
	now     func() time.Time
	pass    func(write.Class)
	suffix  func() string
	seq     int
	hints   bool
	shown   string
	ctx     context.Context
	cancel  context.CancelFunc
	running sync.WaitGroup
	every   time.Duration
}

func newLiveWriter(g *write.Gate, dir string, now func() time.Time, pass func(write.Class)) *liveWriter {
	if g == nil {
		return nil
	}
	w := &liveWriter{gate: g, level: g.Level(), reviews: map[write.Access]*accessState{}, now: now, pass: pass, every: write.DrainEvery}
	w.ctx, w.cancel = context.WithCancel(context.Background())
	w.ledger = write.OpenLedger(dir, g.Context())
	if dir == "" {
		w.ledger = nil
	}
	if old, err := w.ledger.Read(); err == nil && len(old) > 0 {
		for _, e := range old[max(0, len(old)-ledgerShown):] {
			w.entries = append(w.entries, ledgerLine{at: e.At, object: e.Object, code: e.Code, answer: e.Answer, earlier: true})
		}
	}
	w.suffix = jobSuffix
	return w
}

const jobLetters = "bcdfghjklmnpqrstvwxz2456789"

func jobSuffix() string {
	b := make([]byte, 3)
	for i := range b {
		b[i] = jobLetters[rand.IntN(len(jobLetters))]
	}
	return string(b)
}

func (s *Session) liveLevel() string {
	if s.Writer == nil {
		return hud.LevelReadOnly
	}
	return s.Writer.level.String()
}

func (s *Session) stepWriter() {
	w := s.Writer
	if w == nil {
		if s.Live {
			s.Hud.SetLiveOrders(s.liveOrders())
		}
		return
	}
	for {
		w.mu.Lock()
		next := w.inbox
		w.inbox = nil
		w.mu.Unlock()
		if len(next) == 0 {
			break
		}
		for _, fn := range next {
			fn(s)
		}
		s.refreshLive()
	}
	s.Hud.SetLiveOrders(s.liveOrders())
}

func (w *liveWriter) later(fn func(*Session)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.inbox = append(w.inbox, fn)
}

func (w *liveWriter) spawn(fn func()) {
	if w.sync {
		fn()
		return
	}
	w.running.Add(1)
	go func() {
		defer w.running.Done()
		fn()
	}()
}

func (w *liveWriter) end(limit time.Duration) {
	w.cancel()
	done := make(chan struct{})
	go func() {
		w.running.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(limit):
	}
}

func (s *Session) review(a write.Access) *accessState {
	w := s.Writer
	if st, ok := w.reviews[a]; ok {
		return st
	}
	st := &accessState{}
	w.reviews[a] = st
	w.spawn(func() {
		ctx, cancel := context.WithTimeout(w.ctx, liveSendTimeout)
		defer cancel()
		v := w.gate.Review(ctx, a)
		w.later(func(*Session) { st.done, st.v = true, v })
	})
	return st
}

func (s *Session) staleReason() string {
	if s.Writer == nil {
		return ""
	}
	return s.Writer.gate.Stale()
}

const (
	readOnlyWhy  = "READ-ONLY"
	staleWhy     = "STALE"
	askingWhy    = "ASKING"
	refusedWhy   = "REFUSED"
	readOnlyTipf = "This connection only reads. Connect again with READ ONLY unticked to %s."
)

func (s *Session) liveOrders() []hud.LiveOrder {
	if !s.Live {
		return nil
	}
	specs := s.liveSpecs()
	out := make([]hud.LiveOrder, 0, len(specs))
	stale := s.staleReason()
	for _, sp := range specs {
		o := hud.LiveOrder{Key: sp.key, Label: sp.label, Tip: sp.tip, Icon: sp.icon, Cell: sp.cell, Page: sp.page, Chord: sp.chord, Replaces: sp.replaces}
		switch {
		case sp.key == orderMore:
		case sp.read && s.Writer == nil:
		case s.Writer == nil:
			o.Disabled, o.Why, o.Tip = true, readOnlyWhy, sp.tip+"\n"+fmt.Sprintf(readOnlyTipf, sp.verb)
		case stale != "" && !sp.running:
			o.Disabled, o.Why, o.Tip = true, staleWhy, ui.Bridge("The realm is stale: nothing is sent until the watch on the realm recovers.", "The realm is "+stale+". Nothing is sent until the watch recovers.")
		default:
			st := s.review(sp.access)
			switch {
			case st.done && st.v.Err == nil && !st.v.Allowed:
				reason := st.v.Reason
				if reason == "" {
					reason = "no rule allows " + sp.access.Verb + " on " + accessName(sp.access)
				}
				o.Disabled, o.Why, o.Tip = true, refusedWhy, refusedTip(sp.verb, reason)
			case sp.off != "":
				o.Disabled, o.Why, o.Tip = true, sp.offWhy, sp.off
			case !st.done:
				o.Disabled, o.Why, o.Tip = true, askingWhy, sp.tip+"\n"+ui.Bridge("Asking the Chancery whether these credentials may.", "Asking the API server whether these credentials may (SelfSubjectAccessReview).")
			case st.v.Err != nil:
				o.Tip = clusterSaidOr(sp.tip+"\nThe Chancery could not be asked whether these credentials may: switch NAMES to read why.", sp.tip+"\nThe access check could not be asked: ", st.v.Err.Error())
			}
		}
		out = append(out, o)
	}
	return out
}

func refusedTip(verb, reason string) string {
	if !ui.Naming() {
		return "The Chancery refuses these credentials: they may not " + verb + ". Switch NAMES to read its own words."
	}
	return clusterSaid("The API server refuses these credentials: ", reason)
}

func accessName(a write.Access) string {
	name := a.Resource
	if a.Subresource != "" {
		name += "/" + a.Subresource
	}
	if a.Group != "" {
		name += "." + a.Group
	}
	return name
}

func (s *Session) spec(key string) (orderSpec, bool) {
	for _, sp := range s.liveSpecs() {
		if sp.key == key {
			return sp, true
		}
	}
	return orderSpec{}, false
}

func (s *Session) startOrder(key string) []sound.SFX {
	if !s.Live {
		return nil
	}
	if key == orderMore {
		return nil
	}
	sp, ok := s.spec(key)
	if !ok {
		return []sound.SFX{sound.SFXError}
	}
	for _, o := range s.liveOrders() {
		if o.Key == key && o.Disabled {
			s.Hud.Toast(o.Tip, true)
			return []sound.SFX{sound.SFXError}
		}
	}
	w := s.Writer
	if sp.stone != "" && !(sp.askFirst && w != nil && w.level == write.Seamless) {
		s.runStoneOrder(sp)
		return []sound.SFX{sound.SFXClick}
	}
	if sp.running && w.drain != nil {
		w.seq++
		w.run = &orderRun{id: w.seq, spec: sp, stage: stageDraining}
		s.showOrder()
		return []sound.SFX{sound.SFXClick}
	}
	w.seq++
	run := &orderRun{id: w.seq, spec: sp, stage: stageInput}
	for _, in := range sp.inputs {
		run.values = append(run.values, in.Value)
	}
	w.run = run
	if len(sp.inputs) == 0 && !s.asks(sp) {
		return s.advanceOrder()
	}
	s.showOrder()
	return []sound.SFX{sound.SFXClick}
}

func (s *Session) liveScroll(it hud.Intent) []sound.SFX {
	w := s.Writer
	if w == nil || w.run == nil {
		s.Hud.CloseOrder()
		return nil
	}
	run := w.run
	switch it.Command {
	case hud.ScrollCancel:
		w.run = nil
		s.Hud.CloseOrder()
		return []sound.SFX{sound.SFXClick}
	case hud.ScrollStop:
		if w.drain != nil && w.drain.stop != nil {
			w.drain.stopped = true
			w.drain.stop()
		}
		return []sound.SFX{sound.SFXClick}
	}
	if run.stage == stageInput {
		run.values = strings.Split(it.Target, hud.FieldSep)
	}
	return s.advanceOrder()
}

func (s *Session) advanceOrder() []sound.SFX {
	w := s.Writer
	run := w.run
	switch run.stage {
	case stageInput:
		if run.spec.stone != "" {
			w.run = nil
			s.Hud.CloseOrder()
			s.runStoneOrder(run.spec)
			return []sound.SFX{sound.SFXClick}
		}
		if err := s.buildOrder(run); err != nil {
			run.note, run.bad = err.Error(), true
			s.showOrder()
			return []sound.SFX{sound.SFXError}
		}
		return s.afterBuild()
	case stageClass:
		w.gate.Pass(run.need)
		if w.pass != nil {
			w.pass(run.need)
		}
		run.need = ""
		return s.afterBuild()
	case stagePreview:
		return s.confirmOrder()
	}
	return nil
}

func (s *Session) buildOrder(run *orderRun) error {
	if run.spec.plan != nil {
		d, err := run.spec.plan()
		if err != nil {
			return err
		}
		run.drain = &d
		return nil
	}
	o, err := run.spec.build(run.values)
	if err != nil {
		return err
	}
	run.order = o
	return nil
}

func (run *orderRun) classes() []write.Class {
	if run.drain != nil {
		return run.drain.Classes
	}
	return run.order.Classes
}

func (s *Session) afterBuild() []sound.SFX {
	w := s.Writer
	run := w.run
	for _, c := range run.classes() {
		if !w.gate.Passed(c) {
			run.stage, run.need = stageClass, c
			s.showOrder()
			return []sound.SFX{sound.SFXClick}
		}
	}
	if w.level == write.Seamless {
		return s.sendOrder()
	}
	run.stage = stageAsking
	s.showOrder()
	s.previewOrder(run)
	return []sound.SFX{sound.SFXClick}
}

func (s *Session) previewOrder(run *orderRun) {
	w := s.Writer
	id := run.id
	if run.drain != nil {
		d := *run.drain
		w.spawn(func() {
			ctx, cancel := context.WithTimeout(w.ctx, liveSendTimeout)
			defer cancel()
			var lines []string
			bad, noDry := false, false
			if d.Cordon != nil {
				ans, fields := previewSafely(ctx, w.gate, *d.Cordon)
				lines = append(lines, previewLine("cordon "+d.Node, ans, fields))
				bad, noDry = bad || !ans.OK(), noDry || ans.NoDryRun
			}
			for _, o := range d.Evict {
				ans, _ := previewSafely(ctx, w.gate, o)
				lines = append(lines, previewLine("evict "+o.Target.Namespace+"/"+o.Target.Name, ans, nil))
				noDry = noDry || ans.NoDryRun
			}
			w.later(func(s *Session) { s.previewed(id, nil, lines, bad, noDry, "") })
		})
		return
	}
	o := run.order
	w.spawn(func() {
		ctx, cancel := context.WithTimeout(w.ctx, liveSendTimeout)
		defer cancel()
		ans, fields := previewSafely(ctx, w.gate, o)
		note := ""
		if !ans.OK() {
			note = ans.Text()
		}
		w.later(func(s *Session) { s.previewed(id, fields, nil, !ans.OK() && !ans.NoDryRun, ans.NoDryRun, note) })
	})
}

func previewSafely(ctx context.Context, g *write.Gate, o write.Order) (ans write.Answer, fields []write.Field) {
	defer func() {
		if p := recover(); p != nil {
			ans, fields = write.Answer{Err: fmt.Errorf("the preview failed, so nothing was sent: %v", p)}, nil
		}
	}()
	return g.Preview(ctx, o)
}

func previewLine(what string, ans write.Answer, fields []write.Field) string {
	switch {
	case ans.OK() && len(fields) > 0:
		return what + ": " + fields[0].Label + " " + orNone(fields[0].Before) + " -> " + orNone(fields[0].After)
	case ans.OK():
		return what + ": allowed"
	case ans.Blocked():
		return what + ": blocked now, the drain waits and retries: " + strings.Join(ans.Causes, "; ")
	}
	return clusterSaid(what+": ", ans.Text())
}

func orNone(v string) string {
	if v == "" {
		return "(none)"
	}
	return v
}

const noPreviewStory = "The Chancery cannot preview this: switch NAMES to read why."

func (s *Session) previewed(id int, fields []write.Field, lines []string, bad, noDry bool, note string) {
	w := s.Writer
	run := w.run
	if run == nil || run.id != id || run.stage != stageAsking {
		return
	}
	run.fields, run.change = fields, lines
	switch {
	case noDry && !w.gate.Passed(write.Unpreviewed):
		run.stage, run.need = stageClass, write.Unpreviewed
		run.note, run.bad = clusterSaidOr(noPreviewStory, "", note), true
	case noDry:
		run.stage, run.note, run.bad = stagePreview, clusterSaidOr(noPreviewStory, "", note), true
	case bad:
		run.stage, run.note, run.bad = stagePreview, clusterSaid(ui.Bridge("The Chancery refused the preview: ", "The server refused the dry run: "), note), true
	default:
		run.stage, run.note, run.bad = stagePreview, "", false
	}
	s.showOrder()
}

func (s *Session) confirmOrder() []sound.SFX {
	w := s.Writer
	run := w.run
	if run.drain != nil {
		if s.drainBusy() {
			return []sound.SFX{sound.SFXError}
		}
		w.gate.IssueDrain(*run.drain)
		return s.startDrain(*run.drain)
	}
	return s.sendWith(func(ctx context.Context, o write.Order) write.Answer { return w.gate.SendConfirmed(ctx, o) })
}

func (s *Session) sendOrder() []sound.SFX {
	w := s.Writer
	run := w.run
	if run.drain != nil {
		return s.startDrain(*run.drain)
	}
	return s.sendWith(w.gate.Send)
}

func (s *Session) sendWith(send func(context.Context, write.Order) write.Answer) []sound.SFX {
	w := s.Writer
	run := w.run
	run.stage = stageSending
	if s.asks(run.spec) || len(run.spec.inputs) > 0 || w.level == write.Writes {
		s.showOrder()
	}
	o, id, values := run.order, run.id, run.values
	w.spawn(func() {
		ctx, cancel := context.WithTimeout(w.ctx, liveSendTimeout)
		defer cancel()
		ans := send(ctx, o)
		w.later(func(s *Session) { s.sent(id, o, values, ans) })
	})
	return []sound.SFX{sound.SFXClick}
}

func (s *Session) sent(id int, o write.Order, values []string, ans write.Answer) {
	w := s.Writer
	s.recordWrite(o.Command, o.Verb, o.Target, ans)
	run := w.run
	mine := run != nil && run.id == id
	switch {
	case ans.OK():
		s.Hud.Toast(strings.ToUpper(o.Verb)+": "+o.Object()+" "+ui.Bridge("is done", "answered "+strconv.Itoa(ans.Code)), false)
		if mine {
			w.run = nil
			s.Hud.CloseOrder()
		}
	case ans.Conflict():
		s.Hud.Toast(ui.Bridge("Changed since its card was drawn: nothing was sent again. Read the card and order anew.", "409 Conflict: the object changed since it was drawn; nothing was retried. Read the card and order again."), true)
		if mine {
			w.run = nil
			s.Hud.CloseOrder()
		}
	default:
		if mine {
			run.stage, run.note, run.bad = stagePreview, clusterSaid("", ans.Text()), true
			if w.level == write.Seamless {
				run.stage = stageInput
			}
			s.showOrder()
		}
		s.Hud.Toast(clusterSaid(strings.ToUpper(o.Verb)+": ", ans.Text()), true)
	}
}

type ledgerLine struct {
	at        time.Time
	verb      string
	target    write.Ref
	object    string
	code      int
	answer    string
	earlier   bool
	ended     string
	shownVerb string
}

func (l ledgerLine) text() string {
	obj := l.object
	if l.target.Name != "" {
		obj = objectName(l.target.Kind, &metav1.ObjectMeta{Namespace: l.target.Namespace, Name: l.target.Name})
	}
	verb := verbWords(l.verb)
	if l.shownVerb != "" {
		verb = l.shownVerb
	}
	if l.earlier {
		verb = ui.Bridge("EARLIER", "EARLIER WRITE")
	}
	head := l.at.Format("15:04") + " " + verb + " " + obj + ": "
	if l.ended != "" {
		return head + l.ended
	}
	if words, ok := codeWords(l.code, l.answer); ok {
		return head + words
	}
	return clusterSaid(head, write.Mask(l.answer))
}

var storyVerbs = map[string]string{
	write.OrderEvict: "SEND AWAY", write.OrderSetImage: "SET THE CLASS", write.OrderSetResources: "SET THE PLEDGE", write.OrderRollback: "UNDO",
}

func verbWords(verb string) string {
	kube := strings.ToUpper(verb)
	if story, ok := storyVerbs[verb]; ok {
		return ui.Bridge(story, kube)
	}
	return kube
}

func codeWords(code int, raw string) (string, bool) {
	switch {
	case code >= 200 && code < 300:
		return ui.Bridge("done", "OK"), true
	case code == 409:
		return ui.Bridge("changed since it was drawn", "409 Conflict"), true
	case code == 429:
		return ui.Bridge("the Quorum holds it", "429 "+raw), true
	case code == 404:
		return ui.Bridge("already gone", "404 Not Found"), true
	}
	return "", false
}

func (s *Session) recordWrite(command, verb string, target write.Ref, ans write.Answer) {
	w := s.Writer
	what := verb
	if command != "" {
		what = command
	}
	answer := "OK"
	if !ans.OK() {
		answer = ans.Text()
	}
	e := write.Entry{At: w.now().UTC(), Context: w.gate.Context().Name, Level: w.level.String(), Order: what, Object: target.String(), Code: ans.Code, Answer: answer}
	if err := w.ledger.Append(e); err != nil {
		s.Hud.Toast("The ledger file could not be written: "+err.Error(), true)
	}
	line := ledgerLine{at: e.At, verb: verb, target: target, code: ans.Code, answer: answer}
	w.entries = append(w.entries, line)
	if len(w.entries) > ledgerShown {
		w.entries = w.entries[len(w.entries)-ledgerShown:]
	}
	s.Echoes = Chronicled(s.Echoes, "WRITE "+e.Level+" "+line.text(), s.echoClock())
	if s.Writer.hints && command != "" {
		s.EchoSpoken([]string{command}, hud.FromButton, answer, nil)
	}
}

func (s *Session) writerNow() time.Time {
	if s.Writer != nil && s.Writer.now != nil {
		return s.Writer.now()
	}
	return time.Now()
}

func (s *Session) RecordWrite(e write.Entry) {
	if s.Writer == nil {
		return
	}
	w := s.Writer
	if err := w.ledger.Append(e); err != nil {
		s.Hud.Toast("The ledger file could not be written: "+err.Error(), true)
	}
	w.entries = append(w.entries, ledgerLine{at: e.At, object: write.Mask(e.Object), code: e.Code, answer: e.Answer, earlier: true})
	if len(w.entries) > ledgerShown {
		w.entries = w.entries[len(w.entries)-ledgerShown:]
	}
}

func (s *Session) drainBusy() bool {
	w := s.Writer
	if w.drain == nil || w.drain.over {
		return false
	}
	s.Hud.Toast(oneDrain(w.drain.node), true)
	w.run = nil
	s.Hud.CloseOrder()
	return true
}

func (s *Session) startDrain(d write.Drain) []sound.SFX {
	w := s.Writer
	if s.drainBusy() {
		return []sound.SFX{sound.SFXError}
	}
	ctx, stop := context.WithCancel(w.ctx)
	dr := &drainRun{node: d.Node, stop: stop, plan: d}
	w.drain = dr
	if w.run != nil {
		w.run.stage = stageDraining
	} else {
		w.seq++
		w.run = &orderRun{id: w.seq, stage: stageDraining}
	}
	s.showOrder()
	every := w.every
	if w.sync {
		every = time.Millisecond
	}
	w.spawn(func() {
		err := w.gate.RunDrain(ctx, d, every, func(r write.DrainReport) {
			w.later(func(s *Session) { s.drained(dr, r) })
		})
		w.later(func(s *Session) {
			dr.over = true
			if err != nil && ctx.Err() == nil {
				dr.err = err.Error()
			}
			if ctx.Err() != nil {
				dr.err = "STOPPED: nothing more is evicted"
				if !dr.stopped {
					dr.err = "STOPPED: the realm was left or the window closed, so nothing more is evicted"
				}
			}
			s.recordDrainEnd(dr)
			s.showOrder()
		})
	})
	return []sound.SFX{sound.SFXClick}
}

func (s *Session) drained(dr *drainRun, r write.DrainReport) {
	if r.Cordon != "" {
		ans := write.Answer{Code: r.Code}
		if r.Failed != "" {
			ans.Message = r.Failed
			if ans.Code == 0 {
				ans.Err = errors.New(r.Failed)
			}
		}
		s.recordWrite("", write.OrderCordon, write.Ref{GVR: write.Nodes, Kind: "Node", Name: r.Cordon}, ans)
		s.showOrder()
		return
	}
	ans := write.Answer{Code: 201}
	switch {
	case r.Gone:
		ans = write.Answer{Code: 404, Message: "already gone"}
	case r.Blocked != "":
		ans = write.Answer{Code: 429, Message: r.Blocked}
	case r.Failed != "":
		ans = write.Answer{Code: 409, Message: r.Failed}
	}
	if r.Tries == 1 || r.Blocked == "" {
		ns, name, _ := strings.Cut(r.Hut, "/")
		s.recordWrite("", write.OrderEvict, write.Ref{GVR: write.Pods, Kind: "Pod", Namespace: ns, Name: name}, ans)
	}
	switch {
	case r.Done:
		dr.done++
		dr.blocked, dr.budget = "", ""
	case r.Blocked != "":
		dr.blocked = ui.Bridge(r.Hut+" waits: its Quorum lets no more of its crew leave now", r.Hut+": "+r.Blocked)
		dr.budget = s.budgetLine(r.Hut)
	case r.Failed != "":
		dr.failed = append(dr.failed, drainFailure{hut: r.Hut, why: r.Failed})
	}
	s.showOrder()
}

type drainFailure struct {
	hut, why string
}

func (f drainFailure) shown() string {
	return clusterSaid("FAILED: "+f.hut+": ", f.why)
}

func drainEndWords(dr *drainRun) (string, string) {
	why, story := "over", "over"
	switch {
	case dr.err != "" && strings.HasPrefix(dr.err, "STOPPED") && dr.stopped:
		why, story = "stopped by STOP", "stopped by STOP"
	case dr.err != "" && strings.HasPrefix(dr.err, "STOPPED"):
		why, story = "stopped by leaving the realm or closing the window", "stopped by leaving the realm"
	case dr.err != "":
		why, story = "ended: "+dr.err, "ended"
	}
	ended := fmt.Sprintf("drain %s: %d of %d Pods evicted", why, dr.done, len(dr.plan.Evict))
	shown := fmt.Sprintf(ui.Bridge("DRAIN %s: %d of %d huts sent away", "drain %s: %d of %d Pods evicted"), ui.Bridge(story, why), dr.done, len(dr.plan.Evict))
	if n := len(dr.failed); n > 0 {
		ended += fmt.Sprintf(", %d failed", n)
		shown += fmt.Sprintf(ui.Bridge(", %d could not leave", ", %d failed"), n)
	}
	switch {
	case story != "ended":
	case ui.Naming():
		shown = ui.Verbatim(shown)
	default:
		shown += ", switch NAMES to read why"
	}
	return ended, shown
}

func (s *Session) recordDrainEnd(dr *drainRun) {
	if w := s.Writer; w != nil {
		ended, shown := drainEndWords(dr)
		target := write.Ref{GVR: write.Nodes, Kind: "Node", Name: dr.node}
		e := write.Entry{At: w.now().UTC(), Context: w.gate.Context().Name, Level: w.level.String(), Order: write.DrainCommand(dr.node), Object: target.String(), Answer: ended, Ended: ended}
		if err := w.ledger.Append(e); err != nil {
			s.Hud.Toast("The ledger file could not be written: "+err.Error(), true)
		}
		line := ledgerLine{at: e.At, shownVerb: "DRAIN", target: target, ended: shown}
		w.entries = append(w.entries, line)
		if len(w.entries) > ledgerShown {
			w.entries = w.entries[len(w.entries)-ledgerShown:]
		}
		s.Echoes = Chronicled(s.Echoes, "WRITE "+e.Level+" "+line.text(), s.echoClock())
	}
	switch {
	case dr.err != "":
		s.Hud.Toast("DRAIN "+dr.node+": "+dr.err, true)
	case len(dr.failed) > 0:
		s.Hud.Toast(fmt.Sprintf("DRAIN %s: %d evicted, %d failed", dr.node, dr.done, len(dr.failed)), true)
	default:
		s.Hud.Toast(fmt.Sprintf("DRAIN %s: %d evicted", dr.node, dr.done), false)
	}
}

func (s *Session) showOrder() {
	w := s.Writer
	run := w.run
	if run == nil {
		s.Hud.CloseOrder()
		return
	}
	s.Hud.OpenOrder(s.orderForm(run))
}

func (s *Session) liveCardWrites(card *hud.LiveCard) {
	card.Level = s.liveLevel()
	card.Hints = s.hints
	w := s.Writer
	if w == nil {
		return
	}
	card.Ledger = w.ledger.Path()
	if w.shown != "" {
		card.Ledger = w.shown
	}
	card.Writes = nil
	for _, e := range w.entries {
		card.Writes = append(card.Writes, e.text())
	}
	if dr := w.drain; dr != nil && !dr.over {
		card.Draining = fmt.Sprintf("DRAINING %s: %d OF %d", dr.node, dr.done, len(dr.plan.Evict))
		card.DrainWhy = dr.blocked
		if dr.budget != "" {
			card.DrainWhy += "\n" + dr.budget
		}
	}
}

func (s *Session) writeIntent(it hud.Intent) ([]hud.Intent, []sound.SFX, bool) {
	switch it.Kind {
	case hud.IntentLiveOrder:
		return nil, s.startOrder(it.Target), true
	case hud.IntentLiveScroll:
		return nil, s.liveScroll(it), true
	case hud.IntentLiveForwardStop:
		s.stopForward(it.Target)
		return nil, nil, true
	}
	return nil, nil, false
}

func (s *Session) PrepareLiveCapture(now time.Time, ledger string) {
	w := s.Writer
	if w == nil {
		return
	}
	w.sync, w.shown = true, ledger
	w.now = func() time.Time { return now }
	w.suffix = func() string { return "x7k" }
}

func (s *Session) CaptureClassWarning(c write.Class) {
	w := s.Writer
	if w == nil {
		return
	}
	sp, _ := s.spec("delete")
	w.seq++
	w.run = &orderRun{id: w.seq, spec: orderSpec{key: sp.key, title: sp.title, object: sp.object}, stage: stageClass, need: c}
	s.showOrder()
}

func (s *Session) CaptureAsync() {
	if s.Writer != nil {
		s.Writer.sync = false
	}
}

func (s *Session) CaptureDrainWaiting(limit time.Duration) bool {
	w := s.Writer
	if w == nil {
		return false
	}
	deadline := time.Now().Add(limit)
	for {
		s.stepWriter()
		if dr := w.drain; dr != nil && (dr.blocked != "" || dr.over) {
			return dr.blocked != ""
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Millisecond)
	}
}

func (s *Session) PassForCapture(c write.Class) {
	if s.Writer != nil {
		s.Writer.gate.Pass(c)
	}
}
