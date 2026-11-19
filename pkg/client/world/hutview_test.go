package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestHutsAreNotHiddenBehindBuildings(t *testing.T) {
	codex := view.Structure{Kind: view.StructCodex, X: 20, Y: 20, W: 2, H: 2}
	cases := []struct {
		x, y int
		want bool
	}{
		{18, 18, true},
		{19, 17, true},
		{16, 16, false},
		{23, 23, false},
		{12, 12, false},
		{26, 16, false},
		{20, 25, false},
	}
	for _, tc := range cases {
		if got := hutHiddenBy(codex, tc.x, tc.y); got != tc.want {
			t.Errorf("hut at %d,%d hidden by codex = %v, want %v", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestHutsAvoidBuildingShadows(t *testing.T) {
	site := testSite()
	h := NewHamlet()
	h.Place(site, crewPods("porters", "", "lone-tower", 12), 0)
	spots := h.Spots()
	if len(spots) < 8 {
		t.Fatalf("only %d huts found room around the keep", len(spots))
	}
	for name, p := range spots {
		for _, s := range site.Structures {
			if hutHiddenBy(s, p.X, p.Y) {
				t.Fatalf("%s at %v sits in %s's shadow", name, p, s.Kind)
			}
		}
	}
}
