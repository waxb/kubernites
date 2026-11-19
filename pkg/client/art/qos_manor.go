package art

import "image/color"

const (
	manorX0, manorX1 = 0.07, 0.7
	manorY0, manorY1 = 0.07, 0.6
	manorYM          = 0.335
	manorStone       = 11.0
	manorRoof        = 20.0
	manorRise        = 11.0
	manorJetty       = 0.03
	gardenEdge       = 0.95
	gateU0, gateU1   = 0.18, 0.32
)

var (
	bloomColours   = []color.NRGBA{rgb(236, 120, 160), goldR[3], white, rgb(200, 150, 230)}
	hutManorFloors = [AgeTiers]float64{10, 11, 12, 11}
	hutManorEaves  = [AgeTiers]float64{0.1, 0.07, 0.06, 0.06}
)

func paintLanded(s iso, h hut, lit bool) {
	roof, age := tierRamp(h.tier), s.lk.age
	garden(s.hutGround())
	s.box(manorX0-0.03, manorY0-0.03, manorX1+0.03, manorY1+0.03, 0, 1.5, faces{top: flat(stoneR[2]), left: flat(stoneR[1]), right: flat(stoneR[0])})
	hutManorChimney(s)
	low := hutManorFloors[age]
	s.block(manorX0, manorY0, manorX1, manorY1, 1.5, low, hutManorLower(s), hutManorLower(s), matStone)
	j := manorJetty * float64(boolInt(age == ageFeudal))
	s.walls(manorX0-j, manorY0-j, manorX1+j, manorY1+j, low, manorRoof)
	b := roofBox{manorX0 - j, manorY0 - j, manorX1 + j, manorY1 + j, manorRoof, manorRise, hutManorEaves[age]}
	s.roofGableX(b, hutRoofMat(age, roof, s.lk.seed+17), s.kit().wall)
	hutManorCrown(s, b, roof)
	manorFront(s, roof, lit, low)
	gardenGreens(s)
	fence(s)
}

func hutManorLower(s iso) mat {
	switch s.lk.age {
	case ageDark:
		return logMat(s.lk.seed + 1)
	case ageFeudal:
		return rubbleMat(stoneR, s.lk.seed+1)
	case ageCastle:
		return ashlarMat(stoneR, s.lk.seed+1)
	}
	return ashlarMat(marbleR, s.lk.seed+1)
}

func hutManorChimney(s iso) {
	body := hutManorLower(s)
	if s.lk.age == ageDark {
		body = wattleMat(s.lk.seed + 3)
	}
	s.block(0.46, 0.14, 0.58, 0.26, 16, 34, body, flatMat(inkSft), matStone)
	if s.lk.age == ageEmpire {
		s.block(0.44, 0.12, 0.6, 0.28, 32, 34, flatMat(goldR[2]), flatMat(inkSft), matTrim)
	}
}

func hutManorCrown(s iso, b roofBox, roof ramp) {
	top := b.z + b.rise
	switch s.lk.age {
	case ageDark:
		s.hutLip(b.y1+b.eave, b.x0-b.eave, b.x1+b.eave, b.z, roof)
	case ageCastle:
		s.turn(lathe{cx: 0.8, cy: 0.2, h: 27, r: 0.13}, ashlarMat(stoneR, s.lk.seed+5), matStone)
		s.roofCone(0.8, 0.2, 0.16, 27, 10, shapeRound, hutRoofMat(ageCastle, roof, s.lk.seed+19))
		w := s.p(0.89, 0.29, 16)
		s.slit(upright(int(w.x), int(w.y)), 1, 4, s.litWindow(0.8, 16))
	case ageEmpire:
		ym := (b.y0 + b.y1) / 2
		s.pen(matTrim, func() { s.edge(s.p(b.x0, ym, top), s.p(b.x1, ym, top), goldR[2]) })
		s.turn(lathe{cx: 0.24, cy: ym, z: top - 3, h: 4, r: 0.08}, toneMat(marbleR), matStone)
		s.roofDome(0.24, ym, 0.1, top+1, 5, ribbedMat(copperR, s.lk.seed))
		s.finial(0.24, ym, top+6, 2)
	}
}

