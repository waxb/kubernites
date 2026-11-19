package art

import (
	"image/color"
	"math"
)

func init() {
	agedKind("shrine", paintShrine)
	agedKind("signpost", paintSignpost)
	agedKind("foreign", paintForeign)
	agedKind("harbor-gate", paintHarborGate)
	agedKind("sea-chancery", paintSeaChancery)
}

func paintShrine(s iso) {
	s.apron(1, 0.12)
	stone := rubbleMat(stoneR, s.lk.seed+3)
	if s.lk.age == ageDark {
		stone = logMat(s.lk.seed + 3)
	}
	s.block(0.2, 0.2, 0.8, 0.8, 0, 4, stone, flatMat(stoneR[3]), matStone)
	s.block(0.3, 0.3, 0.7, 0.7, 4, 15, s.kit().wall, flatMat(stoneR[2]), matWall)
	s.paint(sideL(0.7), matTrim, 0.38, 0.62, 5, 12, inkSft)
	s.pen(matTrim, func() {
		c := s.p(0.5, 0.7, 7)
		s.c.disc(c.x, c.y, 2.2, withAlpha(fireR[2], 170))
		s.c.rect(int(c.x), int(c.y)-1, 1, 3, cream)
		s.c.dot(int(c.x), int(c.y)-3, fireR[3])
		s.c.dot(int(c.x), int(c.y)-2, fireR[1])
	})
	switch s.lk.variant {
	case 1:
		s.roofDome(0.5, 0.5, 0.26, 15, 7, s.roofMat())
	case 2:
		s.roofBell(0.5, 0.5, 0.27, 15, 9, shapeSquare, s.roofMat())
	default:
		s.roofCone(0.5, 0.5, 0.3, 15, 9, shapeSquare, s.roofMat())
	}
	if s.lk.age == ageEmpire {
		s.finial(0.5, 0.5, 23, 4)
	}
	s.flag(0.82, 0.78, 0, 21, s.lk.id)
}

func paintSignpost(s iso) {
	s.apron(1, 0.16)
	s.block(0.36, 0.36, 0.64, 0.64, 0, 4, rubbleMat(stoneR, s.lk.seed+5), flatMat(stoneR[3]), matStone)
	s.block(0.46, 0.46, 0.54, 0.54, 4, 31, flatMat(woodR[1]), flatMat(woodR[2]), matWall)
	cloth := s.lk.id
	boards := []struct {
		z     float64
		right bool
		w     int
	}{{26, true, 9}, {21, false, 8}, {16, true, 7}}
	if s.lk.variant == 1 {
		boards = boards[:2]
	}
	s.pen(matTrim, func() {
		for i, b := range boards {
			c := s.p(0.5, 0.5, b.z)
			x, y := int(c.x), int(c.y)
			col, edge := cloth[2], cloth[0]
			if i%2 == 1 {
				col = cream
			}
			if b.right {
				s.c.rect(x, y, b.w, 3, col)
				s.c.rect(x, y+3, b.w, 1, edge)
				s.c.dot(x+b.w, y+1, col)
				s.c.dot(x+b.w, y+2, edge)
				continue
			}
			s.c.rect(x-b.w+1, y, b.w, 3, col)
			s.c.rect(x-b.w+1, y+3, b.w, 1, edge)
			s.c.dot(x-b.w, y+1, col)
			s.c.dot(x-b.w, y+2, edge)
		}
		top := s.p(0.5, 0.5, 31)
		s.c.rect(int(top.x)-1, int(top.y)-3, 3, 3, woodR[0])
		s.c.dot(int(top.x), int(top.y)-2, fireR[3])
		ring := s.p(0.5, 0.5, 9)
		s.c.disc(ring.x+1.5, ring.y, 1.6, goldR[1])
		s.c.dot(int(ring.x)+1, int(ring.y), woodR[1])
		rope := s.p(0.5, 0.5, 6)
		s.c.line(int(rope.x)-1, int(rope.y)-1, int(rope.x)+1, int(rope.y)+1, woodR[3])
	})
}

