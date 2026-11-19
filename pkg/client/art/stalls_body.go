package art

type stallPlan struct {
	counter, posts, back, eave float64
}

var stallPlans = [AgeTiers]stallPlan{
	{counter: 7, posts: 14, back: 26, eave: 17},
	{counter: 8, posts: 14, back: 26, eave: 17},
	{counter: 8, posts: 15, back: 27, eave: 18},
	{counter: 8, posts: 16, back: 29, eave: 20},
}

const (
	wallV0   = 0.08
	wallV1   = 0.22
	eaveV    = 0.26
	frontV   = 1.04
	counter0 = 0.56
	counter1 = 0.84
)

func (t stall) plan() stallPlan { return stallPlans[t.lk.age] }

func (t stall) cloth() ramp {
	if t.lk.age == ageDark {
		return mixRamp(t.lk.id, strawR, 0.12)
	}
	return t.lk.id
}

func (t stall) footing() {
	switch t.lk.age {
	case ageFeudal:
		t.slab(0.16, 0.5, 1.84, 0.9, 0, 2, rubbleMat(stoneR, t.lk.seed), flatMat(stoneR[2]), matStone)
	case ageCastle:
		t.slab(0.06, 0.06, 1.94, 0.96, 0, 2, ashlarMat(stoneR, t.lk.seed), flatMat(stoneR[3]), matStone)
	case ageEmpire:
		t.slab(0.04, 0.05, 1.96, 0.97, 0, 3, marbleMat(t.lk.seed), marbleMat(t.lk.seed+1), matStone)
		t.panel(frontAt(0.97), 0.04, 1.96, 0, 1, flat(porphyryR[1]))
		t.panel(backAt(0.05), 0.04, 1.96, 0, 1, flat(porphyryR[1]))
		t.panel(endAt(1.96, 1), 0.05, 0.97, 0, 1, flat(porphyryR[1]))
		t.panel(endAt(0.04, -1), 0.05, 0.97, 0, 1, flat(porphyryR[1]))
	}
}

func (t stall) floorZ() float64 {
	switch t.lk.age {
	case ageCastle:
		return 2
	case ageEmpire:
		return 3
	}
	return 0
}

func (t stall) wallMats() (mat, mat, uint8) {
	switch t.lk.age {
	case ageDark:
		return plankMat(woodR, t.lk.seed+5), flatMat(woodR[3]), matWall
	case ageFeudal:
		return plasterMat(t.lk.seed + 5), flatMat(woodR[2]), matWall
	case ageCastle:
		return ashlarMat(stoneR, t.lk.seed+2), flatMat(stoneR[3]), matStone
	}
	return marbleMat(t.lk.seed + 2), marbleMat(t.lk.seed + 3), matStone
}

func (t stall) backWall(p stallPlan) {
	if !t.facesCamera() {
		t.lowBack(p)
		return
	}
	wall, top, tag := t.wallMats()
	t.slab(0.06, wallV0, 1.94, wallV1, t.floorZ(), p.back, wall, top, tag)
	switch t.lk.age {
	case ageDark:
		t.wallPoles(p)
	case ageFeudal:
		t.wallBeams(p)
	case ageEmpire:
		t.cornice(p)
	}
	t.band(p)
}

func (t stall) wallPoles(p stallPlan) {
	for _, u := range []float64{0.06, 1.94} {
		t.pole(u, wallV1, 0, p.back+2, woodR)
	}
}

func (t stall) wallBeams(p stallPlan) {
	face := t.wallFace()
	t.pen(matTrim, func() {
		t.panel(face, 0.06, 1.94, p.back-1, p.back, flat(woodR[0]))
		t.panel(face, 0.06, 1.94, 3, 4, flat(woodR[0]))
		for _, u := range []float64{0.06, 0.7, 1.3, 1.94} {
			t.panel(face, u-0.04, u+0.04, 0, p.back, flat(woodR[1]))
		}
	})
}

func (t stall) cornice(p stallPlan) {
	face := t.wallFace()
	t.pen(matTrim, func() {
		t.panel(face, 0.06, 1.94, p.back-2, p.back-1, flat(goldR[2]))
		t.panel(face, 0.06, 1.94, p.back-1, p.back, flat(marbleR[3]))
	})
}

func (t stall) wallFace() plane {
	if t.facesCamera() {
		return frontAt(wallV1)
	}
	return backAt(wallV0)
}

