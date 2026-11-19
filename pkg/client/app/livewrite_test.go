package app

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	appsv1 "k8s.io/api/apps/v1"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
	"github.com/waxb/kubernites/pkg/sim/store"
	"github.com/waxb/kubernites/pkg/view"
)

type liveRig struct {
	g    *Game
	s    *Session
	rec  *livetest.Record
	gate *write.Gate
	dir  string
	snap live.Snapshot
}

func connectAt(t *testing.T, level string, o livetest.Options, classes ...write.Class) *liveRig {
	t.Helper()
	snap := livetest.Managed()
	if o.User == "" {
		o.User = "admin@example.com"
	}
	srv, rec := livetest.ServeAs(t, snap, o)
	kubeconfig := livetest.Kubeconfig(t, srv.URL)
	dir := t.TempDir()
	g := NewGame(nil, Options{LiveOnly: true, SettingsDir: dir, Live: LiveSource{Files: []string{kubeconfig}}}, nil, nil)
	if len(classes) > 0 {
		g.choices.Keep(write.Remembered{Context: write.Context{Name: livetest.Context, Server: srv.URL, User: "fake-user"}, Level: write.Writes, Warned: write.Seamless, Classes: classes})
		if err := write.SaveMemory(dir, g.choices); err != nil {
			t.Fatal(err)
		}
	}
	rig := &liveRig{g: g, rec: rec, dir: dir, snap: snap}
	rig.title(hud.Intent{Kind: hud.IntentLive})
	rig.title(hud.Intent{Kind: hud.IntentLiveList})
	if len(g.Picker.Contexts) != 1 {
		t.Fatalf("contexts %+v (%s)", g.Picker.Contexts, g.Picker.Err)
	}
	rig.press(t, "live:row:0")
	rig.press(t, "live:choose")
	switch level {
	case hud.LevelWrites, hud.LevelSeamless:
		rig.press(t, hud.LiveBoxID)
		if g.Picker.Step == hud.LiveStepWrites {
			if level == hud.LevelSeamless {
				rig.press(t, "live:seam")
			}
			rig.press(t, "live:allow")
		}
		if g.Picker.ChosenLevel() == hud.LevelWrites && level == hud.LevelSeamless {
			rig.press(t, "live:level")
			if g.Picker.Step == hud.LiveStepSeam {
				rig.press(t, "live:allow")
			}
		}
	}
	if got := g.Picker.ChosenLevel(); got != level {
		t.Fatalf("the picker chose %s, want %s", got, level)
	}
	rig.press(t, "live:confirm")
	for deadline := time.Now().Add(30 * time.Second); g.Scene != SceneRealm; time.Sleep(5 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("the cluster never opened: %q %s", g.Picker.Step, g.Picker.Err)
		}
		g.StepLive()
	}
	rig.s = g.Sess
	if w := rig.s.Writer; w != nil {
		w.sync = true
		w.suffix = func() string { return "x7k" }
		w.now = func() time.Time { return time.Date(2026, 11, 15, 21, 4, 5, 0, time.UTC) }
		rig.gate = w.gate
	}
	return rig
}

func (r *liveRig) title(it hud.Intent) {
	if err := r.g.TitleIntent(it); err != nil {
		panic(err)
	}
}

func (r *liveRig) press(t *testing.T, id string) {
	t.Helper()
	for _, b := range r.g.Picker.Buttons() {
		if b.ID != id {
			continue
		}
		in := hud.Input{MX: b.Rect.X + b.Rect.W/2, MY: b.Rect.Y + b.Rect.H/2, Click: true}
		for _, it := range r.g.Picker.Update(in) {
			r.title(it)
		}
		return
	}
	t.Fatalf("no picker button %s at step %s", id, r.g.Picker.Step)
}

func (r *liveRig) frame() {
	r.s.Advance(0.05)
	r.s.stepWriter()
	r.s.stepWriter()
}

func (r *liveRig) do(it hud.Intent) {
	r.s.Handle(it)
	r.frame()
}

