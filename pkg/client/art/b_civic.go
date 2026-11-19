package art

import (
	"image/color"
	"math"
)

func init() {
	agedKind("observatory", paintObservatory)
}

func ctlDir(deg float64) (float64, float64) {
	a := deg * math.Pi / 180
	return math.Cos(a), math.Sin(a)
}

const obCx, obCy = 1.0, 1.0

type obSpec struct {
	r, h, domeR, domeH float64
	wall, dome         mat
}

func obLook(s iso) obSpec {
	sd := s.lk.seed
	switch s.lk.age {
	case ageFeudal:
		return obSpec{0.52, 20, 0.5, 13, rubbleMat(stoneR, sd), ribbedMat(slateR, sd)}
	case ageCastle:
		return obSpec{0.5, 28, 0.56, 17, ashlarMat(stoneR, sd), ribbedMat(s.lk.id, sd)}
	}
	return obSpec{0.62, 22, 0.68, 20, marbleMat(sd), ribbedMat(s.lk.id, sd)}
}

func paintObservatory(s iso) {
	s.apron(2, 0.07)
	if s.lk.age == ageDark {
		obPlatform(s)
	} else {
		obDomeTower(s, obLook(s))
	}
	s.props([][2]float64{{0.28, 1.74}, {1.72, 1.55}}, 61)
}

func obDomeTower(s iso, o obSpec) {
	if s.lk.age == ageCastle {
		t := towerSpec{cx: 1.5, cy: 0.5, r: 0.15, h: o.h + 8, roofH: 10}
		s.tower(t)
		s.towerOpenings(t, s.ringPlan(t, o.h, 1))
	}
	s.layer(func(l iso) { l.turn(lathe{cx: obCx, cy: obCy, h: o.h, r: o.r}, o.wall, matStone) })
	drum := towerSpec{cx: obCx, cy: obCy, r: o.r, round: true}
	s.towerOpenings(drum, s.withDoor(s.ringPlan(drum, o.h*0.5, columnCount(o.r)), 100, 12))
	if s.lk.age == ageEmpire {
		s.turn(lathe{cx: obCx, cy: obCy, z: o.h - 2, h: 2, r: o.r + 0.03}, ribbedMat(goldR, s.lk.seed), matTrim)
	}
	s.roofDome(obCx, obCy, o.domeR, o.h, o.domeH, o.dome)
	obSlit(s, o)
	if s.lk.age == ageEmpire {
		s.finial(obCx-0.25, obCy-0.25, o.h+o.domeH-4, 5)
	}
}

func obSlit(s iso, o obSpec) {
	top := s.p(obCx, obCy, o.h+o.domeH)
	bot := s.p(obCx+o.domeR*0.66, obCy+o.domeR*0.66, o.h+2)
	x := int(math.Round(top.x))
	s.pen(matTrim, func() {
		s.c.rect(x-1, int(top.y)+1, 3, int(bot.y-top.y)-1, inkSft)
		s.c.rect(x+2, int(top.y)+2, 1, int(bot.y-top.y)-3, darken(o.dome(surf{light: -0.5}), 0.2))
	})
	obScope(s, float64(x), top.y+(bot.y-top.y)*0.45, s.lk.age >= ageCastle)
}

func obScope(s iso, x, y float64, gilt bool) {
	tube := []color.NRGBA{goldR[1], goldR[2], goldR[3]}
	if !gilt {
		tube = []color.NRGBA{woodR[0], woodR[2], goldR[2]}
	}
	s.pen(matTrim, func() {
		s.c.line(int(x), int(y), int(x)+11, int(y)-9, tube[0])
		s.c.line(int(x), int(y)+1, int(x)+11, int(y)-8, tube[1])
		s.c.disc(x+11.5, y-9, 1.6, tube[2])
	})
}

func obPlatform(s iso) {
	deck := 16.0
	legs := [][2]float64{{0.52, 0.52}, {1.48, 0.52}, {0.52, 1.48}, {1.48, 1.48}}
	for i, l := range legs {
		s.post(l[0], l[1], 0, deck+float64(1-i/2)*12)
	}
	s.pen(matTrim, func() {
		s.edge(s.p(0.52, 1.48, 1), s.p(1.48, 1.48, deck-1), woodR[1])
		s.edge(s.p(1.48, 1.48, 1), s.p(1.48, 0.52, deck-1), woodR[0])
	})
	board := faces{top: flat(woodR[3]), left: s.plankLeft(woodR, 1.56), right: s.plankRight(woodR, 1.56)}
	s.pen(matWall, func() { s.box(0.44, 0.44, 1.56, 1.56, deck, deck+2, board) })
	s.roofShed(0.4, 0.4, 1.6, 0.95, deck+11, deck+15, true, s.roofMat())
	obStaff(s, deck+2)
	s.pen(matTrim, func() {
		for _, u := range []float64{0.5, 0.9, 1.3} {
			s.edge(s.p(u, 1.54, deck+2), s.p(u, 1.54, deck+7), woodR[1])
		}
		s.edge(s.p(0.46, 1.54, deck+6), s.p(1.54, 1.54, deck+6), woodR[2])
		s.edge(s.p(1.54, 1.54, deck+6), s.p(1.54, 1.0, deck+6), woodR[1])
	})
	obLadder(s, deck)
}

func obStaff(s iso, z float64) {
	a, b := s.p(1.1, 1.1, z), s.p(1.1, 1.1, z+18)
	s.pen(matTrim, func() {
		s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[0])
		for deg := 0.0; deg < 360; deg += 20 {
			c, sn := ctlDir(deg)
			s.c.dot(int(math.Round(b.x+c*4)), int(math.Round(b.y+3+sn*4)), goldR[2+int(deg/20)%2])
		}
		s.c.line(int(b.x)-4, int(b.y)+3, int(b.x)+4, int(b.y)+3, goldR[1])
		s.c.line(int(b.x)-3, int(b.y)-1, int(b.x)+3, int(b.y)+7, goldR[3])
	})
}

func obLadder(s iso, deck float64) {
	a0, a1 := s.p(0.62, 1.72, 0), s.p(0.62, 1.52, deck+2)
	b0, b1 := s.p(0.78, 1.72, 0), s.p(0.78, 1.52, deck+2)
	s.pen(matTrim, func() {
		s.c.line(int(a0.x), int(a0.y), int(a1.x), int(a1.y), woodR[1])
		s.c.line(int(b0.x), int(b0.y), int(b1.x), int(b1.y), woodR[1])
		for t := 0.15; t < 1; t += 0.2 {
			l, r := pt{a0.x + (a1.x-a0.x)*t, a0.y + (a1.y-a0.y)*t}, pt{b0.x + (b1.x-b0.x)*t, b0.y + (b1.y-b0.y)*t}
			s.c.line(int(l.x), int(l.y), int(r.x), int(r.y), woodR[2])
		}
	})
}
