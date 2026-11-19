package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var (
	noDepot = view.CalamityView{ID: "no-depot:ledger", Kind: view.CalamityNoDepot, Object: "kind/ledger",
		Cause: "Ledgers spoil on the quay: 0.3 a second are lost",
		Why:   "No ledgers depot takes them: nothing routes ledgers to a crew (the Service ledgers does not exist).",
		Remedies: []view.RemedyView{
			{Label: "Raise the clerks", Command: "kubectl create deployment clerks --image=clerk:1.0 --replicas=2",
				Action: &view.Action{Action: "raiseCrew", Name: "clerks", Image: "clerk:1.0", Replicas: 2}},
			{Label: "Open the ledgers depot", Command: "kubectl expose deployment clerks --name=ledgers --port=80 --target-port=8080",
				Action: &view.Action{Action: "openDepot", Kind: "ledger"}},
		}}
	oom = view.CalamityView{ID: "oomkilled:default/haulers-1", Kind: view.TagOOMKilled, Object: "pod/default/haulers-1",
		Cause: "OOMKilled: a large box needs about 104Mi in hand, over its 64Mi limit",
		Why:   "A worker holds the box it carries in memory; bigger boxes need more than its limit allows.",
		Remedies: []view.RemedyView{{Label: "Give the haulers more memory", Command: "kubectl set resources deployment haulers --limits=memory=192Mi"},
			{Label: "Retire a Keep", Action: &view.Action{Action: "buyKeep"}}}}
)

func calamityFrame(t *testing.T, list ...view.CalamityView) Frame {
	t.Helper()
	f := fullFrame(t)
	f.State.Reveals = append(append([]string(nil), allReveals...), "stone")
	f.State.Calamities = list
	f.State.Objectives = []view.ObjectiveView{{ID: "o1", Text: "Carry boxes", Hint: "Wait"}}
	return f
}

const settleLimit = 240.0

func settled(h *HUD, f Frame) Frame {
	h.Update(Input{MX: -5, MY: -5}, f)
	f.Clock += CalamitySettle + 1
	h.Update(Input{MX: -5, MY: -5}, f)
	for range f.State.Calamities[min(1, len(f.State.Calamities)):] {
		f.Clock += CalamityGap
		h.Update(Input{MX: -5, MY: -5}, f)
	}
	return h.SettleStage(Input{MX: -5, MY: -5}, f, settleLimit)
}

func TestCalamityStateSettlesRemembersAndForgets(t *testing.T) {
	var st CalamityState
	st = st.Step([]view.CalamityView{noDepot}, 10, CalamityGate{})
	if len(st.Ready([]view.CalamityView{noDepot}, 12)) != 0 {
		t.Error("a calamity must last a moment before its card shows")
	}
	st = st.Step([]view.CalamityView{noDepot, oom}, 15, CalamityGate{})
	ready := st.Ready([]view.CalamityView{noDepot, oom}, 16)
	if len(ready) != 1 || ready[0].ID != noDepot.ID {
		t.Errorf("ready = %+v", ready)
	}
	st = st.Dismiss(noDepot.ID).Step([]view.CalamityView{noDepot, oom}, 15+CalamitySettle, CalamityGate{})
	if ready := st.Ready([]view.CalamityView{noDepot, oom}, 15+CalamitySettle); len(ready) != 1 || ready[0].ID != oom.ID {
		t.Errorf("a dismissed calamity stays away and the next opens once settled: %+v", ready)
	}
	later := 15 + CalamitySettle
	st = st.Step([]view.CalamityView{oom}, later+1, CalamityGate{}).Step([]view.CalamityView{oom}, later+1+DismissMemory, CalamityGate{}).Step([]view.CalamityView{noDepot, oom}, later+2+DismissMemory, CalamityGate{})
	st = st.Step([]view.CalamityView{noDepot, oom}, later+2+DismissMemory+CalamitySettle, CalamityGate{})
	if len(st.Ready([]view.CalamityView{noDepot, oom}, later+2+DismissMemory+CalamitySettle)) != 2 {
		t.Error("a calamity that clears for its window and strikes again returns")
	}
}

func TestCalamityCardShowsOneFixAtATime(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := calamityFrame(t, noDepot, oom)
	if _, ok := h.CalamityRect(f); ok {
		t.Fatal("no card before it settles")
	}
	f = settled(h, f)
	r, ok := h.CalamityRect(f)
	if !ok {
		t.Fatal("the card shows once settled")
	}
	buttons := h.CalamityButtons(f)
	for _, id := range []string{CalamityClose, CalamityNext, CalamityShow, "line:cal:0:speak", "line:cal:0:do", "line:cal:1:speak", "line:cal:1:do"} {
		b, found := HitID(buttons, id)
		if id == CalamityShow {
			if found {
				t.Error("no place, no SHOW")
			}
			continue
		}
		if !found || !contains(r, b.Rect) {
			t.Errorf("%s missing or outside the card %+v: %+v", id, r, b.Rect)
		}
	}
	for i, a := range buttons {
		for _, b := range buttons[i+1:] {
			if Overlaps(a.Rect, b.Rect) {
				t.Errorf("%s overlaps %s", a.ID, b.ID)
			}
		}
	}
}

