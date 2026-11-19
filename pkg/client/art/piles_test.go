package art

import (
	"image"
	"image/color"
	"testing"
)

func hasColor(a Atlas, r Region, want color.NRGBA) bool {
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			if a.Image.NRGBAAt(x, y) == want {
				return true
			}
		}
	}
	return false
}

func TestEveryRequestPileUnitIsInTheBuildingAtlas(t *testing.T) {
	a := built().Buildings
	for _, kind := range RequestPileKinds {
		if _, ok := a.Regions[RequestPileName(kind, 0)]; ok {
			t.Errorf("%s: no sprite for zero units", kind)
		}
		for n := 1; n <= RequestPileUnits; n++ {
			if _, ok := a.Regions[RequestPileName(kind, n)]; !ok {
				t.Errorf("no sprite %s", RequestPileName(kind, n))
			}
		}
	}
}

func TestRequestPilesGrowWithTheirUnits(t *testing.T) {
	a := built().Buildings
	for _, kind := range RequestPileKinds {
		last := 0
		for n := 1; n <= RequestPileUnits; n++ {
			r := a.Regions[RequestPileName(kind, n)]
			got := opaquePixels(a, r)
			if got <= last {
				t.Errorf("%s: %d units paint %d pixels, %d units painted %d", kind, n, got, n-1, last)
			}
			last = got
		}
	}
}

func TestRequestPilesUseTheLaborMaterials(t *testing.T) {
	a := built().Buildings
	for n := 1; n <= RequestPileUnits; n++ {
		wood, stone := a.Regions[RequestPileName(PileWood, n)], a.Regions[RequestPileName(PileStone, n)]
		if !hasColor(a, wood, woodR[3]) || !hasColor(a, wood, barkR[1]) || !hasColor(a, wood, ink) {
			t.Errorf("wood %d lacks the woodcutters' log colours", n)
		}
		if !hasColor(a, stone, cutR[3]) || !hasColor(a, stone, cutR[1]) || !hasColor(a, stone, ink) {
			t.Errorf("stone %d lacks the masons' cut stone colours", n)
		}
	}
}

func pileExtent(a Atlas, r Region) image.Rectangle {
	out := image.Rectangle{}
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			if a.Image.NRGBAAt(x, y).A > 96 {
				out = out.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return out
}

func TestRequestPilesStayTinyBesideAHut(t *testing.T) {
	s := hearthSize()
	a := built().Buildings
	for _, kind := range RequestPileKinds {
		for n := 1; n <= RequestPileUnits; n++ {
			b := pileExtent(a, a.Regions[RequestPileName(kind, n)])
			if b.Dx()*2 > s.w || b.Dy()*4 > s.h {
				t.Errorf("%s %d paints %dx%d beside a %dx%d hut", kind, n, b.Dx(), b.Dy(), s.w, s.h)
			}
		}
	}
}
