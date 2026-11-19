package art

import "image/color"

func init() {
	agedKind("gate", paintGateAged)
	agedKind("treasury", paintTreasuryAged)
}

func paintGateAged(s iso) {
	s.apron(2, 0.05)
	switch s.lk.age {
	case ageDark:
		gtPalisade(s)
	case ageFeudal:
		gtFeudal(s)
	case ageCastle:
		gtCastle(s)
	default:
		gtArch(s)
	}
}

func gtPalisade(s iso) {
	lm := logMat(s.lk.seed + 5)
	gtLogTower(s, 0.36, lm)
	pm := plankMat(woodR, s.lk.seed+2)
	s.block(0.82, 0.58, 1.18, 1.42, 20, 23, pm, pm, matWall)
	gtStakes(s, 1.18, 0.6, 1.4, 23)
	s.pen(matWall, func() { s.onRight(1.18, 0.6, 1.4, 0, 20, gtGateDoor(s.lk.chance(8, 0.5))) })
	gtLogTower(s, 1.64, lm)
	gtStakes(s, 1.22, 1.9, 1.98, 0)
	s.aloft(40, func() { s.flag(1.0, 1.64, 40, 7, mixRamp(s.lk.id, strawR, 0.3)) })
	s.props([][2]float64{{1.65, 0.45}, {0.3, 1.85}}, 51)
}

func gtLogTower(s iso, cy float64, lm mat) {
	s.turn(lathe{cx: 1.0, cy: cy, h: 28, r: 0.22, shape: shapeSquare}, lm, matWall)
	s.window(sideR(1.22), cy-0.07, 18)
	s.roofPyramid(roofBox{0.78, cy - 0.22, 1.22, cy + 0.22, 28, 12, 0.12}, s.roofMat())
}

func gtStakes(s iso, x1, v0, v1, z float64) {
	for v := v0; v < v1; v += 0.08 {
		a := s.p(x1, v, z)
		h := 3 + s.lk.pick(int(v*100), 3)
		s.pen(matTrim, func() {
			s.c.rect(int(a.x), int(a.y)-h, 1, h, woodR[1])
			s.c.dot(int(a.x), int(a.y)-h-1, woodR[3])
		})
	}
}

func gtGateDoor(ajar bool) shader {
	return func(x, y int) color.NRGBA {
		if ajar && x%13 < 2 {
			return rgb(40, 30, 30)
		}
		if y%7 == 0 {
			return greyR[0]
		}
		if x%2 == 0 {
			return woodR[0]
		}
		return woodR[1]
	}
}

func gtWing(s iso, y0, y1, top float64, m mat) {
	s.block(0.4, y0, 1.6, y1, 0, top, m, flatMat(stoneR[3]), matStone)
	gtCrenels(s, 0.4, y0, 1.6, y1, top)
}

func gtCrenels(s iso, x0, y0, x1, y1, z float64) {
	m := s.kit().wall
	if s.lk.age == ageFeudal {
		m = s.kit().base
	}
	i := 0
	for x := x0; x < x1-0.01; x += 0.3 {
		if !s.lk.chance(90+i, 0.15) {
			s.block(x, y1-0.14, x+0.15, y1, z, z+4, m, flatMat(stoneR[3]), matStone)
		}
		i++
	}
	for y := y0; y < y1-0.01; y += 0.3 {
		if !s.lk.chance(90+i, 0.15) {
			s.block(x1-0.14, y, x1, y+0.15, z, z+4, m, flatMat(stoneR[3]), matStone)
		}
		i++
	}
}

func gtPassage(s iso, x1, v0, v1, z float64) {
	s.paint(sideR(x1), matTrim, v0, v1, 0, z, inkSft)
	s.pen(matTrim, func() {
		for v := v0 + 0.06; v < v1; v += 0.12 {
			a, b := s.p(x1, v, z), s.p(x1, v, z*0.45)
			s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), greyR[2])
		}
		for zz := z * 0.5; zz < z; zz += 5 {
			a, b := s.p(x1, v0, zz), s.p(x1, v1, zz)
			s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), greyR[1])
		}
	})
}

func gtFeudal(s iso) {
	rm := s.kit().base
	gtSquareTower(s, 0.12, 0.64, rm)
	s.block(0.55, 0.64, 1.45, 1.36, 0, 22, rm, rm, matStone)
	gtPassage(s, 1.45, 0.72, 1.28, 20)
	s.block(0.55, 0.64, 1.45, 1.36, 22, 31, s.kit().wall, s.kit().wall, matWall)
	s.timberFrame(0.55, 0.64, 1.45, 1.36, 22, 31)
	s.window(sideR(1.45), 0.92, 24)
	s.roofGableX(roofBox{0.55, 0.64, 1.45, 1.36, 31, 11, 0.06}, s.roofMat(), s.kit().wall)
	gtSquareTower(s, 1.36, 1.88, rm)
	s.window(sideL(1.88), 0.9, 22)
	s.banner(sideL(1.88), 1.2, 30, 0.16, 11)
}

