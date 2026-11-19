package art

import "testing"

var borderInkBefore = map[string]int{
	BorderStoneName(0): 63, BorderStoneName(1): 59, BorderStoneName(2): 74,
	BorderFlagName(0): 162, BorderFlagName(1): 147,
}

func inkedPixels(a Atlas, r Region) int {
	n := 0
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			if a.Image.NRGBAAt(r.X+x, r.Y+y).A > 128 {
				n++
			}
		}
	}
	return n
}

func TestBorderMarkersAreSlightlySmaller(t *testing.T) {
	a := built().Buildings
	for name, before := range borderInkBefore {
		r := a.Regions[name]
		share := float64(inkedPixels(a, r)) / float64(before)
		if share < 0.7 || share > 0.9 {
			t.Errorf("%s keeps %.0f%% of its former area, want a slight trim", name, share*100)
		}
		if b := solidBounds(a, r); b.Dx() > 14 || b.Dy() > 22 {
			t.Errorf("%s spans %dx%d", name, b.Dx(), b.Dy())
		}
	}
}
