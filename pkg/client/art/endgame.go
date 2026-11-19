package art

import "math"

func hornBody(c *canvas, x, y float64, metal ramp) {
	c.shadePoly([]pt{{x - 6, y - 1}, {x, y - 4}, {x + 1, y + 4}, {x - 6, y + 1}}, flat(metal[2]))
	c.line(int(x)-6, int(y)-1, int(x), int(y)-4, metal[3])
	c.rect(int(x)-3, int(y)-2, 1, 4, metal[1])
	c.ellipse(x+0.5, y, 1.6, 4.2, metal[1])
	c.ellipse(x+1, y, 0.8, 3, metal[0])
}

func paintHornFx(c *canvas, f int) {
	horn := newCanvas(c.w, c.h)
	hornBody(horn, 8, 12, brassR)
	horn.outline(ink)
	c.paste(horn, 0, 0, false)
	for k := 0; k < 3; k++ {
		r := 4 + math.Mod(float64(f)*1.4+float64(k)*4.2, 12.6)
		a := uint8(math.Max(60, 255-(r-4)*15))
		col := withAlpha(tkGoldBright, a)
		if k%2 == 1 {
			col = withAlpha(tkParchment, a)
		}
		span := 0.9 - r*0.02
		for t := -span; t <= span; t += 0.04 {
			c.dot(int(9.5+math.Cos(t)*r), int(12+math.Sin(t)*r), col)
		}
	}
}

func iconSiege(c *canvas) {
	c.disc(12, 4, 2.6, tkGoldBright)
	for _, o := range [][2]int{{12, 0}, {15, 4}, {9, 1}, {15, 1}} {
		c.dot(o[0], o[1], tkGold)
	}
	c.rect(2, 5, 9, 9, stoneR[2])
	c.rect(2, 5, 9, 1, stoneR[3])
	c.rect(2, 4, 2, 1, stoneR[2])
	c.rect(9, 4, 2, 1, stoneR[2])
	c.rect(4, 8, 5, 6, ink)
	c.rect(5, 7, 3, 1, ink)
	c.rect(5, 8, 1, 6, greyR[1])
	c.rect(7, 8, 1, 6, greyR[1])
	for i, x := range []int{1, 4, 7, 10, 13} {
		c.rect(x, 13-i%2, 2, 2, []ramp{skinR, darkSkin}[i%2][2])
		c.rect(x, 15-i%2, 2, 1, []ramp{redR, blueR, greenR}[i%3][2])
	}
}

func iconArchive(c *canvas) {
	c.rect(3, 5, 10, 7, plasterR[3])
	c.rect(3, 11, 10, 1, plasterR[1])
	c.rect(1, 4, 2, 9, plasterR[1])
	c.rect(13, 4, 2, 9, plasterR[1])
	c.dot(1, 4, plasterR[2])
	c.dot(13, 4, plasterR[2])
	for i, w := range []int{5, 3, 4} {
		c.rect(5, 6+i*2, w, 1, blueR[1])
	}
	c.shadePoly([]pt{{14, 0}, {15.5, 1.5}, {9, 9}, {8, 8}}, flat(white))
	c.line(14, 1, 8, 8, plasterR[1])
	c.line(8, 9, 6, 12, inkSft)
}

func iconWarhorn(c *canvas) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			dx, dy := float64(x)+0.5-8, float64(y)+0.5-15
			a := math.Atan2(dy, dx)
			if a > -0.25 || a < -math.Pi+0.35 {
				continue
			}
			t := (a + math.Pi - 0.35) / (math.Pi - 0.6)
			th, r := 1.2+t*3.6, math.Hypot(dx, dy)
			if math.Abs(r-9) > th/2 {
				continue
			}
			col := tkGold
			switch {
			case math.Abs(t-0.45) < 0.05 || math.Abs(t-0.75) < 0.05:
				col = goldR[0]
			case r > 9:
				col = tkGoldBright
			}
			c.dot(x, y, col)
		}
	}
	c.ellipse(15, 11.5, 1.2, 3.2, goldR[1])
	c.ellipse(15, 11.5, 0.6, 2.2, goldR[0])
}

func paintBarMini(c *canvas) {
	c.rect(0, 0, c.w, c.h, tkConsoleEdge)
	ring(c, 0, tkInk, tkInk)
}
