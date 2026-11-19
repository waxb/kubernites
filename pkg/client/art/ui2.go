package art

import "image/color"

func paintScroll(c *canvas) {
	c.rect(4, 6, c.w-8, c.h-12, plasterR[3])
	for i, col := range []color.NRGBA{plasterR[1], plasterR[2]} {
		c.rect(4+i, 6, 1, c.h-12, col)
		c.rect(c.w-5-i, 6, 1, c.h-12, col)
	}
	scrollRoll(c, 1)
	scrollRoll(c, c.h-7)
	c.outline(ink)
}

func scrollRoll(c *canvas, top int) {
	for i, col := range []color.NRGBA{plasterR[1], plasterR[3], cream, plasterR[2], plasterR[1], plasterR[0]} {
		c.rect(3, top+i, c.w-6, 1, col)
	}
	for _, x := range []int{1, c.w - 3} {
		c.rect(x, top+1, 2, 4, woodR[2])
		c.dot(x, top+1, woodR[3])
		c.rect(x, top+4, 2, 1, woodR[0])
	}
}

func paintTooltip(c *canvas) {
	c.rect(0, 0, c.w, c.h, rgb(38, 30, 36))
	ring(c, 0, ink, ink)
	ring(c, 1, goldR[1], goldR[0])
	roundCorners(c, true)
}

func paintBarFrame(c *canvas) {
	c.rect(0, 0, c.w, c.h, rgb(26, 20, 24))
	ring(c, 0, ink, ink)
	ring(c, 1, woodR[3], woodR[1])
	c.rect(2, 2, c.w-4, 1, rgb(14, 10, 14))
}

func paintBarFill(c *canvas, r ramp) {
	for y, col := range []color.NRGBA{r[3], r[2], r[2], r[1], r[1], r[0]} {
		c.rect(0, y, c.w, 1, col)
	}
}

func paintMinimapFrame(c *canvas) {
	ring(c, 0, ink, ink)
	woodBorder(c, 1, 5)
	ring(c, 6, ink, ink)
	ring(c, 7, stoneR[3], stoneR[1])
	ring(c, 8, stoneR[2], stoneR[0])
	ring(c, 9, ink, ink)
	for _, p := range [][2]float64{{5, 5}, {float64(c.w) - 5, 5}, {5, float64(c.h) - 5}, {float64(c.w) - 5, float64(c.h) - 5}} {
		x, y := p[0], p[1]
		c.shadePoly([]pt{{x, y - 4.5}, {x + 4.5, y}, {x, y + 4.5}, {x - 4.5, y}}, flat(goldR[1]))
		c.shadePoly([]pt{{x, y - 3.5}, {x, y}, {x - 3.5, y}}, flat(goldR[3]))
		c.shadePoly([]pt{{x, y - 3.5}, {x + 3.5, y}, {x, y + 3.5}, {x - 3.5, y}}, litByX(goldR, x))
		c.disc(x, y, 1.3, redR[2])
	}
}

func paintWovenStrip(c *canvas) {
	thread := mix(tkConsole, tkParchmentDark, 0.11)
	c.rect(0, 0, c.w, c.h, tkConsole)
	for y := 2; y < c.h-2; y++ {
		for x := 0; x < c.w; x++ {
			across, down := y%4 == 0, x%4 == 0
			over := (x/4+y/4)%2 == 0
			if (across && (!down || over)) || (down && (!across || !over)) {
				c.dot(x, y, thread)
			}
		}
	}
	twist := mix(tkParchment, tkParchmentDark, 0.35)
	for _, y := range []int{0, c.h - 2} {
		c.rect(0, y, c.w, 2, tkParchment)
		for x := 0; x < c.w; x += 4 {
			c.dot(x, y, twist)
			c.dot(x+2, y+1, twist)
		}
	}
}

func paintChip(c *canvas) {
	c.rect(0, 0, c.w, c.h, tkConsoleEdge)
	ring(c, 0, tkParchment, tkParchment)
}