func (r *liveRig) key(in hud.Input) {
	r.frame()
	f := r.s.Frame(true)
	for _, it := range r.s.Hud.Update(in, f) {
		r.s.Handle(it)
	}
	r.frame()
}

func (r *liveRig) selections() []hud.Selection {
	r.frame()
	r.s.Refresh()
	var out []hud.Selection
	for _, c := range r.s.Crews {
		if !c.Bare {
			out = append(out, hud.Selection{Kind: hud.SelectCrew, ID: c.ID()})
		}
	}
	for i, p := range r.s.Pods {
		if i%3 == 0 {
			out = append(out, hud.Selection{Kind: hud.SelectHearth, ID: p.Namespace + "/" + p.Name, Pod: p})
		}
	}
	for _, st := range r.s.Site.Structures {
		if st.Object.Name != "" && (st.Kind == view.StructKeep || st.Object.Kind == "Service") {
			out = append(out, hud.Selection{Kind: "structure", ID: st.ID, Structure: st})
		}
	}
	return out
}

func (r *liveRig) select_(sel hud.Selection) []hud.LiveOrder {
	r.s.Selection = sel
	r.s.Hud.SetLiveOrders(nil)
	r.s.Hud.CloseOrder()
	if r.s.Writer != nil {
		r.s.Writer.run = nil
	}
	r.frame()
	return r.s.Hud.LiveOrders()
}

func (r *liveRig) goAll(order string, steps int) {
	r.do(hud.Intent{Kind: hud.IntentLiveOrder, Target: order})
	for range steps {
		if _, open := r.s.Hud.OrderOpen(); !open {
			return
		}
		r.do(hud.Intent{Kind: hud.IntentLiveScroll, Command: hud.ScrollGo, Target: r.s.Hud.OrderValues()})
	}
}

func (r *liveRig) previewOnly(order string) {
	r.do(hud.Intent{Kind: hud.IntentLiveOrder, Target: order})
	for range 4 {
		form, open := r.s.Hud.OrderOpen()
		if !open || strings.HasPrefix(form.Go, "CONFIRM") || form.Go == "" {
			return
		}
		r.do(hud.Intent{Kind: hud.IntentLiveScroll, Command: hud.ScrollGo, Target: r.s.Hud.OrderValues()})
	}
}

var everyChord = []ebiten.Key{ebiten.KeyD, ebiten.KeyK, ebiten.KeyL, ebiten.KeyT, ebiten.KeyN, ebiten.KeyB, ebiten.KeyO, ebiten.KeyX, ebiten.KeyS, ebiten.KeyE, ebiten.KeyR, ebiten.KeyC, ebiten.KeyU,
	ebiten.KeyF, ebiten.KeyJ, ebiten.KeyV, ebiten.KeyY}

func (r *liveRig) everyKey() {
	for _, k := range everyChord {
		r.key(hud.Input{Keys: []ebiten.Key{k}})
		r.key(hud.Input{Ctrl: true, Keys: []ebiten.Key{k}})
		r.key(hud.Input{Ctrl: true, Control: true, Keys: []ebiten.Key{k}})
		r.s.Hud.CloseOrder()
		if r.s.Writer != nil {
			r.s.Writer.run = nil
		}
	}
	for k := ebiten.Key1; k <= ebiten.Key9; k++ {
		r.key(hud.Input{Keys: []ebiten.Key{k}})
	}
}

func changes(lines []string) []string {
	var out []string
	for _, l := range lines {
		if strings.HasPrefix(l, "GET ") || l == livetest.SelfReview || l == livetest.AccessReview {
			continue
		}
		out = append(out, l)
	}
	return out
}

