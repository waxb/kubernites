package art

import (
	"image/color"
	"math"
)

var hlpBellCorners = [4][2]float64{{0.18, 0.18}, {1.82, 0.18}, {0.18, 1.82}, {1.82, 1.82}}

func paintBellWatch(s iso) {
	s.apron(2, 0.05)
	body := math.Round(44 * s.kit().scale)
	bell := body + 5
	for _, p := range hlpBellCorners[:3] {
		bellLine(s, p[0], p[1], bell)
	}
	t := towerSpec{cx: 1, cy: 1, r: 0.34}
	shape := s.towerShape()
	s.turn(lathe{cx: 1, cy: 1, h: body, r: t.r, shape: shape}, s.towerWall(), matStone)
	s.towerOpenings(t, s.doorPlan(t, body, 100, 11).upTo(1))
	hlpBelfry(s, t, body, shape)
	s.post(hlpBellCorners[3][0], hlpBellCorners[3][1], 0, 14)
	s.props([][2]float64{{0.55, 1.62}, {1.62, 0.5}}, 39)
}

func hlpBelfry(s iso, t towerSpec, body float64, shape int) {
	s.turn(lathe{cx: t.cx, cy: t.cy, z: body, h: 1, r: t.r + 0.03, shape: shape}, flatMat(stoneR[3]), matStone)
	d := t.r * 0.72
	posts := [][2]float64{{-d, -d}, {-d, d}, {d, -d}}
	wall := s.towerWall()
	for _, p := range posts {
		s.block(t.cx+p[0]-0.05, t.cy+p[1]-0.05, t.cx+p[0]+0.05, t.cy+p[1]+0.05, body, body+12, wall, wall, matStone)
	}
	c := s.p(t.cx, t.cy, body+6)
	metal := goldR
	if s.lk.age == ageDark {
		metal = ramp{rgb(92, 62, 40), rgb(140, 96, 52), rgb(176, 128, 70), rgb(214, 176, 110)}
	}
	s.c.ellipse(c.x, c.y, 3.5, 4, metal[1])
	s.c.ellipse(c.x-1, c.y-1, 1.8, 2.4, metal[3])
	s.c.rect(int(c.x)-4, int(c.y)+3, 9, 1, metal[0])
	s.block(t.cx+d-0.05, t.cy+d-0.05, t.cx+d+0.05, t.cy+d+0.05, body, body+12, wall, wall, matStone)
	s.towerRoof(t.cx, t.cy, t.r+0.1, body+12, 12, shape)
}

func bellLine(s iso, gx, gy, bell float64) {
	s.post(gx, gy, 0, 14)
	a, b := s.p(gx, gy, 14), s.p(1.0, 1.0, bell)
	s.aloft(14, func() {
		s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), goldR[1])
		s.c.dot(int(a.x), int(a.y)+1, goldR[3])
		s.c.dot(int(a.x)-1, int(a.y)+2, goldR[2])
		s.c.dot(int(a.x)+1, int(a.y)+2, goldR[2])
	})
}

func paintCacheStore(s iso) {
	s.apron(2, 0.04)
	if s.lk.age < ageEmpire {
		s.box(0.1, 0.1, 1.9, 1.9, 0, 2, faces{top: func(x, y int) color.NRGBA {
			gx, gy := s.ground(x, y, 2)
			return quayPlank(gx, gy, s.lk.variant)
		}, left: flat(quayR[1]), right: flat(quayR[0])})
	}
	x0, y0, x1, y1 := 0.3, 0.3, 1.7, 1.55
	top := s.hall(hallSpec{x0: x0, y0: y0, x1: x1, y1: y1, h: 24, rise: 14, alongY: true})
	s.doorway(sideL(y1), 0.48, 0.34, 12)
	s.window(sideR(x1), 0.65, math.Round(top*0.55))
	if s.lk.variant != 0 {
		s.window(sideR(x1), 1.05, math.Round(top*0.55))
	}
	squidBanner(s, y1, 1.02, top-1)
	s.kindBox(1.9, 1.3, 2, 1, "crate", 1)
	for i := 0; i < 2+s.lk.variant%2; i++ {
		s.kindBox(1.85, 1.78, 2+float64(i)*7, 1, "crate", 1)
	}
	s.props([][2]float64{{0.25, 1.85}}, 35)
}

