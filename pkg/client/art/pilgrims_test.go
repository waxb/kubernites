package art

import "testing"

func saturation(r ramp) float64 {
	c := r[1]
	hi := max(c.R, c.G, c.B)
	lo := min(c.R, c.G, c.B)
	return float64(hi-lo) / float64(max(hi, 1))
}

func TestPilgrimsStayTheSameKindOfTraveller(t *testing.T) {
	tools, heights := map[string]bool{}, map[int]bool{}
	for v := 1; v < WalkerLooks; v++ {
		spec := lookSpec("pilgrim", v, 1)
		if spec.hat != "hood" && spec.hat != "straw" {
			t.Errorf("look %d wears %q", v, spec.hat)
		}
		if !spec.robe {
			t.Errorf("look %d lost the cloak", v)
		}
		if s := saturation(spec.tunic); s > 0.32 {
			t.Errorf("look %d cloak is too loud (saturation %.2f)", v, s)
		}
		tools[spec.tool[:6]] = true
		heights[spec.short] = true
	}
	if !tools["pilgri"] || !tools["bindle"] {
		t.Errorf("pilgrims need both staffs and bundles: %v", tools)
	}
	if len(heights) != 2 {
		t.Errorf("pilgrims need a 1px height difference: %v", heights)
	}
}

func TestShortPilgrimsAreOnePixelShorter(t *testing.T) {
	a := framedWalkers()
	base := contentBounds(a, lookRegion(a, "pilgrim", 1, 0)).Min.Y
	found := false
	for v := 1; v < WalkerLooks; v++ {
		spec := lookSpec("pilgrim", v, 1)
		top := contentBounds(a, lookRegion(a, "pilgrim", 1, v)).Min.Y
		if spec.short == 1 && spec.hat == "hood" {
			found = true
			if top != base+1 {
				t.Errorf("short hooded look %d top %d, base %d", v, top, base)
			}
		}
	}
	if !found {
		t.Skip("no short hooded pilgrim to compare")
	}
}
