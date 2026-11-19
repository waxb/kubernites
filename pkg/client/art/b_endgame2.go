package art

import "image/color"

func init() {
	agedKind("warhorn", paintWarhorn)
	agedDarkKind("warhorn", paintWarhornDark)
}

func paintWarhorn(s iso) {
	z := whTower(s)
	whHorn(s, z, brassR, true)
}

func paintWarhornDark(s iso) {
	z := whTower(s)
	whHorn(s, z, greyR, false)
}

func whTower(s iso) float64 {
	s.apron(2, 0.1)
	switch s.lk.age {
	case ageDark:
		return whStilts(s)
	case ageFeudal:
		return whFeudal(s)
	case ageCastle:
		return whCastle(s)
	}
	return whEmpire(s)
}

func whLog(s iso, a, b pt, k int) {
	s.pen(matTrim, func() {
		s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[1])
		s.c.line(int(a.x)+1, int(a.y), int(b.x)+1, int(b.y), woodR[2+k%2])
	})
}

func whStilts(s iso) float64 {
	legs := [][2]float64{{0.62, 0.62}, {1.38, 0.62}, {0.62, 1.38}, {1.38, 1.38}}
	for i, l := range legs {
		splay := (l[0] - 1) * 0.15
		whLog(s, s.p(l[0]+splay, l[1]+(l[1]-1)*0.15, 0), s.p(l[0], l[1], 31), i)
	}
	whLog(s, s.p(0.62, 1.38, 4), s.p(1.38, 1.38, 26), 0)
	whLog(s, s.p(1.38, 0.62, 4), s.p(1.38, 1.38, 26), 1)
	pm := plankMat(woodR, s.lk.seed+2)
	s.block(0.5, 0.5, 1.5, 1.5, 30, 33, pm, pm, matWall)
	wm := wattleMat(s.lk.seed + 3)
	s.block(0.66, 0.66, 1.34, 1.34, 33, 44, wm, wm, matWall)
	s.cornerPosts(0.66, 0.66, 1.34, 1.34, 33, 44)
	s.roofPyramid(roofBox{0.66, 0.66, 1.34, 1.34, 44, 14, 0.16}, s.roofMat())
	whLadder(s, 1.2, 1.5)
	s.aloft(57, func() { s.flag(1.0, 1.0, 57, 6, mixRamp(s.lk.id, strawR, 0.3)) })
	s.props([][2]float64{{0.3, 1.75}, {1.7, 0.75}}, 41)
	return 38
}

func whLadder(s iso, gx, gy float64) {
	a0, a1 := s.p(gx-0.08, gy+0.08, 0), s.p(gx-0.08, gy-0.1, 30)
	b0, b1 := s.p(gx+0.08, gy+0.08, 0), s.p(gx+0.08, gy-0.1, 30)
	s.pen(matTrim, func() {
		s.c.line(int(a0.x), int(a0.y), int(a1.x), int(a1.y), woodR[2])
		s.c.line(int(b0.x), int(b0.y), int(b1.x), int(b1.y), woodR[1])
		for t := 0.1; t < 1; t += 0.14 {
			ax, ay := a0.x+(a1.x-a0.x)*t, a0.y+(a1.y-a0.y)*t
			bx, by := b0.x+(b1.x-b0.x)*t, b0.y+(b1.y-b0.y)*t
			s.c.line(int(ax), int(ay), int(bx), int(by), woodR[3])
		}
	})
}

func whFeudal(s iso) float64 {
	t := towerSpec{cx: 1, cy: 1, r: 0.3}
	s.turn(lathe{cx: 1, cy: 1, h: 46, r: 0.3, shape: shapeSquare}, s.kit().base, matStone)
	s.towerOpenings(t, s.doorPlan(t, 42, 100, 11))
	pm := plankMat(woodR, s.lk.seed+2)
	s.block(0.5, 0.5, 1.5, 1.5, 46, 49, pm, pm, matWall)
	whLoggia(s, 0.36, 49, 58)
	s.roofPyramid(roofBox{0.5, 0.5, 1.5, 1.5, 58, 13, 0.1}, s.roofMat())
	s.aloft(71, func() { s.flag(1.0, 1.0, 71, 7, s.lk.id) })
	s.props([][2]float64{{0.35, 1.7}}, 43)
	return 52
}

