package art

import (
	"image/color"
	"math"
)

func wideChevron(c *canvas, y, arm int, col color.NRGBA) {
	for i := 0; i <= arm; i++ {
		for t := 0; t < 2; t++ {
			c.dot(11-i, y+i+t, col)
			c.dot(12+i, y+i+t, col)
		}
	}
}

func pageLighthouse(c *canvas) {
	c.shadePoly([]pt{{8, 22}, {10, 7}, {14, 7}, {16, 22}}, func(x, y int) color.NRGBA {
		if (y/4)%2 == 0 {
			return redR[2]
		}
		return white
	})
	c.rect(9, 4, 6, 3, lampLit)
	c.rect(8, 3, 8, 1, inkSft)
	c.shadePoly([]pt{{9, 3}, {12, 0}, {15, 3}}, flat(redR[1]))
	for _, s := range []float64{-1, 1} {
		c.shadePoly([]pt{{12 + s*3, 5}, {12 + s*11, 2}, {12 + s*11, 8}}, flat(withAlpha(goldR[3], 150)))
	}
	c.rect(4, 22, 16, 1, rockR[1])
}

func pageGateways(c *canvas) {
	c.rect(3, 6, 5, 16, stoneR[2])
	c.rect(16, 6, 5, 16, stoneR[1])
	c.shadePoly([]pt{{3, 6}, {12, 1}, {21, 6}, {21, 9}, {3, 9}}, flat(stoneR[3]))
	c.ellipse(12, 12, 4, 4, ink)
	c.rect(8, 12, 8, 10, ink)
	for i, col := range []color.NRGBA{greenR[3], goldR[2], blueR[3]} {
		x := 10 + i*2
		c.rect(x, 14+i, 1, 8-i, col)
	}
	c.dot(12, 3, tkRed)
}

func pageRollouts(c *canvas) {
	ringArc(c, 12, 12, 7, 9.5, func(a float64) bool { return a > -0.4 && a < 0.5 }, greenR)
	ringArc(c, 12, 12, 7, 9.5, func(a float64) bool { return a < 2.6 && a > -2.8 }, greenR)
	c.shadePoly([]pt{{19, 6}, {23, 11}, {17, 11}}, flat(greenR[3]))
	c.shadePoly([]pt{{5, 18}, {1, 13}, {7, 13}}, flat(greenR[2]))
	c.rect(10, 7, 1, 10, woodR[1])
	c.shadePoly([]pt{{11, 7}, {16, 8.5}, {11, 11}}, flat(porterR[2]))
}

func pageBudgets(c *canvas) {
	c.shadePoly([]pt{{2, 2}, {22, 2}, {22, 12}, {12, 23}, {2, 12}}, flat(greyR[2]))
	c.shadePoly([]pt{{4, 4}, {20, 4}, {20, 12}, {12, 20}, {4, 12}}, flat(greenR[1]))
	for i := 0; i < 3; i++ {
		x := 7 + i*5
		col := cream
		if i == 2 {
			col = withAlpha(cream, 90)
		}
		c.disc(float64(x)+0.5, 8, 1.6, col)
		c.rect(x-1, 10, 3, 4, col)
	}
	c.rect(6, 16, 12, 1, goldR[2])
}

func pageRebalancer(c *canvas) {
	c.rect(11, 3, 2, 17, woodR[1])
	c.rect(6, 20, 12, 2, woodR[2])
	c.line(3, 7, 21, 5, goldR[2])
	c.line(3, 8, 21, 6, goldR[1])
	for _, p := range [][2]int{{4, 8}, {20, 6}} {
		c.line(p[0]-2, p[1]+5, p[0], p[1], ropeR[2])
		c.line(p[0]+2, p[1]+5, p[0], p[1], ropeR[2])
		c.rect(p[0]-3, p[1]+5, 7, 2, goldR[1])
		c.rect(p[0]-2, p[1]+1, 5, 4, stoneR[2])
		c.rect(p[0]-2, p[1]+1, 5, 1, stoneR[3])
	}
	c.disc(12, 4, 1.5, goldR[3])
}

