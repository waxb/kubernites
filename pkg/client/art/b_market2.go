package art

import "image/color"

func init() {
	agedKind("wayfinder", paintWayfinder)
}

var wfTops = [AgeTiers]float64{26, 31, 34, 34}

func paintWayfinder(s iso) {
	s.apron(1, 0.1)
	top := wfTops[s.lk.age]
	switch s.lk.age {
	case ageDark, ageFeudal:
		wfCairn(s)
		wfPost(s, top)
	case ageCastle:
		wfPlinth(s)
		wfIron(s, top)
	default:
		wfPlinth(s)
		wfGilded(s, top)
	}
	wfSigns(s, top)
}

func wfCairn(s iso) {
	n := 3 + s.lk.pick(3, 2)
	for i, p := range [][2]float64{{0.3, 0.75}, {0.72, 0.3}, {0.75, 0.72}, {0.25, 0.4}}[:n] {
		q := s.p(p[0], p[1], 0)
		r := 2.5 + s.lk.jit(4+i, 0.6)
		s.c.ellipse(q.x, q.y-1, r, r*0.7, rockR[1+i%2])
		s.c.dot(int(q.x)-1, int(q.y)-2, rockR[3])
	}
}

func wfPost(s iso, top float64) {
	base, t := s.p(0.5, 0.5, 0), s.p(0.5, 0.5, top)
	lean := 0.0
	if s.lk.age == ageDark {
		lean = s.lk.jit(6, 1.5)
	}
	s.pen(matTrim, func() {
		for i, col := range []color.NRGBA{woodR[2], woodR[1], woodR[0]} {
			s.c.line(int(base.x)-1+i, int(base.y), int(t.x+lean)-1+i, int(t.y), col)
		}
		s.c.dot(int(t.x+lean), int(t.y)-1, woodR[3])
	})
}

func wfPlinth(s iso) {
	k := s.kit()
	if s.lk.age == ageEmpire {
		s.block(0.2, 0.2, 0.8, 0.8, 0, 2, k.base, flatMat(marbleR[3]), matStone)
	}
	s.block(0.3, 0.3, 0.7, 0.7, 0, 5, k.wall, flatMat(stoneR[3]), matStone)
}

func wfIron(s iso, top float64) {
	base, t := s.p(0.5, 0.5, 5), s.p(0.5, 0.5, top)
	s.pen(matTrim, func() {
		s.c.rect(int(base.x)-1, int(t.y), 2, int(base.y-t.y), greyR[0])
		s.c.rect(int(base.x)-1, int(t.y), 1, int(base.y-t.y), greyR[2])
	})
	s.aloft(top-6, func() {
		s.c.rect(int(t.x), int(t.y)-1, 6, 1, greyR[0])
		s.c.rect(int(t.x)+4, int(t.y), 1, 2, ink)
		s.c.rect(int(t.x)+3, int(t.y)+2, 3, 4, lampLit)
		s.c.dot(int(t.x)+4, int(t.y)+3, white)
		s.c.rect(int(t.x)+3, int(t.y)+6, 3, 1, greyR[0])
	})
}

func wfGilded(s iso, top float64) {
	s.turn(lathe{cx: 0.5, cy: 0.5, z: 5, h: top - 5, r: 0.045}, toneMat(goldR), matTrim)
	s.turn(lathe{cx: 0.5, cy: 0.5, z: top - 1, h: 2, r: 0.09}, toneMat(goldR), matTrim)
	t := s.p(0.5, 0.5, top+1)
	s.aloft(top, func() {
		s.c.rect(int(t.x)-2, int(t.y)-6, 5, 6, goldR[1])
		s.c.rect(int(t.x)-1, int(t.y)-5, 3, 4, lampLit)
		s.c.dot(int(t.x), int(t.y)-4, white)
		s.c.rect(int(t.x)-2, int(t.y)-7, 5, 1, goldR[3])
		s.c.dot(int(t.x), int(t.y)-8, goldR[2])
	})
}

func wfSignColors(s iso) (color.NRGBA, color.NRGBA) {
	switch s.lk.age {
	case ageDark:
		return woodR[2], woodR[0]
	case ageFeudal:
		return woodR[3], woodR[1]
	case ageCastle:
		return plasterR[3], plasterR[1]
	}
	return goldR[2], goldR[1]
}

func wfSigns(s iso, top float64) {
	t := s.p(0.5, 0.5, top)
	body, edge := wfSignColors(s)
	n := 2 + min(s.lk.age, 1)
	for i := 0; i < n; i++ {
		left := (i+s.lk.variant)%2 == 1
		w := 6 + float64(s.lk.pick(10+i, 3))
		y := t.y + 4 + float64(i)*5 + s.lk.jit(12+i, 0.6) + float64(boolInt(s.lk.age == ageCastle)*4)
		x := t.x + 1
		if left {
			x = t.x - w - 3
		}
		s.aloft(top-8, func() { mkArrow(s.c, x, y, w, body, edge, left) })
	}
}

func mkArrow(c *canvas, x, y, w float64, body, edge color.NRGBA, left bool) {
	pts := []pt{{x, y}, {x + w, y - 1}, {x + w + 3, y + 1}, {x + w, y + 3}, {x, y + 3}}
	if left {
		pts = []pt{{x + w + 3, y}, {x + 3, y - 1}, {x, y + 1}, {x + 3, y + 3}, {x + w + 3, y + 3}}
	}
	c.shadePoly(pts, flat(body))
	c.line(int(x)+1, int(y)+3, int(x+w)+2, int(y)+3, edge)
	for i := 0; i < int(w)-2; i += 2 {
		c.dot(int(x)+2+i+boolInt(left)*2, int(y)+1, ink)
	}
}