func manorFront(s iso, roof ramp, lit bool, low float64) {
	up := manorY1
	if s.lk.age == ageFeudal {
		up += manorJetty
	}
	s.tierDoorL(manorY1, gateU0+0.02, gateU1-gateU0-0.04, 1.5, 8, roof)
	s.hutWindow(sideL(manorY1), 0.48, 4, lit)
	s.hutWindow(sideL(up), 0.16, low+3, lit)
	s.hutWindow(sideL(up), 0.5, low+3, lit)
	s.bannerL(up, 0.35, manorRoof-1, 0.1, 6, roof, goldR[3])
	s.hutWindow(sideR(manorX1), 0.3, 4, lit)
	s.hutWindow(sideR(manorX1), 0.26+s.lk.jit(7, 0.04), low+3, lit)
	if s.lk.variant == 2 {
		s.hutFlowerBox(sideR(manorX1), 0.3, 3)
	}
}

func garden(s iso) {
	top := func(x, y int) color.NRGBA {
		gx, gy := s.ground(x, y, 0)
		edge := min(gx-0.05, gardenEdge-gx, gy-0.05, gardenEdge-gy)
		switch {
		case edge < 0.08 && dither(x, y) > edge/0.08:
			return color.NRGBA{}
		case gx > gateU0+0.02 && gx < gateU1-0.02 && gy > manorY1:
			return gardenPath(x, y, s.lk.age)
		case gy > manorY1 && gy < manorY1+0.12 && gx > gateU1+0.04 && gx < manorX1+0.1:
			return bed(x, y)
		case gx > manorX1 && gx < manorX1+0.12 && gy < manorY1+0.1:
			return bed(x, y)
		case noise(x, y, 71) > 0.82:
			return grassR[3]
		}
		return grassR[2]
	}
	s.quad(s.p(0.05, 0.05, 0), s.p(gardenEdge, 0.05, 0), s.p(gardenEdge, gardenEdge, 0), s.p(0.05, gardenEdge, 0), top)
}

func gardenGreens(s iso) {
	leaf := greenR
	if s.lk.age == ageDark {
		leaf = forestR
	}
	for i, b := range [][3]float64{{0.86, 0.18, 2.6}, {0.84, 0.84, 2.4}} {
		p := s.p(b[0], b[1], 0)
		blob(s.c, p.x, p.y-b[2], b[2], leaf, 5+i+s.lk.variant)
	}
}

func gardenPath(x, y, age int) color.NRGBA {
	if age == ageEmpire {
		return plazaR[2+int(noise(x, y, 73)*1.4)]
	}
	if noise(x, y, 73) > 0.7 {
		return sandR[1]
	}
	return sandR[2]
}

func bed(x, y int) color.NRGBA {
	n := hash(x, y, 79)
	if n%3 == 0 {
		return bloomColours[(n/3)%uint32(len(bloomColours))]
	}
	return dirtR[1]
}

func fence(s iso) {
	for _, run := range [][2][2]float64{
		{{0.06, gardenEdge}, {gateU0, gardenEdge}},
		{{gateU1, gardenEdge}, {gardenEdge, gardenEdge}},
		{{gardenEdge, 0.06}, {gardenEdge, gardenEdge}},
	} {
		s.hutFenceRun(run[0], run[1])
	}
}

func (s iso) hutFenceRun(a, b [2]float64) {
	post, rail, h := woodR[3], woodR[2], 4.0
	switch s.lk.age {
	case ageDark:
		post, rail = woodR[1], woodR[2]
	case ageCastle:
		post, rail, h = stoneR[2], stoneR[3], 3
	case ageEmpire:
		post, rail = inkSft, goldR[1]
	}
	p, q := s.p(a[0], a[1], h-0.5), s.p(b[0], b[1], h-0.5)
	s.c.line(int(p.x), int(p.y), int(q.x), int(q.y), rail)
	if s.lk.age == ageDark || s.lk.age == ageCastle {
		p, q = s.p(a[0], a[1], h-2), s.p(b[0], b[1], h-2)
		s.c.line(int(p.x), int(p.y), int(q.x), int(q.y), post)
	}
	n := int(max(absf(b[0]-a[0]), absf(b[1]-a[1]))/0.22) + 1
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		gx, gy := a[0]+(b[0]-a[0])*t, a[1]+(b[1]-a[1])*t
		lo, hi := s.p(gx, gy, 0), s.p(gx, gy, h)
		s.c.line(int(lo.x), int(lo.y), int(hi.x), int(hi.y), post)
		if s.lk.age == ageEmpire {
			s.c.dot(int(hi.x), int(hi.y)-1, goldR[3])
		}
	}
}
