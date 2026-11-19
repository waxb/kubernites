package art

import (
	"image"
	"image/color"
	"testing"
)

func havenRegion(kind string, v int) Region {
	return built().Buildings.Regions[havenName(kind, v)]
}

func TestHavenPiecesExistWithTheirVariants(t *testing.T) {
	for _, k := range HavenKinds {
		if HavenVariants(k) < 1 {
			t.Fatalf("%s has no variants", k)
		}
		for v := 0; v < HavenVariants(k); v++ {
			r := havenRegion(k, v)
			if r.W == 0 || opaquePixels(built().Buildings, r) == 0 {
				t.Errorf("%s/%d missing or empty", k, v)
			}
		}
		if _, ok := built().Buildings.Regions[havenName(k, HavenVariants(k))]; ok {
			t.Errorf("%s has more variants than HavenVariants says", k)
		}
	}
	if HavenVariants("nope") != 0 || HavenFootprint("warehouse") != 2 || HavenFootprint("crane") != 1 {
		t.Error("unknown kind")
	}
}

func TestHavenTilesAnchorAtTheTileCentre(t *testing.T) {
	for _, k := range []string{"pier-x", "pier-y", "quay"} {
		for v := 0; v < HavenVariants(k); v++ {
			r := havenRegion(k, v)
			if r.W != havenTileW || r.H != havenTileH || r.AnchorX != havenTileOX || r.AnchorY != havenTileOY+8 {
				t.Errorf("%s/%d region %+v", k, v, r)
			}
		}
	}
	for _, k := range []string{"warehouse", "lighthouse"} {
		r := havenRegion(k, 0)
		if r.AnchorX != r.W/2 || r.AnchorY != r.H-2 {
			t.Errorf("%s anchor is not the footprint bottom: %+v", k, r)
		}
	}
}

func composeTiles(a Atlas, name string, steps [][2]float64) *image.NRGBA {
	r := a.Regions[name]
	img := image.NewNRGBA(image.Rect(0, 0, 160, 120))
	for _, s := range steps {
		ox := 60 + (s[0]-s[1])*16 - float64(r.AnchorX)
		oy := 40 + (s[0]+s[1])*8 - float64(r.AnchorY)
		blitOver(img, sheetCell{a: a, r: r}, int(ox), int(oy))
	}
	return img
}

func inkAlong(img *image.NRGBA, from, to [2]float64) int {
	n := 0
	for k := 0; k <= 40; k++ {
		f := float64(k) / 40
		gx, gy := from[0]+(to[0]-from[0])*f, from[1]+(to[1]-from[1])*f
		x := int(60 + (gx-gy)*16)
		y := int(40 + (gx+gy-1)*8 - HavenDeck - 0.5)
		if img.NRGBAAt(x, y) == (color.NRGBA{ink.R, ink.G, ink.B, 255}) {
			n++
		}
	}
	return n
}

func TestPierAndQuayTilesJoinWithoutSeams(t *testing.T) {
	a := built().Buildings
	for v := 0; v < 3; v++ {
		px := composeTiles(a, havenName("pier-x", v), [][2]float64{{0, 0}, {1, 0}, {2, 0}})
		if n := inkAlong(px, [2]float64{0.1, 0.5}, [2]float64{2.4, 0.5}); n > 0 {
			t.Errorf("pier-x/%d: %d outline pixels across the joins", v, n)
		}
		py := composeTiles(a, havenName("pier-y", v), [][2]float64{{0, 0}, {0, 1}, {0, 2}})
		if n := inkAlong(py, [2]float64{0.5, 0.1}, [2]float64{0.5, 2.4}); n > 0 {
			t.Errorf("pier-y/%d: %d outline pixels across the joins", v, n)
		}
		q := composeTiles(a, havenName("quay", v), [][2]float64{{0, 0}, {1, 0}, {0, 1}, {1, 1}})
		if n := inkAlong(q, [2]float64{0.1, 0.5}, [2]float64{1.9, 0.5}) + inkAlong(q, [2]float64{0.5, 0.1}, [2]float64{0.5, 1.9}); n > 0 {
			t.Errorf("quay/%d: %d outline pixels across the joins", v, n)
		}
	}
}

func contentHeight(a Atlas, r Region) int {
	return r.AnchorY - contentBounds(a, r).Min.Y
}

func TestTallHarborPiecesStayWithinTheirDeclaredHeight(t *testing.T) {
	a := built().Buildings
	cases := map[string]int{"warehouse": warehouseHeight, "lighthouse": lighthouseHeight}
	for k, px := range cases {
		n := HavenFootprint(k)
		for v := 0; v < HavenVariants(k); v++ {
			if h := contentHeight(a, havenRegion(k, v)) - n*TileHeight; h > px || h < px-12 {
				t.Errorf("%s/%d rises %dpx above its footprint, declared %d", k, v, h, px)
			}
		}
	}
	if contentHeight(a, havenRegion("lighthouse", 0)) <= contentHeight(a, havenRegion("warehouse", 0)) {
		t.Error("the lighthouse must be the tallest harbor piece")
	}
}

func TestWakeIsTranslucentAndRequestIsABubble(t *testing.T) {
	a := built().Effects
	for f := 0; f < EffectFrames; f++ {
		r := a.Regions[effectName("wake", f)]
		for y := 0; y < r.H; y++ {
			for x := 0; x < r.W; x++ {
				if a.Image.NRGBAAt(r.X+x, r.Y+y).A == 255 {
					t.Fatalf("wake/%d has an opaque pixel at %d,%d", f, x, y)
				}
			}
		}
		b := contentBounds(a, r)
		if cx := (b.Min.X + b.Max.X) / 2; cx < r.AnchorX-2 || cx > r.AnchorX+2 {
			t.Errorf("wake/%d is not centred on its anchor", f)
		}
		q := a.Regions[effectName("request", f)]
		want := paddedSize(size{RequestWidth, RequestHeight, RequestWidth / 2, RequestHeight - 1}, effectPads["request"])
		if q.W != want.w || q.H != want.h || q.AnchorY != want.ay {
			t.Errorf("request/%d region %+v", f, q)
		}
		if a.Image.NRGBAAt(q.X+q.AnchorX, q.Y+q.AnchorY).A == 0 && a.Image.NRGBAAt(q.X+q.AnchorX, q.Y+q.AnchorY-1).A == 0 {
			t.Errorf("request/%d: the tail does not reach the anchor", f)
		}
	}
	for _, k := range []string{"wake", "request"} {
		if samePixels(a, a.Regions[effectName(k, 0)], a.Regions[effectName(k, 2)]) {
			t.Errorf("%s frames do not animate", k)
		}
	}
}

func TestHarborPiecesFitTheirCanvas(t *testing.T) {
	a := built().Buildings
	for _, k := range HavenKinds {
		for v := 0; v < HavenVariants(k); v++ {
			r := havenRegion(k, v)
			b := contentBounds(a, r)
			if b.Min.X < 1 || b.Min.Y < 1 || b.Max.X > r.W-1 || b.Max.Y > r.H-1 {
				t.Errorf("%s/%d: content %v touches the %dx%d canvas edge", k, v, b, r.W, r.H)
			}
			if !image.Pt(r.AnchorX, r.AnchorY).In(b.Inset(-2)) {
				t.Errorf("%s/%d: anchor %d,%d is off the painted piece %v", k, v, r.AnchorX, r.AnchorY, b)
			}
		}
	}
}
