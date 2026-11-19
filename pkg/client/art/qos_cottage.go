package art

const (
	cotX0, cotX1 = 0.2, 0.8
	cotY0, cotY1 = 0.26, 0.8
	cotYM        = 0.53
	cotRoof      = 13.0
	cotRise      = 10.0
	cotSill      = 3.0
)

var (
	hutCotEaves = [AgeTiers]float64{0.13, 0.1, 0.08, 0.08}
	hutCotWalls = [AgeTiers]float64{11, 13, 14, 15}
	hutCotSills = [AgeTiers]float64{1, 3, 3, 3}
)

func paintFreeman(s iso, h hut, lit bool) {
	roof, age := tierRamp(h.tier), s.lk.age
	s.hutGround().apron(1, 0.1)
	hutCottageBack(s)
	sill, top := hutCotSills[age], hutCotWalls[age]
	if age >= ageFeudal {
		s.block(cotX0-0.02, cotY0-0.02, cotX1+0.02, cotY1+0.02, 0, sill, rubbleMat(stoneR, s.lk.seed), flatMat(stoneR[2]), matStone)
	}
	s.walls(cotX0, cotY0, cotX1, cotY1, sill, top)
	hutCotChimney(s, 14)
	b := roofBox{cotX0, cotY0, cotX1, cotY1, top, cotRoof + cotRise - top, hutCotEaves[age]}
	s.roofGableX(b, hutRoofMat(age, roof, s.lk.seed+17), s.kit().wall)
	hutCottageTrim(s, b, roof)
	s.tierDoorL(cotY1, 0.28, 0.16, sill, 8, roof)
	hutCottageWindows(s, lit, sill)
	hutCottageSmoke(s, lit)
	hutCottageProps(s)
}

func hutCottageBack(s iso) {
	if s.lk.variant == 1 && s.lk.age >= ageFeudal {
		s.hutWoodpile(0.1, 0.2)
	}
}

func hutCotChimney(s iso, zb float64) {
	x0, y0, x1, y1, zt := 0.6, 0.34, 0.7, 0.44, 27.0
	k := s.kit()
	body := k.base
	switch s.lk.age {
	case ageDark:
		body = wattleMat(s.lk.seed + 3)
	case ageEmpire:
		body = k.wall
	}
	s.block(x0, y0, x1, y1, zb, zt, body, flatMat(inkSft), matStone)
	switch s.lk.age {
	case ageCastle:
		s.block(x0-0.02, y0-0.02, x1+0.02, y1+0.02, zt-2, zt, flatMat(stoneR[3]), flatMat(inkSft), matStone)
	case ageEmpire:
		s.block(x0-0.02, y0-0.02, x1+0.02, y1+0.02, zt-2, zt, flatMat(goldR[2]), flatMat(inkSft), matTrim)
	}
}

func hutCottageTrim(s iso, b roofBox, roof ramp) {
	switch s.lk.age {
	case ageDark:
		s.hutLip(b.y1+b.eave, b.x0-b.eave, b.x1+b.eave, b.z, roof)
	case ageEmpire:
		ym := (b.y0 + b.y1) / 2
		s.pen(matTrim, func() { s.edge(s.p(b.x0, ym, b.z+b.rise), s.p(b.x1, ym, b.z+b.rise), goldR[2]) })
	}
}

func hutCottageWindows(s iso, lit bool, sill float64) {
	z := sill + 3
	u := 0.56 + s.lk.jit(5, 0.04)
	s.hutWindow(sideL(cotY1), u, z, lit)
	s.hutWindow(sideR(cotX1), 0.4, z, lit)
	if s.lk.variant == 1 {
		s.hutWindow(sideR(cotX1), 0.62, z, lit && s.lk.chance(6, 0.5))
	}
	if s.lk.variant == 2 && s.lk.age >= ageFeudal {
		s.hutFlowerBox(sideL(cotY1), u, z-1)
	}
}

func hutCottageSmoke(s iso, lit bool) {
	if !lit {
		return
	}
	top := s.p(0.65, 0.39, 27)
	s.c.dot(int(top.x), int(top.y)-2, withAlpha(greyR[3], 200))
	s.c.dot(int(top.x)+1, int(top.y)-4, withAlpha(greyR[2], 170))
}

func hutCottageProps(s iso) {
	switch s.lk.variant {
	case 0:
		s.barrel(0.9, 0.3)
	case 1:
		s.hutBucket(0.14, 0.9)
	default:
		s.hutProp(s.lk.pick(4, 4), 0.9, 0.32)
	}
}