func squidBanner(s iso, y1, u, z float64) {
	cloth := blueR
	s.pen(matTrim, func() {
		s.onLeft(y1, u, u+0.42, z-15, z, func(x, y int) color.NRGBA {
			if y%2 == 0 {
				return cloth[1]
			}
			return cloth[2]
		})
	})
	c := s.p(u+0.21, y1, z-5)
	squid := ramp{rgb(120, 50, 110), rgb(170, 80, 150), rgb(214, 124, 186), rgb(244, 186, 220)}
	s.c.ellipse(c.x, c.y-1, 2.5, 3, squid[2])
	s.c.dot(int(c.x)-1, int(c.y)-2, squid[3])
	for i := -2; i <= 2; i++ {
		s.c.line(int(c.x)+i, int(c.y)+2, int(c.x)+i+int(math.Copysign(1, float64(i))), int(c.y)+5, squid[1])
	}
	s.c.dot(int(c.x)-1, int(c.y)-1, ink)
	s.c.dot(int(c.x)+1, int(c.y)-1, ink)
}

func paintCaravanYard(s iso) {
	s.apron(3, 0.06)
	hlpFence(s)
	top := s.hall(hallSpec{x0: 0.45, y0: 0.45, x1: 1.35, y1: 1.2, h: 18, rise: 10})
	if s.lk.age >= ageFeudal && s.lk.variant != 1 {
		s.chimney(0.6, 0.6, top+2, top+10)
	}
	s.doorway(sideL(1.2), 0.75, 0.3, 11)
	s.window(sideR(1.35), 0.62, math.Round(top*0.5))
	wagon(s, 1.45, 1.45)
	signpost(s, 0.55, 2.4)
	s.kindBox(2.55, 0.6, 0, 1, "crate", 2)
	s.kindBox(2.2, 0.55, 0, 0, "parcel", 0)
	s.props([][2]float64{{2.55, 1.2}, {0.7, 1.75}}, 37)
}

func hlpFence(s iso) {
	for i := 0; i < 7; i++ {
		t := 0.2 + float64(i)*0.45
		lean := s.lk.jit(40+i, 0.03)
		hlpFencePost(s, t, 0.19+lean)
		hlpFencePost(s, 0.19+lean, t)
	}
	switch s.lk.age {
	case ageDark:
		s.block(0.2, 0.16, 2.9, 0.22, 1, 8, hlpWeave, flatMat(woodR[3]), matTrim)
		s.block(0.16, 0.2, 0.22, 2.9, 1, 8, hlpWeave, flatMat(woodR[3]), matTrim)
	case ageCastle:
		wall := rubbleMat(stoneR, s.lk.seed+11)
		s.block(0.2, 0.14, 2.9, 0.24, 0, 7, wall, flatMat(stoneR[3]), matStone)
		s.block(0.14, 0.2, 0.24, 2.9, 0, 7, wall, flatMat(stoneR[3]), matStone)
	default:
		rail := faces{top: flat(woodR[3]), left: flat(woodR[2]), right: flat(woodR[0])}
		if s.lk.age == ageEmpire {
			rail = faces{top: flat(goldR[2]), left: flat(inkSft), right: flat(ink)}
		}
		for _, z := range []float64{5, 9} {
			s.box(0.2, 0.17, 2.8, 0.21, z, z+1, rail)
			s.box(0.17, 0.2, 0.21, 2.8, z, z+1, rail)
		}
	}
}

func hlpFencePost(s iso, gx, gy float64) {
	switch s.lk.age {
	case ageCastle:
		return
	case ageEmpire:
		s.turn(lathe{cx: gx, cy: gy, h: 11, r: 0.06, shape: shapeSquare}, toneMat(marbleR), matStone)
		s.finial(gx, gy, 11, 2)
		return
	}
	col := faces{top: flat(woodR[3]), left: flat(woodR[2]), right: flat(woodR[0])}
	s.box(gx-0.04, gy-0.04, gx+0.04, gy+0.04, 0, 11+float64(s.lk.pick(int(gx*50+gy*70), 3)), col)
}