func TestAtReadOnlyEveryOrderAndKeySendsOnlyGetsAndTheSelfReviews(t *testing.T) {
	rig := connectAt(t, hud.LevelReadOnly, livetest.Options{})
	if rig.s.Writer != nil {
		t.Fatal("a READ ONLY connection has a writer")
	}
	if g, ok := rig.s.Realm.(gated); ok && g.Gate() != nil {
		t.Fatal("a READ ONLY connection has a write gate")
	}
	orders := 0
	for _, sel := range rig.selections() {
		for _, o := range rig.select_(sel) {
			orders++
			sp, _ := rig.s.spec(o.Key)
			if sp.read && o.Disabled {
				t.Errorf("%s at READ ONLY is a read, yet greyed: %q", o.Key, o.Why)
			}
			if o.Key != orderMore && !sp.read && (!o.Disabled || o.Why != readOnlyWhy) {
				t.Errorf("%s at READ ONLY: disabled %v why %q", o.Key, o.Disabled, o.Why)
			}
			rig.goAll(o.Key, 3)
		}
		rig.everyKey()
	}
	if orders < 20 {
		t.Fatalf("only %d orders were tried", orders)
	}
	for _, l := range rig.rec.Seen() {
		if !strings.HasPrefix(l, "GET ") && l != livetest.SelfReview {
			t.Errorf("READ ONLY sent %s", l)
		}
	}
	if rig.s.Hud.LiveCard().Level != hud.LevelReadOnly {
		t.Fatalf("level on the card %q", rig.s.Hud.LiveCard().Level)
	}
	for _, it := range []hud.Intent{{Kind: hud.IntentHints}, {Kind: hud.IntentNames}, {Kind: hud.IntentLiveChoices, Command: "forget-all"}, {Kind: hud.IntentHints}, {Kind: hud.IntentNames}} {
		if err := rig.g.RealmIntent(it); err != nil {
			t.Fatal(err)
		}
		rig.frame()
	}
	rig.everyKey()
	if g, ok := rig.s.Realm.(gated); rig.s.Writer != nil || ok && g.Gate() != nil || rig.s.Hud.LiveCard().Level != hud.LevelReadOnly {
		t.Fatal("a READ ONLY connection changed in place")
	}
	for _, l := range rig.rec.Seen() {
		if !strings.HasPrefix(l, "GET ") && l != livetest.SelfReview {
			t.Errorf("READ ONLY sent %s after the settings changed", l)
		}
	}
}

func TestAtWritesWithNothingConfirmedOnlyAccessReviewsAndDryRunsLeave(t *testing.T) {
	rig := connectAt(t, hud.LevelWrites, livetest.Options{}, write.AllClasses...)
	if rig.gate == nil || rig.gate.Level() != write.Writes {
		t.Fatal("no WRITES gate")
	}
	orders := 0
	for _, sel := range rig.selections() {
		for _, o := range rig.select_(sel) {
			if o.Disabled {
				continue
			}
			orders++
			rig.previewOnly(o.Key)
			if form, open := rig.s.Hud.OrderOpen(); open && !strings.HasPrefix(form.Go, "CONFIRM") && !strings.Contains(form.Note, "dry run") && o.Key != orderMore {
				t.Errorf("%s at WRITES did not reach its preview: %+v", o.Key, form)
			}
		}
		rig.everyKey()
	}
	if orders < 20 {
		t.Fatalf("only %d orders were enabled", orders)
	}
	reviews := 0
	for _, l := range rig.rec.Full() {
		switch {
		case strings.HasPrefix(l, "GET "), l == livetest.SelfReview:
		case l == livetest.AccessReview:
			reviews++
		case strings.Contains(l, "dryRun=All"):
		default:
			t.Errorf("WRITES with nothing confirmed sent %s", l)
		}
	}
	if reviews == 0 {
		t.Fatal("no access review was asked")
	}
}

func TestAtSeamlessEachOrderSendsExactlyItsOneRequest(t *testing.T) {
	rig := connectAt(t, hud.LevelSeamless, livetest.Options{}, write.AllClasses...)
	seen := 0
	for _, sel := range rig.selections() {
		for _, o := range rig.select_(sel) {
			if sp, _ := rig.s.spec(o.Key); o.Disabled || o.Key == "drain" || o.Key == orderMore || sp.stone != "" {
				continue
			}
			before := len(changes(rig.rec.Full()))
			rig.goAll(o.Key, 2)
			after := changes(rig.rec.Full())
			if len(after)-before != 1 {
				t.Errorf("%v %s sent %v", sel.ID+sel.Pod.Name+sel.Structure.Object.Name, o.Key, after[before:])
			}
			if len(after) > before && strings.Contains(after[len(after)-1], "dryRun") {
				t.Errorf("SEAMLESS previewed %s", o.Key)
			}
			seen++
		}
	}
	if seen < 20 {
		t.Fatalf("only %d orders", seen)
	}
}

