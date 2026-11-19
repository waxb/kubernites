package art

import "image/color"

func chainLink(c *canvas, cx, cy, rx, ry float64, r ramp) {
	for y := int(cy - ry - 1); y <= int(cy+ry+1); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			d := dx*dx + dy*dy
			if d <= 1 && d >= 0.36 {
				col := r[2]
				if dx+dy < -0.3 {
					col = r[3]
				}
				c.dot(x, y, col)
			}
		}
	}
}

func pageBindings(c *canvas) {
	chainLink(c, 8, 9, 6, 4.5, greyR)
	chainLink(c, 16, 15, 6, 4.5, greyR)
	chainLink(c, 8, 9, 6, 4.5, ramp{greyR[0], greyR[1], greyR[2], greyR[3]})
	c.rect(10, 11, 3, 2, greyR[2])
}

func pageMasonry(c *canvas) {
	for row := 0; row < 4; row++ {
		off := row % 2 * 3
		for x := -off; x < 20; x += 6 {
			x0 := max(x+2, 2)
			w := min(x+7, 21) - x0
			if w <= 0 {
				continue
			}
			y := 10 + row*3
			c.rect(x0, y, w, 2, stoneR[2])
			c.rect(x0, y, w, 1, stoneR[3])
		}
	}
	c.shadePoly([]pt{{12, 2}, {21, 5}, {17, 9}}, flat(greyR[3]))
	c.line(12, 2, 8, 6, woodR[2])
	c.line(8, 6, 6, 8, woodR[1])
}

func pageCharters(c *canvas) {
	scrollSheet(c, 3, 2, 16, 18)
	c.rect(4, 6, 14, 10, blueR[2])
	c.ellipse(10, 11, 4.5, 3, greenR[2])
	c.ellipse(9, 10, 2.5, 1.5, greenR[3])
	c.dot(15, 8, greenR[2])
	c.line(10, 6, 10, 9, woodR[0])
	c.dot(11, 6, redR[2])
	c.disc(17, 19, 3.6, sealR[1])
	c.disc(16.5, 18.5, 2.4, sealR[2])
	c.line(15, 21, 14, 23, twineR[2])
	c.line(18, 21, 19, 23, twineR[3])
}

func pageFamiliars(c *canvas) {
	c.rect(4, 16, 16, 6, woodR[2])
	c.rect(4, 16, 16, 1, woodR[3])
	c.rect(18, 17, 2, 5, woodR[1])
	raven := ramp{rgb(22, 20, 30), rgb(40, 38, 54), rgb(66, 62, 86), rgb(110, 104, 140)}
	c.ellipse(12, 11, 5.5, 4, raven[1])
	c.ellipse(10, 10, 3, 2, raven[2])
	c.disc(16, 6, 3, raven[1])
	c.shadePoly([]pt{{18, 5}, {23, 7}, {18, 8}}, flat(inkSft))
	c.dot(16, 5, goldR[3])
	c.shadePoly([]pt{{7, 12}, {1, 15}, {3, 11}}, flat(raven[0]))
	c.rect(11, 15, 1, 2, goldR[1])
	c.rect(14, 15, 1, 2, goldR[1])
}

func pageRanks(c *canvas) {
	c.shadePoly([]pt{{4, 2}, {20, 2}, {20, 12}, {12, 22}, {4, 12}}, flat(blueR[1]))
	c.shadePoly([]pt{{4, 2}, {12, 2}, {12, 22}, {4, 12}}, flat(blueR[2]))
	for i, y := range []int{4, 9, 14} {
		wideChevron(c, y, 5, []color.NRGBA{goldR[3], goldR[2], goldR[1]}[i])
	}
}

func pageKeepFloor(c *canvas) {
	s := iso{c: c, ox: 12, oy: 8}
	s.box(-0.35, -0.35, 0.35, 0.35, 0, 3, faces{top: func(x, y int) color.NRGBA {
		if (x+y)%4 == 0 {
			return woodR[1]
		}
		return woodR[2]
	}, left: flat(woodR[1]), right: flat(woodR[0])})
	for i, p := range [][2]int{{8, 9}, {12, 7}, {15, 10}} {
		c.rect(p[0]-2, p[1]-3, 4, 3, []color.NRGBA{pineR[2], plasterR[3], crateR[2]}[i])
		c.rect(p[0]-2, p[1]-3, 4, 1, cream)
	}
	c.rect(2, 20, 20, 3, ink)
	c.rect(3, 21, 12, 1, greenR[2])
	c.rect(15, 21, 4, 1, redR[2])
}

