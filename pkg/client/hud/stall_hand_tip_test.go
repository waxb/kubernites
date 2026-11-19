package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheHandDepotTipDescribesTheRollNotABadge(t *testing.T) {
	f := depotBoardFrame()
	f.State.Unlocks = map[string]bool{}
	seen := false
	for _, r := range stallRows(f) {
		if r.Kind != soulstone.KindParcel {
			continue
		}
		seen = true
		tip := stallTip(r, f)
		if strings.Contains(tip, "wearing app=") || strings.Contains(tip, "expose") {
			t.Errorf("before Badges the parcels depot keeps a hand roll, but its tip reads %q", tip)
		}
		if !strings.Contains(tip, "roll you keep by hand") {
			t.Errorf("the hand depot tip does not name the hand roll: %q", tip)
		}
		f.State.Unlocks = map[string]bool{view.BadgesPage: true}
		if tip := stallTip(r, f); !strings.Contains(tip, "wearing app=") {
			t.Errorf("after Badges the depot looks for a badge, but its tip reads %q", tip)
		}
	}
	if !seen {
		t.Fatal("no parcels stall row in the frame")
	}
}