func (t stall) lowBack(p stallPlan) {
	wall, top, tag := t.wallMats()
	for _, u := range []float64{0.08, 1.92} {
		t.backPost(u, p.eave+1)
	}
	t.slab(0.06, wallV0, 1.94, wallV1, t.floorZ(), p.counter+3, wall, top, tag)
	board := wallV0 - 0.05
	t.slab(0.48, board, 1.52, wallV0, p.eave-8, p.eave-1, wall, top, tag)
	t.signBand(backAt(board), board, p.eave-8, p.eave-1)
}

func (t stall) backPost(u, top float64) {
	switch t.lk.age {
	case ageDark:
		t.pole(u, 0.15, 0, top, woodR)
	case ageCastle:
		gx, gy := t.g(u, 0.15)
		t.block(gx-0.06, gy-0.06, gx+0.06, gy+0.06, t.floorZ(), top, ashlarMat(stoneR, t.lk.seed+6), flatMat(stoneR[3]), matStone)
	case ageEmpire:
		t.column(u, 0.15, t.floorZ(), top)
	default:
		t.post(u, 0.15, 0, top)
	}
}

func (t stall) band(p stallPlan) {
	t.signBand(t.wallFace(), wallV1, p.eave+0.5, p.back-1)
}

func (t stall) signBand(face plane, v, z0, z1 float64) {
	rim, inset := woodR[0], plasterR[3]
	switch t.lk.age {
	case ageFeudal:
		inset = t.cloth()[3]
	case ageCastle:
		rim, inset = stoneR[1], t.cloth()[3]
	case ageEmpire:
		rim, inset = goldR[2], t.cloth()[2]
	}
	t.pen(matTrim, func() {
		t.panel(face, 0.5, 1.5, z0, z1, flat(rim))
		t.panel(face, 0.56, 1.44, z0+1, z1-1, flat(inset))
	})
	t.aloft(z1, func() { t.emblem(t.at(1, v, (z0+z1)/2)) })
}

func (t stall) counter(p stallPlan) {
	z0, u0, u1 := t.floorZ(), 0.24, 1.76
	switch t.lk.age {
	case ageDark:
		t.slab(u0, counter0, u1, counter1, 0, p.counter, plankMat(woodR, t.lk.seed+7), flatMat(woodR[3]), matWall)
	case ageFeudal:
		t.slab(u0, counter0, u1, counter1, 0, p.counter-2, rubbleMat(stoneR, t.lk.seed+3), flatMat(stoneR[2]), matStone)
		t.flatSlab(u0-0.03, counter0-0.03, u1+0.03, counter1+0.03, p.counter-2, p.counter, woodR)
	case ageCastle:
		t.slab(u0, counter0, u1, counter1, z0, p.counter, ashlarMat(stoneR, t.lk.seed+4), flatMat(stoneR[3]), matStone)
	default:
		t.slab(u0, counter0, u1, counter1, z0, p.counter, marbleMat(t.lk.seed+4), marbleMat(t.lk.seed+5), matStone)
		t.panel(frontAt(counter1), u0+0.25, u1-0.25, z0+1.5, p.counter-1.5, flat(porphyryR[2]))
	}
	if t.lk.age <= ageFeudal {
		t.panel(frontAt(counter1), u0+0.1, u1-0.1, p.counter-3, p.counter-2, flat(t.cloth()[1]))
	}
}

func (t stall) frontPosts(p stallPlan) {
	z0 := t.floorZ()
	us := []float64{0.12, 1.88}
	if t.lk.age >= ageCastle {
		us = []float64{0.12, 1.0, 1.88}
	}
	for _, u := range us {
		switch t.lk.age {
		case ageDark:
			t.pole(u, 0.94, 0, p.posts+1, woodR)
		case ageCastle:
			gx, gy := t.g(u, 0.92)
			t.block(gx-0.07, gy-0.07, gx+0.07, gy+0.07, z0, p.posts, ashlarMat(stoneR, t.lk.seed+6), flatMat(stoneR[3]), matStone)
		case ageEmpire:
			t.column(u, 0.9, z0, p.posts)
		default:
			t.post(u, 0.92, z0, p.posts)
		}
	}
}

func (t stall) column(u, v, z0, z1 float64) {
	gx, gy := t.g(u, v)
	t.turn(lathe{cx: gx, cy: gy, z: z0, h: 2, r: 0.08}, toneMat(marbleR), matTrim)
	t.turn(lathe{cx: gx, cy: gy, z: z0 + 2, h: z1 - z0 - 4, r: 0.06}, toneMat(marbleR), matTrim)
	t.turn(lathe{cx: gx, cy: gy, z: z1 - 2, h: 2, r: 0.09}, toneMat(marbleR), matTrim)
}
