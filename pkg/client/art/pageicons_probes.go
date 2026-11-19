package art

import "image/color"

func pageShutters(c *canvas) {
	c.rect(7, 3, 10, 18, stoneR[1])
	c.rect(8, 4, 8, 16, lampLit)
	c.rect(8, 4, 8, 2, goldR[3])
	c.rect(11, 4, 2, 16, woodR[1])
	c.rect(8, 11, 8, 2, woodR[1])
	c.rect(7, 21, 10, 2, stoneR[2])
	c.rect(6, 21, 12, 1, stoneR[3])
	for _, side := range []struct{ x, dir int }{{1, 1}, {18, -1}} {
		c.rect(side.x, 4, 5, 16, woodR[2])
		for y := 6; y < 20; y += 3 {
			c.rect(side.x, y, 5, 1, woodR[0])
		}
		edge := side.x
		if side.dir < 0 {
			edge = side.x + 4
		}
		c.rect(edge, 4, 1, 16, woodR[3])
		c.dot(side.x+2, 11, goldR[2])
	}
	c.dot(9, 8, cream)
	c.dot(14, 15, cream)
}

func pageBell(c *canvas) {
	c.rect(2, 2, 20, 3, woodR[1])
	c.rect(2, 2, 20, 1, woodR[3])
	c.rect(3, 5, 2, 3, woodR[0])
	c.rect(19, 5, 2, 3, woodR[0])
	c.rect(11, 5, 2, 3, woodR[0])
	c.shadePoly([]pt{{6.5, 19}, {8, 12}, {9.5, 9}, {12, 8}, {14.5, 9}, {16, 12}, {17.5, 19}}, func(x, y int) color.NRGBA {
		switch {
		case x < 10:
			return goldR[3]
		case x > 14:
			return goldR[1]
		}
		return goldR[2]
	})
	c.rect(5, 19, 14, 2, goldR[1])
	c.rect(5, 19, 14, 1, goldR[2])
	c.disc(12, 22, 1.5, ink)
	for _, d := range []struct{ x, y int }{{3, 11}, {2, 13}, {3, 15}, {21, 11}, {22, 13}, {21, 15}} {
		c.dot(d.x, d.y, cream)
	}
	c.dot(10, 11, cream)
}
