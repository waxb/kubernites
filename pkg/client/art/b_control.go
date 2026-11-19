package art

import "image/color"

func init() {
	agedKind("codex", paintCodex)
	agedKind("chancery", paintChancery)
	agedKind("stewards-hall", paintStewardsHall)
}

var ctlWindowZ = [AgeTiers]float64{7, 12, 11, 9}

func paintCodex(s iso) {
	s.apron(2, 0.04)
	s.hall(hallSpec{x0: 1.15, y0: 0.25, x1: 1.8, y1: 0.9, h: 20, rise: 10, alongY: true})
	if s.lk.age >= ageFeudal && s.lk.chance(3, 0.6) {
		s.chimney(1.5, 0.35, 16, 28)
	}
	s.layer(codexTower)
	if s.lk.age >= ageCastle {
		s.banner(sideL(0.9), 1.3, 17, 0.14, 9)
	}
	giantBook(s, 1.7, 1.45, 10)
	s.props([][2]float64{{0.3, 1.75}, {1.92, 1.9}}, 11)
}

func codexTower(s iso) {
	t := towerSpec{cx: 0.75, cy: 0.8, r: 0.45, h: 50, roofH: 18}
	top := s.tower(t)
	s.towerOpenings(t, s.doorPlan(t, top, 105, 12))
}

func giantBook(s iso, gx, gy, z float64) {
	c := s.p(gx, gy, z)
	x, y := c.x, c.y
	s.c.shadePoly([]pt{{x - 11, y - 2}, {x, y + 3}, {x, y + 5}, {x - 11, y}}, flat(woodR[0]))
	s.c.shadePoly([]pt{{x, y + 3}, {x + 11, y - 2}, {x + 11, y}, {x, y + 5}}, flat(woodR[0]))
	s.c.shadePoly([]pt{{x - 10, y - 8}, {x, y - 3}, {x, y + 3}, {x - 10, y - 2}}, pageShader(true))
	s.c.shadePoly([]pt{{x, y - 3}, {x + 10, y - 8}, {x + 10, y - 2}, {x, y + 3}}, pageShader(false))
	s.c.line(int(x), int(y-3), int(x), int(y+3), inkSft)
	s.c.rect(int(x)+2, int(y), 1, 6, redR[2])
	leg := s.p(gx, gy, 0)
	s.c.rect(int(leg.x)-1, int(leg.y)-int(z), 3, int(z), woodR[1])
}

func pageShader(left bool) shader {
	return func(x, y int) color.NRGBA {
		if y%2 == 0 && (x+y)%5 != 0 {
			return rgb(120, 100, 80)
		}
		if left {
			return cream
		}
		return rgb(236, 220, 176)
	}
}

const (
	chX0, chY0, chY1    = 0.35, 0.85, 2.0
	chWingX0, chWingX1  = 1.0, 1.7
	chWingY0, chWingY1  = 1.9, 2.66
	chHallH, chHallRise = 26.0, 16.0
)

func chanceryEnd(s iso) float64 {
	if s.lk.age >= ageCastle {
		return 2.25
	}
	return 2.62
}

func paintChancery(s iso) {
	s.apron(3, 0.06)
	if s.lk.age == ageCastle {
		t := towerSpec{cx: 0.55, cy: 0.72, r: 0.26, h: 44, roofH: 15}
		crest := s.tower(t)
		s.towerOpenings(t, s.windowPlan(t, chHallH+4, crest))
	}
	x1 := chanceryEnd(s)
	var top float64
	s.layer(func(l iso) {
		top = l.hall(hallSpec{x0: chX0, y0: chY0, x1: x1, y1: chY1, h: chHallH, rise: chHallRise})
	})
	if s.lk.age >= ageCastle {
		s.layer(func(l iso) { l.apse(x1, 1.05, 1.8, 21, 11) })
	}
	if s.lk.age == ageFeudal && s.lk.chance(5, 0.7) {
		s.chimney(2.3+s.lk.jit(6, 0.08), 1.1, top, top+chHallRise+3)
	}
	chanceryWindows(s, x1)
	if s.lk.age == ageEmpire {
		chanceryDome(s, top)
	}
	chanceryFront(s)
	s.props([][2]float64{{0.45, 2.5}, {2.6, 2.45}}, 21)
}