func TestWriteKeysAreControlChordsOnTheSelectedObjectOnly(t *testing.T) {
	rig := connectAt(t, hud.LevelSeamless, livetest.Options{}, write.AllClasses...)
	var hut hud.Selection
	for _, sel := range rig.selections() {
		if sel.Kind == hud.SelectHearth {
			hut = sel
			break
		}
	}
	rig.select_(hut)
	for _, in := range []hud.Input{
		{Keys: []ebiten.Key{ebiten.KeyK}}, {Keys: []ebiten.Key{ebiten.KeyD}}, {Ctrl: true, Keys: []ebiten.Key{ebiten.KeyK}},
		{Shift: true, Keys: []ebiten.Key{ebiten.KeyK}}, {Alt: true, Keys: []ebiten.Key{ebiten.KeyK}}, {Keys: []ebiten.Key{ebiten.Key2}},
	} {
		rig.key(in)
		if c := changes(rig.rec.Full()); len(c) != 0 {
			t.Fatalf("%+v sent %v", in, c)
		}
	}
	rig.key(hud.Input{Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyD}})
	if _, open := rig.s.Hud.OrderOpen(); !open {
		t.Fatal("ctrl-d at SEAMLESS did not ask, as k9s does")
	}
	if c := changes(rig.rec.Full()); len(c) != 0 {
		t.Fatalf("ctrl-d sent %v before its answer", c)
	}
	rig.s.Hud.CloseOrder()
	rig.s.Writer.run = nil
	rig.key(hud.Input{Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyK}})
	c := changes(rig.rec.Full())
	if len(c) != 1 || !strings.HasPrefix(c[0], "DELETE /api/v1/namespaces/"+hut.Pod.Namespace+"/pods/"+hut.Pod.Name) {
		t.Fatalf("ctrl-k sent %v", c)
	}
}

func TestNoWriteKeyFiresWhileTheStoneHoldsTheKeyboard(t *testing.T) {
	rig := connectAt(t, hud.LevelSeamless, livetest.Options{}, write.AllClasses...)
	for _, sel := range rig.selections() {
		if sel.Kind == hud.SelectHearth {
			rig.select_(sel)
			break
		}
	}
	rig.s.Hud.OpenStone("")
	rig.s.Hud.FocusStone()
	rig.key(hud.Input{Ctrl: true, Control: true, Keys: []ebiten.Key{ebiten.KeyK}, Chars: []rune{'k'}})
	if c := changes(rig.rec.Full()); len(c) != 0 {
		t.Fatalf("a chord typed into the Stone sent %v", c)
	}
}

func TestTheCameraDoesNotPanWhileControlIsHeld(t *testing.T) {
	held := HeldKey
	defer func() { HeldKey = held }()
	HeldKey = func(k ebiten.Key) bool { return k == ebiten.KeyD || k == ebiten.KeyControl }
	if dx, dy := PanVector(1); dx != 0 || dy != 0 {
		t.Fatalf("ctrl-d panned %v,%v", dx, dy)
	}
	HeldKey = func(k ebiten.Key) bool { return k == ebiten.KeyD }
	if dx, _ := PanVector(1); dx <= 0 {
		t.Fatal("D alone no longer pans")
	}
}

func TestAStaleRealmGreysEveryOrderAndSendsNothing(t *testing.T) {
	rig := connectAt(t, hud.LevelSeamless, livetest.Options{}, write.AllClasses...)
	rig.gate.StaleWhen(func() string { return "stale since 21:00:00 UTC: pods: watch failed" })
	for _, sel := range rig.selections() {
		for _, o := range rig.select_(sel) {
			if o.Key == orderMore {
				continue
			}
			if !o.Disabled || o.Why != staleWhy {
				t.Errorf("%s while stale: disabled %v why %q", o.Key, o.Disabled, o.Why)
			}
			rig.goAll(o.Key, 2)
		}
		rig.everyKey()
	}
	if c := changes(rig.rec.Full()); len(c) != 0 {
		t.Fatalf("a stale realm sent %v", c)
	}
}

