package art

import (
	"image/color"
	"math"
)

var (
	sealR     = ramp{rgb(90, 14, 24), rgb(142, 24, 34), rgb(190, 40, 44), rgb(232, 100, 88)}
	freshCols = []color.NRGBA{rgb(112, 176, 88), rgb(214, 196, 64), rgb(236, 142, 50), rgb(214, 72, 60)}
	mouldR    = ramp{rgb(62, 78, 50), rgb(92, 104, 74), rgb(128, 138, 102), rgb(170, 178, 138)}
)

var crownGlyphs = [][]string{
	{"#.#.#", "#####"},
	{"#.#.#", "#####", "#####"},
	{"#..#..#", "#.###.#", "#######", "#######"},
	{"#..#..#", "##.#.##", "#######", "#######", "#######"},
}

var hourglassSmall = []string{"###", "sss", ".n.", "SSS", "###"}
var hourglassLarge = []string{"#####", ".sss.", ".sss.", "..n..", ".SSS.", ".SSS.", "#####"}

func paintSealMark(c *canvas, sz size, d boxDims, n int) {
	cx := float64(sz.ax)
	cy := float64(sz.ay) + d.hh() - d.h*0.45
	r := []float64{1.6, 2.1, 2.6, 3.2}[clampSize(n)]
	x, y := int(math.Floor(cx)), int(math.Floor(cy+r))
	c.line(x-1, y-1, x-2, y+1, twineR[2])
	c.line(x, y-1, x+1, y+1, twineR[3])
	c.disc(cx, cy, r, sealR[1])
	c.disc(cx-0.4, cy-0.4, math.Max(r-0.9, 0.8), sealR[2])
	c.dot(int(cx-r*0.5), int(cy-r*0.5), sealR[3])
	if r >= 2 {
		c.dot(int(cx), int(cy), sealR[0])
	}
	c.outline(ink)
}

func paintCrownMark(c *canvas, sz size, d boxDims, n int) {
	g := crownGlyphs[clampSize(n)]
	w := len(g[0])
	x0 := sz.ax - w/2
	y0 := sz.ay - int(d.h) - len(g)/2 - 1
	for j, row := range g {
		col := goldR[2]
		switch {
		case j == 0:
			col = goldR[3]
		case j == len(g)-1:
			col = goldR[1]
		}
		glyph(c, []string{row}, x0, y0+j, col)
	}
	if len(g) >= 3 {
		c.dot(sz.ax, y0+len(g)-2, redR[2])
	}
	c.outline(ink)
}

func paintFreshMark(c *canvas, sz size, d boxDims, n, level int) {
	g := hourglassSmall
	if n >= 2 {
		g = hourglassLarge
	}
	cx := float64(sz.ax) + 0.72*d.hw()
	cy := float64(sz.ay) + 0.5*d.hh() - 0.62*d.h
	x0 := int(math.Round(cx)) - len(g[0])/2
	y0 := int(math.Round(cy)) - len(g)/2
	top, bottom := sandSplit(g, level)
	sand := freshCols[min(max(level, 0), FreshLevels-1)]
	for j, row := range g {
		for i, ch := range row {
			if col, ok := hourglassPixel(ch, j, top, bottom, level, sand); ok {
				c.dot(x0+i, y0+j, col)
			}
		}
	}
	c.outline(ink)
}

func sandSplit(g []string, level int) (map[int]bool, map[int]bool) {
	var tops, bottoms []int
	for j, row := range g {
		switch {
		case row[len(row)/2] == 's':
			tops = append(tops, j)
		case row[len(row)/2] == 'S':
			bottoms = append(bottoms, j)
		}
	}
	keep := max(0, int(math.Round(float64(len(tops))*(1-float64(level)/2))))
	top, bottom := map[int]bool{}, map[int]bool{}
	for i := len(tops) - keep; i < len(tops); i++ {
		top[tops[i]] = true
	}
	for i := len(bottoms) - (len(tops) - keep); i < len(bottoms); i++ {
		bottom[bottoms[max(i, 0)]] = true
	}
	return top, bottom
}

func hourglassPixel(ch rune, j int, top, bottom map[int]bool, level int, sand color.NRGBA) (color.NRGBA, bool) {
	switch ch {
	case '#':
		return woodR[1], true
	case 's':
		if top[j] {
			return sand, true
		}
		return glassR[3], true
	case 'S':
		if bottom[j] {
			return sand, true
		}
		return glassR[3], true
	case 'n':
		if level > 0 && level < FreshLevels-1 {
			return sand, true
		}
		return glassR[2], true
	}
	return color.NRGBA{}, false
}

func rot(c *canvas, kind string, v int) {
	seed := 700 + v*13 + len(kind)
	c.mapPixels(func(col color.NRGBA) color.NRGBA {
		return darken(mix(col, mouldR[2], 0.55), 0.08)
	})
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.at(x, y).A == 0 {
				continue
			}
			switch hash(x, y, seed) % 13 {
			case 0:
				c.dot(x, y, mouldR[0])
			case 1:
				c.dot(x, y, mouldR[3])
			}
		}
	}
}

func paintSpoil(c *canvas, f int) {
	stink := rgb(128, 170, 58)
	for i, base := range []float64{4.5, 10.5} {
		for y := 3; y <= 14; y++ {
			x := base + 1.3*math.Sin(float64(y+f*2+i*3)*0.75)
			c.dot(int(x), y, withAlpha(stink, uint8(70+13*(y-3))))
		}
	}
	for i := 0; i < 3; i++ {
		a := float64(f)*math.Pi/3 + float64(i)*2*math.Pi/3
		x := 8 + 5*math.Cos(a)
		y := 7 + 3*math.Sin(a) - float64(i)
		c.dot(int(x), int(y), ink)
		wing := int(y) - 1 + (f+i)%2
		c.dot(int(x)-1+(f+i)%2, wing, withAlpha(white, 190))
	}
}
