package art

import (
	"image/color"
	"testing"
)

var paletteRamps = []ramp{
	waterR, shallowR, sandR, grassR, meadowR, forestR, rockR, dirtR, plazaR, stoneR, woodR, plasterR, redR, blueR,
	purpleR, greenR, goldR, tealR, orangeR, greyR, skinR, fireR, voidR, glassR, slateR, strawR, marbleR, copperR,
	daubR, mossR, dryR, soilR, crateR, quayR, pierR, ropeR, barkR, paperR,
}

var paletteSingles = []color.NRGBA{ink, inkSft, cream, white, lampLit, rgb(44, 34, 36), rgb(40, 30, 28), rgb(60, 30, 26)}

func paletteDistance(c color.NRGBA) float64 {
	best := 1e9
	for _, r := range paletteRamps {
		for _, p := range r {
			best = min(best, colorDist(c, p))
		}
	}
	for _, p := range paletteSingles {
		best = min(best, colorDist(c, p))
	}
	return best
}

func agedReadyNames(kind string) []string {
	out := []string{}
	for age := 0; age < AgeTiers; age++ {
		for v := 0; v < variantsOf(kind); v++ {
			out = append(out, BuildingLookName(kind, "ready", age, v))
		}
	}
	return out
}

func TestAgedSpritesExistForEveryAgeVariantAndState(t *testing.T) {
	a := built().Buildings
	for _, kind := range agedKinds() {
		names := agedReadyNames(kind)
		states := []string{"dark"}
		if !Unbuilt[kind] {
			names = append(names, buildingName(kind, "construction-0"))
			states = append(states, "construction-1", "construction-2")
		}
		for age := 0; age < AgeTiers; age++ {
			for _, st := range states {
				names = append(names, BuildingLookName(kind, st, age, 0))
			}
		}
		for _, n := range names {
			if _, ok := a.Regions[n]; !ok {
				t.Errorf("missing %s", n)
			}
		}
	}
}

func regionAt(a Atlas, r Region, x, y int) color.NRGBA { return a.Image.NRGBAAt(r.X+x, r.Y+y) }

func TestAgedSpritesStayInsideCanvasAndFootprint(t *testing.T) {
	a := built().Buildings
	for _, kind := range agedKinds() {
		for _, name := range agedReadyNames(kind) {
			r := a.Regions[name]
			for x := 0; x < r.W; x++ {
				if regionAt(a, r, x, 0).A > 0 {
					t.Errorf("%s is clipped at the top (x=%d)", name, x)
					break
				}
			}
			for y := 0; y < r.H; y++ {
				for x := 0; x < r.W; x++ {
					if regionAt(a, r, x, y).A == 0 {
						continue
					}
					if x == 0 || x == r.W-1 || y > r.AnchorY+2 {
						t.Errorf("%s spills out of its footprint at %d,%d", name, x, y)
						return
					}
				}
			}
		}
	}
}

func TestAgedSpritesKeepInkOutlines(t *testing.T) {
	a := built().Buildings
	for _, kind := range agedKinds() {
		n := Footprints[kind]
		for _, name := range agedReadyNames(kind) {
			r := a.Regions[name]
			bad := 0
			for x := 0; x < r.W; x++ {
				apronTop := r.AnchorY - n*TileHeight + abs(x-r.AnchorX)/2
				for y := 0; y < apronTop-1; y++ {
					col := regionAt(a, r, x, y)
					if col.A == 0 {
						continue
					}
					if col != ink {
						bad++
					}
					break
				}
			}
			if bad > 0 {
				t.Errorf("%s: %d columns lose the ink outline on top", name, bad)
			}
		}
	}
}

func TestAgedVariantsDiffer(t *testing.T) {
	a := built().Buildings
	for _, kind := range agedKinds() {
		for age := 0; age < AgeTiers; age++ {
			for v := 1; v < variantsOf(kind); v++ {
				p, q := a.Regions[BuildingLookName(kind, "ready", age, v-1)], a.Regions[BuildingLookName(kind, "ready", age, v)]
				diff := 0
				for y := 0; y < p.H; y++ {
					for x := 0; x < p.W; x++ {
						if regionAt(a, p, x, y) != regionAt(a, q, x, y) {
							diff++
						}
					}
				}
				if diff < 40 {
					t.Errorf("%s age %d: variants %d and %d differ in only %d pixels", kind, age, v-1, v, diff)
				}
			}
		}
	}
}

func TestAgedSpritesStayInThePalette(t *testing.T) {
	a := built().Buildings
	for _, kind := range agedKinds() {
		for _, name := range agedReadyNames(kind) {
			r := a.Regions[name]
			worst, at := 0.0, [2]int{}
			for y := 0; y < r.H; y++ {
				for x := 0; x < r.W; x++ {
					if col := regionAt(a, r, x, y); col.A == 255 {
						if d := paletteDistance(col); d > worst {
							worst, at = d, [2]int{x, y}
						}
					}
				}
			}
			if worst > 64 {
				t.Errorf("%s: colour %v at %v is %.0f away from the palette", name, regionAt(a, r, at[0], at[1]), at, worst)
			}
		}
	}
}

func TestAgesLookDifferent(t *testing.T) {
	a := built().Buildings
	for _, kind := range agedKinds() {
		for age := 1; age < AgeTiers; age++ {
			p, q := a.Regions[BuildingLookName(kind, "ready", age-1, 0)], a.Regions[BuildingLookName(kind, "ready", age, 0)]
			if meanOf(a, p).R == meanOf(a, q).R && meanOf(a, p).G == meanOf(a, q).G {
				t.Errorf("%s: ages %d and %d look the same", kind, age-1, age)
			}
		}
	}
}

func meanOf(a Atlas, r Region) color.NRGBA {
	var s [3]int
	n := 1
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			if col := regionAt(a, r, x, y); col.A == 255 {
				s[0], s[1], s[2], n = s[0]+int(col.R), s[1]+int(col.G), s[2]+int(col.B), n+1
			}
		}
	}
	return color.NRGBA{uint8(s[0] / n), uint8(s[1] / n), uint8(s[2] / n), 255}
}

func BenchmarkBuildBuildings(b *testing.B) {
	for i := 0; i < b.N; i++ {
		buildBuildings()
	}
}