func pageTerritory(c *canvas) {
	c.shadePoly([]pt{{1, 16}, {12, 10}, {23, 16}, {12, 22}}, flat(grassR[2]))
	for i := 0; i < 12; i += 3 {
		c.dot(1+i, 16-i/2, cream)
		c.dot(12+i, 22-i/2, cream)
	}
	c.rect(13, 4, 1, 14, woodR[1])
	c.shadePoly([]pt{{14, 4}, {21, 6}, {14, 9}}, flat(purpleR[2]))
	c.rect(4, 12, 4, 5, markerR[2])
	c.rect(4, 12, 4, 1, markerR[3])
	c.rect(10, 15, 5, 3, markerR[1])
}

func pageCertManager(c *canvas) {
	scrollSheet(c, 2, 2, 15, 17)
	inkLines(c, 4, 6, 10, 3, mapInk)
	c.disc(18, 17, 5, white)
	ringArc(c, 18, 17, 4, 5.2, func(float64) bool { return false }, greyR)
	c.line(18, 17, 18, 14, ink)
	c.line(18, 17, 20, 17, ink)
	c.disc(7, 18, 3, sealR[1])
	c.disc(6.5, 17.5, 2, sealR[2])
	c.line(6, 20, 5, 23, goldR[2])
	c.line(8, 20, 9, 23, goldR[1])
}

func pageHerald(c *canvas) {
	scrollSheet(c, 12, 4, 10, 16)
	inkLines(c, 14, 8, 6, 3, mapInk)
	for i := 0; i < 8; i++ {
		t := float64(i) / 7
		c.disc(3+t*8, 14-t*5, 1.2+t*2.2, goldR[2-i%2])
	}
	c.disc(11, 9, 3, goldR[1])
	c.disc(11, 9, 1.6, ink)
	c.rect(2, 15, 2, 4, redR[2])
}

func pageKeycloak(c *canvas) {
	glyph(c, []string{"#..#..#", "##.#.##", "#######"}, 3, 2, goldR[3])
	c.rect(3, 5, 7, 2, goldR[1])
	c.dot(6, 5, redR[2])
	c.disc(6.5, 12, 4.4, goldR[1])
	c.disc(6, 11.5, 3.2, goldR[2])
	c.disc(6.5, 12, 1.5, inkSft)
	c.rect(10, 11, 12, 2, goldR[2])
	c.rect(10, 11, 12, 1, goldR[3])
	c.rect(17, 13, 2, 4, goldR[1])
	c.rect(20, 13, 2, 3, goldR[1])
}

func pageGoldpinger(c *canvas) {
	for _, x := range []float64{5, 19} {
		c.ellipse(x, 14, 3.2, 3.8, goldR[1])
		c.ellipse(x-0.8, 13, 1.6, 2.2, goldR[3])
		c.rect(int(x)-4, 17, 8, 1, goldR[0])
		c.rect(int(x), 8, 1, 3, woodR[1])
	}
	for x := 9; x <= 15; x += 2 {
		c.dot(x, 13, goldR[3])
	}
	for i := 0; i < 9; i++ {
		y := 6 - int(3*math.Sin(float64(i)*math.Pi/4))
		c.dot(8+i, y, greenR[3])
	}
}

func pageSquid(c *canvas) {
	c.ellipse(12, 7, 6, 6, squidR[2])
	c.ellipse(10.5, 5.5, 3, 3, squidR[3])
	for i := 0; i < 6; i++ {
		x := 6 + i*2
		for j := 0; j < 9; j++ {
			c.dot(x+int(1.5*math.Sin(float64(j)*0.9+float64(i))), 12+j, squidR[1+i%2])
		}
	}
	c.dot(9, 8, ink)
	c.dot(15, 8, ink)
}

func pageContracts(c *canvas) {
	scrollSheet(c, 2, 2, 14, 19)
	inkLines(c, 4, 6, 9, 3, mapInk)
	c.line(4, 17, 12, 16, sealR[1])
	goldCoin(c, 18, 17, 4.5)
	c.dot(18, 17, goldR[0])
	c.shadePoly([]pt{{22, 1}, {23, 3}, {15, 12}, {14, 11}}, flat(white))
}

func pageBlank(c *canvas) {
	c.rect(5, 2, 14, 20, tkParchment)
	c.rect(17, 2, 2, 20, tkParchmentDark)
	glyph(c, []string{".###.", "#...#", "...#.", "..#..", ".....", "..#.."}, 9, 7, mapInk)
}