func TestAConflictReadsTheCardAgainAndIsNeverRetried(t *testing.T) {
	rig := connectAt(t, hud.LevelSeamless, livetest.Options{Answer: func(r *http.Request, _ []byte) (int, string, bool) {
		if r.Method == http.MethodPatch {
			return http.StatusConflict, `{"kind":"Status","code":409,"reason":"Conflict","message":"the object has been modified"}`, true
		}
		return 0, "", false
	}}, write.AllClasses...)
	d := rig.snap.Deployments[0]
	rig.select_(hud.Selection{Kind: hud.SelectCrew, ID: d.Namespace + "/" + d.Name})
	rig.goAll("restart", 2)
	c := changes(rig.rec.Full())
	if len(c) != 1 || !strings.HasPrefix(c[0], "PATCH ") {
		t.Fatalf("the conflicting restart sent %v", c)
	}
	if _, open := rig.s.Hud.OrderOpen(); open {
		t.Fatal("the order stayed open after a conflict")
	}
	toasts := rig.s.Hud.Toasts()
	if len(toasts) == 0 || !strings.Contains(toasts[len(toasts)-1].Text, "409") && !strings.Contains(toasts[len(toasts)-1].Text, "Changed since") {
		t.Fatalf("toasts %+v", toasts)
	}
}

func TestEveryWriteGoesToTheLedgerFileTheLiveCardNames(t *testing.T) {
	rig := connectAt(t, hud.LevelSeamless, livetest.Options{}, write.AllClasses...)
	d := rig.snap.Deployments[0]
	rig.select_(hud.Selection{Kind: hud.SelectCrew, ID: d.Namespace + "/" + d.Name})
	rig.goAll("restart", 2)
	card := rig.s.Hud.LiveCard()
	if card.Ledger == "" || filepath.Dir(card.Ledger) != rig.dir || len(card.Writes) != 1 {
		t.Fatalf("card ledger %q writes %v", card.Ledger, card.Writes)
	}
	data, err := os.ReadFile(card.Ledger)
	if err != nil || strings.Count(string(data), "\n") != 1 || !strings.Contains(string(data), "rollout restart") {
		t.Fatalf("ledger file %q %v", data, err)
	}
	found := false
	for _, e := range rig.s.Echoes {
		if strings.Contains(e.Text, "WRITE ") && strings.Contains(e.Text, "RESTART") {
			found = true
		}
	}
	if !found {
		t.Fatal("the write is not in the Stone's history")
	}
}

func TestReadOnlyRBACGreysEveryOrderWithTheServersReason(t *testing.T) {
	rig := connectAt(t, hud.LevelWrites, livetest.Options{Deny: func(map[string]any) string {
		return `RBAC: user "viewer" cannot change anything`
	}}, write.AllClasses...)
	n := 0
	for _, sel := range rig.selections() {
		for _, o := range rig.select_(sel) {
			if o.Key == orderMore {
				continue
			}
			n++
			if !o.Disabled || o.Why != refusedWhy || !strings.Contains(o.Tip, `cannot change anything`) {
				t.Errorf("%s: disabled %v why %q tip %q", o.Key, o.Disabled, o.Why, o.Tip)
			}
		}
	}
	if n == 0 {
		t.Fatal("no orders")
	}
}

