package art

import "math"

func ringArc(c *canvas, cx, cy, r0, r1 float64, skip func(a float64) bool, r ramp) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			d := math.Hypot(dx, dy)
			if d < r0 || d > r1 || skip(math.Atan2(dy, dx)) {
				continue
			}
			col := r[2]
			if dx+dy < 0 {
				col = r[3]
			}
			c.dot(x, y, col)
		}
	}
}

func iconRoll(c *canvas) {
	ringArc(c, 8, 8, 3.6, 6, func(a float64) bool { return (a > -0.5 && a < 0.2) || a > 2.64 || a < -2.94 }, blueR)
	c.shadePoly([]pt{{11, 7}, {15.5, 7}, {13.2, 10.5}}, flat(blueR[2]))
	c.shadePoly([]pt{{5, 9}, {0.5, 9}, {2.8, 5.5}}, flat(blueR[3]))
}

func iconTech(c *canvas) {
	for k := 0; k < 8; k++ {
		a := float64(k) * math.Pi / 4
		c.disc(8+math.Cos(a)*5.3, 8+math.Sin(a)*5.3, 1.5, greyR[1])
	}
	c.disc(8, 8, 5, greyR[2])
	c.disc(7.5, 7.5, 3.5, greyR[3])
	c.disc(8, 8, 1.8, greyR[0])
}

func iconBook(c *canvas) { bookIcon(c, tealR) }

func iconTome(c *canvas) { bookIcon(c, redR) }

func bookIcon(c *canvas, r ramp) {
	c.rect(4, 2, 9, 11, r[2])
	c.rect(3, 2, 2, 12, r[0])
	c.rect(5, 13, 8, 1, cream)
	c.shadePoly([]pt{{9, 4.5}, {11, 7.5}, {9, 10.5}, {7, 7.5}}, flat(goldR[2]))
	c.dot(9, 7, goldR[3])
	c.dot(12, 2, goldR[2])
	c.dot(12, 12, goldR[2])
}

func iconChronicle(c *canvas) {
	c.rect(3, 3, 10, 10, plasterR[2])
	c.rect(2, 2, 12, 2, plasterR[1])
	c.rect(2, 12, 12, 2, plasterR[1])
	c.rect(1, 2, 1, 2, woodR[1])
	c.rect(14, 2, 1, 2, woodR[1])
	c.rect(1, 12, 1, 2, woodR[1])
	c.rect(14, 12, 1, 2, woodR[1])
	for i, w := range []int{6, 5, 6} {
		c.rect(5, 5+i*2, w, 1, inkSft)
	}
}

func iconSoulstone(c *canvas) {
	c.shadePoly([]pt{{8, 1}, {3, 5}, {5, 14}, {8, 14}}, flat(voidR[3]))
	c.shadePoly([]pt{{8, 1}, {13, 5}, {11, 14}, {8, 14}}, flat(voidR[1]))
	c.shadePoly([]pt{{8, 3}, {10, 6}, {8, 12}, {6, 6}}, flat(voidR[2]))
	c.dot(6, 5, white)
}

func iconBell(c *canvas) {
	c.dot(8, 1, goldR[1])
	c.disc(8, 5, 3.2, goldR[2])
	c.shadePoly([]pt{{4.8, 5}, {11.2, 5}, {12.8, 11.5}, {3.2, 11.5}}, litByX(goldR, 7))
	c.rect(2, 11, 12, 2, goldR[1])
	c.disc(8, 14, 1.4, inkSft)
	c.dot(6, 4, goldR[3])
}

func iconAge(c *canvas) {
	c.shadePoly([]pt{{4, 3}, {12, 3}, {8.6, 8}, {12, 13}, {4, 13}, {7.4, 8}}, flat(withAlpha(glassR[3], 200)))
	c.shadePoly([]pt{{5.5, 4.5}, {10.5, 4.5}, {8, 7.5}}, flat(sandR[2]))
	c.shadePoly([]pt{{4.5, 13}, {11.5, 13}, {8, 10}}, flat(sandR[2]))
	c.rect(8, 8, 1, 3, sandR[3])
	c.rect(3, 1, 10, 2, woodR[2])
	c.rect(3, 13, 10, 2, woodR[1])
	c.rect(3, 3, 1, 10, woodR[1])
	c.rect(12, 3, 1, 10, woodR[0])
}

func iconSpeed(c *canvas) {
	c.shadePoly([]pt{{1.5, 2.5}, {8, 8}, {1.5, 13.5}}, flat(goldR[3]))
	c.shadePoly([]pt{{8, 2.5}, {14.5, 8}, {8, 13.5}}, flat(goldR[2]))
}

func iconPause(c *canvas) {
	c.rect(3, 2, 4, 12, cream)
	c.rect(9, 2, 4, 12, plasterR[2])
}

func iconAlert(c *canvas) {
	c.shadePoly([]pt{{8, 0.5}, {15.5, 14.5}, {0.5, 14.5}}, litByX(redR, 8))
	c.rect(7, 5, 2, 5, cream)
	c.rect(7, 11, 2, 2, cream)
}

func iconLens(c *canvas) {
	c.line(9, 10, 14, 15, woodR[1])
	c.line(10, 10, 14, 14, woodR[2])
	c.line(9, 11, 13, 15, woodR[0])
	c.disc(6.5, 6.5, 5, goldR[1])
	c.disc(6.5, 6.5, 3.6, glassR[2])
	c.disc(5.5, 5.5, 1.5, glassR[3])
}

func iconLogs(c *canvas) {
	c.rect(2, 2, 10, 12, plasterR[3])
	c.rect(11, 2, 1, 12, plasterR[1])
	for i, w := range []int{6, 4, 6, 3} {
		c.rect(4, 4+i*2, w, 1, inkSft)
	}
	c.line(9, 14, 14, 4, white)
	c.shadePoly([]pt{{14, 1}, {15, 3}, {12, 9}, {11.5, 8}}, flat(plasterR[3]))
}

func iconYAML(c *canvas) {
	c.shadePoly([]pt{{2, 1}, {10, 1}, {14, 5}, {14, 15}, {2, 15}}, flat(cream))
	c.shadePoly([]pt{{10, 1}, {10, 5}, {14, 5}}, flat(plasterR[1]))
	for _, l := range [][3]int{{4, 4, 4}, {6, 6, 5}, {6, 8, 4}, {4, 10, 4}, {6, 12, 6}} {
		c.dot(l[0], l[1], redR[2])
		c.rect(l[0]+1, l[1], l[2]-1, 1, blueR[1])
	}
}

func iconBack(c *canvas) {
	c.shadePoly([]pt{{1, 8}, {7.5, 1.5}, {7.5, 5.5}, {14.5, 5.5}, {14.5, 10.5}, {7.5, 10.5}, {7.5, 14.5}}, litByX(plasterR, 6))
}

func iconClose(c *canvas) {
	for d := -1; d <= 1; d++ {
		c.line(3+d, 3, 13+d, 13, redR[2])
		c.line(13+d, 3, 3+d, 13, redR[2])
	}
	c.line(3, 3, 7, 7, redR[3])
}
