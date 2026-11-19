package art

import (
	"image"
	"image/color"
	"math"
)

const (
	AppIconName = "appicon"
	AppIconSize = 32
)

var tkInk = rgb(43, 37, 32)

func AppIcon() *image.NRGBA {
	c := newCanvas(AppIconSize, AppIconSize)
	paintAppIcon(c)
	return c.img
}

func paintAppIcon(c *canvas) {
	appTile(c)
	shield := newCanvas(c.w, c.h)
	appShield(shield)
	shield.outline(tkInk)
	c.paste(shield, 0, 0, false)
	wheel := newCanvas(c.w, c.h)
	appWheel(wheel)
	wheelOutline(wheel)
	c.paste(wheel, 0, 0, false)
}

func inRoundedSquare(x, y, inset int, r float64) bool {
	lo, hi := float64(inset), float64(AppIconSize-inset)
	px, py := float64(x)+0.5, float64(y)+0.5
	if px < lo || px > hi || py < lo || py > hi {
		return false
	}
	cx, cy := math.Max(lo+r, math.Min(px, hi-r)), math.Max(lo+r, math.Min(py, hi-r))
	return math.Hypot(px-cx, py-cy) <= r
}

func appTile(c *canvas) {
	band := mix(tkParchment, tkParchmentDark, 0.45)
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			switch {
			case inRoundedSquare(x, y, 2, 2.5):
				c.dot(x, y, tkParchment)
			case inRoundedSquare(x, y, 1, 3.5):
				c.dot(x, y, band)
			case inRoundedSquare(x, y, 0, 4.5):
				c.dot(x, y, tkInk)
			}
		}
	}
}

func shieldHalfWidth(y int) float64 {
	py := float64(y) + 0.5
	if py < 17 {
		return 11
	}
	t := (py - 17) / 11.5
	return 11 * (1 - math.Pow(t, 1.8))
}

func inShield(x, y int) bool {
	top := 4
	if x >= 11 && x <= 20 {
		top = 5
	}
	return y >= top && y <= 28 && math.Abs(float64(x)+0.5-16) <= shieldHalfWidth(y)
}

func appShield(c *canvas) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if !inShield(x, y) {
				continue
			}
			col := blueR[1]
			if !inShield(x-1, y) || !inShield(x+1, y) || !inShield(x, y-1) || !inShield(x, y+1) {
				col = blueR[0]
			}
			c.dot(x, y, col)
		}
	}
	c.dot(7, 6, blueR[3])
}

const wheelX, wheelY = 16.0, 15.0

func appWheel(c *canvas) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			dx, dy := float64(x)+0.5-wheelX, float64(y)+0.5-wheelY
			d := math.Hypot(dx, dy)
			if (d >= 6.0 && d <= 7.9) || onSpoke(dx, dy, d) {
				c.dot(x, y, wheelGold(dx, dy))
			}
		}
	}
	for k := 0; k < 8; k++ {
		a := float64(k) * math.Pi / 4
		kx, ky := wheelX+math.Cos(a)*10.2, wheelY+math.Sin(a)*10.2
		c.disc(kx, ky, 1.3, tkGold)
		c.dot(int(kx-0.5), int(ky-0.5), tkGoldBright)
	}
	c.disc(wheelX, wheelY, 2.3, tkGold)
	c.disc(wheelX, wheelY, 1.5, tkRed)
	c.dot(15, 14, mix(tkRed, white, 0.35))
}

func wheelOutline(c *canvas) {
	src := newCanvas(c.w, c.h)
	src.paste(c, 0, 0, false)
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if src.at(x, y).A > 0 || !(src.at(x-1, y).A > 0 || src.at(x+1, y).A > 0 || src.at(x, y-1).A > 0 || src.at(x, y+1).A > 0) {
				continue
			}
			if math.Hypot(float64(x)+0.5-wheelX, float64(y)+0.5-wheelY) > 7.9 {
				c.dot(x, y, tkInk)
			}
		}
	}
}

func onSpoke(dx, dy, d float64) bool {
	if d < 2 || d > 9.6 {
		return false
	}
	for k := 0; k < 4; k++ {
		a := float64(k) * math.Pi / 4
		width := 0.5
		if k%2 == 1 {
			width = 0.62
		}
		if math.Abs(dx*math.Sin(a)-dy*math.Cos(a)) <= width {
			return true
		}
	}
	return false
}

func wheelGold(dx, dy float64) color.NRGBA {
	if dx+dy < -3 {
		return tkGoldBright
	}
	return tkGold
}