var foreignCloth = []ramp{purpleR, tealR, orangeR}

func paintForeign(s iso) {
	s.apron(2, 0.05)
	x0, y0, x1, y1 := 0.35, 0.45, 1.65, 1.6
	top := 18 * s.kit().scale
	s.walls(x0, y0, x1, y1, 0, top)
	s.crenels(x0, y0, x1, y1, top, stoneR)
	cx, cy := (x0+x1)/2, (y0+y1)/2
	cloth := foreignCloth[s.lk.variant%len(foreignCloth)]
	dome := flatMat(cloth[2])
	switch s.lk.variant {
	case 1:
		s.roofCone(cx, cy, 0.62, top, 7, shapeSquare, flatMat(cloth[1]))
		s.roofCone(cx, cy, 0.42, top+6, 7, shapeSquare, flatMat(cloth[2]))
		s.roofCone(cx, cy, 0.26, top+12, 9, shapeSquare, flatMat(cloth[3]))
		s.finial(cx, cy, top+21, 5)
	case 2:
		s.turn(lathe{cx: cx, cy: cy, z: top, h: 6, r: 0.34}, s.kit().wall, matStone)
		s.roofBell(cx, cy, 0.4, top+6, 14, shapeRound, dome)
		s.finial(cx, cy, top+20, 5)
	default:
		s.turn(lathe{cx: cx, cy: cy, z: top, h: 6, r: 0.34}, s.kit().wall, matStone)
		s.roofOnion(cx, cy, 0.42, top+6, 18, dome)
		s.finial(cx, cy, top+24, 5)
	}
	s.doorway(sideL(y1), 0.8, 0.42, 11)
	s.window(sideR(x1), 0.9, math.Round(top*0.55))
	s.bannerL(y1, 0.42, top-2, 0.16, 9, cloth, goldR[3])
	s.bannerL(y1, 1.38, top-2, 0.16, 9, cloth, goldR[3])
	s.props([][2]float64{{0.35, 1.8}, {1.8, 1.8}}, 41)
}

func paintHarborGate(s iso) {
	s.apron(2, 0.04)
	x0, y0, x1, y1 := 0.3, 0.6, 1.7, 1.4
	top := 24 * s.kit().scale
	s.layer(func(l iso) {
		l.tower(towerSpec{cx: 0.42, cy: 0.95, r: 0.3, h: 36, roofH: 12 + 4*float64(l.lk.variant)})
	})
	s.walls(x0, y0, x1, y1, 0, top)
	s.crenels(x0, y0, x1, y1, top, stoneR)
	s.layer(func(l iso) {
		l.tower(towerSpec{cx: 1.58, cy: 0.95, r: 0.3, h: 36, roofH: 12 + 4*float64(l.lk.variant)})
	})
	arch := math.Round(min(top*0.62, 15))
	s.paint(sideL(y1), matTrim, 0.72, 1.28, 0, arch+2, stoneR[3])
	s.pen(matTrim, func() {
		if s.lk.age < ageCastle {
			s.on(sideL(y1), 0.76, 1.24, 0, arch, s.plankDoor(woodR))
			return
		}
		s.on(sideL(y1), 0.76, 1.24, 0, arch, func(x, y int) color.NRGBA {
			if x%3 == 0 || y%3 == 0 {
				return greyR[1]
			}
			return inkSft
		})
	})
	if s.lk.age >= ageCastle {
		s.turret(1.0, 1.0, 0.2, top, 8)
	}
	cloth := []ramp{blueR, tealR, redR}[s.lk.variant%3]
	s.bannerL(y1, 0.4, top-1, 0.16, 10, cloth, cream)
	s.bannerL(y1, 1.44, top-1, 0.16, 10, cloth, cream)
	s.pen(matTrim, func() {
		c := s.p(1.0, y1, arch+6)
		x, y := int(c.x), int(c.y)
		s.c.rect(x, y-3, 1, 6, goldR[2])
		s.c.rect(x-2, y-2, 5, 1, goldR[2])
		s.c.line(x-3, y+1, x, y+3, goldR[1])
		s.c.line(x+3, y+1, x, y+3, goldR[1])
	})
	s.props([][2]float64{{0.55, 1.75}, {1.75, 1.7}}, 47)
}

