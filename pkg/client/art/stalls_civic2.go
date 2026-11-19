package art

import (
	"image/color"
	"math"
)

const archiveRise = 6.0

func init() {
	stallPainters["stall-archive"] = paintArchiveStall
	stallPainters["stall-herald"] = paintHeraldStall
}

func paintArchiveStall(t stall) {
	p := t.plan()
	p.back += archiveRise
	t.shadowFloor()
	t.footing()
	if t.facesCamera() {
		t.archiveRack(p)
		t.counter(p)
		t.goods(p)
		t.frontPosts(p)
		t.roof(p)
	} else {
		t.frontPosts(p)
		t.counter(p)
		t.goods(p)
		t.roof(p)
		t.archiveRack(p)
	}
}

func (t stall) archiveRack(p stallPlan) {
	if !t.facesCamera() {
		low := p
		low.back -= archiveRise
		t.lowBack(low)
		return
	}
	wall, top, tag := t.wallMats()
	t.slab(0.06, wallV0, 1.94, wallV1, t.floorZ(), p.back, wall, top, tag)
	if t.facesCamera() {
		t.pigeonholes(p)
	}
	band := p
	band.eave = p.back - 7
	t.band(band)
	if t.lk.age >= ageFeudal {
		t.rackCoping(p)
	}
}

func (t stall) rackCoping(p stallPlan) {
	x0, y0, x1, y1 := t.grid(0.02, wallV0-0.04, 1.98, wallV1+0.05)
	r := woodR
	switch t.lk.age {
	case ageCastle:
		r = stoneR
	case ageEmpire:
		r = marbleR
	}
	t.pen(matTrim, func() {
		t.box(x0, y0, x1, y1, p.back, p.back+1.5, faces{top: flat(r[3]), left: flat(r[2]), right: flat(r[1])})
	})
}

func (t stall) pigeonholes(p stallPlan) {
	face := frontAt(wallV1)
	hole, scroll := rgb(46, 38, 44), paperR[3]
	if t.lk.age == ageEmpire {
		hole = mix(hole, porphyryR[0], 0.3)
	}
	for row := 0; row < 2; row++ {
		zb := p.eave + 0.5 + float64(row)*3.5
		for i := 0; i < 7; i++ {
			u := 0.16 + float64(i)*0.245
			tone := scroll
			if (i+row+t.lk.variant)%3 == 0 {
				tone = t.lk.id[3]
			}
			t.pen(matTrim, func() {
				t.panel(face, u, u+0.19, zb, zb+3, flat(hole))
				t.panel(face, u+0.05, u+0.13, zb+1, zb+2, flat(tone))
			})
		}
	}
}

func paintHeraldStall(t stall) {
	t.shadowFloor()
	t.dais()
	t.bunting()
	post, bell := 0.5, 1.45
	if t.depthOf(post) > t.depthOf(bell) {
		t.bellFrame(bell)
		t.signpost(post)
		return
	}
	t.signpost(post)
	t.bellFrame(bell)
}

func (t stall) bunting() {
	if t.lk.age >= ageCastle {
		return
	}
	for _, f := range []plane{frontAt(0.9), backAt(0.12)} {
		if !t.sees(f) {
			continue
		}
		for i := 0; i < 7; i++ {
			a := 0.14 + float64(i)*0.245
			col := t.lk.id[2]
			if i%2 == 1 {
				col = plasterR[3]
			}
			t.pen(matTrim, func() { t.panel(f, a, a+0.245, t.daisTop()-2-float64(i%2), t.daisTop(), flat(col)) })
		}
	}
}

func (t stall) depthOf(u float64) float64 {
	gx, gy := t.g(u, 0.5)
	return gx + gy
}

func (t stall) daisTop() float64 { return 4 + t.floorZ() }

