package art

import "image/color"

func drawPennant(c *canvas, hx, hy int) {
	c.line(hx, hy+2, hx+1, hy-9, woodR[1])
	c.dot(hx+1, hy-10, goldR[3])
	c.shadePoly([]pt{{float64(hx) + 2, float64(hy) - 9}, {float64(hx) + 6.5, float64(hy) - 7.5}, {float64(hx) + 2, float64(hy) - 5.5}}, func(x, y int) color.NRGBA {
		if y%2 == 0 {
			return goldR[2]
		}
		return rgb(140, 146, 70)
	})
}

func iconPennant(c *canvas) {
	c.line(3, 15, 5, 1, woodR[2])
	c.line(4, 15, 6, 1, woodR[1])
	c.dot(5, 0, goldR[3])
	c.shadePoly([]pt{{6, 1.5}, {15, 4.5}, {6, 8.5}}, func(x, y int) color.NRGBA {
		if y%2 == 0 {
			return goldR[2]
		}
		return rgb(140, 146, 70)
	})
	c.dot(9, 4, redR[2])
}

func iconQuillLedger(c *canvas) {
	c.rect(1, 6, 9, 9, redR[1])
	c.rect(2, 7, 7, 7, redR[2])
	c.rect(1, 6, 9, 1, cream)
	c.rect(4, 9, 3, 2, goldR[2])
	c.shadePoly([]pt{{13, 0}, {15, 2}, {9, 9}, {7.5, 8}}, flat(plasterR[3]))
	c.line(14, 1, 8, 8, plasterR[1])
	c.line(8, 8, 6, 11, inkSft)
}