func gtSquareTower(s iso, y0, y1 float64, m mat) {
	s.block(0.45, y0, 1.55, y1, 0, 34, m, m, matStone)
	s.roofPyramid(roofBox{0.45, y0, 1.55, y1, 34, 16, 0.08}, s.roofMat())
	s.aloft(50, func() { s.flag(1.0, (y0+y1)/2, 50, 7, s.lk.id) })
}

func gtCastle(s iso) {
	s.tower(towerSpec{cx: 1.0, cy: 0.42, r: 0.33, h: 44, roofH: 16})
	k := s.kit()
	s.block(0.6, 0.62, 1.4, 1.38, 0, 34, k.wall, flatMat(stoneR[3]), matStone)
	gtCrenels(s, 0.6, 0.62, 1.4, 1.38, 34)
	gtPassage(s, 1.4, 0.76, 1.24, 22)
	s.banner(sideR(1.4), 0.66, 32, 0.12, 12)
	t := towerSpec{cx: 1.0, cy: 1.58, r: 0.33, h: 44, roofH: 16}
	top := s.tower(t)
	s.towerOpenings(t, s.windowPlan(t, 12, top))
	s.banner(sideR(1.4), 1.22, 32, 0.12, 12)
	s.props([][2]float64{{1.7, 0.6}}, 53)
}

func gtArch(s iso) {
	s.block(0.48, 0.22, 1.52, 1.78, 0, 4, s.kit().base, flatMat(marbleR[3]), matStone)
	m := marbleMat(s.lk.seed)
	s.block(0.58, 0.3, 1.42, 1.7, 4, 34, m, m, matStone)
	f := sideR(1.42)
	s.paint(f, matTrim, 0.64, 1.36, 4, 28, marbleR[3])
	gtThrough(s, 1.42, 0.7, 1.3, 4, 25)
	for _, v := range [][2]float64{{0.38, 0.56}, {1.44, 1.62}} {
		s.paint(f, matTrim, v[0], v[1], 4, 15, inkSft)
		s.paint(f, matTrim, v[0]+0.03, v[1]-0.03, 15, 16, inkSft)
	}
	s.paint(f, matTrim, 0.3, 1.7, 28, 30, goldR[2])
	s.paint(sideL(1.7), matTrim, 0.58, 1.42, 28, 30, goldR[2])
	s.paint(f, matTrim, 0.96, 1.04, 25, 28, goldR[3])
	s.paint(sideL(1.7), matTrim, 0.86, 1.14, 8, 20, marbleR[0])
	for _, v := range []float64{0.34, 0.62, 1.38, 1.66} {
		mkColumn(s, 1.49, v, 4, 28)
	}
	s.block(0.64, 0.4, 1.36, 1.6, 34, 40, m, flatMat(marbleR[3]), matStone)
	s.paint(sideR(1.36), matTrim, 0.75, 1.25, 35, 38, goldR[1])
	gtQuadriga(s, 1.0, 1.0, 40)
}

func gtQuadriga(s iso, gx, gy, z float64) {
	p := s.p(gx, gy, z)
	s.pen(matTrim, func() {
		s.c.ellipse(p.x, p.y-2, 6, 2.5, goldR[1])
		s.c.ellipse(p.x-1, p.y-3, 4, 1.5, goldR[2])
		s.c.rect(int(p.x)-1, int(p.y)-9, 2, 6, goldR[2])
		s.c.dot(int(p.x), int(p.y)-10, goldR[3])
		s.c.dot(int(p.x)-4, int(p.y)-5, goldR[3])
		s.c.dot(int(p.x)+4, int(p.y)-5, goldR[2])
	})
}

func paintTreasuryAged(s iso) {
	s.apron(1, 0.03)
	switch s.lk.age {
	case ageDark:
		tsShed(s)
	case ageFeudal:
		tsVault(s)
	case ageCastle:
		tsRound(s)
	default:
		tsTemple(s)
	}
	s.goldPile(0.33, 0.88, s.lk.age >= ageCastle)
	s.goldPile(0.8, 0.86, false)
	if s.lk.variant == 2 {
		s.goldPile(0.1, 0.72, false)
	}
}

