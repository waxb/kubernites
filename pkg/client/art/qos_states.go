package art

import "image/color"

var qosKept = map[string]int{"vagrant": 10, "freeman": 16, "landed": 20}

var qosScaffold = map[string]float64{"vagrant": 16, "freeman": 24, "landed": 32}

func paintQoSKindling(c *canvas, s size, h hut, m *hutMemo) {
	house := m.house(s, h, false)
	cut := s.ay - qosKept[h.qos]
	for y := cut; y < s.h; y++ {
		for x := 0; x < s.w; x++ {
			if col := house.at(x, y); col.A > 0 {
				c.dot(x, y, col)
			}
		}
	}
	fr := hearthFrame(c, s)
	top := qosScaffold[h.qos]
	scaffold(fr, 1, top)
	tierPennant(fr, 0.8, 0.8, top*0.9, tierRamp(h.tier))
	for i := 0; i < 3; i++ {
		a, b := fr.p(0.3+float64(i)*0.05, 0.93, float64(1+i)), fr.p(0.55+float64(i)*0.05, 0.93, float64(1+i))
		c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[2-i%2])
	}
}

func paintQoSPending(s iso, h hut) {
	switch h.qos {
	case "vagrant":
		roughSleeper(s, h)
	case "landed":
		pavilion(s, h)
		stakes(s, 1)
		for _, b := range [][2]float64{{0.76, 0.12}, {0.8, 0.3}} {
			s.block(b[0], b[1], b[0]+0.13, b[1]+0.13, 0, 4, s.kit().base, flatMat(stoneR[3]), matStone)
		}
	default:
		hutTent(s)
		tierPennant(s, 0.45, 0.45, 18, tierRamp(h.tier))
		logPile(s, 0.84, 0.26)
	}
}

func roughSleeper(s iso, h hut) {
	trodden(s.hutGround(), 11, 4.5)
	cloth := tentRamp(h.tier)
	a, b := s.p(0.3, 0.55, 0), s.p(0.75, 0.55, 0)
	for i := 0; i <= 10; i++ {
		t := float64(i) / 10
		x, y := a.x+(b.x-a.x)*t, a.y+(b.y-a.y)*t
		col := cloth[2]
		if i%3 == 0 {
			col = cloth[1]
		}
		s.c.disc(x, y-2.5, 2.6, col)
	}
	s.c.ellipse(b.x, b.y-2.5, 1.6, 2.6, cloth[3])
	s.c.dot(int(b.x), int(b.y)-3, cloth[0])
	s.c.disc(a.x-2, a.y-3.5, 2.2, skinR[2])
	s.c.rect(int(a.x)-4, int(a.y)-6, 4, 2, brownHair)
	bindle(s, s.p(0.22, 0.82, 0))
	fire := s.p(0.86, 0.86, 1)
	s.c.ellipse(fire.x, fire.y, 3, 1.5, rockR[1])
	s.c.shadePoly([]pt{{fire.x - 2, fire.y}, {fire.x, fire.y - 5}, {fire.x + 2, fire.y}}, flat(fireR[1]))
	s.c.dot(int(fire.x), int(fire.y)-2, fireR[3])
}

func bindle(s iso, p pt) {
	s.c.line(int(p.x)-4, int(p.y), int(p.x)+3, int(p.y)-9, woodR[1])
	s.c.disc(p.x+3.5, p.y-9, 2.4, rgb(150, 124, 88))
	s.c.dot(int(p.x)+3, int(p.y)-10, rgb(190, 168, 128))
}

func pavilion(s iso, h hut) {
	s.hutGround().apron(1, 0.2)
	r := tierRamp(h.tier)
	apex := s.p(0.48, 0.48, 22)
	stripes := func(lit bool) shader {
		return func(x, y int) color.NRGBA {
			switch {
			case (x/2)%2 == 0 && lit:
				return r[2]
			case (x/2)%2 == 0:
				return r[1]
			case lit:
				return plasterR[3]
			}
			return plasterR[1]
		}
	}
	s.c.shadePoly([]pt{s.p(0.2, 0.78, 2), s.p(0.78, 0.78, 2), apex}, stripes(true))
	s.c.shadePoly([]pt{s.p(0.78, 0.2, 2), s.p(0.78, 0.78, 2), apex}, stripes(false))
	door := s.p(0.5, 0.78, 2)
	s.c.shadePoly([]pt{{door.x - 3, door.y}, {door.x + 2, door.y + 1}, {door.x, door.y - 9}}, flat(inkSft))
	s.c.rect(int(apex.x)-1, int(apex.y)-3, 1, 3, woodR[0])
	s.c.disc(apex.x-0.5, apex.y-3.5, 1.2, goldR[2])
}

