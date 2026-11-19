package art

func iconThread(c *canvas) {
	for y := 4; y <= 12; y++ {
		col := tkGold
		if y%2 == 0 {
			col = tkGoldBright
		}
		c.rect(4, y, 8, 1, col)
		c.dot(11, y, goldR[0])
	}
	c.rect(4, 5, 1, 7, goldR[3])
	c.ellipse(8, 13, 5.5, 1.8, woodR[1])
	c.ellipse(8, 3.5, 5.5, 2, woodR[1])
	c.ellipse(8, 3, 5.5, 1.5, woodR[2])
	c.dot(8, 3, woodR[0])
	c.line(12, 8, 14, 11, tkGoldBright)
	c.dot(14, 12, tkGold)
}

func iconLoomTreasury(c *canvas) {
	c.shadePoly([]pt{{2, 7}, {14, 7}, {12.5, 1.5}, {3.5, 1.5}}, flat(woodR[0]))
	for _, p := range [][3]float64{{5, 6, 2.2}, {11, 6, 2.2}, {8, 5, 2.4}} {
		c.disc(p[0], p[1], p[2], tkGold)
		c.dot(int(p[0])-1, int(p[1])-1, tkGoldBright)
	}
	c.rect(2, 7, 12, 7, woodR[2])
	c.rect(10, 8, 4, 6, woodR[1])
	c.rect(2, 7, 12, 1, woodR[3])
	c.rect(4, 7, 1, 7, tkGold)
	c.rect(11, 7, 1, 7, goldR[1])
	c.rect(7, 9, 2, 3, tkGoldBright)
	c.dot(7, 10, ink)
}

func iconLoomBuilders(c *canvas) {
	iconHammer(c)
	c.rect(0, 10, 4, 1, tkGoldBright)
	c.rect(1, 12, 4, 1, tkParchment)
	c.rect(0, 14, 3, 1, tkGold)
}

func iconLoomFame(c *canvas) {
	c.shadePoly([]pt{{4, 10}, {11, 7.5}, {11, 15}, {8.5, 12.5}, {5.5, 15}}, flat(tkRed))
	c.line(4, 10, 11, 8, tkGold)
	c.shadePoly([]pt{{1.5, 11}, {2.5, 10}, {11, 3}, {15, 7.5}, {3, 12}}, flat(tkGold))
	c.ellipse(13, 5, 2.2, 3.4, tkGoldBright)
	c.dot(13, 5, goldR[0])
	c.line(3, 10, 10, 4, goldR[3])
}

func paintLoomPortrait(c *canvas) {
	for y := 0; y < PortraitSize; y++ {
		col := mix(woodR[1], woodR[0], float64(y)/PortraitSize)
		if y%9 == 0 {
			col = woodR[0]
		}
		c.rect(0, y, PortraitSize, 1, col)
	}
	c.rect(0, 40, PortraitSize, 8, woodR[2])
	c.rect(0, 40, PortraitSize, 1, woodR[3])
	loom := newCanvas(PortraitSize, PortraitSize)
	loomFrame(loom)
	loomCloth(loom)
	loom.outline(ink)
	c.paste(loom, 0, 0, false)
	portraitFrame(c)
}

func loomFrame(c *canvas) {
	c.rect(8, 8, 3, 35, woodR[3])
	c.rect(10, 8, 1, 35, woodR[2])
	c.rect(37, 8, 3, 35, woodR[2])
	c.rect(39, 8, 1, 35, woodR[1])
	c.rect(6, 7, 36, 3, woodR[3])
	c.rect(6, 9, 36, 1, woodR[1])
	c.rect(6, 35, 36, 3, woodR[2])
	c.rect(6, 37, 36, 1, woodR[0])
	c.rect(6, 42, 7, 2, woodR[0])
	c.rect(35, 42, 7, 2, woodR[0])
	for x := 13; x <= 35; x += 2 {
		c.rect(x, 10, 1, 25, tkParchment)
	}
	c.rect(11, 17, 26, 1, woodR[0])
}

func loomCloth(c *canvas) {
	for y := 23; y < 35; y++ {
		for x := 12; x <= 36; x++ {
			col := tkRedDark
			switch {
			case (x-12+abs(y-28))%8 < 2:
				col = tkGold
			case (y/4)%2 == 0:
				col = blueR[0]
			}
			c.dot(x, y, col)
		}
	}
	c.rect(12, 23, 25, 1, tkGoldBright)
	c.shadePoly([]pt{{15, 20}, {29, 19}, {32, 20.5}, {29, 22}, {15, 21.5}}, flat(woodR[3]))
	c.line(24, 21, 36, 23, tkGoldBright)
	c.ellipse(41, 40, 3, 1.2, woodR[1])
	c.rect(39, 36, 5, 4, tkGold)
	c.rect(39, 37, 5, 1, tkGoldBright)
	c.ellipse(41, 35.5, 3, 1.2, woodR[2])
}
