package art

import "math"

var magisterRobe = ramp{rgb(30, 34, 74), rgb(44, 52, 108), rgb(64, 78, 146), rgb(104, 122, 186)}

func magisterClasp(c *canvas) {
	for i := 0; i <= 10; i++ {
		t := float64(i) / 10
		c.dot(int(17+t*14), int(38+math.Sin(t*math.Pi)*3), goldR[2])
	}
	c.disc(24, 41.5, 2.2, goldR[2])
	c.dot(24, 41, tkGoldBright)
	for _, p := range [][2]int{{12, 44}, {35, 46}, {16, 47}} {
		c.dot(p[0], p[1], goldR[3])
	}
}

func pageTreasure(c *canvas) {
	for i := 0; i < 6; i++ {
		c.rect(2+i*2, 20-i*2, 1, 1, mapInk)
	}
	stroke := func(x0, y0, x1, y1 int) {
		for _, d := range []int{0, 1} {
			c.line(x0+d, y0, x1+d, y1, tkRed)
			c.line(x0, y0+d, x1, y1+d, redR[1])
		}
	}
	stroke(9, 4, 20, 15)
	stroke(20, 4, 9, 15)
	c.dot(14, 9, tkGoldBright)
}