func logPile(s iso, gx, gy float64) {
	for i, o := range [][3]float64{{0, 0, 1.5}, {0, 0.14, 1.5}, {0, 0.28, 1.5}, {0.01, 0.07, 4.5}, {0.01, 0.21, 4.5}} {
		p := s.p(gx+o[0], gy+o[1], o[2])
		s.c.disc(p.x, p.y, 1.7, woodR[1+i%2])
		s.c.dot(int(p.x), int(p.y), woodR[3])
	}
}

func paintQoSRubble(s iso, h hut) {
	switch h.qos {
	case "vagrant":
		shackRubble(s, h)
	case "landed":
		s.hutGround().apron(1, 0.06)
		rubbleStones(s, h.tier, 5)
		roofSlab(s, tierRamp(h.tier), 0.2, 0.3)
		brokenFence(s)
	default:
		s.hutGround().apron(1, 0.12)
		roofSlab(s, tierRamp(h.tier), 0.25, 0.35)
		for i, l := range [][6]float64{{0.2, 0.7, 3, 0.8, 0.8, 5}, {0.3, 0.5, 5, 0.85, 0.35, 3}, {0.6, 0.2, 3, 0.7, 0.85, 7}} {
			a, b := s.p(l[0], l[1], l[2]), s.p(l[3], l[4], l[5])
			s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[1+i%2])
			s.c.line(int(a.x), int(a.y)-1, int(b.x), int(b.y)-1, woodR[2+i%2])
		}
	}
}

func rubbleStones(s iso, tier, n int) {
	for i, p := range [][2]float64{{0.25, 0.5}, {0.5, 0.3}, {0.7, 0.6}, {0.4, 0.72}, {0.62, 0.42}}[:n] {
		h := float64(3 + (i+tier)%4*2)
		s.block(p[0], p[1], p[0]+0.14, p[1]+0.14, 1, 1+h, s.kit().base, flatMat(stoneR[3]), matStone)
	}
}

func roofSlab(s iso, r ramp, gx, gy float64) {
	s.pen(matRoof, func() {
		s.quad(s.p(gx, gy, 9), s.p(gx+0.45, gy, 11), s.p(gx+0.5, gy+0.4, 3), s.p(gx+0.05, gy+0.4, 3), s.planeX(gy+0.4, -0.4, 3, 7, litFront, hutRoofMat(s.lk.age, r, s.lk.seed+17)))
	})
}

func hutTent(s iso) {
	s.hutGround().apron(1, 0.2)
	apex := s.p(0.45, 0.45, 18)
	s.c.shadePoly([]pt{s.p(0.2, 0.75, 2), s.p(0.75, 0.75, 2), apex}, flat(plasterR[2]))
	s.c.shadePoly([]pt{s.p(0.75, 0.2, 2), s.p(0.75, 0.75, 2), apex}, flat(plasterR[1]))
	door := s.p(0.48, 0.75, 2)
	s.c.shadePoly([]pt{{door.x - 3, door.y}, {door.x + 2, door.y + 1}, {door.x, door.y - 9}}, flat(inkSft))
	fire := s.p(0.9, 0.95, 2)
	s.c.ellipse(fire.x, fire.y, 3, 1.5, rockR[1])
	s.c.shadePoly([]pt{{fire.x - 2, fire.y}, {fire.x, fire.y - 6}, {fire.x + 2, fire.y}}, flat(fireR[1]))
	s.c.dot(int(fire.x), int(fire.y)-2, fireR[3])
}

func brokenFence(s iso) {
	for i := 0; i < 5; i++ {
		gx := 0.12 + float64(i)*0.18
		lean := float64(i%3-1) * 2
		a := s.p(gx, gardenEdge, 0)
		s.c.line(int(a.x), int(a.y), int(a.x+lean), int(a.y)-4+i%2, plasterR[3])
	}
}
