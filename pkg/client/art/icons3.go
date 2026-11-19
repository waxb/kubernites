package art

import "image/color"

var caesarIconPainters = map[string]func(c *canvas){
	"decree": iconDecree, "rating-reliability": iconReliability, "rating-efficiency": iconEfficiency,
	"rating-resilience": iconResilience, "rating-favor": iconFavor, "overlay-midden": iconOverlayMidden,
	"overlay-labor": iconOverlayLabor, "overlay-unhoused": iconOverlayUnhoused, "overlay-heraldry": iconOverlayHeraldry, "council": iconCouncil,
	"rank-high": iconRankHigh, "rank-normal": iconRankNormal, "rank-low": iconRankLow,
}

func iconDecree(c *canvas) {
	c.rect(3, 3, 10, 10, tkParchment)
	c.rect(12, 3, 1, 10, tkParchmentDark)
	c.rect(2, 1, 12, 3, tkParchmentDark)
	c.rect(2, 1, 12, 1, tkParchment)
	c.rect(2, 12, 12, 2, tkParchmentDark)
	c.rect(6, 5, 5, 1, tkGold)
	for _, x := range []int{6, 8, 10} {
		c.dot(x, 4, tkGoldBright)
	}
	c.rect(5, 7, 6, 1, inkSft)
	c.rect(5, 9, 4, 1, inkSft)
	c.line(10, 13, 9, 15, redR[1])
	c.line(12, 13, 13, 15, redR[0])
	c.disc(11, 12, 2.6, redR[1])
	c.disc(10.6, 11.6, 1.6, redR[2])
	c.dot(10, 11, redR[3])
}

var nineGlyph = []string{"###", "#.#", "###", "..#", "##."}

func glyph(c *canvas, rows []string, x, y int, col color.NRGBA) {
	for j, row := range rows {
		for i, ch := range row {
			if ch == '#' {
				c.dot(x+i, y+j, col)
			}
		}
	}
}

func iconReliability(c *canvas) {
	c.dot(7, 0, greyR[1])
	c.dot(8, 0, greyR[1])
	c.shadePoly([]pt{{1.5, 4.5}, {8, 0.5}, {14.5, 4.5}}, litByX(greyR, 8))
	c.rect(2, 4, 12, 9, greyR[0])
	c.rect(3, 5, 10, 7, lampLit)
	c.rect(3, 5, 10, 1, goldR[3])
	c.rect(3, 5, 1, 7, goldR[3])
	glyph(c, nineGlyph, 4, 6, redR[0])
	glyph(c, nineGlyph, 9, 6, redR[0])
	c.rect(2, 13, 12, 2, greyR[1])
	c.rect(2, 13, 12, 1, greyR[2])
}

func iconEfficiency(c *canvas) {
	c.rect(7, 3, 2, 10, goldR[1])
	c.rect(7, 3, 1, 10, tkGoldBright)
	c.rect(4, 13, 8, 2, woodR[1])
	c.rect(4, 13, 8, 1, woodR[2])
	c.rect(1, 4, 14, 1, tkGold)
	c.rect(1, 3, 14, 1, tkGoldBright)
	c.disc(8, 2.5, 1.6, tkGoldBright)
	for _, x := range []int{3, 12} {
		c.line(x, 5, x-2, 9, greyR[2])
		c.line(x, 5, x+2, 9, greyR[2])
		c.ellipse(float64(x)+0.5, 10, 3.2, 1.4, goldR[1])
		c.rect(x-2, 9, 5, 1, tkGoldBright)
	}
}

func iconResilience(c *canvas) {
	c.shadePoly([]pt{{1.5, 1.5}, {14.5, 1.5}, {14.5, 8}, {8, 15}, {1.5, 8}}, flat(tkGold))
	c.shadePoly([]pt{{3, 3}, {13, 3}, {13, 8}, {8, 13.4}, {3, 8}}, func(x, y int) color.NRGBA {
		if y < 5 {
			return blueR[3]
		}
		return blueR[2]
	})
	c.ellipse(6, 8, 2.4, 1.4, greenR[2])
	c.dot(5, 7, greenR[3])
	c.ellipse(10.5, 6.5, 1.8, 1.1, greenR[2])
	c.dot(10, 6, greenR[3])
	c.rect(4, 9, 4, 1, sandR[3])
	for _, p := range [][2]int{{5, 11}, {7, 11}, {10, 10}, {11, 9}} {
		c.dot(p[0], p[1], white)
	}
	c.rect(1, 1, 14, 1, tkGoldBright)
}

func iconFavor(c *canvas) {
	c.shadePoly([]pt{{1.5, 12}, {1.5, 4.5}, {5, 8}, {8, 2.5}, {11, 8}, {14.5, 4.5}, {14.5, 12}}, litByX(ramp{goldR[0], goldR[1], tkGold, tkGoldBright}, 8))
	c.rect(2, 11, 12, 3, goldR[1])
	c.rect(2, 11, 12, 1, tkGoldBright)
	for _, p := range [][2]float64{{2, 3.5}, {8, 1.5}, {14, 3.5}} {
		c.disc(p[0], p[1], 1.2, tkGoldBright)
	}
	c.dot(8, 12, redR[2])
	c.dot(5, 12, blueR[2])
	c.dot(11, 12, blueR[2])
	c.dot(8, 7, redR[2])
}