func chanceryWindows(s iso, x1 float64) {
	z := ctlWindowZ[s.lk.age]
	for i, u := range []float64{0.5, 1.9, 2.25} {
		if u+0.2 > x1 || (i > 0 && s.lk.chance(30+i, 0.25)) {
			continue
		}
		s.window(sideL(chY1), u, z)
	}
	if s.lk.age <= ageFeudal {
		s.window(sideR(x1), 1.2+s.lk.jit(33, 0.15), z)
	}
	if s.lk.age >= ageCastle {
		s.banner(sideL(chY1), 0.8, z+13, 0.12, 10)
		s.banner(sideL(chY1), 1.74, z+13, 0.12, 10)
	}
}

func chanceryDome(s iso, top float64) {
	cx, cy := 1.3, (chY0+chY1)/2
	z := top + chHallRise - 6
	s.turn(lathe{cx: cx, cy: cy, z: z, h: 9, r: 0.34}, s.kit().wall, matStone)
	drum := towerSpec{cx: cx, cy: cy, r: 0.34, round: true}
	s.towerOpenings(drum, s.ringPlan(drum, z+2, 3))
	s.turn(lathe{cx: cx, cy: cy, z: z + 8, h: 1, r: 0.37}, ribbedMat(goldR, s.lk.seed), matTrim)
	s.roofDome(cx, cy, 0.4, z+9, 18, s.domeMat())
	s.finial(cx, cy, z+27, 6)
}

func chanceryFront(s iso) {
	if s.lk.age == ageDark {
		s.porch(chY1, chWingX0+0.08, chWingX1-0.08, 0.45, 11)
		s.doorway(sideL(chY1), 1.2, 0.3, 13)
		greatSeal(s, 1.35, chY1+0.47, 17)
		return
	}
	wy1 := chWingY1
	if s.lk.age == ageEmpire {
		wy1 = chWingY1 - 0.2
	}
	top := 22 * s.kit().scale
	s.layer(func(l iso) { chanceryWing(l, wy1, top) })
	s.doorway(sideL(wy1), 1.22, 0.26, 13)
	greatSeal(s, 1.35, chWingY1, top+5)
}

func chanceryWing(s iso, wy1, top float64) {
	k := s.kit()
	s.walls(chWingX0, chWingY0, chWingX1, wy1, 0, top)
	rb := roofBox{chWingX0, chWingY0, chWingX1, chWingY1, top, 11 * k.rise, k.eave}
	s.roofGableY(rb, s.roofMat(), k.wall)
	if s.lk.age != ageEmpire {
		return
	}
	s.paint(sideL(wy1), matTrim, chWingX0, chWingX1, 0, top, marbleR[0])
	for _, u := range []float64{chWingX0 + 0.05, chWingX0 + 0.19, chWingX1 - 0.19, chWingX1 - 0.05} {
		s.turn(lathe{cx: u, cy: chWingY1 - 0.05, h: top, r: 0.05}, toneMat(marbleR), matStone)
	}
	s.giltRidge(rb, true, false)
}

func greatSeal(s iso, gx, gy, z float64) {
	c := s.p(gx, gy, z)
	s.c.disc(c.x, c.y, 4.5, goldR[1])
	s.c.disc(c.x-0.5, c.y-0.5, 3.5, goldR[2])
	s.c.disc(c.x, c.y, 1.8, redR[2])
	s.c.dot(int(c.x)-2, int(c.y)-3, goldR[3])
}

