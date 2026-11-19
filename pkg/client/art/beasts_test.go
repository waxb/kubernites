package art

import "testing"

func TestEveryBeastHasAPainter(t *testing.T) {
	for _, role := range append(append([]string{}, CritterRoles...), "raven", "hound") {
		if beastPainters[role] == nil {
			t.Errorf("%s has no painter", role)
		}
	}
}

func TestBeastGaitsAnimate(t *testing.T) {
	a := framedWalkers()
	for role := range beastPainters {
		for _, action := range WalkerActions {
			frames := map[int]bool{}
			for f := 0; f < WalkerFrames; f++ {
				if !samePixels(a, a.Regions[walkerName(role, action, "se", 0)], a.Regions[walkerName(role, action, "se", f)]) {
					frames[f] = true
				}
			}
			if len(frames) == 0 && action != "idle" {
				t.Errorf("%s/%s never moves", role, action)
			}
		}
	}
}

func TestFlyingBirdsCarryNoGroundShadow(t *testing.T) {
	a := built().Walkers
	for _, role := range []string{"finch", "gull"} {
		for f := 0; f < WalkerFrames; f++ {
			r := a.Regions[walkerName(role, "walk", "se", f)]
			for x := 0; x < r.W; x++ {
				if a.Image.NRGBAAt(r.X+x, r.Y+r.AnchorY).A != 0 {
					t.Fatalf("%s flight frame %d has a pixel on the ground row", role, f)
				}
			}
		}
	}
}

func TestPerchedBeastsStandOnTheAnchor(t *testing.T) {
	a := built().Walkers
	for role := range beastPainters {
		r := a.Regions[walkerName(role, "idle", "se", 0)]
		feet := 0
		for x := 0; x < r.W; x++ {
			for y := max(r.AnchorY-2, 0); y <= r.AnchorY; y++ {
				if a.Image.NRGBAAt(r.X+x, r.Y+y).A > 200 {
					feet++
				}
			}
		}
		if feet == 0 {
			t.Errorf("%s idle floats above its anchor", role)
		}
	}
}