func wagon(s iso, gx, gy float64) {
	bed := faces{top: flat(woodR[2]), left: s.plankLeft(woodR, gy+0.6), right: s.plankRight(woodR, gx+1.1)}
	s.box(gx, gy, gx+1.1, gy+0.6, 6, 12, bed)
	for _, u := range []float64{0.2, 0.9} {
		wheelAt(s, gx+u, gy+0.62)
	}
	wagonCover(s, gx, gy)
	pole := s.p(gx+1.1, gy+0.3, 7)
	s.c.line(int(pole.x), int(pole.y), int(pole.x)+10, int(pole.y)+5, woodR[1])
}

func wagonCover(s iso, gx, gy float64) {
	const n = 8
	cloth := []color.NRGBA{plasterR[1], plasterR[2], plasterR[3], cream, cream, plasterR[3], plasterR[2], plasterR[1]}
	if s.lk.variant == 2 {
		for i := range cloth {
			cloth[i] = mix(cloth[i], s.lk.id[2], 0.35)
		}
	}
	for i := 0; i < n; i++ {
		t0, t1 := float64(i)/n, float64(i+1)/n
		z0, z1 := 12+12*math.Sin(math.Pi*t0), 12+12*math.Sin(math.Pi*t1)
		s.quad(s.p(gx+0.05, gy+0.6*t0, z0), s.p(gx+1.05, gy+0.6*t0, z0), s.p(gx+1.05, gy+0.6*t1, z1), s.p(gx+0.05, gy+0.6*t1, z1), flat(cloth[i]))
	}
	for _, u := range []float64{0.3, 0.7} {
		for i := 0; i < n; i++ {
			t0, t1 := float64(i)/n, float64(i+1)/n
			a := s.p(gx+u, gy+0.6*t0, 12+12*math.Sin(math.Pi*t0))
			b := s.p(gx+u, gy+0.6*t1, 12+12*math.Sin(math.Pi*t1))
			s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[1])
		}
	}
	mouth := s.p(gx+1.05, gy+0.3, 16)
	s.c.ellipse(mouth.x, mouth.y, 3, 4, rgb(60, 44, 36))
}

func wheelAt(s iso, gx, gy float64) {
	c := s.p(gx, gy, 5)
	s.c.ellipse(c.x, c.y, 3.2, 4.4, woodR[0])
	s.c.ellipse(c.x, c.y, 2, 3, woodR[2])
	s.c.dot(int(c.x), int(c.y), ink)
}

func signpost(s iso, gx, gy float64) {
	base, top := s.p(gx, gy, 0), s.p(gx, gy, 22)
	s.c.rect(int(base.x), int(top.y), 2, int(base.y-top.y), woodR[1])
	s.c.rect(int(base.x), int(top.y), 1, int(base.y-top.y), woodR[2])
	s.aloft(18, func() {
		hlpArrow(s.c, top.x-6, top.y+3, 7, woodR[3])
		hlpArrow(s.c, top.x+1, top.y+8, 7, plasterR[2])
	})
	if s.lk.age == ageEmpire {
		s.c.dot(int(top.x), int(top.y)-1, goldR[3])
	}
}

func hlpArrow(c *canvas, x, y, w float64, body color.NRGBA) {
	c.shadePoly([]pt{{x, y}, {x + w, y - 2}, {x + w + 3, y}, {x + w, y + 3}, {x, y + 3}}, flat(body))
	c.dot(int(x)+2, int(y)+1, ink)
	c.dot(int(x)+4, int(y)+1, ink)
}

func hlpWeave(p surf) color.NRGBA {
	row := int(math.Floor(p.v / 2))
	if (int(math.Floor(p.u/3))+row)%2 == 0 {
		return rampAt(woodR, level(p.light)+0.4)
	}
	return rampAt(woodR, level(p.light)-0.5)
}