func TestAClassWarningIsAskedOncePerContextAndRemembered(t *testing.T) {
	rig := connectAt(t, hud.LevelSeamless, livetest.Options{})
	var hut hud.Selection
	for _, sel := range rig.selections() {
		if sel.Kind == hud.SelectHearth {
			hut = sel
			break
		}
	}
	rig.select_(hut)
	rig.do(hud.Intent{Kind: hud.IntentLiveOrder, Target: "delete"})
	rig.s.Hud.OpenOrder(func() hud.OrderForm { f, _ := rig.s.Hud.OrderOpen(); f.Inputs[1].Value = "on"; return f }())
	rig.do(hud.Intent{Kind: hud.IntentLiveScroll, Command: hud.ScrollGo, Target: rig.s.Hud.OrderValues()})
	form, open := rig.s.Hud.OrderOpen()
	if !open || !form.Warning || !strings.Contains(strings.ToLower(form.Title), "destroying") {
		t.Fatalf("a force delete did not warn: %+v", form)
	}
	if c := changes(rig.rec.Full()); len(c) != 0 {
		t.Fatalf("sent before the warning: %v", c)
	}
	rig.do(hud.Intent{Kind: hud.IntentLiveScroll, Command: hud.ScrollGo})
	if c := changes(rig.rec.Full()); len(c) != 1 {
		t.Fatalf("after the warning: %v", c)
	}
	mem, err := write.LoadMemory(rig.dir)
	if err != nil {
		t.Fatal(err)
	}
	r, _, ok := mem.Recall(rig.gate.Context())
	if !ok || !slices.Contains(r.Classes, write.Destroy) || r.Level != write.Seamless {
		t.Fatalf("remembered %+v %v", r, ok)
	}
}

func TestADrainStopsWhenTold(t *testing.T) {
	rig := connectAt(t, hud.LevelSeamless, livetest.Options{Answer: func(r *http.Request, _ []byte) (int, string, bool) {
		if strings.HasSuffix(r.URL.Path, "/eviction") {
			return http.StatusTooManyRequests, `{"kind":"Status","code":429,"message":"Cannot evict pod as it would violate the pod's disruption budget.","details":{"causes":[{"reason":"DisruptionBudget","message":"The disruption budget web needs 1 healthy pods and has 1 currently"}]}}`, true
		}
		return 0, "", false
	}}, write.AllClasses...)
	rig.s.Writer.sync = false
	node := rig.snap.Nodes[len(rig.snap.Nodes)-1].Name
	i := slices.IndexFunc(rig.s.Site.Structures, func(st view.Structure) bool {
		return st.Kind == view.StructKeep && st.Object.Name == node
	})
	if i < 0 {
		t.Fatalf("no Keep for node %s", node)
	}
	keep := rig.s.Site.Structures[i]
	rig.select_(hud.Selection{Kind: "structure", ID: keep.ID, Structure: keep})
	for i := 0; i < 400 && slices.ContainsFunc(rig.s.Hud.LiveOrders(), func(o hud.LiveOrder) bool { return o.Why == askingWhy }); i++ {
		time.Sleep(5 * time.Millisecond)
		rig.frame()
	}
	rig.goAll("drain", 1)
	deadline := time.Now().Add(10 * time.Second)
	for rig.s.Writer.drain == nil || rig.s.Writer.drain.blocked == "" {
		if time.Now().After(deadline) {
			t.Fatalf("the drain never waited on its budget: %+v", rig.s.Writer.drain)
		}
		rig.frame()
		time.Sleep(5 * time.Millisecond)
	}
	if card := rig.s.Hud.LiveCard(); !strings.HasPrefix(card.Draining, "DRAINING") || !strings.Contains(card.DrainWhy, "disruption budget web") {
		t.Fatalf("the card does not show the blocking budget: %q %q", card.Draining, card.DrainWhy)
	}
	rig.do(hud.Intent{Kind: hud.IntentLiveScroll, Command: hud.ScrollStop})
	for !rig.s.Writer.drain.over {
		if time.Now().After(deadline) {
			t.Fatal("STOP did not end the drain")
		}
		rig.frame()
		time.Sleep(5 * time.Millisecond)
	}
	n := len(changes(rig.rec.Full()))
	time.Sleep(50 * time.Millisecond)
	if len(changes(rig.rec.Full())) != n {
		t.Fatal("the drain sent more after STOP")
	}
}

