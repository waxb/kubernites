package art

import "image/color"

const (
	plaqueV0 = 0.42
	plaqueV1 = 0.58
)

func init() {
	stallPainters["stall-plaque"] = paintPlaqueStall
	stallEmblems["stall-plaque"] = routeGlyph
}

func paintPlaqueStall(t stall) {
	t.shadowFloor()
	t.footing()
	t.plaqueBase()
	top := t.plaqueSlab()
	t.plaqueFace(top)
	if t.lk.age <= ageFeudal {
		t.plaqueHood(top)
	}
}

func (t stall) plaqueBase() {
	switch t.lk.age {
	case ageFeudal:
		t.slab(0.2, 0.3, 1.8, 0.7, 0, 4, rubbleMat(stoneR, t.lk.seed), flatMat(stoneR[2]), matStone)
	case ageCastle:
		t.slab(0.16, 0.26, 1.84, 0.74, t.floorZ(), t.floorZ()+3, ashlarMat(stoneR, t.lk.seed+1), flatMat(stoneR[3]), matStone)
	case ageEmpire:
		t.slab(0.16, 0.26, 1.84, 0.74, t.floorZ(), t.floorZ()+3, flatMat(porphyryR[1]), flatMat(porphyryR[2]), matStone)
	}
}

func (t stall) plaqueFoot() float64 {
	switch t.lk.age {
	case ageFeudal:
		return 4
	case ageCastle, ageEmpire:
		return t.floorZ() + 3
	}
	return 0
}

func (t stall) plaqueSlab() float64 {
	z0 := t.plaqueFoot()
	switch t.lk.age {
	case ageDark, ageFeudal:
		for _, u := range []float64{0.3, 1.7} {
			t.post(u, 0.5, 0, z0+22)
		}
		t.slab(0.24, plaqueV0, 1.76, plaqueV1, z0+7, z0+20, plankMat(woodR, t.lk.seed+3), flatMat(woodR[3]), matWall)
		return z0 + 20
	case ageCastle:
		t.slab(0.3, plaqueV0, 1.7, plaqueV1, z0, z0+22, ashlarMat(stoneR, t.lk.seed+2), flatMat(stoneR[3]), matStone)
		return z0 + 22
	}
	t.slab(0.3, plaqueV0, 1.7, plaqueV1, z0, z0+23, marbleMat(t.lk.seed+2), marbleMat(t.lk.seed+3), matStone)
	return z0 + 23
}

func (t stall) plaqueFace(top float64) {
	face, v := frontAt(plaqueV1), plaqueV1
	if !t.facesCamera() {
		face, v = backAt(plaqueV0), plaqueV0
	}
	switch t.lk.age {
	case ageDark, ageFeudal:
		t.notes(face, top)
	default:
		t.inscription(face, top)
	}
	t.aloft(top, func() { t.emblem(t.at(1, v, top-6)) })
}

func (t stall) notes(face plane, top float64) {
	for i, u := range []float64{0.34, 0.62, 1.3, 1.52} {
		z := top - 11 + float64(i%2)*4
		t.pen(matTrim, func() {
			t.panel(face, u, u+0.16, z, z+4, flat(paperR[3]))
			t.panel(face, u+0.04, u+0.12, z+2, z+3, flat(inkSft))
		})
	}
}

func (t stall) inscription(face plane, top float64) {
	rim, field := stoneR[1], t.cloth()[3]
	if t.lk.age == ageEmpire {
		rim, field = goldR[2], t.cloth()[2]
	}
	t.pen(matTrim, func() {
		t.panel(face, 0.66, 1.34, top-10, top-2, flat(rim))
		t.panel(face, 0.72, 1.28, top-9, top-3, flat(field))
		for _, z := range []float64{top - 16, top - 13} {
			t.panel(face, 0.5, 1.5, z, z+1, flat(mix(rim, inkSft, 0.5)))
		}
	})
}

func (t stall) plaqueHood(top float64) {
	m := t.roofMat()
	t.pen(matRoof, func() {
		t.quad(t.at(0.18, 0.3, top+2), t.at(1.82, 0.3, top+2), t.at(1.82, 0.7, top+2), t.at(0.18, 0.7, top+2), t.topMat(top+2, m))
	})
	t.slab(0.18, 0.3, 1.82, 0.7, top, top+2, m, m, matRoof)
}

func routeGlyph(c *canvas, x, y int, fg, ac color.NRGBA) {
	c.rect(x-3, y, 3, 1, fg)
	c.dot(x, y-1, fg)
	c.dot(x, y+1, fg)
	for _, dy := range []int{-2, 0, 2} {
		c.dot(x+1, y+dy, fg)
		c.dot(x+2, y+dy, ac)
	}
}
