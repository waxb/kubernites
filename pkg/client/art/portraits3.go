package art

func ravenFace(c *canvas) {
	shadedEllipse(c, 22, 44, 15, 12, ravenR)
	shadedEllipse(c, 22, 24, 11, 11, ravenR)
	c.shadePoly([]pt{{29, 20}, {45, 26}, {29, 30}}, flat(inkSft))
	c.line(30, 25, 44, 26, ravenR[0])
	c.disc(25, 20, 3, goldR[3])
	c.disc(25.8, 20.3, 1.6, ink)
	c.dot(24, 19, white)
	for _, p := range [][2]int{{14, 30}, {17, 34}, {12, 38}, {20, 40}} {
		c.dot(p[0], p[1], ravenR[3])
	}
	c.rect(14, 36, 14, 2, ravenR[2])
}

func houndFace(c *canvas) {
	shadedEllipse(c, 24, 48, 16, 12, houndR)
	shadedEllipse(c, 22, 24, 11, 10, houndR)
	shadedEllipse(c, 33, 29, 8, 5.5, houndR)
	c.disc(40, 27, 2.4, ink)
	c.dot(39, 26, greyR[3])
	c.shadePoly([]pt{{12, 16}, {17, 14}, {15, 34}, {10, 30}}, flat(houndR[0]))
	c.disc(25, 21, 2.2, ink)
	c.dot(24, 20, white)
	c.line(30, 33, 38, 32, houndR[0])
	c.rect(15, 35, 18, 3, redR[2])
	c.rect(15, 35, 18, 1, redR[3])
	c.disc(24, 39.5, 2, goldR[2])
	c.dot(23, 39, goldR[3])
}