func (t stall) dais() {
	t.footing()
	z0, z1 := t.floorZ(), t.daisTop()
	u0, v0, u1, v1 := 0.1, 0.12, 1.9, 0.9
	switch t.lk.age {
	case ageDark:
		t.slab(u0, v0, u1, v1, z0, z1, plankMat(woodR, t.lk.seed+2), plankMat(woodR, t.lk.seed+3), matWall)
	case ageFeudal:
		t.slab(u0, v0, u1, v1, z0, z1-1, rubbleMat(stoneR, t.lk.seed+2), flatMat(stoneR[2]), matStone)
		t.flatSlab(u0-0.03, v0-0.03, u1+0.03, v1+0.03, z1-1, z1, woodR)
	case ageCastle:
		t.slab(u0, v0, u1, v1, z0, z1, ashlarMat(stoneR, t.lk.seed+2), flatMat(stoneR[3]), matStone)
	default:
		t.slab(u0, v0, u1, v1, z0, z1, marbleMat(t.lk.seed+2), marbleMat(t.lk.seed+3), matStone)
		t.panel(frontAt(v1), u0+0.2, u1-0.2, z0+1, z1-1, flat(porphyryR[1]))
		t.panel(backAt(v0), u0+0.2, u1-0.2, z0+1, z1-1, flat(porphyryR[1]))
	}
}

func (t stall) signpost(u float64) {
	top := float64(tallStallHeight) - 4
	z0 := t.daisTop()
	switch t.lk.age {
	case ageEmpire:
		t.column(u, 0.5, z0, top-6)
		t.finialAt(t.at(u, 0.5, top-6), 5)
	case ageCastle:
		gx, gy := t.g(u, 0.5)
		t.pen(matTrim, func() {
			t.box(gx-0.04, gy-0.04, gx+0.04, gy+0.04, z0, top, faces{top: flat(greyR[3]), left: flat(greyR[2]), right: flat(greyR[1])})
		})
	default:
		gx, gy := t.g(u, 0.5)
		t.pen(matTrim, func() {
			t.box(gx-0.05, gy-0.05, gx+0.05, gy+0.05, z0, top, faces{top: flat(woodR[3]), left: flat(woodR[2]), right: flat(woodR[0])})
		})
	}
	t.fingers(u, top)
}

func (t stall) fingerTones() (color.NRGBA, color.NRGBA) {
	switch t.lk.age {
	case ageDark:
		return woodR[3], woodR[1]
	case ageFeudal:
		return plasterR[3], t.lk.id[1]
	case ageCastle:
		return t.lk.id[2], greyR[1]
	}
	return goldR[3], goldR[1]
}

func (t stall) fingers(u, top float64) {
	body, edge := t.fingerTones()
	at := t.at(u, 0.5, top)
	for i, w := range []float64{7, 6, 5} {
		left := (i+t.lk.variant)%2 == 1
		y := at.y + 2 + float64(i)*5
		x := at.x + 1
		if left {
			x = at.x - w - 3
		}
		t.aloft(top-float64(i)*5, func() { mkArrow(t.c, x, y, w, body, edge, left) })
	}
}

func (t stall) bellFrame(u float64) {
	z0, top := t.daisTop(), t.daisTop()+18
	for _, v := range []float64{0.3, 0.7} {
		switch t.lk.age {
		case ageDark:
			t.pole(u, v, z0, top+1, woodR)
		case ageEmpire:
			t.column(u, v, z0, top)
		default:
			t.post(u, v, z0, top)
		}
	}
	x0, y0, x1, y1 := t.grid(u-0.05, 0.26, u+0.05, 0.74)
	t.flatSlab2(x0, y0, x1, y1, top, top+2)
	c := t.at(u, 0.5, top-5)
	t.aloft(top, func() { t.bell(c) })
}

func (t stall) flatSlab2(x0, y0, x1, y1, z0, z1 float64) {
	r := woodR
	switch t.lk.age {
	case ageCastle:
		r = stoneR
	case ageEmpire:
		r = marbleR
	}
	t.pen(matTrim, func() { t.box(x0, y0, x1, y1, z0, z1, faces{top: flat(r[3]), left: flat(r[2]), right: flat(r[1])}) })
}

func (t stall) bell(c pt) {
	metal := brassR
	if t.lk.age == ageEmpire {
		metal = goldR
	}
	t.c.line(int(c.x), int(c.y)-5, int(c.x), int(c.y)-3, woodR[0])
	t.c.ellipse(c.x+0.5, c.y, 3, 3.2, metal[1])
	t.c.ellipse(c.x-0.3, c.y-0.8, 1.5, 1.8, metal[3])
	t.c.rect(int(c.x)-3, int(c.y)+2, 7, 1, metal[0])
	t.c.dot(int(math.Round(c.x)), int(c.y)+3, inkSft)
}
