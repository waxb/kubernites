package art

import (
	"fmt"
	"image/color"
	"math"
)

const (
	FishFrames    = 6
	LeyFrames     = 6
	CharterFrames = 4
	Blots         = 3
)

var silverR = ramp{rgb(58, 78, 98), rgb(118, 146, 168), rgb(188, 208, 218), white}

func BlotName(size int) string {
	return fmt.Sprintf("blot/%d", min(max(size, 0), Blots-1))
}

func addBlots(b *atlasBuilder) {
	for i := 0; i < Blots; i++ {
		rx, ry := 1.6+float64(i)*1.2, 0.8+float64(i)*0.35
		w, h := int(rx*2)+2, int(ry*2)+2
		b.add(BlotName(i), w, h, w/2, h/2, func(c *canvas) {
			c.ellipse(float64(w)/2, float64(h)/2, rx, ry, withAlpha(shadow, 70))
		})
	}
}

func ripple(c *canvas, rx, ry float64, a uint8) {
	for k := 0; k < 24; k++ {
		ang := float64(k) * math.Pi / 12
		c.dot(int(8+math.Cos(ang)*rx), int(13+math.Sin(ang)*ry), withAlpha(shallowR[3], a))
	}
}

func fishBody(c *canvas, x, y float64, tilt float64) {
	for i := -3; i <= 3; i++ {
		px, py := x+float64(i), y+float64(i)*tilt
		col := silverR[2]
		if i > 1 {
			col = silverR[1]
		}
		c.dot(int(px), int(py), col)
		if i > -3 && i < 2 {
			c.dot(int(px), int(py)+1, silverR[1])
		}
	}
	c.dot(int(x-4), int(y-4*tilt)-1, orangeR[2])
	c.dot(int(x-4), int(y-4*tilt)+1, orangeR[2])
	c.dot(int(x+2), int(y+2*tilt), ink)
}

func paintFish(c *canvas, f int) {
	switch f {
	case 0:
		ripple(c, 2.5, 0.8, 200)
	case 1:
		ripple(c, 3.5, 1.1, 170)
		fishBody(c, 7, 10, -0.6)
		c.dot(4, 11, white)
		c.dot(11, 12, white)
	case 2:
		fishBody(c, 8, 7, 0)
		c.dot(5, 10, withAlpha(white, 200))
		c.dot(12, 11, withAlpha(white, 200))
	case 3:
		fishBody(c, 9, 10, 0.6)
		ripple(c, 2, 0.7, 150)
	case 4:
		for _, p := range [][2]int{{6, 10}, {8, 9}, {10, 10}, {5, 12}, {11, 12}} {
			c.dot(p[0], p[1], white)
		}
		ripple(c, 3, 1, 220)
	default:
		ripple(c, 5, 1.6, 120)
	}
}

func paintLeyGlow(c *canvas, f int) {
	pulse := 0.5 + 0.5*math.Sin(float64(f)*math.Pi/3)
	c.ellipse(8, 14, 5+pulse, 1.8, withAlpha(runeR[1], uint8(70+60*pulse)))
	c.ellipse(8, 14, 2.5, 0.9, withAlpha(runeR[2], uint8(90+80*pulse)))
	for k := 0; k < 4; k++ {
		rise := float64((f+k*2)%LeyFrames) * 2
		x := 4 + (k*5)%9
		y := int(12 - rise)
		a := uint8(250 - rise*12)
		c.dot(x, y, withAlpha(runeR[3], a))
		c.dot(x, y+1, withAlpha(runeR[2], a/2))
		if rise < 4 {
			c.dot(x-1, y, withAlpha(runeR[2], a/2))
			c.dot(x+1, y, withAlpha(runeR[2], a/2))
		}
	}
}

func paintCharter(c *canvas, f int) {
	for _, s := range [][3]float64{{9, 37, 3}, {15, 37.5, 2.5}, {12, 35, 2.4}} {
		c.ellipse(s[0], s[1], s[2], s[2]*0.7, stoneR[1+int(s[2])%2])
	}
	c.rect(11, 6, 2, 31, woodR[1])
	c.rect(11, 6, 1, 31, woodR[2])
	c.rect(6, 8, 13, 2, woodR[1])
	c.dot(12, 4, goldR[3])
	c.rect(12, 5, 1, 1, goldR[2])
	wave := []float64{0, 1, 0, -1}[f]
	c.shadePoly([]pt{{7, 10}, {17, 10}, {17 + wave, 24}, {12 + wave*0.5, 21}, {7 + wave, 24}}, func(x, y int) color.NRGBA {
		switch {
		case x == 7 || x == 16 || y == 10:
			return tkGold
		case (x+y)%7 == 0:
			return tkParchmentDark
		}
		return tkParchment
	})
	c.disc(12+wave*0.3, 15.5, 2.2, redR[2])
	c.dot(int(12+wave*0.3), 15, redR[3])
	c.outline(ink)
}
