package art

import (
	"image/color"
	"math"
)

func init() {
	stallPainters["stall-guildhall"] = paintGuildStall
}

func paintGuildStall(t stall) {
	p := t.plan()
	t.shadowFloor()
	t.footing()
	if t.facesCamera() {
		t.guildBack(p)
		t.counter(p)
		t.goods(p)
		t.guildPosts(p)
	} else {
		t.guildPosts(p)
		t.counter(p)
		t.goods(p)
		t.guildBack(p)
	}
	t.guildRoof(p)
}

func (t stall) guildBack(p stallPlan) {
	wall, top, tag := t.wallMats()
	t.slab(0.04, wallV0, 1.96, wallV1, t.floorZ(), p.posts+1, wall, top, tag)
	if t.lk.age == ageFeudal {
		face := t.wallFace()
		t.pen(matTrim, func() { t.panel(face, 0.04, 1.96, p.posts, p.posts+1, flat(woodR[0])) })
	}
}

func (t stall) guildPosts(p stallPlan) {
	us := []float64{0.1, 1.9}
	if t.lk.age == ageEmpire {
		us = []float64{0.1, 0.7, 1.3, 1.9}
	}
	for _, u := range us {
		switch t.lk.age {
		case ageDark:
			t.pole(u, 0.95, 0, p.posts+2, woodR)
		case ageEmpire:
			t.column(u, 0.9, t.floorZ(), p.posts+1)
		default:
			t.post(u, 0.93, t.floorZ(), p.posts+1)
		}
	}
}

func (t stall) guildRoof(p stallPlan) {
	x0, y0, x1, y1 := t.grid(0.08, 0.06, 1.92, 0.94)
	rise := 10.0
	b := roofBox{x0, y0, x1, y1, p.posts + 1, rise, 0.06}
	m := t.roofMat()
	if t.face == "se" || t.face == "nw" {
		t.roofGableX(b, m, t.kit().wall)
	} else {
		t.roofGableY(b, m, t.kit().wall)
	}
	if t.lk.age == ageDark {
		t.thatchLip(b, t.face == "sw" || t.face == "ne")
	}
	t.guildCrest(p, rise)
}

func (t stall) guildCrest(p stallPlan, rise float64) {
	v := 1.0
	if !t.facesCamera() {
		v = 0.02
	}
	apex := t.at(1, v, p.posts+1+rise)
	c := t.at(1, v, p.posts+1+rise*0.42)
	t.aloft(p.posts+rise, func() {
		t.c.disc(c.x+0.5, c.y+0.5, 4.2, t.crestRim())
		t.c.disc(c.x+0.5, c.y+0.5, 3.4, plasterR[3])
		hornGlyph(t.c, int(math.Round(c.x)), int(math.Round(c.y)), t.lk.id[0], goldR[2])
	})
	t.aloft(p.posts+rise+2, func() {
		if t.lk.age == ageEmpire {
			t.finialAt(apex, 4)
			return
		}
		t.pennant(apex)
	})
}

func (t stall) crestRim() color.NRGBA {
	if t.lk.age >= ageCastle {
		return goldR[2]
	}
	return woodR[0]
}

func (t stall) pennant(at pt) {
	x, y := int(at.x), int(at.y)
	t.c.rect(x, y-6, 1, 6, woodR[0])
	t.c.shadePoly([]pt{{float64(x) + 1, float64(y) - 6}, {float64(x) + 6, float64(y) - 4.5}, {float64(x) + 1, float64(y) - 3}}, flat(t.lk.id[2]))
}

func (t stall) finialAt(at pt, h int) {
	x, y := int(at.x), int(at.y)
	t.c.rect(x, y-h, 1, h, goldR[1])
	t.c.dot(x, y-h-1, goldR[3])
}
