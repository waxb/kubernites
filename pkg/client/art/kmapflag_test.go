package art

import "testing"

func TestEveryBranchFliesAFlagInItsBannerCloth(t *testing.T) {
	a := built().Icons
	for i, x := range KMapBranches {
		r, ok := a.Regions[KMapFlagName(x)]
		if !ok || r.W != FlagW+2*kmapPad || r.H != FlagH+2*kmapPad {
			t.Fatalf("%s flag: %+v (present %v)", x, r, ok)
		}
		if share := float64(opaquePixels(a, r)) / float64(r.W*r.H); share < 0.7 {
			t.Errorf("the %s flag fills only %.0f%%", x, share*100)
		}
		for _, y := range KMapBranches[i+1:] {
			if differingShare(a, r, a.Regions[KMapFlagName(y)]) < 0.3 {
				t.Errorf("flags %s and %s look alike", x, y)
			}
		}
	}
}

func TestTopBarSupplyIconsStayAboveTheBarRule(t *testing.T) {
	for _, name := range []string{"food", "housing"} {
		img, ok := PaintIcon(name)
		if !ok {
			t.Fatalf("no %s icon", name)
		}
		for x := 0; x < IconSize; x++ {
			if img.NRGBAAt(x, IconSize-1).A != 0 {
				t.Fatalf("the %s icon reaches its last row and would cross the top bar rule", name)
			}
		}
	}
}
