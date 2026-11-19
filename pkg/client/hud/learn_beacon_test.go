package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestAReadyLearnStepLeadsThroughTheGrimoireToItsPage(t *testing.T) {
	f := navFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: view.BadgeObjective, Text: "Let the depot look for a badge: the Stewards keep its roll",
		Target: "cmd:grimoire", Page: "marks", Progress: "ready: read about Badges in the MANUAL"}}
	f.Guide.Target = "cmd:grimoire"
	h := New(nil, (&fakeRunner{}).backend())
	if path := NavPath(h.GuideGoal(f), h.NavState(f)); len(path) == 0 || path[0] != "cmd:grimoire" {
		t.Fatalf("with the manual shut the guide walks %v, want cmd:grimoire first", path)
	}
	h.OverlayID = OverlayGrimoire
	if path := NavPath(h.GuideGoal(f), h.NavState(f)); len(path) == 0 || path[0] != "grim:marks" {
		t.Fatalf("inside the manual the guide walks %v, want the Badges page", path)
	}
}

func TestAPagesOpenCardTakesTheGuideStraightToItsLearnButton(t *testing.T) {
	f := navFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: view.BadgeObjective, Text: "Let the depot look for a badge: the Stewards keep its roll",
		Target: "cmd:grimoire", Page: "muster", Progress: "ready: read about Muster in the MANUAL"}}
	f.Guide.Target = "cmd:grimoire"
	h := New(nil, (&fakeRunner{}).backend())
	h.OverlayID = OverlayGrimoire
	plaque, ok := h.Knowledge(f).Find("muster")
	if !ok {
		t.Fatal("no Muster plaque in the manual")
	}
	r, seen := clipRect(h.ScreenOf(plaque.Rect()), KnowledgeView)
	if !seen {
		t.Fatal("the page plaque is out of view")
	}
	h.Update(Input{MX: r.X + r.W/2, MY: r.Y + r.H/2}, f)
	if h.focusedPage(h.Knowledge(f)) != "muster" {
		t.Fatalf("pointing at the page plaque shows the card of %q", h.focusedPage(h.Knowledge(f)))
	}
	if path := NavPath(h.GuideGoal(f), h.NavState(f)); len(path) == 0 || path[0] != "learn:muster" {
		t.Fatalf("with the page card open the guide walks %v, want its LEARN button", path)
	}
}
