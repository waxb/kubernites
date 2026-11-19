package art

func pageDaemons(c *canvas) {
	for i, x := range []int{3, 11, 19} {
		base := 20 - (i%2)*2
		c.rect(x-2, base-6, 5, 6, stoneR[2])
		c.rect(x-2, base-6, 1, 6, stoneR[3])
		c.rect(x-2, base-7, 5, 1, stoneR[1])
		c.rect(x, base-10, 1, 3, woodR[2])
		c.rect(x-1, base-12, 3, 2, purpleR[2])
		c.dot(x-1, base-12, purpleR[3])
		c.dot(x+2, base-9, lampLit)
	}
	c.line(3, 3, 19, 3, goldR[2])
	c.dot(2, 4, goldR[2])
	c.dot(20, 4, goldR[2])
	c.dot(11, 2, goldR[3])
	c.rect(1, 21, 22, 1, grassR[2])
}

func pageVaults(c *canvas) {
	c.shadePoly([]pt{{4, 9}, {12, 5}, {20, 9}, {20, 21}, {4, 21}}, flat(stoneR[2]))
	c.rect(4, 9, 1, 12, stoneR[3])
	c.rect(19, 9, 1, 12, stoneR[1])
	c.shadePoly([]pt{{3, 9}, {12, 3}, {21, 9}}, flat(slateR[2]))
	c.rect(7, 12, 10, 7, ink)
	for i, x := range []int{8, 10, 12, 14} {
		c.rect(x, 13, 1, 2, spines[i])
		c.rect(x+1, 16, 1, 2, spines[i+2])
	}
	for x := 7; x <= 16; x += 3 {
		c.rect(x, 12, 1, 7, greyR[2])
	}
	c.rect(9, 19, 6, 2, tkParchment)
	c.rect(11, 19, 2, 2, redR[1])
}

func pageStatefulCrews(c *canvas) {
	for i := 0; i < 3; i++ {
		x := 3 + i*7
		c.rect(x, 9, 5, 9, blueR[2])
		c.rect(x, 9, 1, 9, blueR[3])
		c.shadePoly([]pt{{float64(x) - 1, 9}, {float64(x) + 2.5, 5}, {float64(x) + 6, 9}}, flat(redR[2]))
		for p := 0; p <= i; p++ {
			c.dot(x+1+p, 12, cream)
		}
		c.rect(x+1, 19, 3, 2, tkParchment)
		c.dot(x+3, 20, redR[1])
	}
	c.line(3, 22, 21, 22, greyR[2])
	for x := 4; x < 21; x += 3 {
		c.dot(x, 21, greyR[3])
	}
}

func pageVaultWrights(c *canvas) {
	c.shadePoly([]pt{{6, 12}, {12, 9}, {18, 12}, {18, 21}, {6, 21}}, flat(stoneR[2]))
	c.rect(10, 15, 4, 6, woodR[1])
	c.rect(10, 17, 4, 1, greyR[1])
	c.line(3, 3, 12, 12, woodR[2])
	c.line(4, 3, 13, 12, woodR[1])
	c.shadePoly([]pt{{1, 5}, {3, 1}, {7, 3}, {5, 6}}, flat(greyR[3]))
	c.line(21, 3, 14, 10, woodR[2])
	c.shadePoly([]pt{{18, 2}, {23, 1}, {22, 5}, {20, 6}}, flat(greyR[2]))
	c.rect(1, 21, 22, 1, grassR[2])
	c.dot(12, 7, goldR[3])
}
