package art

import (
	"image/color"
	"math"
)

var runeR = ramp{rgb(96, 70, 170), rgb(140, 110, 230), rgb(186, 160, 255), rgb(236, 224, 255)}

func facet(c *canvas, pts []pt, r ramp, seed int) {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, p := range pts {
		minX, minY, maxX, maxY = math.Min(minX, p.x), math.Min(minY, p.y), math.Max(maxX, p.x), math.Max(maxY, p.y)
	}
	cx, cy := (minX+maxX)/2, (minY+maxY)/2
	w, h := math.Max(maxX-minX, 1), math.Max(maxY-minY, 1)
	c.shadePoly(pts, func(x, y int) color.NRGBA {
		l := -((float64(x)-cx)/w*1.2 + (float64(y)-cy)/h*1.4) + (noise(x, y, seed)-0.5)*0.4
		switch {
		case l > 0.55:
			return r[3]
		case l > 0.05:
			return r[2]
		case l < -0.55:
			return r[0]
		}
		return r[1]
	})
}

func paintRock(c *canvas, s size, v int) {
	shapes := [][]pt{
		{{1, 9}, {3, 4}, {7, 2}, {11, 4}, {13, 9}},
		{{2, 9}, {4, 3}, {9, 3}, {12, 8}},
		{{1, 9}, {2, 6}, {6, 4}, {9, 5}, {12, 9}},
	}
	facet(c, shapes[v], rockR, 90+v)
	if v == 1 {
		c.shadePoly([]pt{{9, 9}, {11, 6}, {14, 7}, {14, 9}}, flat(rockR[1]))
	}
}

func paintBoulder(c *canvas, s size, v int) {
	shapes := [][]pt{
		{{1, 14}, {2, 7}, {7, 2}, {14, 1}, {19, 5}, {21, 13}},
		{{2, 14}, {3, 5}, {10, 2}, {17, 4}, {20, 14}},
		{{1, 14}, {4, 6}, {9, 4}, {12, 6}, {16, 3}, {21, 8}, {20, 14}},
	}
	facet(c, shapes[v], rockR, 95+v)
	c.line(9+v, 5, 11+v, 10, rockR[0])
	for i := 0; i < 9; i++ {
		x, y := 4+int(noise(i, v, 97)*12), 2+int(noise(v, i, 98)*4)
		if c.at(x, y).A > 0 {
			c.dot(x, y, mossR[2+i%2])
			c.dot(x+1, y, mossR[2])
		}
	}
}

func paintLog(c *canvas, s size, v int) {
	c.shadePoly([]pt{{3, 7}, {17, 1}, {22, 4}, {8, 10}}, func(x, y int) color.NRGBA {
		switch {
		case (x+2*y)%5 == 0:
			return barkR[0]
		case y+x/3 < 7:
			return barkR[2]
		}
		return barkR[1]
	})
	c.ellipse(5.5, 8, 2.6, 2.2, woodR[3])
	c.ellipse(5.5, 8, 1.4, 1.1, woodR[2])
	c.dot(5, 8, woodR[1])
	for i := 0; i < 4; i++ {
		c.dot(10+i*3, 4-i/2, mossR[2+i%2])
	}
	if v > 0 {
		c.dot(15, 7, []color.NRGBA{clear, redR[2], goldR[2]}[v])
		c.dot(15, 8, cream)
	}
}

func paintStump(c *canvas, s size, v int) {
	c.rect(3, 4, 8, 5, barkR[1])
	c.rect(3, 4, 2, 5, barkR[2])
	c.rect(9, 4, 2, 5, barkR[0])
	c.ellipse(7, 4, 4.5, 2, woodR[3])
	c.ellipse(7, 4, 2.6, 1.1, woodR[2])
	c.dot(7, 4, woodR[1])
	roots(c, 3, 8, 8)
	if v == 1 {
		c.dot(11, 7, mossR[3])
		c.dot(2, 7, mossR[2])
	}
	if v == 2 {
		c.ellipse(12, 6, 1.6, 1, redR[2])
		c.dot(12, 7, cream)
	}
}

func paintLeyStone(c *canvas, s size, v int) {
	shapes := [][]pt{
		{{3, 17}, {3, 9}, {5, 6}, {8, 6}, {9, 10}, {9, 17}},
		{{2, 17}, {3, 4}, {6, 1}, {9, 3}, {10, 17}},
		{{1, 17}, {1, 11}, {3, 9}, {5, 11}, {5, 17}},
	}
	facet(c, shapes[v], greyR, 110+v)
	if v == 2 {
		facet(c, []pt{{7, 17}, {7, 8}, {9, 6}, {11, 8}, {11, 17}}, greyR, 113)
	}
	glyph := [][][2]int{{{6, 9}, {6, 13}, {5, 11}, {7, 11}}, {{6, 6}, {6, 12}, {5, 8}, {7, 10}, {5, 12}}, {{3, 12}, {3, 15}, {9, 10}, {9, 14}}}[v]
	for i, p := range glyph {
		c.dot(p[0], p[1], runeR[2+i%2])
	}
	c.dot(4, s.ay-1, mossR[2])
}
