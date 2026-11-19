package art

import (
	"fmt"
	"image/color"
	"math"
)

const (
	KMapCell       = 16
	KMapParchment  = 64
	PageIconSize   = 24
	MedallionSize  = 30
	BannerW        = 32
	BannerH        = 48
	FlagW          = 46
	FlagH          = 26
	FlagTuck       = 10
	flagNotch      = 7
	PlaqueW        = 48
	PlaqueH        = 40
	PlaqueSlice    = 12
	KMapFrameSize  = 96
	KMapFrameSlice = 24
	kmapPad        = 2
)

var kmapMargin = margin{kmapPad, kmapPad, kmapPad, kmapPad}

var KMapStates = []string{"locked", "available", "researching", "learned"}

var KMapRoadStates = []string{"locked", "open", "learned"}

var KMapBranches = []string{"networking", "scheduling", "reliability", "upkeep"}

var KMapDecor = []string{"compass", "hills", "forest", "waves", "mountains"}

var kmapDecorSizes = map[string][2]int{"compass": {40, 40}, "hills": {28, 14}, "forest": {26, 16}, "waves": {16, 6}, "mountains": {32, 18}}

var (
	mapInk   = rgb(92, 64, 40)
	mapInkLt = color.NRGBA{92, 64, 40, 150}
)

func KMapRoadName(state, mask string) string { return fmt.Sprintf("kmap/road/%s/%s", state, mask) }
func KMapBannerName(branch string) string    { return "kmap/banner/" + branch }
func KMapFlagName(branch string) string      { return "kmap/flag/" + branch }
func KMapMedallionName(state string) string  { return "kmap/medallion/" + state }
func KMapDecorName(kind string) string       { return "kmap/deco/" + kind }
func PageIconName(id string) string          { return "page/" + id }
func PlaqueName(state string) string         { return "plaque-" + state }
func KMapParchmentName() string              { return "kmap/parchment" }

func addKnowledgeMap(b *atlasBuilder) {
	b.add(KMapParchmentName(), KMapParchment, KMapParchment, 0, 0, paintParchmentTile)
	for _, st := range KMapRoadStates {
		for _, m := range KMapRoadMasks {
			state, mask := st, m
			b.add(KMapRoadName(st, m), KMapCell, KMapCell, 0, 0, func(c *canvas) { paintRoadTile(c, state, mask) })
		}
	}
	for _, br := range KMapBranches {
		branch := br
		b.add(KMapBannerName(br), BannerW+2*kmapPad, BannerH+2*kmapPad, kmapPad, kmapPad, func(c *canvas) { paintBranchBanner(c, branch) })
		b.add(KMapFlagName(br), FlagW+2*kmapPad, FlagH+2*kmapPad, kmapPad, kmapPad, func(c *canvas) { paintBranchFlag(c, branch) })
	}
	for _, st := range KMapStates {
		state := st
		b.add(KMapMedallionName(st), MedallionSize, MedallionSize, 0, 0, func(c *canvas) { paintMedallion(c, state) })
	}
	for _, k := range KMapDecor {
		kind, sz := k, kmapDecorSizes[k]
		b.add(KMapDecorName(k), sz[0]+2*kmapPad, sz[1]+2*kmapPad, 0, 0, func(c *canvas) {
			paintInset(c, sz[0], sz[1], kmapMargin, kmapDecorPainters[kind])
		})
	}
	for _, id := range PageIcons {
		paint := pagePainters[id]
		b.add(PageIconName(id), PageIconSize, PageIconSize, PageIconSize/2, PageIconSize/2, func(c *canvas) {
			paint(c)
			c.outline(ink)
		})
	}
}

func wrapNoise(x, y, period, cell, seed int) float64 {
	n := period / cell
	fx, fy := float64(x)/float64(cell), float64(y)/float64(cell)
	x0, y0 := int(math.Floor(fx)), int(math.Floor(fy))
	tx, ty := fx-float64(x0), fy-float64(y0)
	v := func(i, j int) float64 { return noise(((i%n)+n)%n, ((j%n)+n)%n, seed) }
	sx, sy := tx*tx*(3-2*tx), ty*ty*(3-2*ty)
	a := v(x0, y0)*(1-sx) + v(x0+1, y0)*sx
	b := v(x0, y0+1)*(1-sx) + v(x0+1, y0+1)*sx
	return a*(1-sy) + b*sy
}

func parchmentAt(x, y int) color.NRGBA {
	n := wrapNoise(x, y, KMapParchment, 16, 801)*0.7 + wrapNoise(x, y, KMapParchment, 8, 803)*0.3
	col := mix(tkParchment, tkParchmentDark, math.Max(0, n-0.35)*0.34)
	switch h := hash(x%KMapParchment, y%KMapParchment, 807) % 97; {
	case h == 0:
		return mix(col, tkParchmentDark, 0.45)
	case h < 3 && (x+y)%3 == 0:
		return mix(col, cream, 0.35)
	}
	return col
}

func paintParchmentTile(c *canvas) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			c.dot(x, y, parchmentAt(x, y))
		}
	}
}

func paintKMapFrame(c *canvas) {
	c.rect(0, 0, c.w, c.h, tkParchment)
	ring(c, 0, ink, ink)
	woodBorder(c, 1, 3)
	ring(c, 4, ink, ink)
	ring(c, 5, mix(tkParchment, tkParchmentDark, 0.5), mix(tkParchment, tkParchmentDark, 0.5))
	ring(c, 8, mapInk, mapInk)
	for y := 11; y < c.h-11; y++ {
		for x := 11; x < c.w-11; x++ {
			erase(c, x, y)
		}
	}
	ring(c, 10, mapInkLt, mapInkLt)
	for _, f := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
		cornerRose(c, f[0], f[1])
	}
}

func cornerRose(c *canvas, fx, fy bool) {
	px := func(v float64) float64 {
		if fx {
			return float64(c.w) - v
		}
		return v
	}
	py := func(v float64) float64 {
		if fy {
			return float64(c.h) - v
		}
		return v
	}
	cx, cy := px(15), py(15)
	c.shadePoly([]pt{{cx, cy - 6}, {cx + 1.5, cy}, {cx, cy + 6}, {cx - 1.5, cy}}, flat(tkGold))
	c.shadePoly([]pt{{cx - 6, cy}, {cx, cy - 1.5}, {cx + 6, cy}, {cx, cy + 1.5}}, flat(tkGold))
	c.disc(cx, cy, 1.6, tkRed)
	c.dot(int(cx), int(py(8.5)), tkGoldBright)
	c.dot(int(px(8.5)), int(cy), tkGoldBright)
}

func paintKMapRibbon(c *canvas) {
	c.shadePoly([]pt{{0, 3}, {8, 3}, {8, 15}, {0, 15}, {4, 9}}, flat(tkParchmentDark))
	c.shadePoly([]pt{{float64(c.w), 3}, {float64(c.w) - 8, 3}, {float64(c.w) - 8, 15}, {float64(c.w), 15}, {float64(c.w) - 4, 9}}, flat(tkParchmentDark))
	c.rect(5, 1, c.w-10, c.h-3, tkParchment)
	c.rect(5, 1, c.w-10, 1, cream)
	c.rect(5, c.h-3, c.w-10, 1, tkParchmentDark)
	c.rect(5, 2, c.w-10, 1, mix(tkParchment, cream, 0.5))
	c.outline(ink)
}