func TestCalamityRemediesSpeakActAndPage(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := settled(h, calamityFrame(t, noDepot, oom))
	clickOn(t, h, f, "line:cal:1:speak")
	if h.Overlay() != OverlayStone || string(h.Stone.Input) != "expose deployment clerks --name=ledgers --port=80 --target-port=8080" {
		t.Fatalf("remedy chip pre-fills the Stone: %q", string(h.Stone.Input))
	}
	h.OverlayID = OverlayNone
	intents := clickOn(t, h, f, "line:cal:0:do")
	if it, ok := hasIntent(intents, IntentAction); !ok || it.Action.Action != "raiseCrew" {
		t.Errorf("DO IT raises the clerks: %+v", intents)
	}
	clickOn(t, h, f, CalamityNext)
	if v, i, n, _ := h.CurrentCalamity(f); v.ID != oom.ID || i != 1 || n != 2 {
		t.Errorf("NEXT shows the second card: %s %d/%d", v.ID, i, n)
	}
	if _, ok := HitID(h.CalamityButtons(f), "line:cal:1:do"); !ok {
		t.Error("a remedy with only an action is a button")
	}
	clickOn(t, h, f, CalamityClose)
	for range 4 {
		f.Clock++
		h.Update(Input{MX: -5, MY: -5}, f)
	}
	if v, _, n, _ := h.CurrentCalamity(f); n != 1 || v.ID != noDepot.ID {
		t.Errorf("X dismisses the card shown: %s %d", v.ID, n)
	}
}

func TestCalamityCardWaitsForCalm(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := settled(h, calamityFrame(t, noDepot))
	f.Guide = Guide{Staged: true, Intro: &Intro{ID: "gate", Target: SurfaceGate, Via: ViaAck}}
	if _, ok := h.CalamityRect(f); ok {
		t.Error("an intro speaks alone")
	}
	f.Guide = Guide{Staged: true, Introduced: []string{SurfaceTidings}}
	if _, ok := h.CalamityRect(f); ok {
		t.Error("cards wait until the rail is taught")
	}
	f.Guide = Guide{}
	f.State.Objectives[0].Command = "kubectl create deployment clerks --image=clerk:1.0 --replicas=2"
	if _, ok := h.CalamityRect(f); ok {
		t.Error("the step already says it: one voice")
	}
	f.State.Objectives[0].Command = ""
	h.Open(OverlayGrimoire)
	if _, ok := h.CalamityRect(f); ok {
		t.Error("windows hide the card")
	}
}

func TestCalamityCardShowsWhere(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := settled(h, calamityFrame(t, oom))
	f.Places = map[string][2]float64{oom.Object: {12, 14}}
	intents := clickOn(t, h, f, CalamityShow)
	if it, ok := hasIntent(intents, IntentFocus); !ok || it.X != 12 || it.Y != 14 {
		t.Errorf("SHOW glides to the worker: %+v", intents)
	}
	if New(nil, (&fakeRunner{}).backend()).PressCalamity("next", Frame{}) != nil {
		t.Error("no card, nothing to press")
	}
}

func TestCalamityCardStaysClearOfCardsRailAndTargets(t *testing.T) {
	notices := []Notice{{ID: "a", Kind: NoticeDanger, Icon: "icon/alert"}, {ID: "b", Kind: NoticeDecision, Icon: "icon/tech"}, {ID: "c", Kind: NoticeNews, Icon: "icon/chronicle"}}
	cases := []struct {
		name    string
		beacons []ui.Rect
		lesson  bool
	}{
		{"calm", nil, false},
		{"beacon bottom right", []ui.Rect{{X: 700, Y: 330, W: 40, H: 60}}, false},
		{"beacons in both bottom corners", []ui.Rect{{X: 700, Y: 330, W: 40, H: 60}, {X: 40, Y: 340, W: 40, H: 50}}, true},
	}
	for _, c := range cases {
		h := New(nil, (&fakeRunner{}).backend())
		f := settled(h, calamityFrame(t, noDepot))
		f.Notices, f.Beacons = notices, c.beacons
		if c.lesson {
			h.Lesson("Raise a Guildhall", "A Guildhall spreads the boxes over every ready worker.")
		}
		r, ok := h.CalamityRect(f)
		if !ok {
			t.Fatalf("%s: no card", c.name)
		}
		if r.X < 0 || r.Y < TopH || r.X+r.W > Width || r.Y+r.H > ConsoleY {
			t.Errorf("%s: card %+v leaves the map", c.name, r)
		}
		blocked := append(append([]ui.Rect{h.RailRect(f), h.ObjectiveRectOf(f)}, c.beacons...), FreightRects(f)...)
		if c.lesson {
			blocked = append(blocked, h.LessonRect(f))
		}
		for _, b := range blocked {
			if b.W > 0 && Overlaps(r, b) {
				t.Errorf("%s: card %+v covers %+v", c.name, r, b)
			}
		}
		if !h.Captures(r.X+4, r.Y+4, f) {
			t.Errorf("%s: the card captures clicks", c.name)
		}
		dst := ebiten.NewImage(Width, Height)
		h.Draw(dst, f)
		dst.Deallocate()
	}
}