func TestReadonlyForcesTheBoxTickedWhateverWasRemembered(t *testing.T) {
	g := NewGame(nil, Options{LiveOnly: true, ReadOnly: true, SettingsDir: t.TempDir()}, nil, nil)
	g.Picker.Contexts = []hud.LiveContext{{Name: "c", Server: "https://s", User: "u"}}
	g.choices.Keep(write.Remembered{Context: write.Context{Name: "c", Server: "https://s", User: "u"}, Level: write.Seamless, Warned: write.Seamless})
	g.saveChoices()
	g.loadChoices()
	g.Picker.Open(nil)
	g.Picker.Contexts, g.Picker.Chosen, g.Picker.Step = []hud.LiveContext{{Name: "c", Server: "https://s", User: "u"}}, 0, hud.LiveStepConfirm
	for _, b := range g.Picker.Buttons() {
		if b.ID == hud.LiveBoxID && (!b.Disabled || b.Label != "[x] READ ONLY") {
			t.Fatalf("box %+v", b)
		}
	}
	if gate, err := g.gateFor("c", hud.LevelSeamless); gate != nil || err != nil {
		t.Fatalf("--readonly built a gate: %v %v", gate, err)
	}
}

func TestARemovedHutCarriesTheVersionItWasDrawnFrom(t *testing.T) {
	rig := connectAt(t, hud.LevelSeamless, livetest.Options{}, write.AllClasses...)
	p := rig.snap.Pods[0]
	rig.select_(hud.Selection{Kind: hud.SelectHearth, Pod: model.PodView{Namespace: p.Namespace, Name: p.Name}})
	spec, ok := rig.s.spec("kill")
	if !ok {
		t.Fatal("no kill")
	}
	o, err := spec.build(nil)
	if err != nil || !strings.Contains(string(o.Request.Body), `"uid":"`+string(p.UID)+`"`) || !strings.Contains(string(o.Request.Body), `"resourceVersion":"`+p.ResourceVersion+`"`) {
		t.Fatalf("kill %s %v", o.Request.Body, err)
	}
	if _, ok := rig.s.Cache.Get(store.KindDeployment, rig.snap.Deployments[0].Namespace, rig.snap.Deployments[0].Name); !ok {
		t.Fatal("no deployment in the cache")
	}
	_ = appsv1.Deployment{}
	_ = context.Background()
}

func TestAtSeamlessADrainAsksFirstAsK9sDoesAndTheWarningNamesEveryOneChordWrite(t *testing.T) {
	rig := connectAt(t, hud.LevelSeamless, livetest.Options{}, write.AllClasses...)
	warning := strings.Join(hud.SeamlessWarningLines(), " ")
	drained := false
	for _, sel := range rig.selections() {
		for _, o := range rig.select_(sel) {
			sp, _ := rig.s.spec(o.Key)
			if o.Disabled || o.Chord == "" || sp.stone != "" || sp.read {
				continue
			}
			oneChord := !rig.s.asks(sp) && len(sp.inputs) == 0
			if oneChord && !strings.Contains(warning, "Ctrl-"+o.Chord) {
				t.Errorf("%s (Ctrl-%s) changes the cluster from one chord and the SEAMLESS warning does not name it", o.Key, o.Chord)
			}
			if o.Key != "drain" || drained {
				continue
			}
			drained = true
			if oneChord {
				t.Errorf("DRAIN fires from one chord; k9s asks first")
			}
			before := len(changes(rig.rec.Full()))
			rig.do(hud.Intent{Kind: hud.IntentLiveOrder, Target: o.Key})
			if after := changes(rig.rec.Full()); len(after) != before {
				t.Fatalf("Ctrl-X sent %v before any answer", after[before:])
			}
			form, open := rig.s.Hud.OrderOpen()
			if !open || form.Go != "DRAIN [Y]" {
				t.Fatalf("the drain asks %+v", form)
			}
		}
	}
	if !drained {
		t.Fatal("no Keep to drain")
	}
	for _, order := range []string{"a restart", "a rollback", "a trigger", "a cordon", "a delete", "a drain"} {
		if !strings.Contains(warning, order) {
			t.Errorf("the SEAMLESS warning does not say %s asks first", order)
		}
	}
}
