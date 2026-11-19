package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func stoneObjective(t *testing.T, cmd string) Frame {
	t.Helper()
	f := offerFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "three-hearths", Text: "Scale", Hint: "Scale", Target: SurfaceStone, Command: cmd}}
	return f
}

func TestButtonFirstObjectivesHideDoIt(t *testing.T) {
	if !ButtonFirst(view.ObjectiveView{Target: CmdDepot}) || !ButtonFirst(view.ObjectiveView{Target: "structure:chancery"}) {
		t.Error("a button or a building leads")
	}
	if ButtonFirst(view.ObjectiveView{Target: SurfaceStone}) || ButtonFirst(view.ObjectiveView{Target: SurfaceScriptorium}) || ButtonFirst(view.ObjectiveView{}) {
		t.Error("the Stone and the Scriptorium lead themselves")
	}
	f := offerFrame(t, openParcels)
	f.State.Objectives = []view.ObjectiveView{{ID: "open-depot", Text: "Open", Target: CmdDepot, Command: openParcels.Command}}
	l, ok := f.ObjectiveLine()
	if !ok || l.Runnable() {
		t.Fatalf("a DEPOT task = %+v", l)
	}
	h := New(nil, (&fakeRunner{}).backend())
	chip, do := ObjectiveChipRects(h.ObjectiveRectOf(f), l)
	if do.W != 0 || chip.W == 0 {
		t.Fatalf("chip %+v do %+v", chip, do)
	}
	if _, ok := HitID(h.Buttons(f), ObjDo); ok {
		t.Error("no DO IT beside a button-first task")
	}
}

func TestStoneFirstObjectivesRunTheirCommandWithDoIt(t *testing.T) {
	runner := &fakeRunner{}
	h := New(nil, runner.backend())
	f := stoneObjective(t, "kubectl scale deployment porters --replicas=3")
	l, _ := f.ObjectiveLine()
	if !l.Runnable() || l.Action != nil {
		t.Fatalf("stone-first line = %+v", l)
	}
	intents := clickOn(t, h, f, ObjDo)
	if runner.commands[len(runner.commands)-1] != "scale deployment porters --replicas=3" {
		t.Fatalf("DO IT ran %v", runner.commands)
	}
	if echo, ok := hasIntent(intents, IntentEcho); !ok || echo.Command != "scale deployment porters --replicas=3" {
		t.Fatalf("DO IT echoes: %+v", intents)
	}
	last := h.StoneLines()
	found := false
	for _, s := range last {
		found = found || strings.HasSuffix(s.Text, "(from the objective card)")
	}
	if !found || len(h.Toasts()) == 0 {
		t.Error("a mutating DO IT toasts and marks its source")
	}
	read := stoneObjective(t, "kubectl describe node node-a")
	clickOn(t, h, read, ObjDo)
	if lines := h.Details().Lines; h.Overlay() != OverlayDetails || len(lines) < 3 || !strings.HasPrefix(lines[0], "Your step: ") || lines[2] != "$ kubectl describe node node-a" {
		t.Fatalf("a reading DO IT shows its answer: %+v", h.Details())
	}
	if h.doObjective(CommandLine{Command: "kubectl get pods"}) != nil || h.doObjective(CommandLine{Run: true}) != nil {
		t.Error("nothing to run, nothing done")
	}
	if !readCommand("kubectl get pods") || readCommand("kubectl scale deployment x") || readCommand("") {
		t.Error("read verbs")
	}
}
