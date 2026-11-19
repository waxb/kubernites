package gfx

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/waxb/kubernites/pkg/client/art"
)

const (
	gildPrefix    = "gild/"
	GildedClass   = "caravel"
	GildedVariant = 2
	GildedBoxSize = 2
	sailLum       = .68
	sailSpread    = 80
	trimBlue      = 40
	trimOverGreen = 20
	pineLum       = .56
	battenLum     = .3
	goldLo        = .56
	goldHi        = .95
)

var (
	gildRim   = color.NRGBA{255, 244, 176, 255}
	sailScale = [3]float64{1, .93, .6}
	trimGold  = [2]color.NRGBA{{150, 104, 28, 255}, {236, 190, 70, 255}}
	faceGold  = [2]color.NRGBA{{214, 164, 56, 255}, {255, 236, 136, 255}}
	battenTo  = [3]float64{.75, .66, .4}
)

func GildedName(name string) string { return gildPrefix + name }

func lum(c color.NRGBA) float64 {
	return (.3*float64(c.R) + .59*float64(c.G) + .11*float64(c.B)) / 255
}

func warm(c color.NRGBA) bool { return c.R >= c.G && c.G >= c.B }

func spread(c color.NRGBA) int { return int(c.R) - int(c.B) }

func scaled(c color.NRGBA, k [3]float64) color.NRGBA {
	return color.NRGBA{uint8(float64(c.R) * k[0]), uint8(float64(c.G) * k[1]), uint8(float64(c.B) * k[2]), c.A}
}

func ramp(pair [2]color.NRGBA, t float64, a uint8) color.NRGBA {
	t = min(max(t, 0), 1)
	mix := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.NRGBA{mix(pair[0].R, pair[1].R), mix(pair[0].G, pair[1].G), mix(pair[0].B, pair[1].B), a}
}

func blueTrim(c color.NRGBA) bool {
	return int(c.B)-int(c.R) > trimBlue && int(c.B)-int(c.G) > trimOverGreen
}

func GildShipPixel(c color.NRGBA) color.NRGBA {
	switch {
	case c.A < 255:
		return c
	case warm(c) && lum(c) >= sailLum && spread(c) <= sailSpread:
		return scaled(c, sailScale)
	case blueTrim(c):
		return ramp(trimGold, lum(c)/.5, c.A)
	}
	return c
}

func GildBoxPixel(c color.NRGBA) color.NRGBA {
	switch l := lum(c); {
	case c.A < 255 || !warm(c) || l < battenLum:
		return c
	case l < pineLum:
		return scaled(c, battenTo)
	default:
		return ramp(faceGold, (l-goldLo)/(goldHi-goldLo), c.A)
	}
}

func gildRegion(src *image.NRGBA, r art.Region, pixel func(color.NRGBA) color.NRGBA, rim bool) (*image.NRGBA, art.Region) {
	pad := 0
	if rim {
		pad = 1
	}
	out := image.NewNRGBA(image.Rect(0, 0, r.W+2*pad, r.H+2*pad))
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			out.SetNRGBA(x+pad, y+pad, pixel(src.NRGBAAt(r.X+x, r.Y+y)))
		}
	}
	if rim {
		addRim(out)
	}
	return out, art.Region{W: r.W + 2*pad, H: r.H + 2*pad, AnchorX: r.AnchorX + pad, AnchorY: r.AnchorY + pad}
}

func solid(img *image.NRGBA, x, y int) bool {
	return image.Pt(x, y).In(img.Bounds()) && img.NRGBAAt(x, y).A == 255
}

func addRim(img *image.NRGBA) {
	b := img.Bounds()
	marks := []image.Point{}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.NRGBAAt(x, y).A == 0 && (solid(img, x-1, y) || solid(img, x+1, y) || solid(img, x, y-1) || solid(img, x, y+1)) {
				marks = append(marks, image.Pt(x, y))
			}
		}
	}
	for _, p := range marks {
		img.SetNRGBA(p.X, p.Y, gildRim)
	}
}

func gilder(name string) (func(color.NRGBA) color.NRGBA, bool, bool) {
	switch {
	case strings.HasPrefix(name, fmt.Sprintf("ship/%s/%d/", GildedClass, GildedVariant)):
		return GildShipPixel, false, true
	case name == BoxName(GildedBoxSize, "parcel", 0):
		return GildBoxPixel, true, true
	}
	return nil, false, false
}

func (l *Library) addGilded(a art.Atlas) {
	if a.Image == nil {
		return
	}
	for _, name := range art.Names(a) {
		r := a.Regions[name]
		pixel, rim, ok := gilder(name)
		if !ok {
			continue
		}
		img, region := gildRegion(a.Image, r, pixel, rim)
		l.index[GildedName(name)] = newEntry(newImage(img), img, region)
	}
}
