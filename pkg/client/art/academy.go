package art

func paintBoat(c *canvas, f int) {
	y := 17 + []int{0, -1, 0, 1}[f]
	c.shadePoly([]pt{{2, float64(y - 4)}, {22, float64(y - 4)}, {19, float64(y)}, {5, float64(y)}}, flat(woodR[1]))
	c.rect(3, y-4, 19, 1, woodR[3])
	c.rect(5, y-2, 15, 1, woodR[0])
	c.rect(12, y-15, 1, 11, woodR[0])
	c.shadePoly([]pt{{13, float64(y - 14)}, {20, float64(y - 5)}, {13, float64(y - 5)}}, flat(white))
	c.rect(13, y-9, 5, 1, blueR[2])
	c.shadePoly([]pt{{12, float64(y - 15)}, {12, float64(y - 13)}, {float64(8 - f%2), float64(y - 14)}}, flat(blueR[2]))
	for _, x := range []int{5, 8} {
		c.rect(x, y-7, 2, 2, skinR[2])
		c.rect(x-1, y-8, 4, 1, inkSft)
		c.dot(x+2, y-7, goldR[2])
		c.rect(x, y-5, 2, 1, blueR[1])
	}
	c.outline(ink)
	for i := 0; i < 3; i++ {
		c.rect(1+(f*2+i*7)%20, 20, 3, 1, withAlpha(waterR[3], 210))
	}
	c.rect(4, 19, 16, 1, withAlpha(waterR[3], 150))
}

func iconAcademy(c *canvas) {
	board, cap, edge := rgb(44, 48, 78), rgb(30, 32, 54), rgb(96, 104, 150)
	c.shadePoly([]pt{{4, 6.5}, {12, 6.5}, {12, 11}, {8, 12.5}, {4, 11}}, flat(cap))
	c.shadePoly([]pt{{8, 1.5}, {15, 5}, {8, 8.5}, {1, 5}}, flat(board))
	c.line(1, 5, 8, 2, edge)
	c.dot(8, 5, goldR[3])
	c.line(8, 5, 13, 7, goldR[2])
	c.rect(13, 7, 1, 5, goldR[2])
	c.rect(12, 12, 3, 2, goldR[3])
}

func iconDiploma(c *canvas) {
	for d, col := range []ramp{plasterR, plasterR, plasterR, plasterR} {
		c.line(3, 10+d, 13, 3+d, col[3-d])
	}
	c.ellipse(3, 12, 1.8, 2.2, plasterR[1])
	c.dot(3, 12, woodR[2])
	c.line(7, 6, 9, 11, redR[2])
	c.line(8, 6, 10, 11, redR[2])
	c.disc(10, 12, 2.6, redR[1])
	c.dot(9, 11, redR[3])
	c.dot(12, 15, redR[2])
	c.dot(8, 15, redR[2])
}

func paintAcademyPortrait(c *canvas) {
	for y := 0; y < PortraitSize; y++ {
		c.rect(0, y, PortraitSize, 1, mix(rgb(118, 166, 212), rgb(250, 222, 186), min(float64(y)/30, 1)))
	}
	for y := 30; y < PortraitSize; y++ {
		for x := 0; x < PortraitSize; x++ {
			col := waterR[min((PortraitSize-y)/6, 2)]
			if hash(x/3, y, 71)%11 == 0 {
				col = waterR[3]
			}
			c.dot(x, y, col)
		}
	}
	c.shadePoly([]pt{{5, 30}, {12, 25}, {36, 24}, {44, 30}}, flat(grassR[1]))
	c.shadePoly([]pt{{5, 30}, {44, 30}, {41, 32}, {8, 32}}, flat(rockR[1]))
	academyHalls(c)
	c.shadePoly([]pt{{7, 39}, {17, 39}, {15, 41}, {9, 41}}, flat(woodR[1]))
	c.rect(12, 32, 1, 7, woodR[0])
	c.shadePoly([]pt{{12.5, 33}, {16, 38}, {12.5, 38}}, flat(white))
	c.rect(8, 42, 9, 1, waterR[3])
	portraitFrame(c)
}

func academyHalls(c *canvas) {
	c.rect(12, 18, 9, 9, plasterR[2])
	c.shadePoly([]pt{{11, 18}, {16.5, 13}, {22, 18}}, flat(blueR[1]))
	c.rect(28, 19, 10, 8, plasterR[1])
	c.shadePoly([]pt{{27, 19}, {33, 14}, {39, 19}}, flat(blueR[1]))
	c.rect(21, 10, 7, 17, plasterR[3])
	c.rect(25, 10, 3, 17, plasterR[2])
	c.shadePoly([]pt{{20, 10}, {24.5, 4}, {29, 10}}, flat(blueR[2]))
	c.rect(24, 4, 1, 1, goldR[3])
	for _, p := range [][2]int{{23, 13}, {23, 18}, {23, 23}, {14, 21}, {17, 21}, {30, 22}, {34, 22}} {
		c.dot(p[0], p[1], lampLit)
	}
	c.rect(20, 26, 9, 1, ink)
}
