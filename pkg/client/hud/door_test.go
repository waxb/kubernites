package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func doorFrame(t *testing.T) Frame {
	f := fullFrame(t)
	f.State.Reveals = []string{"serve", "gold", "build", "door"}
	f.Pods = nil
	return f
}

func hasButton(buttons []Button, id string) (Button, bool) {
	for _, b := range buttons {
		if b.ID == id {
			return b, true
		}
	}
	return Button{}, false
}

func TestDoorHearthCommandAppearsUntilTheReeveRunsIt(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := doorFrame(t)
	b, ok := hasButton(h.Buttons(f), "cmd:door")
	if !ok {
		t.Fatal("no PORTER AT THE DOOR once the door is revealed")
	}
	if b.Label != "PORTER AT THE DOOR" || !strings.Contains(b.Tip, "static Pod") || !strings.Contains(b.Tip, "no gold") {
		t.Errorf("door button = %+v", b)
	}
	intents := clickOn(t, h, f, "cmd:door")
	it, ok := hasIntent(intents, IntentAction)
	if !ok || it.Action != (view.Action{Action: "doorHearth"}) {
		t.Fatalf("door intents = %+v", intents)
	}
	f.Pods = []model.PodView{{Namespace: "default", Name: "porter-lone-tower", Node: view.TowerName, Static: true, State: model.HearthReady}}
	if _, ok := hasButton(h.Buttons(f), "cmd:door"); ok {
		t.Error("the door button stays after the door hut burns")
	}
	f.Pods = []model.PodView{{Namespace: "default", Name: "porter-x", Node: view.TowerName, State: model.HearthReady}}
	if _, ok := hasButton(h.Buttons(f), "cmd:door"); !ok {
		t.Error("a declared hut is not the door hut")
	}
	f.State.Reveals = []string{"serve", "gold", "build"}
	if _, ok := hasButton(h.Buttons(f), "cmd:door"); ok {
		t.Error("the door button shows before its reveal")
	}
}

func TestTheDoorCardBringsTheGridBackToItsButton(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	h.SetStaging(false)
	f := doorFrame(t)
	f.Guide = Guide{Staged: true, Introduced: []string{"build", "cmd:build"}}
	h.Update(Input{}, f)
	h.PageID = PageBuild
	f.Guide.Intro = &Intro{ID: "door", Title: "Porter at the Door", Try: "Press " + view.ButtonToken + ".", Target: "cmd:door", Via: ViaAct}
	h.Update(Input{}, f)
	if h.PageID != PageMain {
		t.Fatalf("the door card came while the grid stayed on %q", h.PageID)
	}
	if _, ok := hasButton(h.Buttons(f), "cmd:door"); !ok {
		t.Fatal("the door card asks for a button that is not on screen")
	}
	f.Guide.Intro = &Intro{ID: "codex", Title: "The Codex", Target: "build:codex", Via: ViaAct}
	h.PageID = PageBuild
	h.Update(Input{}, f)
	if h.PageID != PageBuild {
		t.Fatal("a card for a building tile closed the build page")
	}
}

func TestDoorButtonFindsAFreeCellBesideTheOthers(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := doorFrame(t)
	f.State.Reveals = append(f.State.Reveals, "summon", "berths", "chronicles", "tidings", "stone")
	buttons := h.Buttons(f)
	door, ok := hasButton(buttons, "cmd:door")
	if !ok {
		t.Fatal("no free cell for the door")
	}
	for _, b := range buttons {
		if b.ID != door.ID && b.Rect == door.Rect {
			t.Errorf("door shares a cell with %s", b.ID)
		}
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
}

func TestBeaconPointsAtTheDoorWhenItIsTheObjective(t *testing.T) {
	f := doorFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "door-hearth", Text: "Light a hut at the door.", Target: "cmd:door"}}
	f.Guide.Target = "cmd:door"
	if got := BeaconButton(f, PageMain); got != "cmd:door" {
		t.Errorf("beacon = %q, want cmd:door", got)
	}
}

func TestSummonCardNamesTheHandsCost(t *testing.T) {
	f := fullFrame(t)
	f.Site.Structures = append(f.Site.Structures, view.Structure{ID: "keep-1", Kind: view.StructKeep, Variant: "tower", Label: "Lone Tower",
		State: view.StateReady, Object: view.ObjectRef{Kind: "Node", Name: "lone-tower"}})
	lines := strings.Join(SummonLines(f), " | ")
	for _, want := range []string{"A hut costs no gold: its worker pledges a Keep's food (CPU).", "The Lone Tower has 600 rations free."} {
		if !strings.Contains(lines, want) {
			t.Errorf("summon card %q lacks %q", lines, want)
		}
	}
	f.Metrics.Nodes = nil
	if strings.Contains(strings.Join(SummonLines(f), " "), "free") {
		t.Error("free food shown without metrics")
	}
	h := New(nil, (&fakeRunner{}).backend())
	h.PageID = PageSummon
	for _, b := range h.Buttons(f) {
		if strings.HasPrefix(b.ID, "summon:") && !strings.Contains(b.Tip, "food") {
			t.Errorf("summon tip %q does not name the food", b.Tip)
		}
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.Draw(dst, f)
	rows := 1
	for _, l := range SummonLines(fullFrame(t)) {
		wrapped := h.Font.Wrap(l, CardBox.W-16, 1)
		if len(wrapped) > 2 {
			t.Errorf("summon line wraps %d times: %q", len(wrapped), l)
		}
		rows += len(wrapped)
	}
	if rows*11+8 > CardBox.H {
		t.Errorf("summon card needs %d rows", rows)
	}
}

func TestStaticPodReadsAsTheHearthAtTheDoor(t *testing.T) {
	p := model.PodView{Namespace: "default", Name: "porter-lone-tower", Node: "lone-tower", Static: true, Profession: "Porter", Sprite: "porter", State: model.HearthReady, Image: "porter:1.0"}
	c := PodCard(p, view.PodMetrics{})
	text := c.Title + " " + c.K8s
	for _, l := range c.Lines {
		text += " " + l.Text
	}
	for _, want := range []string{"HUT AT THE DOOR", "STATIC POD", "RUN BY ITS KEEP"} {
		if !strings.Contains(strings.ToUpper(text), want) {
			t.Errorf("card %q lacks %q", text, want)
		}
	}
	sel := Selection{Kind: "hearth", ID: "default/porter-lone-tower", Pod: p}
	if got := SelectionTitle(sel); !strings.Contains(got, "Hut at the door (static Pod)") {
		t.Errorf("details title = %q", got)
	}
	h := New(nil, (&fakeRunner{}).backend())
	h.OpenDetails(sel, DetailDescribe)
	joined := strings.Join(h.Details().Lines, " ")
	if !strings.Contains(joined, "Keep runs it") || !strings.Contains(joined, "manifests") {
		t.Errorf("details do not say who runs it: %q", joined)
	}
}