var seaRock = ramp{rgb(64, 60, 62), rgb(92, 86, 84), rgb(124, 116, 108), rgb(160, 150, 136)}

func paintSeaChancery(s iso) {
	rock := rubbleMat(seaRock, s.lk.seed+11)
	for i, r := range [][5]float64{{0.1, 0.35, 1.05, 1.5, 5}, {0.9, 0.1, 2.9, 1.15, 4}, {0.15, 1.3, 1.5, 2.9, 6}, {1.35, 1.0, 2.9, 2.9, 7}, {0.6, 0.6, 2.4, 2.4, 8}} {
		j := s.lk.jit(80+i, 0.07)
		s.block(r[0]+j, r[1]-j, r[2]+j, r[3], 0, r[4]+s.lk.jit(90+i, 1.5), rock, flatMat(seaRock[2]), matStone)
	}
	s.pen(matTrim, func() {
		for i := 0; i < 34; i++ {
			a := 2 * math.Pi * float64(i) / 34
			reach := 1.42 + s.lk.jit(100+i, 0.06)
			p := s.p(1.5+reach*math.Cos(a), 1.5+reach*math.Sin(a), 0)
			s.c.dot(int(p.x), int(p.y), withAlpha(cream, 230))
			if i%2 == 0 {
				s.c.dot(int(p.x)+1, int(p.y), withAlpha(waterR[3], 210))
			}
		}
		for i := 0; i < 6; i++ {
			a := 2*math.Pi*float64(i)/6 + 0.4
			p := s.p(1.5+1.3*math.Cos(a), 1.5+1.3*math.Sin(a), 2)
			s.c.disc(p.x, p.y, 2.2, seaRock[1])
			s.c.dot(int(p.x)-1, int(p.y)-1, seaRock[3])
		}
	})
	base := 8.0
	k := s.kit()
	x0, y0, x1, y1 := 0.55, 0.55, 2.45, 2.45
	s.layer(func(l iso) {
		t := towerSpec{cx: 1.0, cy: 1.0, r: 0.42, z0: base, h: 44, roofH: 16}
		crest := l.tower(t)
		l.towerOpenings(t, l.windowPlan(t, base+22, crest))
	})
	s.walls(x0, y0, x1, y1, base, base+11)
	s.crenels(x0, y0, x1, y1, base+11, stoneR)
	hx0, hy0, hx1, hy1 := 1.35, 1.3, 2.3, 2.2
	htop := base + 11 + 11*k.scale
	s.walls(hx0, hy0, hx1, hy1, base+11, htop)
	s.roofOver(roofBox{hx0, hy0, hx1, hy1, htop, 12 * k.rise, k.eave}, false, false)
	s.window(sideL(hy1), 1.6, htop-6)
	s.window(sideR(hx1), 1.75, htop-6)
	for _, c := range [][2]float64{{0.62, 2.38}, {2.38, 0.62}, {2.38, 2.38}} {
		s.layer(func(l iso) {
			l.tower(towerSpec{cx: c[0], cy: c[1], r: 0.24, z0: base, h: 20, roofH: 9, crown: l.lk.variant == 2})
		})
	}
	s.bannerL(y1, 1.1, base+10, 0.16, 8, redR, sealR[2])
	s.bannerL(y1, 1.7, base+10, 0.16, 8, redR, sealR[2])
	s.flag(1.0, 1.0, base+44*k.scale+16, 10, redR)
}
