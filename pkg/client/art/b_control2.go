package art

import (
	"image/color"
	"math"
)

func init() {
	agedKind("quartermaster", paintQuartermaster)
}

const qmCx, qmCy = 0.88, 0.88

var (
	qmHideR = ramp{rgb(92, 66, 46), rgb(136, 100, 68), rgb(172, 134, 92), rgb(204, 170, 122)}
	qmCanvR = ramp{plasterR[1], plasterR[2], plasterR[3], cream}
	qmSpot  = [][2]float64{{0.25, 1.72}, {1.7, 0.62}}
)

func qmHideMat(seed int) mat {
	return func(p surf) color.NRGBA {
		if fmod(p.u+float64(int(p.v/6)%2)*3, 10) < 0.9 {
			return qmHideR[0]
		}
		f := level(p.light)
		if vnoise(p.u/6, p.v/5, seed) > 0.62 {
			f -= 0.6
		}
		return rampAt(qmHideR, f)
	}
}

func paintQuartermaster(s iso) {
	s.apron(2, 0.08)
	switch s.lk.age {
	case ageDark:
		qmHide(s)
	case ageFeudal:
		qmTent(s)
	default:
		qmPavilion(s)
	}
	if s.lk.age == ageDark {
		qmStumpMap(s, 1.5, 1.62)
	} else {
		mapTable(s, 1.36, 1.5)
	}
	s.props(qmSpot, 41)
}

func qmFlap(s iso, r, h, deg, frac float64) {
	a := deg * math.Pi / 180
	c, sn := math.Cos(a), math.Sin(a)
	base := s.p(qmCx+r*c, qmCy+r*sn, 0)
	tip := s.p(qmCx+r*(1-frac)*c, qmCy+r*(1-frac)*sn, h*frac)
	s.pen(matTrim, func() {
		s.c.shadePoly([]pt{{base.x - 4, base.y + 1}, {base.x + 4, base.y + 1}, {tip.x, tip.y}}, flat(inkSft))
		s.c.line(int(base.x)+4, int(base.y), int(tip.x), int(tip.y), qmCanvR[3])
	})
}

func qmHide(s iso) {
	r, h := 0.6, 26.0
	s.turn(lathe{cx: qmCx, cy: qmCy, h: h, r: r, profile: coneProfile}, qmHideMat(s.lk.seed), matRoof)
	top := s.p(qmCx, qmCy, h)
	s.pen(matTrim, func() {
		for i, dx := range []float64{-4, -1, 3} {
			s.c.line(int(top.x), int(top.y)+2, int(top.x+dx), int(top.y)-5-i%2, woodR[1+i%2])
		}
	})
	qmFlap(s, r, h, 100, 0.55)
	fire := s.p(1.65, 1.15, 0)
	s.c.ellipse(fire.x, fire.y-1, 3.5, 1.6, rockR[1])
	s.c.shadePoly([]pt{{fire.x - 2, fire.y - 1}, {fire.x, fire.y - 7}, {fire.x + 2, fire.y - 1}}, flat(fireR[1]))
	s.c.dot(int(fire.x), int(fire.y)-3, fireR[3])
}

func qmTent(s iso) {
	r, h := 0.64, 31.0
	stripe := stripeMat(s.lk.id, qmCanvR, 5)
	s.turn(lathe{cx: qmCx, cy: qmCy, h: h, r: r, shape: shapeSquare, profile: coneProfile}, stripe, matRoof)
	qmFlap(s, r, h, 90, 0.5)
	s.flag(qmCx, qmCy, h, 7, goldR)
}

func qmPavilion(s iso) {
	shape, wall, roof, cap := shapeRound, 9.0, 20.0, bellProfile
	if s.lk.age == ageEmpire {
		shape, wall, roof, cap = shapeOct, 11, 21, flareProfile
	}
	r := 0.58
	trim := qmCanvR
	if s.lk.age == ageEmpire {
		trim = goldR
	}
	s.turn(lathe{cx: qmCx, cy: qmCy, h: wall, r: r, shape: shape}, stripeMat(qmCanvR, s.lk.id, 6), matWall)
	qmFlap(s, r, wall*2, 95, 0.45)
	s.turn(lathe{cx: qmCx, cy: qmCy, z: wall, h: roof, r: r + 0.08, shape: shape, profile: cap}, stripeMat(s.lk.id, trim, 6), matRoof)
	qmValance(s, r+0.08, wall, trim)
	if s.lk.age == ageEmpire {
		s.finial(qmCx, qmCy, wall+roof, 7)
		return
	}
	s.flag(qmCx, qmCy, wall+roof, 7, goldR)
}

func qmValance(s iso, r, z float64, trim ramp) {
	s.pen(matTrim, func() {
		for deg := -40.0; deg <= 130; deg += 17 {
			a := deg * math.Pi / 180
			p := s.p(qmCx+r*math.Cos(a), qmCy+r*math.Sin(a), z)
			s.c.dot(int(p.x), int(p.y)+1, trim[2])
			s.c.dot(int(p.x), int(p.y)+2, trim[1])
		}
	})
}

func qmStumpMap(s iso, gx, gy float64) {
	p := s.p(gx, gy, 0)
	s.c.ellipse(p.x, p.y-5, 5, 2.5, woodR[3])
	s.c.rect(int(p.x)-5, int(p.y)-5, 10, 5, barkR[1])
	s.c.rect(int(p.x)-5, int(p.y)-5, 2, 5, barkR[2])
	s.c.ellipse(p.x, p.y-5, 3.5, 1.6, woodR[2])
	s.c.rect(int(p.x)-3, int(p.y)-7, 6, 2, cream)
	s.c.dot(int(p.x)-1, int(p.y)-7, redR[2])
	s.c.dot(int(p.x)+2, int(p.y)-6, blueR[2])
}
