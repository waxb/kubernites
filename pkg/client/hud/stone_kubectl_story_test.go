package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestInStoryTheStoneThatRunsKubectlSpeaksKubectlsOwnWords(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(lexicon.StoryText)
	h, f := liveHUD(t, true)
	if p := h.stonePrompt(); p != "$ kubectl " {
		t.Errorf("STONE: KUBECTL prompts %q in story", p)
	}
	h.Stone.SetInput("get huts -n shop")
	h.Speak()
	if len(f.started) != 1 || f.started[0] != "get huts -n shop" {
		t.Fatalf("kubectl was handed %v, want the words as typed", f.started)
	}
	if echo := h.Stone.Output[len(h.Stone.Output)-1]; ConsoleText(echo) != "> kubectl get huts -n shop" {
		t.Errorf("the echo reads %q", ConsoleText(echo))
	}
	story, _ := liveHUD(t, false)
	if p := story.stonePrompt(); !strings.Contains(p, lexicon.StoryCommand) {
		t.Errorf("STONE: STORY prompts %q in story", p)
	}
}

func TestTheConfirmBoxLeavesTheEchoAboveIt(t *testing.T) {
	h, f := liveHUD(t, true)
	for i := range 40 {
		h.Stone.Append(LineOutput, strings.Repeat("x", i%7+1))
	}
	h.Stone.SetInput("scale deployment web-b -n parity --replicas=4")
	h.Speak()
	f.jobs[0].ask = &StoneAsk{Title: "CONFIRM: kubectl scale deployment web-b -n parity --replicas=4", Lines: []string{"Request: PATCH /apis/apps/v1/namespaces/parity/deployments/web-b/scale", "Body: {}", "Dry run (dryRun=All): OK"}, Confirm: true}
	h.pumpKubectl()
	a, ok := h.StoneAsking()
	if !ok {
		t.Fatal("no ask on the Stone")
	}
	box, out := h.stoneAskRect(a), h.stoneOutput()
	rows, shown := h.StoneRows(), h.stoneShownRows()
	if shown < 1 {
		t.Fatalf("no row is shown above the box")
	}
	last := out.Y + 3 + (shown-1)*ui.LineHeight + ui.GlyphHeight
	if last >= box.Y {
		t.Errorf("the last row shown ends at y %d, the box starts at %d", last, box.Y)
	}
	if got := rows[len(rows)-1].Text; !strings.Contains(got, "scale deployment web-b") {
		t.Errorf("the newest row is %q, want the echo", got)
	}
	h.answerStone(false)
	if h.stoneShownRows() != h.StoneVisible() {
		t.Errorf("with no ask the Stone shows %d rows of %d", h.stoneShownRows(), h.StoneVisible())
	}
}
