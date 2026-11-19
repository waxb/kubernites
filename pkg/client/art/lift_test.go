package art

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestPlacementKnowsHowTallEachBuildingIs(t *testing.T) {
	for kind, px := range buildingHeights {
		variant := ""
		name := kind
		if v, ok := strings.CutPrefix(kind, "keep-"); ok {
			name, variant = view.StructKeep, v
		}
		if want := (px + 7) / 8; view.StructureLift(name, variant) != want {
			t.Errorf("%s is %dpx tall: placement lift %d, want %d", kind, px, view.StructureLift(name, variant), want)
		}
	}
}