func pagePalimpsest(c *canvas) {
	c.shadePoly([]pt{{2, 12}, {12, 15}, {12, 22}, {2, 19}}, flat(tealR[1]))
	c.shadePoly([]pt{{12, 15}, {22, 12}, {22, 19}, {12, 22}}, flat(tealR[0]))
	c.shadePoly([]pt{{3, 11}, {12, 14}, {12, 20}, {3, 17}}, flat(cream))
	c.shadePoly([]pt{{12, 14}, {21, 11}, {21, 17}, {12, 20}}, flat(plasterR[3]))
	ringArc(c, 12, 7, 4, 5.8, func(a float64) bool { return a > 0.2 && a < 1.4 }, goldR)
	c.shadePoly([]pt{{15.5, 7}, {19.5, 7}, {17.5, 10}}, flat(goldR[2]))
}

func pageCrews(c *canvas) {
	c.rect(4, 1, 1, 21, woodR[1])
	c.shadePoly([]pt{{5, 2}, {20, 2}, {17, 6}, {20, 10}, {5, 10}}, flat(porterR[2]))
	c.rect(5, 2, 15, 1, porterR[3])
	c.dot(9, 6, goldR[3])
	c.dot(12, 6, goldR[3])
	c.dot(15, 6, goldR[3])
	for i := 0; i < 3; i++ {
		x := 9 + i*5
		c.disc(float64(x)+0.5, 14, 2, skinR[2])
		c.rect(x-1, 16, 4, 5, porterR[1+i%2])
	}
}

func pageDepots(c *canvas) {
	c.rect(3, 10, 14, 11, plasterR[3])
	c.rect(3, 10, 3, 11, cream)
	c.rect(15, 10, 2, 11, plasterR[1])
	c.shadePoly([]pt{{1, 11}, {10, 3}, {19, 11}}, func(x, y int) color.NRGBA {
		if (x+y)%3 == 0 {
			return blueR[1]
		}
		return blueR[2]
	})
	c.rect(8, 14, 4, 7, woodR[0])
	c.rect(4, 13, 3, 3, lampLit)
	c.rect(13, 13, 2, 3, lampLit)
	drawBox(c, 18.5, 19, boxDims{0.3125, 5}, "parcel", 0)
}

func pageCarters(c *canvas) {
	c.line(0, 21, 23, 16, dirtR[1])
	c.line(0, 22, 23, 17, dirtR[2])
	c.rect(4, 12, 13, 3, woodR[2])
	c.rect(4, 12, 13, 1, woodR[3])
	c.line(17, 13, 22, 9, woodR[1])
	drawBox(c, 10, 9.5, boxDims{0.3125, 5}, "crate", 0)
	for _, x := range []float64{7, 14} {
		c.disc(x, 16.5, 2.6, woodR[0])
		c.dot(int(x), 16, woodR[3])
	}
}

func pageRoadSigns(c *canvas) {
	c.rect(11, 3, 2, 19, woodR[1])
	c.rect(11, 3, 1, 19, woodR[2])
	arrowSign(c, 13, 5, 9, false, blueR[2])
	arrowSign(c, 11, 11, 9, true, amberR[2])
	c.rect(8, 21, 8, 2, grassR[2])
}

func arrowSign(c *canvas, x, y, w int, left bool, mark color.NRGBA) {
	fx, fy, fw := float64(x), float64(y), float64(w)
	if left {
		c.shadePoly([]pt{{fx, fy}, {fx - fw, fy}, {fx - fw - 3, fy + 2.5}, {fx - fw, fy + 5}, {fx, fy + 5}}, flat(woodR[3]))
		c.rect(x-w+1, y+2, 2, 2, mark)
		return
	}
	c.shadePoly([]pt{{fx, fy}, {fx + fw, fy}, {fx + fw + 3, fy + 2.5}, {fx + fw, fy + 5}, {fx, fy + 5}}, flat(woodR[3]))
	c.rect(x+w-3, y+2, 2, 2, mark)
}

func pageSealedDocks(c *canvas) {
	for i := 0; i < 5; i++ {
		c.line(1+i*4, 20, 7+i*4, 17, pierR[2+i%2])
		c.line(1+i*4, 21, 7+i*4, 18, pierR[1])
	}
	c.rect(3, 21, 1, 2, pierR[0])
	c.rect(19, 18, 1, 4, pierR[0])
	drawBox(c, 11, 14.5, boxDims{0.5, 8}, "crate", 0)
	c.disc(11, 13, 3.4, sealR[1])
	c.disc(10.5, 12.5, 2.3, sealR[2])
	c.dot(11, 13, sealR[0])
}
