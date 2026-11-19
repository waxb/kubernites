package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestATroubleCardSetAsideByAStepDoneWaitsWhileTheStepDoneIsUp(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetAttention(true)
	h.SetTaught([]string{TeachTopBar})
	f := calamityFrame(t, noDepot)
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Progress: "ready: press it", Target: "cmd:door"}}
	f.Guide.Target = "cmd:door"
	key := ActTrouble + noDepot.ID
	waitUntil(t, h, &f, paceLimit, "the trouble card", func(Frame) bool { return h.Director.Stage.Slot.Key == key })
	tick(h, &f, 10)
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Done: true},
		{ID: "build-codex", Text: "Found the Codex", Progress: "ready: press BUILD", Build: "codex"}}
	f.State.Enjoying = true
	f.Guide.Target = "build:codex"
	waitUntil(t, h, &f, paceLimit, "the STEP DONE", shownStep("door-hearth", true))
	if !h.Aside(key) {
		t.Fatalf("the STEP DONE did not set the trouble card aside: %s", h.StageDebug())
	}
	for waited := 0.0; waited < 240; waited += .25 {
		f.Clock += .25
		f.EventClock = f.Clock
		h.Update(Input{MX: -10, MY: -10}, f)
		if !shownStep("door-hearth", true)(h.Stage(f)) {
			t.Fatalf("the STEP DONE left unanswered after %.1f s: %s", waited, h.StageDebug())
		}
		if h.Director.Stage.Has(key) {
			t.Fatalf("the trouble card came back %.1f s after the STEP DONE set it aside, with the STEP DONE still up: %s", waited, h.StageDebug())
		}
	}
	b, ok := nextStepOn(h, h.Stage(f))
	if !ok {
		t.Fatal("the done step offers no NEXT STEP")
	}
	h.Update(Input{MX: b.Rect.X + 1, MY: b.Rect.Y + 1, Click: true}, f)
	f.State.Enjoying = false
	waitUntil(t, h, &f, paceLimit, "the trouble card back", func(Frame) bool { return h.Director.Stage.Slot.Key == key })
}
