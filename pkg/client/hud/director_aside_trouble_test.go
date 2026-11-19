package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestATroubleCardSetAsideByAStepDoneComesBackOnceTheStepIsAnswered(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetAttention(true)
	h.SetTaught([]string{TeachTopBar})
	f := calamityFrame(t, noDepot)
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Progress: "ready: press it", Target: "cmd:door"}}
	f.Guide.Target = "cmd:door"
	key := ActTrouble + noDepot.ID
	onStage := func(Frame) bool { return h.Director.Stage.Slot.Key == key }
	waitUntil(t, h, &f, paceLimit, "the trouble card", onStage)
	tick(h, &f, 10)
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Set a porter at the door", Done: true},
		{ID: "build-codex", Text: "Found the Codex", Progress: "ready: press BUILD", Build: "codex"}}
	f.State.Enjoying = true
	f.Guide.Target = "build:codex"
	waitUntil(t, h, &f, paceLimit, "the STEP DONE", shownStep("door-hearth", true))
	if !h.Aside(key) {
		t.Fatalf("the STEP DONE did not set the trouble card aside: %s", h.StageDebug())
	}
	b, ok := nextStepOn(h, h.Stage(f))
	if !ok {
		t.Fatal("the done step offers no NEXT STEP")
	}
	h.Update(Input{MX: b.Rect.X + 1, MY: b.Rect.Y + 1, Click: true}, f)
	f.State.Enjoying = false
	waitUntil(t, h, &f, paceLimit, "the trouble card back", onStage)
}
