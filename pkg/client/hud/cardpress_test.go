package hud

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestPressingACardButtonRunsAndEchoesIt(t *testing.T) {
	f := selectStructure(staged(fullFrame(t)), stoneKeep())
	runner := &fakeRunner{}
	h := New(nil, runner.backend())
	intents := clickOn(t, h, f, "act:cordon")
	if runner.commands[len(runner.commands)-1] != "cordon stone-keep-2" {
		t.Fatalf("ran %v", runner.commands)
	}
	if echo, ok := hasIntent(intents, IntentEcho); !ok || echo.Command != "cordon stone-keep-2" || echo.Tab != "ok" {
		t.Fatalf("echo = %+v", intents)
	}
	found := false
	for _, l := range h.StoneLines() {
		found = found || l.Text == "> kubectl cordon stone-keep-2  (from the Keep card)"
	}
	if !found {
		t.Fatalf("the console never shows the command: %+v", h.StoneLines())
	}
	clickOn(t, h, f, "act:describe")
	if h.Overlay() != OverlayDetails || h.Details().Lines[0] != "$ kubectl describe node stone-keep-2" || h.Details().Tab != DetailDescribe {
		t.Fatalf("describe = %+v", h.Details())
	}
	if h.PressAction("nope", f) != nil || h.PressAction(KeyDrain, f) != nil {
		t.Error("an unknown or disabled action does nothing")
	}
}

func TestChoosersOpenPickAndCloseWithTheSelection(t *testing.T) {
	f := selectStructure(staged(fullFrame(t)), stoneKeep())
	f.Nodes = map[string]model.NodeView{"stone-keep-2": {TaintSpecs: []string{"ward=true:NoSchedule"}}}
	runner := &fakeRunner{}
	h := New(nil, runner.backend())
	clickOn(t, h, f, "act:taint")
	if h.Overlay() != OverlayChooser || len(h.Ops.Chooser.Options) != 4 || !strings.Contains(h.Ops.Chooser.title, "TAINT") {
		t.Fatalf("chooser = %+v", h.Ops.Chooser)
	}
	box := h.OverlayRect()
	if box.Y+box.H > CardBox.Y || box.Y < 0 || !h.Captures(box.X+2, box.Y+2, f) {
		t.Fatalf("chooser box %+v", box)
	}
	if h.pressPick("0", f) != nil {
		t.Error("a Ward that stands cannot be picked")
	}
	intents := clickOn(t, h, f, "pick:3")
	if runner.commands[len(runner.commands)-1] != "taint node stone-keep-2 ward:NoSchedule-" || h.Overlay() != OverlayNone {
		t.Fatalf("pick ran %v, overlay %q", runner.commands, h.Overlay())
	}
	if echo, ok := hasIntent(intents, IntentEcho); !ok || !strings.HasPrefix(echo.Command, "taint node") {
		t.Errorf("pick echo = %+v", intents)
	}
	clickOn(t, h, f, "act:label")
	other := f
	other.Selection = Selection{Kind: "hearth", ID: "default/porter-a", Pod: f.Pods[0]}
	h.Update(Input{}, other)
	if h.Overlay() != OverlayNone || h.pressPick("0", other) != nil {
		t.Error("a new selection closes the chooser")
	}
}

func TestGuideWalksFromTheBuildingToItsButton(t *testing.T) {
	f := fullFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "speak-get", Target: "structure:chancery", Button: "act:get-pods", Command: "kubectl get pods"}}
	f.Guide.Target, f.Guide.Button = "structure:chancery", "act:get-pods"
	if ObjectiveGoal(f) != "structure:chancery" {
		t.Fatalf("before selection the goal is the building: %q", ObjectiveGoal(f))
	}
	f = selectStructure(f, view.Structure{ID: "chancery-1", Kind: view.StructChancery, State: view.StateReady})
	if ObjectiveGoal(f) != "act:get-pods" {
		t.Fatalf("with the Chancery selected the goal is its button: %q", ObjectiveGoal(f))
	}
	h := New(nil, (&fakeRunner{}).backend())
	step, ok := h.Step(f)
	if !ok || step.ID != "act:get-pods" || !inside(step.Rect, Command) {
		t.Fatalf("step = %+v %v", step, ok)
	}
	for _, b := range h.CommandButtons(f) {
		if b.Selected != (b.ID == "act:get-pods") {
			t.Errorf("%s selected = %v", b.ID, b.Selected)
		}
	}
	f = selectStructure(f, stoneKeep())
	if ObjectiveGoal(f) != "structure:chancery" {
		t.Error("another building keeps the goal on the Chancery")
	}
}

