package art

import (
	"image/color"
	"math"
)

var caskR = ramp{rgb(84, 50, 34), rgb(128, 80, 48), rgb(170, 114, 66), rgb(206, 152, 94)}

type cask struct {
	ox, oy, r, ry, h float64
	v                int
}

func (k cask) radius(t float64) float64 {
	return k.r * (0.84 + 0.16*math.Sin(math.Pi*clamp01(t)))
}

func drawCask(c *canvas, ox, oy float64, d boxDims, v int) {
	k := cask{ox: ox, oy: oy, r: d.hw(), ry: d.hh(), h: d.h, v: v}
	for y := int(oy - k.h - k.ry - 1); y <= int(oy+k.ry+1); y++ {
		for x := int(ox - k.r - 1); x <= int(ox+k.r+1); x++ {
			if col, ok := k.body(float64(x)+0.5-ox, float64(y)+0.5); ok {
				c.dot(x, y, col)
			}
		}
	}
	k.lid(c)
}

func (k cask) body(dx, py float64) (color.NRGBA, bool) {
	t := (k.oy - py) / k.h
	if t > 1 || math.Abs(dx) > k.radius(t) {
		return color.NRGBA{}, false
	}
	e := 1 - (dx/k.r)*(dx/k.r)
	if e < 0 {
		return color.NRGBA{}, false
	}
	z := k.oy + k.ry*math.Sqrt(e) - py
	if z < 0 {
		return color.NRGBA{}, false
	}
	return k.shade(dx, int(math.Floor(z))), true
}

func (k cask) shade(dx float64, zi int) color.NRGBA {
	light := -dx / k.r
	tone := func(r ramp) color.NRGBA {
		switch {
		case light > 0.45:
			return r[3]
		case light > -0.2:
			return r[2]
		case light > -0.7:
			return r[1]
		}
		return r[0]
	}
	switch {
	case caskHoop(k.v, zi, int(k.h)):
		return tone(greyR)
	case caskBand(k.v, zi, int(k.h)):
		return tone(purpleR)
	case k.v == 2 && caskStencil(dx, zi, k):
		return tone(purpleR)
	case k.v == 1 && math.Abs(dx) < 1 && zi == 2:
		return ink
	case k.r >= 6 && caskStave(dx, k.r):
		return darken(tone(caskR), 0.22)
	}
	return tone(caskR)
}

func caskHoop(v, z, h int) bool {
	if z == 1 || z == h-2 {
		return true
	}
	return v == 1 && h >= 7 && z == h/2+1
}

func caskBand(v, z, h int) bool {
	if v == 2 {
		return false
	}
	mid := h/2 - 1
	if v == 1 {
		mid = h/2 - 1 - min(h/5, 1)
	}
	return z == mid || (h >= 10 && z == mid+1)
}

func caskStencil(dx float64, zi int, k cask) bool {
	cx := -k.r * 0.45
	cz := float64(k.h) / 2
	return math.Abs(dx-cx)+math.Abs(float64(zi)+0.5-cz)*0.9 < math.Max(k.r*0.22, 1)
}

func caskStave(dx, r float64) bool {
	theta := math.Asin(math.Max(-1, math.Min(1, dx/r)))
	n := math.Max(3, math.Round(r/2.5))
	f := (theta/math.Pi + 0.5) * n
	return f-math.Floor(f) < 0.12
}

func (k cask) lid(c *canvas) {
	rx := k.radius(1)
	cy := k.oy - k.h
	for y := int(cy - k.ry - 1); y <= int(cy+k.ry+1); y++ {
		for x := int(k.ox - rx - 1); x <= int(k.ox+rx+1); x++ {
			dx, dy := (float64(x)+0.5-k.ox)/rx, (float64(y)+0.5-cy)/(k.ry*rx/k.r)
			q := dx*dx + dy*dy
			switch {
			case q > 1:
			case q > 0.55:
				c.dot(x, y, caskR[2])
			default:
				c.dot(x, y, caskR[3])
			}
		}
	}
	k.lidMarks(c, cy)
}

func (k cask) lidMarks(c *canvas, cy float64) {
	x, y := int(math.Floor(k.ox)), int(math.Floor(cy))
	switch k.v {
	case 1:
		c.dot(x, y, purpleR[2])
	case 3:
		for i, dx := range []int{-2, 0, 2} {
			if k.r < 6 && i != 1 {
				continue
			}
			c.dot(x+dx-1, y, twineR[3])
			c.dot(x+dx-1, y+1, twineR[1])
		}
		c.dot(x, y+1, redR[2])
	}
}
