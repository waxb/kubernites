package art

import "math"

const (
	ivyVines   = 2
	ivyMinH    = 10
	ivySpanH   = 14
	ivyLeafGap = 3
	ivyMargin  = 2
)

func wallPixel(c *canvas, x, y int) bool {
	t := c.tagAt(x, y)
	return (t == matStone || t == matWall) && c.at(x-1, y).A > 0 && c.at(x+1, y).A > 0 && c.at(x, y-1).A > 0
}

func ivyRoot(c *canvas, x int) (int, bool) {
	for y := c.h - 1; y >= 0; y-- {
		if c.at(x, y).A == 0 {
			continue
		}
		if !wallPixel(c, x, y) || c.liftAt(x, y) > 3 {
			return 0, false
		}
		return y, true
	}
	return 0, false
}

func ivyRoots(c *canvas, l look) [][2]int {
	cands := [][2]int{}
	for x := ivyMargin; x < c.w-ivyMargin; x++ {
		y, ok := ivyRoot(c, x)
		_, okL := ivyRoot(c, x-ivyMargin)
		_, okR := ivyRoot(c, x+ivyMargin)
		if ok && okL && okR {
			cands = append(cands, [2]int{x, y})
		}
	}
	out := [][2]int{}
	for i := 0; i < ivyVines && len(cands) > 0; i++ {
		k := l.pick(400+i, len(cands))
		out = append(out, cands[k])
		cands = append(cands[:max(k-6, 0)], cands[min(k+7, len(cands)):]...)
	}
	return out
}

func ivy(c *canvas, l look) {
	if l.age != ageCastle {
		return
	}
	for i, r := range ivyRoots(c, l) {
		vine(c, r[0], r[1], ivyMinH+l.pick(410+i, ivySpanH), float64(i)*1.7+l.roll(420+i)*3)
	}
}

func vine(c *canvas, x0, y0, h int, phase float64) {
	for k := 0; k < h; k++ {
		y := y0 - k
		x := x0 + int(math.Round(math.Sin(float64(k)*0.45+phase)))
		if !wallPixel(c, x, y) {
			return
		}
		c.img.SetNRGBA(x, y, forestR[1])
		if k%ivyLeafGap == 1 {
			side := 1 - 2*(k/ivyLeafGap%2)
			leaf(c, x+side, y, side)
		}
	}
}

func leaf(c *canvas, x, y, side int) {
	for _, p := range [][3]int{{0, -1, 3}, {0, 0, 2}, {side, 0, 2}, {side, -1, 3}, {0, 1, 0}, {side, 1, 1}} {
		if wallPixel(c, x+p[0], y+p[1]) {
			c.img.SetNRGBA(x+p[0], y+p[1], greenR[p[2]])
		}
	}
}