func TestGuideReachesTheCrewCardButtonFromAHut(t *testing.T) {
	f := fullFrame(t)
	f.Guide.Target, f.Guide.Button = "pod:porter-a", "crew:scale:up"
	f.Selection = Selection{Kind: "hearth", ID: "default/porter-a", Pod: f.Pods[0]}
	if CardGoal(f) != "crew:scale:up" {
		t.Fatalf("goal = %q", CardGoal(f))
	}
	f.Guide.Button = "act:previous"
	f.Selection.Pod.Restarts = 0
	if CardGoal(f) != "pod:porter-a" {
		t.Error("a disabled card button leaves the goal on the hut")
	}
	if x := chooserX(nil); x != CardBox.X+cardLineX {
		t.Errorf("chooser x = %d", x)
	}
	if x := chooserX([]ui.Rect{{X: CardBox.X, W: 404}}); x != CardBox.X+408 {
		t.Errorf("the chooser steps beside the crew card: %d", x)
	}
	if x := chooserX([]ui.Rect{{X: 800, W: 150}}); x+chooserW > Width {
		t.Errorf("the chooser stays on screen: %d", x)
	}
}

func TestSelectionMatchesMapTargets(t *testing.T) {
	keep := Selection{Kind: "structure", Structure: stoneKeep()}
	pod := Selection{Kind: "hearth", Pod: model.PodView{Name: "porter-lone-tower"}}
	cases := []struct {
		sel    Selection
		target string
		want   bool
	}{
		{keep, "keep:stone-keep-2", true}, {keep, "structure:keep-2", true}, {keep, "structure:keep", true},
		{keep, "keep:lone-tower", false}, {pod, "pod:default/porter-lone-tower", true}, {pod, "pod:porter-lone-tower", true},
		{pod, "structure:keep", false}, {keep, "cmd:depot", false},
		{Selection{Kind: "crew", ID: "default/porters"}, "crew:default/porters", true},
		{Selection{Kind: "hearth", Pod: model.PodView{Name: "porters-a", Crew: "porters"}}, "crew:default/porters", true},
		{pod, "crew:default/porters", false},
	}
	for _, c := range cases {
		if got := SelectionMatches(c.sel, c.target); got != c.want {
			t.Errorf("SelectionMatches(%s) = %v", c.target, got)
		}
	}
}

func TestDepotCardShowsItsServiceType(t *testing.T) {
	if got := ServiceTypeLine("parcels"); got != "DOOR 80 -> WORKER DOOR 8080" {
		t.Fatalf("story type line = %q", got)
	}
	ui.SetNamer(func(s string) string { return s })
	defer ui.SetNamer(nil)
	if got := ServiceTypeLine("parcels"); got != "ClusterIP  PORT 80 -> 8080" {
		t.Fatalf("NAMES type line = %q", got)
	}
	ui.SetNamer(nil)
	if ServiceTypeLine("kubernetes") != "" {
		t.Error("no type line for a Service that is no depot")
	}
	f := fullFrame(t)
	hall := view.Structure{ID: "guildhall-1", Kind: view.StructGuildhall, State: view.StateReady, Object: view.ObjectRef{Kind: "Service", Name: "parcels"}}
	if c := StructureCard(hall, f); !strings.HasPrefix(c.Lines[1].Text, "DOOR 80") {
		t.Fatalf("guildhall lines = %+v", c.Lines)
	}
}

func TestCardsAndChoosersDraw(t *testing.T) {
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h := New(nil, (&fakeRunner{}).backend())
	f := selectStructure(staged(fullFrame(t)), stoneKeep())
	f.Nodes = map[string]model.NodeView{"stone-keep-2": {TaintSpecs: []string{"node-role.kubernetes.io/control-plane:NoSchedule"}, Labels: map[string]string{"tier": "front"}}}
	clickOn(t, h, f, "act:taint")
	h.Update(Input{MX: h.ChooserRect().X + 20, MY: h.ChooserRect().Y + chooserHead + 2}, f)
	h.Draw(dst, f)
	if !h.Hovering || !strings.HasPrefix(h.Hover.ID, pickPrefix) {
		t.Errorf("hover = %+v", h.Hover)
	}
	if taintName("dedicated") != "dedicated" || taintName("node-role.kubernetes.io/control-plane") != "the Royal Ward" {
		t.Error("taint names")
	}
	if answerLine("") != "(no output)" || soundFor(false) != "error" {
		t.Error("empty answers and refused sounds")
	}
}
