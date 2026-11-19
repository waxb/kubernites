package art

import "image/color"

const (
	forgeTop   = 46.0
	chimneyW   = 0.22
	sawRadius  = 3.6
	logEndSize = 2.1
)

var (
	sawdustR = ramp{rgb(170, 132, 78), rgb(206, 170, 108), rgb(232, 204, 146), rgb(248, 230, 184)}
	brickR   = ramp{rgb(92, 36, 32), rgb(138, 58, 44), rgb(174, 84, 58), rgb(208, 122, 86)}
)

func (r yardRect) alongX() bool { return r.x1-r.x0 >= r.y1-r.y0 }

func (r yardRect) inset(d float64) yardRect { return yardRect{r.x0 + d, r.y0 + d, r.x1 - d, r.y1 - d} }

func (r yardRect) mid() (float64, float64) { return (r.x0 + r.x1) / 2, (r.y0 + r.y1) / 2 }

func (r yardRect) cut(from, to float64) yardRect {
	if r.alongX() {
		w := r.x1 - r.x0
		return yardRect{r.x0 + w*from, r.y0, r.x0 + w*to, r.y1}
	}
	h := r.y1 - r.y0
	return yardRect{r.x0, r.y0 + h*from, r.x1, r.y0 + h*to}
}

func (s iso) swapYards(y keepYard) (yardRect, yardRect) {
	if yardsSwapped(s.lk) {
		return y.right, y.left
	}
	return y.left, y.right
}

func (s iso) keepMotifs(k KeepLook, y keepYard) {
	a, b := s.swapYards(y)
	switch k.Class {
	case "compute":
		s.keepSawmill(a)
		s.keepForge(b)
	case "memory":
		s.keepGranary(a)
		s.keepVault(b)
		s.keepStones(y.front[0], y.front[1], 5)
	case "rich":
		s.keepSawmill(a)
		s.keepGranary(b)
		s.keepStones(y.front[0], y.front[1], 3)
		s.keepBanners(k, a, b)
	default:
		s.keepWoodpile(a)
		gx, gy := b.mid()
		s.keepStones(gx, gy, 4)
	}
}

func (r yardRect) depth(from, to float64) yardRect {
	if r.alongX() {
		d := r.y1 - r.y0
		return yardRect{r.x0, r.y0 + d*from, r.x1, r.y0 + d*to}
	}
	d := r.x1 - r.x0
	return yardRect{r.x0 + d*from, r.y0, r.x0 + d*to, r.y1}
}

func (s iso) keepSawmill(r yardRect) {
	r = r.inset(0.05)
	shed := r.depth(0, 0.6)
	h := 13 * s.kit().scale
	for _, c := range [4][2]float64{{shed.x0, shed.y0}, {shed.x1, shed.y0}, {shed.x0, shed.y1}, {shed.x1, shed.y1}} {
		s.post(c[0], c[1], 0, h)
	}
	s.roofShed(shed.x0-0.06, shed.y0-0.06, shed.x1+0.06, shed.y1+0.06, h-3, h+3, r.alongX(), plankMat(woodR, s.lk.seed+41))
	s.keepSawLog(r.depth(0.62, 1))
}

func (s iso) keepSawLog(r yardRect) {
	gx, gy := r.mid()
	bark := faces{top: flat(woodR[2]), left: flat(woodR[1]), right: flat(woodR[0])}
	var end pt
	s.pen(matTrim, func() {
		if r.alongX() {
			s.box(r.x0+0.08, gy-0.07, r.x1-0.08, gy+0.07, 3, 7, bark)
			end = s.p(r.x1-0.08, gy, 5)
		} else {
			s.box(gx-0.07, r.y0+0.08, gx+0.07, r.y1-0.08, 3, 7, bark)
			end = s.p(gx, r.y1-0.08, 5)
		}
		s.c.disc(end.x, end.y, logEndSize, woodR[1])
		s.c.disc(end.x, end.y, logEndSize-0.8, woodR[3])
	})
	s.keepSawBlade(s.p(gx, gy, 10))
	dust := s.p(gx+0.12, gy+0.12, 0)
	s.pen(matTrim, func() {
		s.c.ellipse(dust.x, dust.y, 3.5, 1.4, sawdustR[2])
		s.c.dot(int(dust.x)-1, int(dust.y)-1, sawdustR[3])
	})
}

