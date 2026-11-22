package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestTheLevelBoxStartsBelowTheSeenAsLine(t *testing.T) {
	p := LivePicker{Step: LiveStepConfirm, Level: LevelWrites, Contexts: []LiveContext{{Name: "kind-kubernites"}}, Chosen: 0}
	seenEnds := LiveBox.Y + 48 + 4*ui.LineHeight
	found := false
	for _, b := range p.levelButtons() {
		if b.ID != liveLevel {
			continue
		}
		found = true
		if b.Rect.Y < seenEnds {
			t.Errorf("the level box %+v reaches into the SEEN AS line, which ends at y=%d", b.Rect, seenEnds)
		}
	}
	if !found {
		t.Fatal("no level box for WRITES")
	}
}
