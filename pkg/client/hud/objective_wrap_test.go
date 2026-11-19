package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestAStepsProgressWrapsWholeAndIsNeverCut(t *testing.T) {
	h := New(nil, Backend{})
	f := crewFrame(Selection{}, allReveals...)
	progress := "first make room: the roll-out needs room for 4 more huts at once (it sends 3 old huts home first); the Keeps have room for 4; while it rolls, the huts still at work carry fewer boxes than land, and the Nines dip until it ends; once it ends, the Keeps hold all but 2 huts at the new pledge; buy a Keep, or scale the porters down by 2"
	f.State.Objectives = []view.ObjectiveView{{ID: view.MakeRoomObjective, Text: "Make room for the pledge.", Purpose: "An honest pledge needs Keeps with room for every hut it rolls over", Progress: progress}}
	text := linesText(h.ObjectiveLines(f))
	if strings.Contains(text, "...") || !strings.Contains(strings.ReplaceAll(text, "\n", " "), "down by 2") {
		t.Fatalf("the step's progress is cut:\n%s", text)
	}
}
