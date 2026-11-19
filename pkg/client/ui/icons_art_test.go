package ui

import (
	"image"
	"image/color"
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
)

func nearestDown(src *image.NRGBA) map[[2]int]color.RGBA {
	b := image.Rectangle{}
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		for x := src.Rect.Min.X; x < src.Rect.Max.X; x++ {
			if src.NRGBAAt(x, y).A >= 128 {
				b = b.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	f := min(float64(iconWidth)/float64(b.Dx()), float64(iconRows)/float64(b.Dy()), 1)
	w, h := int(float64(b.Dx())*f+.5), int(float64(b.Dy())*f+.5)
	ox, oy := (iconWidth-w)/2, (iconRows-h)/2
	out := map[[2]int]color.RGBA{}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := src.NRGBAAt(b.Min.X+int((float64(x)+.5)/f), b.Min.Y+int((float64(y)+.5)/f))
			if c.A >= 128 {
				out[[2]int{ox + x, oy + y}] = color.RGBA{c.R, c.G, c.B, 255}
			}
		}
	}
	return out
}

func TestInlineIconsAreTheOriginalSpritesScaledDown(t *testing.T) {
	top := func(name string) func() (*image.NRGBA, bool) {
		return func() (*image.NRGBA, bool) { return art.PaintIcon(name) }
	}
	box := func(kind string, sealed bool) func() (*image.NRGBA, bool) {
		return func() (*image.NRGBA, bool) { return art.PaintBoxIcon(kind, sealed) }
	}
	for icon, paint := range map[string]func() (*image.NRGBA, bool){
		IconGold: top("gold"), IconWisdom: top("wisdom"), IconRenown: top("renown"), IconFood: top("food"), IconHousing: top("housing"),
		IconBox: box("parcel", false), IconParcel: box("parcel", false), IconLedger: box("ledger", false), IconCrate: box("crate", false),
		IconDispatch: box("dispatch", false), IconSealed: box("parcel", true), IconCellar: top("overlay-midden"),
	} {
		src, ok := paint()
		if !ok {
			t.Fatalf("no original sprite for %q", icon)
		}
		want := nearestDown(src)
		for y := 0; y < iconRows; y++ {
			for x := 0; x < iconWidth; x++ {
				got := iconPixel(icon, x, y)
				w, inked := want[[2]int{x, y}]
				switch {
				case inked && got != w:
					t.Fatalf("%q pixel %d,%d = %v, want the sprite's %v", icon, x, y, got, w)
				case !inked && got != nil:
					t.Fatalf("%q pixel %d,%d = %v, want empty", icon, x, y, got)
				}
			}
		}
	}
}

func TestTheWheatArtStaysForASwitchBack(t *testing.T) {
	if src, ok := art.PaintIcon("wheat"); !ok || len(nearestDown(src)) < 20 {
		t.Fatal("the wheat sheaf art is gone")
	}
}
