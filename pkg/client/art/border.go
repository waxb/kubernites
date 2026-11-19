package art

import (
	"fmt"
	"image/color"
)

const (
	BorderStones = 3
	BorderFlags  = 2
	TintW        = TileWidth
	TintH        = TileHeight
)

var GroundTints = []string{"good", "poor", "own", "forum"}

var tintColors = map[string]color.NRGBA{
	"good": {176, 244, 112, 104}, "poor": {226, 74, 58, 112}, "own": {255, 226, 140, 58}, "forum": {250, 244, 226, 150},
}

var markerR = ramp{rgb(112, 108, 112), rgb(160, 156, 152), rgb(206, 202, 192), rgb(238, 234, 222)}

func BorderStoneName(v int) string { return fmt.Sprintf("border/stone/%d", v) }
func BorderFlagName(f int) string  { return fmt.Sprintf("border/flag/%d", f) }
func GroundTintName(k string) string {
	return "ground/tint/" + k
}

func addBorders(b *atlasBuilder) {
	for v := 0; v < BorderStones; v++ {
		vv := v
		b.add(BorderStoneName(v), 12, 16, 6, 13, func(c *canvas) { paintBorderStone(c, vv) })
	}
	for f := 0; f < BorderFlags; f++ {
		ff := f
		b.add(BorderFlagName(f), 18, 30, 5, 27, func(c *canvas) { paintBorderFlag(c, ff) })
	}
}

func addGroundTints(b *atlasBuilder) {
	for _, k := range GroundTints {
		col := tintColors[k]
		b.add(GroundTintName(k), TintW, TintH, TintW/2, TintH/2, func(c *canvas) { paintTint(c, col) })
	}
}

func paintTint(c *canvas, col color.NRGBA) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			a, b := diamondCoords(x, y)
			if absf(a)+absf(b) < 1 {
				c.dot(x, y, col)
			}
		}
	}
}

func markerStone(c *canvas, x, base, w, h int) {
	c.rect(x, base-h, w, h, markerR[1])
	c.rect(x, base-h, max(w/2, 1), h, markerR[2])
	c.rect(x+1, base-h-1, w-2, 1, markerR[3])
	c.rect(x, base-h, w, 1, markerR[3])
	c.rect(x, base-2, w, 1, markerR[0])
}

func paintBorderStone(c *canvas, v int) {
	switch v {
	case 1:
		markerStone(c, 4, 13, 5, 5)
		c.dot(6, 10, markerR[0])
	case 2:
		markerStone(c, 3, 13, 4, 5)
		markerStone(c, 6, 14, 4, 4)
	default:
		markerStone(c, 4, 13, 4, 7)
		c.rect(4, 9, 4, 1, purpleR[2])
	}
	c.dot(3, 13, greenR[3])
	c.outline(ink)
	propShadow(c, 6, 13, 3.5)
}

func paintBorderFlag(c *canvas, f int) {
	markerStone(c, 3, 27, 5, 4)
	c.rect(5, 8, 1, 15, woodR[1])
	c.dot(5, 7, goldR[3])
	wave := []float64{0, 1}[f]
	c.shadePoly([]pt{{6, 8}, {11, 9 + wave}, {15, 9}, {12.5, 11.5 + wave}, {15, 14}, {10.5, 13 - wave}, {6, 13.5}}, func(x, y int) color.NRGBA {
		if (x+y+f)%4 == 0 {
			return purpleR[1]
		}
		return purpleR[2]
	})
	c.rect(6, 10, 3, 2, goldR[2])
	c.outline(ink)
	propShadow(c, 5.5, 27, 3.5)
}
