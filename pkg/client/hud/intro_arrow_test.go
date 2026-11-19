package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const arrowWatch = 30.0

func nextStepWithItsCard(f Frame) Frame {
	next := f
	next.State.Objectives = []view.ObjectiveView{
		{ID: "raise-keep", Text: "Raise a Keep", Done: true, Build: "keep"},
		{ID: "build-codex", Text: "Inspect the Codex", Progress: "ready: press BUILD", Build: "codex"},
	}
	next.Guide.Target = "build:codex"
	next.Guide.Intro = &Intro{ID: "codex", Title: "The Codex", Purpose: "Records every order.", Try: "Open BUILD, then pick the Codex.", Target: "build:codex", Via: ViaAct}
	return next
}

func TestAStepsOwnCardOpensAtItsArrowFromItsFirstFrame(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := directorFrame(t)
	tick(h, &f, 1)
	f = nextStepWithItsCard(f)
	var first ui.Rect
	seen := false
	for waited := 0.0; waited < arrowWatch; waited += .1 {
		tick(h, &f, .1)
		s := h.Stage(f)
		if !h.IntroShown(s) {
			continue
		}
		card := h.IntroCard(s)
		if _, ok := h.IntroHole(s); !ok {
			t.Fatalf("at %.1f s the card is drawn at %+v with no arrow (step held for %q)", waited, card, s.Guide.Held)
		}
		if !seen {
			first, seen = card, true
			continue
		}
		if card != first {
			t.Fatalf("at %.1f s the card jumped from %+v to %+v", waited, first, card)
		}
	}
	if !seen {
		t.Fatal("the step's own card never opened")
	}
}