func mapTable(s iso, gx, gy float64) {
	leg := faces{top: flat(woodR[2]), left: flat(woodR[1]), right: flat(woodR[0])}
	for _, d := range [][2]float64{{0.04, 0.26}, {0.4, 0.26}, {0.4, 0.04}} {
		s.box(gx+d[0], gy+d[1], gx+d[0]+0.06, gy+d[1]+0.06, 0, 6, leg)
	}
	s.box(gx, gy, gx+0.5, gy+0.35, 6, 8, faces{top: flat(woodR[2]), left: flat(woodR[2]), right: flat(woodR[1])})
	s.quad(s.p(gx+0.05, gy+0.04, 8), s.p(gx+0.45, gy+0.04, 8), s.p(gx+0.45, gy+0.31, 8), s.p(gx+0.05, gy+0.31, 8), flat(cream))
	for _, p := range [][2]float64{{0.15, 0.1}, {0.3, 0.2}, {0.4, 0.12}} {
		q := s.p(gx+p[0], gy+p[1], 8)
		s.c.dot(int(q.x), int(q.y), redR[2])
		s.c.dot(int(q.x), int(q.y)-1, redR[3])
	}
	a, b := s.p(gx+0.1, gy+0.25, 8), s.p(gx+0.4, gy+0.08, 8)
	s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), blueR[2])
}

const (
	stX0, stY0, stX1, stY1 = 0.28, 0.3, 1.72, 1.1
	stWingX1, stWingY1     = 0.95, 1.74
)

func paintStewardsHall(s iso) {
	s.apron(2, 0.05)
	mainH := 28.0
	if s.lk.age == ageEmpire {
		mainH = 32
	}
	top := s.hall(hallSpec{x0: stX0, y0: stY0, x1: stX1, y1: stY1, h: mainH, rise: 13})
	if s.lk.age >= ageFeudal && s.lk.chance(7, 0.65) {
		s.chimney(0.55+s.lk.jit(8, 0.15), 0.45, top, top+15)
	}
	if s.lk.age >= ageFeudal {
		s.dormer(1.35+s.lk.jit(9, 0.1), stY1-0.1, top+2, 0.26, 6)
	}
	stewardsMain(s, top)
	s.layer(stewardsWing)
	ledgers(s, 1.5, 1.62)
	s.props([][2]float64{{1.85, 1.85}, {1.15, 1.9}}, 31)
}

func stewardsWing(s iso) {
	wing := s.hall(hallSpec{x0: stX0, y0: stY1 - 0.05, x1: stWingX1, y1: stWingY1, h: 22, rise: 10, alongY: true})
	s.window(sideL(stWingY1), 0.5, ctlWindowZ[s.lk.age])
	s.window(sideR(stWingX1), 1.38, ctlWindowZ[s.lk.age])
	if s.lk.age >= ageCastle {
		s.banner(sideL(stWingY1), 0.36, wing-3, 0.1, 8)
	}
}

func stewardsMain(s iso, top float64) {
	z := ctlWindowZ[s.lk.age]
	s.doorway(sideL(stY1), 1.05, 0.3, 12)
	s.window(sideL(stY1), 1.48, z)
	s.window(sideR(stX1), 0.34, z)
	if s.lk.age != ageCastle && s.lk.chance(12, 0.5) {
		s.window(sideR(stX1), 0.84, z)
	}
	if s.lk.age == ageCastle {
		s.layer(func(l iso) { l.turret(stX1-0.04, stY1-0.04, 0.17, 12, top-4) })
	}
	bigKey(s, stX1, 0.6, top-6)
}

func bigKey(s iso, gx, gy, z float64) {
	c := s.p(gx, gy, z)
	x, y := int(c.x)+3, int(c.y)
	s.c.line(x-3, y-2, x, y, woodR[0])
	s.c.disc(float64(x)+1.5, float64(y)+2.5, 2.5, goldR[2])
	s.c.dot(x+1, y+2, inkSft)
	s.c.rect(x+1, y+4, 1, 7, goldR[1])
	s.c.rect(x+2, y+8, 2, 1, goldR[1])
	s.c.rect(x+2, y+10, 2, 1, goldR[1])
}

func ledgers(s iso, gx, gy float64) {
	base := s.p(gx, gy, 0)
	cols := []color.NRGBA{redR[1], blueR[2], greenR[2], goldR[1]}
	for i, col := range cols {
		s.c.rect(int(base.x)-4, int(base.y)-3-i*2, 8, 2, col)
		s.c.rect(int(base.x)-4, int(base.y)-3-i*2, 8, 1, cream)
	}
}
