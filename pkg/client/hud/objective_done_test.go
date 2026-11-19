package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheObjectiveCardKeepsAFinishedStepUntilTheNextArrives(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Objectives = []view.ObjectiveView{
		{ID: "raise-keep", Text: "Raise a Keep", Why: "so that workers carry beside you", Done: true},
		{ID: "door-hearth", Text: "Set a porter at the door", Why: "so that a porter carries for you", Done: true},
	}
	lines := h.ObjectiveLines(f)
	if !h.ObjectiveShown(f) || len(lines) == 0 {
		t.Fatal("the task card went empty between steps")
	}
	if !strings.HasPrefix(lines[0].Text, doneMark) || !strings.Contains(lines[0].Text, "Set a porter at the door") {
		t.Errorf("the card does not show the last finished step, marked done: %q", lines[0].Text)
	}
	f.State.Objectives = append(f.State.Objectives, view.ObjectiveView{ID: "build-codex", Text: "Inspect the Codex"})
	if lines := h.ObjectiveLines(f); strings.HasPrefix(lines[0].Text, doneMark) || !strings.Contains(lines[0].Text, "Codex") {
		t.Errorf("the next step does not replace the finished one: %q", lines[0].Text)
	}
}

func TestTheObjectiveCardRaisesNoWindowsAndKeepsItsAlmanacLeft(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	r := h.ObjectiveRectOf(f)
	for _, b := range h.ObjectiveButtons(f) {
		if b.Tip != "" || b.Page != "" {
			t.Errorf("%s raises a window over the card: %q", b.ID, b.Tip)
		}
		if b.ID == ObjectiveAlmanac && b.Rect.X+b.Rect.W > r.X+r.W/2 {
			t.Errorf("ALMANAC %+v is not on the card's left %+v", b.Rect, r)
		}
	}
}
