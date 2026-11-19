package art

import "math"

func fireball(c *canvas, x, y, r float64) {
	if r <= 0 {
		return
	}
	c.disc(x, y, r, fireR[0])
	c.disc(x-r*0.15, y-r*0.15, r*0.78, fireR[1])
	c.disc(x-r*0.25, y-r*0.25, r*0.52, fireR[2])
	c.disc(x-r*0.3, y-r*0.3, math.Max(r*0.26, 0.8), fireR[3])
}

func paintExplosion(c *canvas, f int) {
	c.ellipse(12, 20, 8, 2, shadow)
	switch {
	case f == 0:
		for k := 0; k < 8; k++ {
			a := float64(k) * math.Pi / 4
			c.line(int(12+math.Cos(a)*4), int(14+math.Sin(a)*4), int(12+math.Cos(a)*8), int(14+math.Sin(a)*8), fireR[3])
		}
		fireball(c, 12, 14, 3.5)
	case f <= 2:
		fireball(c, 12, 14-float64(f), 5.5+float64(f)*1.5)
	case f <= 4:
		puff(c, 12, 11-float64(f-3)*2, 7.5, uint8(235-(f-3)*45), greyR)
		fireball(c, 12, 15, float64(5-f)*2.4)
	default:
		for _, p := range [][2]float64{{8, 9}, {16, 10}, {12, 6}} {
			puff(c, p[0], p[1], 3.6, 120, greyR)
		}
	}
	if f <= 2 {
		c.outline(darken(fireR[0], 0.5))
	}
	if f >= 2 {
		for k := 0; k < 6; k++ {
			a := float64(k)*math.Pi/3 + 0.4
			d := 5 + float64(f)*1.7
			c.dot(int(12+math.Cos(a)*d), int(13+math.Sin(a)*d*0.7+float64(f*f)*0.2), []ramp{woodR, rockR}[k%2][0])
		}
	}
}

func paintHeal(c *canvas, f int) {
	for i, x := range []int{4, 12, 8} {
		ph := (f + i*2) % 6
		y := 13 - ph*2
		a := uint8(255 - ph*28)
		arm := 1
		if ph < 3 {
			arm = 2
		}
		c.rect(x-arm, y, arm*2+1, 1, withAlpha(greenR[2], a))
		c.rect(x, y-arm, 1, arm*2+1, withAlpha(greenR[2], a))
		c.dot(x, y, withAlpha(white, a))
	}
	c.outline(darken(greenR[0], 0.3))
	if f < 4 {
		ground := newCanvas(c.w, c.h)
		ground.ellipse(8, 14, 6-float64(f)*0.4, 2, withAlpha(greenR[3], uint8(110-f*20)))
		underlay(c, ground)
	}
	twinkle(c, []int{2, 14, 6, 10, 3, 13}[f], 2+f%3, 1, withAlpha(greenR[3], 220))
}

func underlay(c, src *canvas) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if c.at(x, y).A == 0 {
				c.dot(x, y, src.at(x, y))
			}
		}
	}
}