func iconOverlayMidden(c *canvas) {
	c.ellipse(8, 12.5, 7, 3.2, rubbishR[1])
	c.ellipse(7, 11.5, 5, 2.2, rubbishR[2])
	c.rect(2, 5, 6, 6, crateR[2])
	c.rect(2, 5, 6, 1, crateR[3])
	c.line(2, 5, 7, 10, crateR[0])
	c.line(7, 5, 2, 10, crateR[0])
	c.rect(10, 6, 4, 6, woodR[1])
	c.rect(10, 6, 1, 6, woodR[2])
	c.rect(10, 8, 4, 1, greyR[1])
	c.rect(5, 11, 5, 2, tkParchment)
	c.rect(5, 12, 5, 1, tkParchmentDark)
	c.dot(12, 1, inkSft)
	c.dot(15, 3, inkSft)
}

func iconOverlayLabor(c *canvas) {
	c.shadePoly([]pt{{8, 5}, {15.5, 9}, {8, 13}, {0.5, 9}}, func(x, y int) color.NRGBA {
		d := absf(float64(x)+0.5-8)/7.5 + absf(float64(y)+0.5-9)/4
		switch {
		case d < 0.35:
			return fireR[3]
		case d < 0.6:
			return fireR[2]
		case d < 0.8:
			return fireR[1]
		}
		return redR[1]
	})
	c.rect(4, 1, 6, 3, greyR[2])
	c.rect(4, 1, 6, 1, greyR[3])
	c.line(8, 4, 12, 8, woodR[1])
	c.line(9, 4, 13, 8, woodR[2])
}

func iconOverlayUnhoused(c *canvas) {
	pale := plasterR[1]
	for _, l := range [][4]float64{{2, 7, 8, 1}, {8, 1, 14, 7}, {3, 7, 3, 14}, {13, 7, 13, 14}, {3, 14, 13, 14}} {
		stitch(c, pt{l[0], l[1]}, pt{l[2], l[3]}, pale)
	}
	c.line(4, 14, 10, 6, woodR[1])
	c.disc(10.5, 6, 2.6, redR[2])
	c.dot(9, 5, redR[3])
	c.dot(11, 8, redR[1])
}

func iconOverlayHeraldry(c *canvas) {
	c.rect(2, 1, 1, 14, woodR[1])
	c.dot(2, 0, goldR[3])
	shield := []pt{{4.5, 2.5}, {14.5, 2.5}, {14.5, 8}, {9.5, 14.5}, {4.5, 8}}
	c.shadePoly(shield, func(x, y int) color.NRGBA {
		if x < 10 {
			return blueR[2]
		}
		return redR[2]
	})
	c.shadePoly([]pt{{5.5, 3.5}, {9.5, 3.5}, {9.5, 5}, {5.5, 5}}, flat(blueR[3]))
	c.shadePoly([]pt{{10.5, 3.5}, {13.5, 3.5}, {13.5, 5}, {10.5, 5}}, flat(redR[3]))
	c.rect(9, 6, 2, 2, tkGoldBright)
	c.dot(9, 8, tkGold)
	c.dot(10, 8, tkGold)
}

func iconCouncil(c *canvas) {
	for i, f := range [][3]float64{{3.5, 10, 0}, {12.5, 10, 2}, {8, 9, 1}} {
		col := []color.NRGBA{redR[2], purpleR[2], blueR[2]}[int(f[2])]
		c.ellipse(f[0], f[1], 2.6, 3, col)
		c.disc(f[0], f[1]-4.5, 1.9, skinR[2])
		c.dot(int(f[0])-1, int(f[1])-5, skinR[3])
		if i == 2 {
			for _, x := range []int{6, 8, 10} {
				c.dot(x, 2, tkGoldBright)
			}
			c.rect(6, 3, 5, 1, tkGold)
		}
	}
	c.ellipse(8, 12.5, 7, 2.4, woodR[2])
	c.ellipse(8, 12, 6, 1.6, woodR[3])
	c.rect(3, 14, 1, 2, woodR[0])
	c.rect(12, 14, 1, 2, woodR[0])
}

func rankBanner(c *canvas, cloth ramp) {
	c.rect(2, 1, 12, 1, woodR[1])
	c.shadePoly([]pt{{3, 2}, {13, 2}, {13, 15}, {8, 12}, {3, 15}}, litByX(cloth, 6))
	c.rect(3, 2, 10, 1, cloth[3])
}

func chevron(c *canvas, y, arm, thick int, up bool, col color.NRGBA) {
	for i := 0; i <= arm; i++ {
		dy := i
		if !up {
			dy = arm - i
		}
		for t := 0; t < thick; t++ {
			c.dot(7-i, y+dy+t, col)
			c.dot(8+i, y+dy+t, col)
		}
	}
}

func iconRankHigh(c *canvas) {
	rankBanner(c, redR)
	for _, y := range []int{3, 6, 9} {
		chevron(c, y, 3, 1, true, tkGoldBright)
	}
}

func iconRankNormal(c *canvas) {
	rankBanner(c, blueR)
	chevron(c, 5, 3, 2, true, cream)
}

func iconRankLow(c *canvas) {
	rankBanner(c, greyR)
	chevron(c, 4, 3, 2, false, plasterR[1])
}
