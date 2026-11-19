package art

import (
	"bytes"
	"testing"
)

var (
	hutVentSpots = map[string][2]int{"vagrant": {-3, -29}, "freeman": {5, -40}, "landed": {5, -47}}
	hutAgeShift  = map[string]float64{"vagrant": 0.25, "freeman": 0.3, "landed": 0.3}
)

func eachHut(fn func(tier int, qos string, age, v int, state string)) {
	for tier := 1; tier <= 3; tier++ {
		for _, q := range HearthQoS {
			for age := 0; age < AgeTiers; age++ {
				for _, st := range HearthStates {
					for v := 0; v < hutVariants(st); v++ {
						fn(tier, q, age, v, st)
					}
				}
			}
		}
	}
}

func TestEveryAgedHutExists(t *testing.T) {
	a, n := built().Buildings, 0
	eachHut(func(tier int, q string, age, v int, st string) {
		n++
		if _, ok := a.Regions[HearthLookName(tier, st, q, age, v)]; !ok {
			t.Errorf("missing %s", HearthLookName(tier, st, q, age, v))
		}
	})
	if n != 3*3*AgeTiers*(3*BuildingVariants+3) {
		t.Fatalf("%d aged huts", n)
	}
	if _, ok := a.Regions[HearthLookName(1, "pending", "vagrant", 2, 1)]; ok {
		t.Fatal("pending huts come in one variant only")
	}
}

func topOpaque(a Atlas, r Region, x int) int {
	for y := 0; y < r.H; y++ {
		if a.Image.NRGBAAt(r.X+x, r.Y+y).A > 0 {
			return y
		}
	}
	return r.H
}

func TestHutChimneysStayUnderTheirVents(t *testing.T) {
	a := built().Buildings
	eachHut(func(tier int, q string, age, v int, st string) {
		if st != "ready" {
			return
		}
		r := a.Regions[HearthLookName(tier, st, q, age, v)]
		vent := hutVentSpots[q]
		x, y := r.AnchorX+vent[0], r.AnchorY+vent[1]
		if top := topOpaque(a, r, x); top < y-3 || top > y+7 {
			t.Errorf("%s: the chimney top %d is far from the smoke vent %d", HearthLookName(tier, st, q, age, v), top, y)
		}
	})
}

func TestTheQoSLadderHoldsInEveryAge(t *testing.T) {
	a := built().Buildings
	for tier := 1; tier <= 3; tier++ {
		for age := 0; age < AgeTiers; age++ {
			for v := 0; v < BuildingVariants; v++ {
				h := func(q string) int { return contentHeight(a, a.Regions[HearthLookName(tier, "ready", q, age, v)]) }
				shack, cottage, manor := h("vagrant"), h("freeman"), h("landed")
				if !(shack < cottage && cottage < manor) {
					t.Errorf("tier %d age %d v%d: shack %d, cottage %d, manor %d", tier, age, v, shack, cottage, manor)
				}
			}
		}
	}
}

func TestHutRoofsKeepTheirTierInEveryAge(t *testing.T) {
	a := built().Buildings
	eachHut(func(tier int, q string, age, v int, st string) {
		if st == "ghost" || st == "banished" {
			return
		}
		r := a.Regions[HearthLookName(tier, st, q, age, v)]
		own := accentCount(a, r, tier)
		for other := 1; other <= 3; other++ {
			if other != tier && accentCount(a, r, other) >= own {
				t.Errorf("%s: tier %d colours (%d) rival its own (%d)", HearthLookName(tier, st, q, age, v), other, accentCount(a, r, other), own)
			}
		}
	})
}

func TestHutVariantsAndAgesDiffer(t *testing.T) {
	a := built().Buildings
	for tier := 1; tier <= 3; tier++ {
		for _, q := range HearthQoS {
			for age := 0; age < AgeTiers; age++ {
				base := a.Regions[HearthLookName(tier, "ready", q, age, 0)]
				for v := 1; v < BuildingVariants; v++ {
					if d := differingShare(a, base, a.Regions[HearthLookName(tier, "ready", q, age, v)]); d < 0.03 {
						t.Errorf("tier %d %s age %d: variant %d differs in %.1f%% of pixels", tier, q, age, v, d*100)
					}
				}
				if age == 0 {
					continue
				}
				if d := differingShare(a, a.Regions[HearthLookName(tier, "ready", q, age-1, 0)], base); d < hutAgeShift[q] {
					t.Errorf("tier %d %s: age %d differs from age %d in %.0f%% of pixels", tier, q, age, age-1, d*100)
				}
			}
		}
	}
}

func TestAgedHutsStandOnTheirOwnTile(t *testing.T) {
	a := built().Buildings
	eachHut(func(tier int, q string, age, v int, st string) {
		name := HearthLookName(tier, st, q, age, v)
		r := a.Regions[name]
		for y := r.AnchorY - TileHeight/2; y < r.H; y++ {
			for x := 0; x < r.W; x++ {
				if a.Image.NRGBAAt(r.X+x, r.Y+y).A > 0 && !insideTile(x, y, r.AnchorX, r.AnchorY, 2) {
					t.Fatalf("%s: pixel %d,%d spills off the tile", name, x, y)
				}
			}
		}
	})
}

func TestHutPaintingIsDeterministic(t *testing.T) {
	s := hearthSize()
	for _, q := range HearthQoS {
		h := hut{tier: 2, qos: q, age: ageCastle, variant: 1}
		one, two := qosHouse(s, h, false), qosHouse(s, h, false)
		if !bytes.Equal(one.img.Pix, two.img.Pix) {
			t.Errorf("%s: the same seed paints different pixels", q)
		}
	}
}
