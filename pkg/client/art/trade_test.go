package art

import (
	"slices"
	"testing"
)

var tradeActions = append(append([]string{}, WalkerActions...), HoldAction)

func tradeRegion(a Atlas, role string, v int, action, dir string, f int) Region {
	if v == 0 {
		return a.Regions[walkerName(role, action, dir, f)]
	}
	return a.Regions[walkerLookName(role, 1, v, action, dir, f)]
}

func TestTradeRolesAreRegisteredAsVariedWalkers(t *testing.T) {
	for _, role := range TradeRoles {
		if !slices.Contains(WalkerRoles, role) || !slices.Contains(VariedRoles, role) {
			t.Errorf("%s is not a varied walker role", role)
		}
		if roleSpecs[role].trade != role {
			t.Errorf("%s spec is not a trade", role)
		}
		if _, ok := TradeKinds[role]; !ok {
			t.Errorf("%s has no box kind", role)
		}
		if want := map[string]int{"porter": 3}[role]; WalkerTiers(role) != max(want, 1) {
			t.Errorf("%s has %d tiers", role, WalkerTiers(role))
		}
	}
	for role, kind := range TradeKinds {
		if kind != "" && !slices.Contains(BoxKinds, kind) {
			t.Errorf("%s carries unknown kind %s", role, kind)
		}
	}
}

func TestTradeWalkersKeepTheFeetAnchor(t *testing.T) {
	a := framedWalkers()
	for _, role := range TradeRoles {
		for v := 0; v < WalkerLooks; v++ {
			for _, action := range tradeActions {
				for f := 0; f < WalkerFrames; f++ {
					r := tradeRegion(a, role, v, action, "se", f)
					if r.W != walkerFrameW || r.H != walkerFrameH || r.AnchorX != walkerFrameAX || r.AnchorY != walkerFrameAY {
						t.Fatalf("%s v%d %s %d: region %+v", role, v, action, f, r)
					}
					b := solidBounds(a, r)
					if b.Max.Y-1 != r.AnchorY {
						t.Errorf("%s v%d %s %d: feet end on row %d", role, v, action, f, b.Max.Y-1)
					}
					if action == "idle" && b.Dy() > 21 {
						t.Errorf("%s v%d idle is %d px tall", role, v, b.Dy())
					}
				}
			}
		}
	}
}

func TestTradeFramesAnimateAndMirror(t *testing.T) {
	a := framedWalkers()
	for _, role := range TradeRoles {
		for _, action := range []string{"walk", "work", "carry", HoldAction} {
			if samePixels(a, tradeRegion(a, role, 0, action, "se", 0), tradeRegion(a, role, 0, action, "se", 1)) {
				t.Errorf("%s %s frames 0 and 1 are identical", role, action)
			}
		}
		se, sw := tradeRegion(a, role, 2, HoldAction, "se", 1), tradeRegion(a, role, 2, HoldAction, "sw", 1)
		for y := 0; y < se.H; y++ {
			for x := 0; x < se.W; x++ {
				if a.Image.NRGBAAt(se.X+x, se.Y+y) != a.Image.NRGBAAt(sw.X+se.W-1-x, sw.Y+y) {
					t.Fatalf("%s hold: sw is not a mirror of se at %d,%d", role, x, y)
				}
			}
		}
	}
}

func TestTradesAndTheirLooksAreDistinct(t *testing.T) {
	a := framedWalkers()
	for i, role := range TradeRoles {
		for _, other := range TradeRoles[i+1:] {
			for _, action := range []string{"idle", "carry"} {
				if d := differingShare(a, tradeRegion(a, role, 0, action, "se", 0), tradeRegion(a, other, 0, action, "se", 0)); d < 0.3 {
					t.Errorf("%s and %s %s differ in only %.0f%%", role, other, action, d*100)
				}
			}
		}
		for v := 1; v < WalkerLooks; v++ {
			if samePixels(a, tradeRegion(a, role, 0, "walk", "se", 1), tradeRegion(a, role, v, "walk", "se", 1)) {
				t.Errorf("%s look %d is the base look", role, v)
			}
		}
		if samePixels(a, tradeRegion(a, role, 0, "carry", "se", 1), tradeRegion(a, role, 0, HoldAction, "se", 1)) {
			t.Errorf("%s: hold is the carry pose", role)
		}
	}
}

func TestHaulerIsSturdierAndCarriesHigh(t *testing.T) {
	a := framedWalkers()
	hauler := solidBounds(a, a.Regions[walkerName("hauler", "idle", "se", 0)])
	waist := walkerPadTop + figureTop + 10
	if hw, pw := rowWidth(a, a.Regions[walkerName("hauler", "idle", "se", 0)], waist), rowWidth(a, a.Regions[walkerName("porter", "idle", "se", 0)], waist); hw <= pw {
		t.Errorf("hauler waist %d px wide, porter %d", hw, pw)
	}
	back := solidBounds(a, a.Regions[walkerName("hauler", "carry", "se", 0)])
	if back.Min.Y >= hauler.Min.Y {
		t.Errorf("the hauler's crate does not rise over the head: %v vs %v", back, hauler)
	}
}

func TestTradePortraitsExist(t *testing.T) {
	a := built().Portraits
	for i, role := range TradeRoles {
		r, ok := a.Regions[portraitName(role)]
		if !ok {
			t.Fatalf("portrait/%s missing", role)
		}
		for _, other := range TradeRoles[i+1:] {
			if differingShare(a, r, a.Regions[portraitName(other)]) < 0.1 {
				t.Errorf("portraits %s and %s barely differ", role, other)
			}
		}
	}
}

func rowWidth(a Atlas, r Region, y int) int {
	n := 0
	for x := 0; x < r.W; x++ {
		if a.Image.NRGBAAt(r.X+x, r.Y+y).A > 200 {
			n++
		}
	}
	return n
}

func TestPorterClassesShowTheirTier(t *testing.T) {
	a := framedWalkers()
	for v := 0; v < WalkerLooks; v++ {
		for _, action := range []string{"walk", HoldAction} {
			base := tradeRegion(a, "porter", v, action, "se", 1)
			for tier := 2; tier <= WalkerTiers("porter"); tier++ {
				r, ok := a.Regions[walkerLookName("porter", tier, v, action, "se", 1)]
				if !ok {
					t.Fatalf("porter tier %d look %d %s missing", tier, v, action)
				}
				if samePixels(a, base, r) {
					t.Errorf("porter tier %d look %d %s is the tier 1 look", tier, v, action)
				}
			}
		}
	}
}