func tsSign(s iso, f side, u, z float64) {
	s.paint(f, matTrim, u, u+0.12, z, z+4, goldR[1])
	s.paint(f, matTrim, u+0.03, u+0.09, z+1, z+3, goldR[3])
}

func tsShed(s iso) {
	s.leanTo(0.14, 0.12, 0.8, 0.6, 10, 15, true)
	s.doorway(sideL(0.6), 0.32, 0.22, 9)
	pm := plankMat(woodR, s.lk.seed+4)
	s.block(0.6, 0.62, 0.9, 0.8, 0, 6, pm, flatMat(woodR[3]), matWall)
	s.paint(sideL(0.8), matTrim, 0.6, 0.9, 2, 3, greyR[1])
	s.paint(sideL(0.8), matTrim, 0.73, 0.77, 2, 5, goldR[2])
	s.paint(sideR(0.9), matTrim, 0.62, 0.8, 2, 3, greyR[0])
	s.sacks(0.12, 0.82)
}

func tsVault(s iso) {
	rm := s.kit().base
	s.block(0.12, 0.1, 0.88, 0.8, 0, 16, rm, rm, matStone)
	s.roofPyramid(roofBox{0.12, 0.1, 0.88, 0.8, 16, 9, 0.06}, tileMat(goldR, s.lk.seed))
	s.paint(sideL(0.8), matTrim, 0.3, 0.6, 0, 12, woodR[0])
	if s.lk.variant > 0 {
		tsSign(s, sideR(0.88), 0.4, 9)
	}
	s.pen(matTrim, func() {
		s.onLeft(0.8, 0.33, 0.57, 0, 11, func(x, y int) color.NRGBA {
			if y%4 == 0 {
				return greyR[0]
			}
			return woodR[1]
		})
	})
}

func tsRound(s iso) {
	t := towerSpec{cx: 0.5, cy: 0.44, r: 0.34}
	k := s.kit()
	s.turn(lathe{cx: 0.5, cy: 0.44, h: 16, r: 0.34}, k.wall, matStone)
	s.turn(lathe{cx: 0.5, cy: 0.44, z: 14, h: 3, r: 0.38}, k.wall, matStone)
	s.towerOpenings(t, s.doorPlan(t, 14, 80, 10))
	if s.lk.variant != 1 {
		tsSign(s, sideR(0.84), 0.38, 9)
	}
	s.roofCone(0.5, 0.44, 0.42, 17, 12, shapeRound, slateMat(mixRamp(slateR, s.lk.id, 0.6), s.lk.seed))
	s.finial(0.5, 0.44, 29, 4)
}

func tsTemple(s iso) {
	s.block(0.08, 0.06, 0.92, 0.84, 0, 3, s.kit().base, flatMat(marbleR[3]), matStone)
	for _, c := range [][2]float64{{0.16, 0.14}, {0.84, 0.14}} {
		mkColumn(s, c[0], c[1], 3, 16)
	}
	m := marbleMat(s.lk.seed)
	s.block(0.28, 0.24, 0.72, 0.66, 3, 15, m, m, matStone)
	s.paint(sideL(0.66), matTrim, 0.42, 0.58, 3, 12, goldR[1])
	for _, c := range [][2]float64{{0.16, 0.76}, {0.84, 0.76}} {
		mkColumn(s, c[0], c[1], 3, 16)
	}
	s.block(0.1, 0.08, 0.9, 0.82, 16, 18, m, flatMat(marbleR[3]), matStone)
	s.paint(sideL(0.82), matTrim, 0.1, 0.9, 16, 17, goldR[2])
	s.paint(sideR(0.9), matTrim, 0.08, 0.82, 16, 17, goldR[1])
	s.roofDome(0.5, 0.45, 0.28, 18, 11, ribbedMat(goldR, s.lk.seed))
	s.finial(0.5, 0.45, 29, 5)
}

func gtThrough(s iso, x1, v0, v1, z0, z1 float64) {
	hole := map[int]bool{}
	carve := func(x, y int) color.NRGBA {
		s.c.erase(x, y)
		hole[y*s.c.w+x] = true
		return color.NRGBA{}
	}
	s.on(sideR(x1), v0, v1, z0, z1, carve)
	s.on(sideR(x1), v0+0.08, v1-0.08, z1, z1+2, carve)
	inner := s.leftMat(v0, marbleMat(s.lk.seed+9))
	s.pen(matTrim, func() {
		s.onLeft(v0, 0.5, x1, z0, z1+2, func(x, y int) color.NRGBA {
			if !hole[y*s.c.w+x] {
				return color.NRGBA{}
			}
			_, z := s.leftUV(x, y, v0)
			return darken(inner(x, y), 0.4+0.3*(z-z0)/(z1-z0))
		})
	})
}