func TestCauseTipsExplainTheTags(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.Causes = []CauseTip{{Rect: ui.Rect{X: 300, Y: 200, W: 60, H: 11}, Text: "OOMKilled: a large box"}, {Rect: ui.Rect{X: 300, Y: 500, W: 60, H: 11}, Text: "under the console"}}
	buttons := h.CauseButtons(f)
	if len(buttons) != 1 || buttons[0].Tip != "OOMKilled: a large box" {
		t.Fatalf("cause buttons = %+v", buttons)
	}
	h.Update(Input{MX: 310, MY: 204}, f)
	if !h.Hovering || h.Hover.Tip != "OOMKilled: a large box" {
		t.Error("hovering a tag shows why")
	}
	if len(h.Press(buttons[0].ID, f)) != 0 {
		t.Error("a tag click falls through to the world")
	}
}

func TestCalamityCardAndSurveyStepAroundTheFreightBoard(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := settled(h, calamityFrame(t, noDepot))
	f.State.Reveals = append(f.State.Reveals, "freight", "losses")
	f.State.Freight.Kinds = []view.KindView{{Kind: "parcel", Open: true, Waiting: 4}, {Kind: "ledger", Open: true}}
	f.Beacons = []ui.Rect{{X: 600, Y: 300, W: 300, H: 100}, {X: 10, Y: 300, W: 300, H: 100}}
	board := FreightRects(f)
	if len(board) == 0 {
		t.Fatal("the board shows")
	}
	r, ok := h.CalamityRect(f)
	for _, b := range board {
		if ok && Overlaps(r, b) {
			t.Errorf("card %+v covers the freight board %+v", r, b)
		}
	}
	p := keepSurvey(true)
	p.At = [2]int{480, 60}
	f.Survey = &p
	if s, ok := h.SurveyRect(f); ok {
		for _, b := range board {
			if Overlaps(s, b) {
				t.Errorf("survey %+v covers the freight board %+v", s, b)
			}
		}
	}
}

func TestACalamityWithNoFixStillOffersAButton(t *testing.T) {
	crash := view.CalamityView{ID: "crashloop:default/porter-door", Kind: view.TagCrashLoop, Object: "pod/default/porter-door",
		Cause: "The door porter keeps falling and rising again", Remedies: []view.RemedyView{}}
	h := New(nil, (&fakeRunner{}).backend())
	f := settled(h, calamityFrame(t, crash))
	r, ok := h.CalamityRect(f)
	if !ok {
		t.Fatal("no card")
	}
	b, found := HitID(h.CalamityButtons(f), CalamityOK)
	if !found || !contains(r, b.Rect) || b.Label == "" {
		t.Fatalf("the fixless card offers %+v inside %+v", b, r)
	}
	h.Update(Input{MX: b.Rect.X + 2, MY: b.Rect.Y + 2, Click: true}, f)
	f.Clock += CalamityGap
	h.Update(Input{MX: -5, MY: -5}, f)
	if _, ok := h.CalamityRect(f); ok {
		t.Fatal("OK did not dismiss the card")
	}
	withFix := settled(New(nil, (&fakeRunner{}).backend()), calamityFrame(t, noDepot))
	if _, found := HitID(New(nil, (&fakeRunner{}).backend()).CalamityButtons(withFix), CalamityOK); found {
		t.Fatal("a card with a fix still shows OK")
	}
}

func TestABlockedRemedyHandsTheStoneNoCommand(t *testing.T) {
	a := &view.Action{Action: view.ActCrewResources, Name: "clerks"}
	l := CommandLine{Label: "Pledge the housing the clerks really take: 448 rooms", Command: "kubectl set resources deployment clerks --requests=memory=448Mi", Action: a,
		Blocked: "make room first: the roll-out needs room for 4 more huts at once"}
	for _, b := range LineButtons("line:x", ui.Rect{W: 300, H: 16}, l) {
		if strings.HasSuffix(b.ID, ":"+SpeakOp) {
			t.Errorf("a blocked remedy still writes %q on the Stone", b.Label)
		}
		if strings.HasSuffix(b.ID, ":"+DoOp) && !b.Disabled {
			t.Error("a blocked remedy's DO IT is pressable")
		}
	}
}
