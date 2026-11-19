package art

import "testing"

func lookRegion(a Atlas, role string, tier, variant int) Region {
	if tier == 1 && variant == 0 {
		return a.Regions[walkerName(role, "walk", "se", 1)]
	}
	return a.Regions[walkerLookName(role, tier, variant, "walk", "se", 1)]
}

func maskOverlap(a Atlas, x, y Region) float64 {
	same, total := 0, 0
	for yy := 0; yy < x.H; yy++ {
		for xx := 0; xx < x.W; xx++ {
			p, q := a.Image.NRGBAAt(x.X+xx, x.Y+yy).A > 200, a.Image.NRGBAAt(y.X+xx, y.Y+yy).A > 200
			if p || q {
				total++
				if p && q {
					same++
				}
			}
		}
	}
	return float64(same) / float64(max(total, 1))
}

func TestLooksVaryButKeepTheSilhouette(t *testing.T) {
	a := framedWalkers()
	for _, role := range VariedRoles {
		base := lookRegion(a, role, 1, 0)
		seen := map[string]bool{}
		for v := 1; v < WalkerLooks; v++ {
			r := lookRegion(a, role, 1, v)
			if r.W == 0 {
				t.Fatalf("%s look %d missing", role, v)
			}
			if samePixels(a, base, r) {
				t.Errorf("%s look %d equals the base look", role, v)
			}
			if o := maskOverlap(a, base, r); o < 0.7 {
				t.Errorf("%s look %d changes the silhouette too much (%.2f)", role, v, o)
			}
			seen[string(a.Image.Pix[r.Y*a.Image.Stride+r.X*4:(r.Y+r.H-1)*a.Image.Stride+(r.X+r.W)*4])] = true
		}
		if len(seen) < WalkerLooks-2 {
			t.Errorf("%s has only %d distinct looks", role, len(seen))
		}
	}
}

func TestTierCuesStayDistinct(t *testing.T) {
	a := framedWalkers()
	for v := 0; v < WalkerLooks; v++ {
		one, two, three := lookRegion(a, "lamplighter", 1, v), lookRegion(a, "lamplighter", 2, v), lookRegion(a, "lamplighter", 3, v)
		if samePixels(a, one, two) || samePixels(a, two, three) || samePixels(a, one, three) {
			t.Errorf("look %d: lamplighter tiers are not distinguishable", v)
		}
	}
	if WalkerTiers("scribe") != 1 || WalkerTiers("lamplighter") != 3 {
		t.Error("tier counts")
	}
}