func whLoggia(s iso, r, z0, z1 float64) {
	for _, d := range [][2]float64{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
		mkColumn(s, 1+d[0]*r, 1+d[1]*r, z0, z1)
	}
}

func whCastle(s iso) float64 {
	t := towerSpec{cx: 1, cy: 1, r: 0.34}
	k := s.kit()
	s.turn(lathe{cx: 1, cy: 1, h: 48, r: 0.34}, k.wall, matStone)
	s.turn(lathe{cx: 1, cy: 1, z: 45, h: 4, r: 0.42, profile: func(t float64) float64 { return 0.8 + 0.2*t }}, k.wall, matStone)
	s.towerOpenings(t, s.doorPlan(t, 45, 100, 12))
	whLoggia(s, 0.28, 49, 58)
	s.towerRoof(1, 1, 0.48, 58, 18, shapeRound)
	whPennant(s, 0.72, 1.28, 42)
	return 53
}

func whPennant(s iso, gx, gy, z float64) {
	p := s.p(gx, gy, z)
	cloth := s.lk.id
	s.aloft(z-10, func() {
		s.c.rect(int(p.x)-2, int(p.y), 5, 1, woodR[0])
		for i := 0; i < 9; i++ {
			w := 5 - i/3
			s.c.rect(int(p.x)-2+(5-w)/2, int(p.y)+1+i, w, 1, []color.NRGBA{cloth[2], cloth[1]}[i%2])
		}
		s.c.dot(int(p.x), int(p.y)+4, goldR[3])
	})
}

func whEmpire(s iso) float64 {
	t := towerSpec{cx: 1, cy: 1, r: 0.33}
	shape := s.towerShape()
	s.turn(lathe{cx: 1, cy: 1, h: 54, r: 0.33, shape: shape}, s.kit().wall, matStone)
	s.turn(lathe{cx: 1, cy: 1, z: 52, h: 2, r: 0.36, shape: shape}, flatMat(goldR[2]), matTrim)
	s.towerOpenings(t, s.doorPlan(t, 52, 100, 13))
	whLoggia(s, 0.25, 54, 64)
	s.turn(lathe{cx: 1, cy: 1, z: 64, h: 3, r: 0.38, shape: shape}, marbleMat(s.lk.seed), matStone)
	s.roofDome(1, 1, 0.38, 67, 16, s.domeMat())
	s.finial(1, 1, 83, 6)
	return 59
}

func whHorn(s iso, z float64, metal ramp, lit bool) {
	mouth, bell := s.p(1.0, 0.95, z), s.p(2.1, 1.3, z+3)
	s.aloft(z-6, func() {
		s.c.rect(int(mouth.x)-1, int(mouth.y), 3, 5, woodR[0])
		s.c.shadePoly([]pt{{mouth.x - 2, mouth.y - 1}, {bell.x - 1, bell.y - 7}, {bell.x, bell.y + 6}, {mouth.x - 2, mouth.y + 1.5}}, flat(metal[2]))
		s.c.line(int(mouth.x)-2, int(mouth.y)-1, int(bell.x)-2, int(bell.y)-7, metal[3])
		for _, t := range []float64{0.3, 0.55, 0.8} {
			x, y := mouth.x+(bell.x-mouth.x)*t, mouth.y+(bell.y-mouth.y)*t
			h := 3 + t*11
			s.c.rect(int(x), int(y-h/2), 1, int(h), metal[1])
		}
		s.c.ellipse(bell.x+1, bell.y-0.5, 2.6, 7, metal[1])
		s.c.ellipse(bell.x+1.6, bell.y-0.5, 1.4, 5.2, metal[0])
	})
	if lit {
		s.c.dot(int(bell.x)-1, int(bell.y)-6, white)
		s.c.dot(int(bell.x)-3, int(bell.y)-5, metal[3])
	}
}
