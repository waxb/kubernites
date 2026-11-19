package gfx

import (
	"image"
	"image/color"
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
)

func TestSailsGildAndTheHullKeepsItsWood(t *testing.T) {
	cases := []struct {
		name string
		in   color.NRGBA
		gold bool
	}{
		{"sail", color.NRGBA{252, 246, 224, 255}, true},
		{"sail shade", color.NRGBA{214, 190, 146, 255}, true},
		{"trim", color.NRGBA{82, 126, 172, 255}, true},
		{"trim shade", color.NRGBA{56, 88, 138, 255}, true},
		{"hull", color.NRGBA{176, 122, 76, 255}, false},
		{"hull dark", color.NRGBA{90, 42, 30, 255}, false},
		{"ink", color.NRGBA{33, 27, 34, 255}, false},
		{"foam", color.NRGBA{170, 222, 220, 215}, false},
	}
	for _, c := range cases {
		out := GildShipPixel(c.in)
		if out.A != c.in.A {
			t.Errorf("%s: alpha %d -> %d", c.name, c.in.A, out.A)
		}
		if !c.gold && out != c.in {
			t.Errorf("%s: %v changed to %v", c.name, c.in, out)
		}
		if c.gold && !golden(out) {
			t.Errorf("%s: %v gilds to %v, not gold", c.name, c.in, out)
		}
	}
}

func golden(c color.NRGBA) bool {
	return c.R > c.G && c.G > c.B && int(c.R)-int(c.B) > 60
}

func TestBoxFacesGildAndTheBattenStaysDark(t *testing.T) {
	face := GildBoxPixel(color.NRGBA{228, 198, 146, 255})
	batten := GildBoxPixel(color.NRGBA{160, 120, 74, 255})
	if !golden(face) || face.R < 220 || face.G < 170 {
		t.Errorf("face gilds to %v", face)
	}
	if batten.R > 135 || batten.G > 95 || batten.B > 45 || int(face.G)-int(batten.G) < 80 {
		t.Errorf("batten %v does not contrast with the face %v", batten, face)
	}
	for _, keep := range []color.NRGBA{{33, 27, 34, 255}, {20, 16, 24, 90}, {82, 126, 172, 255}} {
		if got := GildBoxPixel(keep); got != keep {
			t.Errorf("%v changed to %v", keep, got)
		}
	}
}

func TestGildedBoxHasAOnePixelRim(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	src.SetNRGBA(1, 1, color.NRGBA{228, 198, 146, 255})
	src.SetNRGBA(2, 1, color.NRGBA{160, 120, 74, 255})
	src.SetNRGBA(1, 3, color.NRGBA{20, 16, 24, 90})
	out, r := gildRegion(src, art.Region{X: 0, Y: 0, W: 4, H: 4, AnchorX: 2, AnchorY: 3}, GildBoxPixel, true)
	if r.W != 6 || r.H != 6 || r.AnchorX != 3 || r.AnchorY != 4 {
		t.Fatalf("region %+v", r)
	}
	rim := 0
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			if out.NRGBAAt(x, y) == gildRim {
				rim++
			}
		}
	}
	if rim != 6 {
		t.Fatalf("a two-pixel box has %d rim pixels, want 6 (4-neighbours, no corners, none for the shadow)", rim)
	}
	if out.NRGBAAt(1, 1) != (color.NRGBA{}) || out.NRGBAAt(4, 3) != (color.NRGBA{}) {
		t.Fatal("the rim rounds the corners")
	}
}

func TestGildedSpritesAreRegistered(t *testing.T) {
	lib := Load(art.Build())
	for _, name := range []string{GildedName(ShipName(GildedClass, GildedVariant, "se", 0)), GildedName(BoxName(GildedBoxSize, "parcel", 0))} {
		if !lib.Has(name) {
			t.Errorf("%s is missing", name)
		}
	}
	w, h := lib.Size(GildedName(BoxName(GildedBoxSize, "parcel", 0)))
	bw, bh := lib.Size(BoxName(GildedBoxSize, "parcel", 0))
	if w != bw+2 || h != bh+2 {
		t.Errorf("gilded box %dx%d, plain %dx%d", w, h, bw, bh)
	}
}