func (s iso) keepSawBlade(c pt) {
	s.pen(matTrim, func() {
		s.c.ellipse(c.x, c.y, sawRadius, sawRadius+0.6, greyR[1])
		s.c.ellipse(c.x-0.5, c.y-0.5, sawRadius-1, sawRadius-0.6, greyR[3])
		s.c.dot(int(c.x), int(c.y), ink)
		for _, d := range [][2]int{{0, -5}, {4, -2}, {4, 2}, {0, 5}, {-4, 2}, {-4, -2}} {
			s.c.dot(int(c.x)+d[0], int(c.y)+d[1], greyR[0])
		}
	})
}

func forgeSpot(r yardRect) (float64, float64) {
	gx, gy := r.cut(0.42, 0.62).mid()
	if r.alongX() {
		return gx - chimneyW/2, r.y1 - chimneyW - 0.02
	}
	return r.x1 - chimneyW - 0.02, gy - chimneyW/2
}

func (s iso) keepForge(r yardRect) {
	r = r.inset(0.04)
	hut := r.cut(0, 0.4)
	s.leanTo(hut.x0, hut.y0, hut.x1, hut.y1, 9*s.kit().scale, 13*s.kit().scale, hut.alongX())
	s.keepForgeMouth(hut)
	cx, cy := forgeSpot(r)
	top := forgeTop * s.kit().scale
	s.block(cx, cy, cx+chimneyW, cy+chimneyW, 0, top, rubbleMat(brickR, s.lk.seed+61), flatMat(ink), matStone)
	s.block(cx-0.03, cy-0.03, cx+chimneyW+0.03, cy+chimneyW+0.03, top, top+chimneyRim, flatMat(brickR[3]), flatMat(ink), matStone)
	s.keepForgeGlow(cx, cy, top)
	gx, gy := r.cut(0.72, 1).mid()
	s.keepLumber(gx, gy)
}

func (s iso) keepForgeGlow(cx, cy, top float64) {
	mouth := s.p(cx+chimneyW/2, cy+chimneyW/2, top+2)
	s.pen(matTrim, func() {
		s.c.rect(int(mouth.x)-3, int(mouth.y)-1, 6, 2, fireR[1])
		s.c.rect(int(mouth.x)-2, int(mouth.y)-2, 4, 1, fireR[2])
		s.c.dot(int(mouth.x), int(mouth.y)-3, fireR[3])
		for i, d := range [][2]int{{-2, -5}, {2, -7}, {0, -10}, {3, -12}, {-3, -9}} {
			if s.lk.chance(120+i, 0.75) {
				s.c.dot(int(mouth.x)+d[0], int(mouth.y)+d[1], []color.NRGBA{fireR[2], goldR[3], fireR[3]}[i%3])
			}
		}
	})
}

func (s iso) keepForgeMouth(r yardRect) {
	gx, gy := r.mid()
	s.pen(matGlass, func() {
		if r.alongX() {
			s.onLeft(r.y1, gx-0.1, gx+0.1, 1, 6, emberGlass(true))
			return
		}
		s.onRight(r.x1, gy-0.1, gy+0.1, 1, 6, emberGlass(true))
	})
}

func (s iso) keepLumber(gx, gy float64) {
	s.pen(matTrim, func() {
		for row, n := range []int{3, 2, 1} {
			for i := range n {
				p := s.p(gx+0.12, gy-0.13+float64(i)*0.13+float64(row)*0.065, 2+float64(row)*3.4)
				s.c.disc(p.x, p.y, logEndSize, woodR[1])
				s.c.disc(p.x, p.y, logEndSize-0.8, woodR[3])
				s.c.dot(int(p.x), int(p.y), woodR[2])
			}
		}
	})
}

func (s iso) keepWoodpile(r yardRect) {
	gx, gy := r.mid()
	logPile(s, gx-0.15, gy-0.1)
	stump := s.p(gx+0.25, gy+0.15, 0)
	s.pen(matTrim, func() {
		s.c.rect(int(stump.x)-2, int(stump.y)-4, 5, 4, woodR[1])
		s.c.ellipse(stump.x+0.5, stump.y-4, 2.5, 1, woodR[3])
		s.c.line(int(stump.x)+1, int(stump.y)-5, int(stump.x)+3, int(stump.y)-9, woodR[2])
		s.c.dot(int(stump.x)+3, int(stump.y)-10, greyR[2])
	})
}
