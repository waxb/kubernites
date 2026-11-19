package hud

import (
	"github.com/waxb/kubernites/pkg/client/model"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestALockedTilesReasonNeverCoversItsName(t *testing.T) {
	for _, label := range []string{"KEEP", "[parcel] PARCELS DEPOT", "STONE KEEP"} {
		b := Button{ID: "build:x", Label: label, Disabled: true, Why: "ASK STONE", Rect: CellRect(5)}
		for i, row := range labelRows(b) {
			if Overlaps(WhyStrip(b), row) {
				t.Errorf("%q: the lock reason covers label row %d, so the tile reads ASK STONE / DEPOT", label, i)
			}
		}
	}
}

func TestALockedTilesReasonNeverCoversItsCellNumber(t *testing.T) {
	for _, label := range []string{"KEEP", "[parcel] PARCELS DEPOT", "STONE KEEP"} {
		b := Button{ID: "build:x", Label: label, Disabled: true, Why: "ASK STONE", Rect: CellRect(5)}
		key := ui.Rect{X: b.Rect.X + 3, Y: b.Rect.Y + 3, W: 5, H: 7}
		if Overlaps(WhyStrip(b), key) {
			t.Errorf("%q: the lock reason covers the cell number", label)
		}
	}
}

func TestTheKeepsOutOfFoodCauseNeverClaimsWaitingHuts(t *testing.T) {
	var f Frame
	f.State.Bottleneck = "labor"
	if got := Cause(f); strings.Contains(got, "wait") {
		t.Fatalf("a drought with no hut Unhoused still says huts wait: %q", got)
	}
}

func TestTheSliderForecastNamesTheSeasoningTheNextHutBrings(t *testing.T) {
	var f Frame
	f.State.Crews = []view.CrewView{{Name: "porters", MilestoneView: view.MilestoneView{Next: 10, NextMultiplier: 3}}}
	v := BoardView{Chain: model.CrewChain{Name: "porters", Desired: 9}, Target: 10}
	got := payPiece(v, f, 1).Text
	if strings.Contains(got, "KEEPING UP") || !strings.Contains(got, "x3") {
		t.Fatalf("the 10th hut seasons the crew, the forecast says %q", got)
	}
}
