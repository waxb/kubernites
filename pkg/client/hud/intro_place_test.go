package hud

import (
	"slices"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestAFoundingCardStepsAsideWhileItsBuildingIsPlaced(t *testing.T) {
	h := New(nil, Backend{})
	f := introFrame(t, Intro{ID: "codex", Title: "The Codex", Purpose: "Stores the cluster state.", Try: "Open BUILD, then pick the Codex.", Target: "build:codex", Via: ViaAct})
	if !h.IntroShown(f) {
		t.Fatal("the Codex card did not show before the Codex was picked")
	}
	f.Placing = "keep"
	if !h.IntroShown(f) {
		t.Fatal("placing another building hid the Codex card")
	}
	f.Placing = "codex"
	if h.IntroShown(f) || len(h.GuideTargets(f)) != 0 || len(h.Voices(f)) != 0 {
		t.Fatalf("the Codex card, its shade or its pointer stayed while the Codex was placed: voices %v", h.Voices(f))
	}
	f.Placing = ""
	if !h.IntroShown(f) {
		t.Fatal("cancelling the placement did not bring the Codex card back")
	}
}

func TestTheKeepsCardShowsTheBuildButtonItPointsThrough(t *testing.T) {
	h := New(nil, Backend{})
	f := stagedFrame(t, "cmd:serve", MeterGold)
	f.Guide.Intro = &Intro{ID: "build", Title: "Build", Try: "Press BUILD, then the Keep.", Target: "build:keep", Via: ViaAct}
	if !f.Shown("cmd:build") {
		t.Fatal("the Keep's card hides the BUILD button it points through")
	}
	if _, ok := HitID(h.Buttons(f), "cmd:build"); !ok {
		t.Fatal("no BUILD button in the command box while the Keep's card is up")
	}
}

func TestGetPodsIsTheFirstKubectlCommandAHutOffers(t *testing.T) {
	acts := []CardAction{{Key: KeyDescribe, Command: "describe pod x"}, {Key: KeyLogs, Command: "logs x"}, {Key: keyGetPods, Command: "get pods"}}
	f := stagedFrame(t, "cmd:build")
	f.State.Objectives = []view.ObjectiveView{{ID: "speak-get"}}
	for _, a := range taughtActions(acts, f) {
		if a.Key == KeyDescribe || a.Key == KeyLogs {
			t.Fatalf("%s is offered before kubectl get pods was asked", a.Key)
		}
	}
	f.State.Objectives = []view.ObjectiveView{{ID: "speak-get", Done: true}}
	if got := taughtActions(acts, f); len(got) != len(acts) {
		t.Fatalf("after get pods the hut offers %+v", got)
	}
}

func TestThePorterCellWarnsGentlyWhenNoKeepHasRoom(t *testing.T) {
	f := stagedFrame(t, "cmd:summon")
	b := summonCell(Button{ID: "summon:porter:1.0", Cost: "1.0", Tip: "Summon a porter"}, f)
	if b.Cost != "1.0" {
		t.Fatalf("with room the porter cell reads %q", b.Cost)
	}
	f.State.Summon.Waiting = 2
	b = summonCell(Button{ID: "summon:porter:1.0", Cost: "1.0", Tip: "Summon a porter"}, f)
	if b.Cost != summonNoRoom || b.Disabled || !strings.HasPrefix(b.Tip, summonNoRoomTip) {
		t.Fatalf("with porters waiting the cell reads %+v", b)
	}
}

func TestResearchIsNamedNotPointedAt(t *testing.T) {
	h := New(nil, Backend{})
	f := stagedFrame(t, "cmd:build", "cmd:grimoire")
	f.State.Objectives = []view.ObjectiveView{{ID: "augury", Text: "Read the Pod usage: kubectl top pods", Progress: "ready: read about Augury in the MANUAL", Target: "cmd:grimoire", Page: "augury"}}
	f.Guide.Target = "cmd:grimoire"
	if _, ok := h.objectiveSpot(f); ok {
		t.Fatal("a research step dims the screen and points at the book")
	}
	f.Selection = Selection{Kind: "structure", ID: "keep", Structure: view.Structure{Kind: view.StructKeep}}
	if _, ok := h.objectiveSpot(f); ok {
		t.Fatalf("with a card open, a research step points at the way to the book (step %q)", h.StepID(f))
	}
}

func TestTheResearchPageIsHighlightedInsideTheBook(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "augury", Text: "Read the Pod usage", Progress: "ready: read about Pledges in the MANUAL", Target: "cmd:grimoire", Page: "pledges"}}
	if _, ok := h.researchPage(f); ok {
		t.Fatal("the page is highlighted while the book is closed")
	}
	clickOn(t, h, f, "cmd:grimoire")
	r, ok := h.researchPage(f)
	page, found := HitID(h.GrimoireButtons(f), "grim:pledges")
	if !ok || !found || r != page.Rect {
		t.Fatalf("inside the book the research page is not highlighted: %+v (page %+v)", r, page.Rect)
	}
}

func TestAStepIsOneBoxItsPanelWithARingOnItsTargetAndNoGuideCard(t *testing.T) {
	h := New(nil, Backend{})
	f := stagedFrame(t, "cmd:serve", "cmd:build", MeterGold)
	f.State.Objectives = []view.ObjectiveView{{ID: "build-quartermaster", Text: "Raise the Quartermaster: someone must place every new worker", Progress: "ready: press BUILD", Build: "quartermaster", Target: "cmd:build"}}
	if _, ok := h.objectiveSpot(f); ok {
		t.Fatal("the step brings a guide card beside its panel: two boxes saying the same thing")
	}
	build, found := HitID(h.Buttons(f), "cmd:build")
	if !found {
		t.Fatal("the BUILD cell is not on the grid")
	}
	if !slices.Contains(h.guideBeacons(f), build.Rect) {
		t.Fatalf("the step's target %+v has no ring: %v", build.Rect, h.guideBeacons(f))
	}
}
