package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestWelcomeExplainsTheRealmAndCloses(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.State.Road.Ages = []view.AgeStep{{ID: view.AgeFeudal, Asks: "99% ready (9 in 10 fresh), a port for caravels, 5000 MiB, 65 points and an incident report"}}
	text := strings.Join(WelcomeStory, " ") + " " + WelcomeGoal(f)
	for _, want := range []string{"request", "GOAL", "9 in 10", "caravels", "5000 MiB", "65 points", "incident", "carry", "spoils"} {
		if !strings.Contains(text, want) {
			t.Errorf("welcome story lacks %q", want)
		}
	}
	if lines := (ui.Font{}).WrapLines(WelcomeGoal(f), WelcomeBox.W-2*WelcomeInset, 1, 3); len(lines) > 2 {
		t.Errorf("the welcome's goal takes %d lines: %q", len(lines), lines)
	}
	legend := map[string]bool{}
	for _, row := range WelcomeLegend {
		legend[row[0]] = true
	}
	for _, want := range []string{"Box", "Keep", "Hut", "Speaking Stone"} {
		if !legend[want] {
			t.Errorf("legend lacks %s", want)
		}
	}
	h.Open(OverlayWelcome)
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
	clickOn(t, h, f, "welcome:begin")
	if h.Overlay() != OverlayNone {
		t.Fatal("BEGIN did not close the welcome")
	}
	h.Open(OverlayWelcome)
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, f)
	if h.Overlay() != OverlayNone {
		t.Fatal("Enter did not close the welcome")
	}
}

func objectiveFrame(t *testing.T, o view.ObjectiveView) Frame {
	f := fullFrame(t)
	f.State.Objectives = []view.ObjectiveView{o}
	f.Guide.Target = o.Target
	return f
}

func TestObjectiveShowsProgressAndTheWholeHint(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	hint := "Requests queue at the Gate. Click one, or press SERVE: +1 MiB. Leave them waiting and they walk away."
	f := objectiveFrame(t, view.ObjectiveView{ID: "serve-hand", Text: "Serve 5 requests by hand.", Hint: hint, Progress: "2/5 served"})
	var shown []string
	progress := false
	for _, l := range h.ObjectiveLines(f) {
		shown = append(shown, l.Text)
		progress = progress || l.Text == "2/5 served"
	}
	if !progress {
		t.Errorf("progress missing: %q", shown)
	}
	if joined := strings.Join(shown, " "); !strings.Contains(joined, "they walk away.") || strings.Contains(joined, "...") {
		t.Errorf("hint cut short: %q", shown)
	}
	cmd := "kubectl get nodes"
	f.State.Objectives[0].Command = cmd
	r, chip := h.ObjectiveRectOf(f), ChipRect(h.ObjectiveRectOf(f), cmd)
	if chip.Y+chip.H > r.Y+r.H || chip.Y < r.Y+ObjectiveTop+len(h.ObjectiveLines(f))*9 {
		t.Errorf("chip %+v does not sit under the text in %+v", chip, r)
	}
}

func TestBeaconPointsAtTheNextThingToPress(t *testing.T) {
	cases := []struct {
		o    view.ObjectiveView
		gate bool
		page string
		want string
	}{
		{view.ObjectiveView{ID: "serve-hand", Target: view.ParcelPile}, true, PageMain, ""},
		{view.ObjectiveView{ID: "raise-keep", Progress: "gold 3/10"}, true, PageMain, ""},
		{view.ObjectiveView{ID: "raise-keep", Progress: "ready: press BUILD"}, true, PageMain, "cmd:build"},
		{view.ObjectiveView{ID: "raise-keep", Progress: "ready: press BUILD"}, true, PageBuild, "build:keep"},
		{view.ObjectiveView{ID: "build-stewards", Progress: "being built"}, true, PageMain, ""},
		{view.ObjectiveView{ID: "first-hearth", Target: "cmd:summon"}, true, PageMain, "cmd:summon"},
		{view.ObjectiveView{ID: "door-hearth", Progress: "the Reeve wakes the porter"}, false, PageMain, ""},
		{view.ObjectiveView{ID: "one-nine"}, false, PageMain, ""},
	}
	for _, tc := range cases {
		f := objectiveFrame(t, tc.o)
		crewCarries(&f, !tc.gate)
		if got := BeaconButton(f, tc.page); got != tc.want {
			t.Errorf("%s (%s, gate %v) beacon = %q, want %q", tc.o.ID, tc.o.Progress, tc.gate, got, tc.want)
		}
	}
	f := objectiveFrame(t, view.ObjectiveView{ID: "serve-hand", Target: view.ParcelPile})
	crewCarries(&f, false)
	if ObjectiveGoal(f) != view.ParcelPile || !MapTarget(ObjectiveGoal(f)) {
		t.Errorf("the opening goal is the pile on the map: %q", ObjectiveGoal(f))
	}
	gold := objectiveFrame(t, view.ObjectiveView{ID: "raise-keep", Progress: "gold 3/10"})
	crewCarries(&gold, false)
	if ObjectiveGoal(gold) != view.ParcelPile {
		t.Errorf("short of memory, the guide points at the pile: %q", ObjectiveGoal(gold))
	}
	if !ServeBeckons(f) {
		t.Error("the Gate should beckon while the first parcels wait")
	}
	f.State.Freight.Kinds[0].Waiting = 0
	if ServeBeckons(f) {
		t.Error("an empty queue has nobody to point at")
	}
}

func TestGateCardWhileYouServeByHand(t *testing.T) {
	f := fullFrame(t)
	crewCarries(&f, false)
	f.State.Freight.Kinds[0].Waiting = 3
	f.State.Pilgrims.TotalServed, f.State.Pilgrims.TotalDropped = 12, 4
	text := strings.Join(GateLines(f), " | ")
	for _, want := range []string{"3 parcels wait", "delivered 12", "spoiled 4", "GOAL"} {
		if !strings.Contains(text, want) {
			t.Errorf("gate card %q lacks %q", text, want)
		}
	}
	for _, l := range GateLines(f) {
		if len(l) > cardChars {
			t.Errorf("gate line too long for the card: %q", l)
		}
	}
	if share, ok := ServedShare(f); !ok || share != .75 {
		t.Errorf("served share = %v %v", share, ok)
	}
	f.State.Pilgrims.TotalServed, f.State.Pilgrims.TotalDropped = 0, 0
	if _, ok := ServedShare(f); ok {
		t.Error("a share before any request came")
	}
	h := New(nil, (&fakeRunner{}).backend())
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
}

func TestLessonCardShowsWhatHappenedUntilGotIt(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	h.Lesson("Serve 5 requests by hand.", "You answered every request yourself: each one needed your hands. That toil is what Kubernetes takes off you.")
	lines := h.LessonLines()
	if len(lines) < 3 || !strings.HasPrefix(lines[1].Text, "DONE: Serve 5") {
		t.Fatalf("hint lines = %+v", lines)
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
	f.Clock += 60
	h.Update(Input{}, f)
	if len(h.LessonLines()) == 0 {
		t.Fatal("the hint faded before the player answered it")
	}
	clickOn(t, h, f, LessonAck)
	if len(h.LessonLines()) != 0 {
		t.Fatal("GOT IT left the hint up")
	}
}
