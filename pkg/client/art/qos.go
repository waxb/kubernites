package art

import (
	"fmt"
	"image/color"
)

var HearthQoS = []string{"vagrant", "freeman", "landed"}

type hut struct {
	tier    int
	qos     string
	age     int
	variant int
}

var tierRamps = [...]ramp{strawR, strawR, redR, blueR}

func tierRamp(tier int) ramp { return tierRamps[min(max(tier, 0), 3)] }

func tentRamp(tier int) ramp {
	r := tierRamp(tier)
	out := ramp{}
	for i, c := range r {
		out[i] = mix(c, greyR[i], 0.3)
	}
	return out
}

func hearthQoSName(tier int, state, qos string) string {
	return fmt.Sprintf("hearth/%d/%s/%s", tier, state, qos)
}

func addQoSHearths(b *atlasBuilder) {
	s, m := hearthSize(), newHutMemo()
	for t := 1; t <= 3; t++ {
		for _, q := range HearthQoS {
			for age := 0; age < AgeTiers; age++ {
				for _, st := range HearthStates {
					for v := 0; v < hutVariants(st); v++ {
						h, state := hut{tier: t, qos: q, age: age, variant: v}, st
						b.add(HearthLookName(t, st, q, age, v), s.w, s.h, s.ax, s.ay, func(c *canvas) { paintQoSHearth(c, s, h, state, m) })
					}
				}
			}
		}
	}
}

func paintQoSHearth(c *canvas, s size, h hut, state string, m *hutMemo) {
	switch state {
	case "pending":
		c.paste(hutRough(s, h.look(), func(fr iso) { paintQoSPending(fr, h) }), 0, 0, false)
	case "kindling":
		paintQoSKindling(c, s, h, m)
	case "banished":
		c.paste(hutRough(s, h.look(), func(fr iso) { paintQoSRubble(fr, h) }), 0, 0, false)
		c.mapPixels(func(col color.NRGBA) color.NRGBA { return mix(col, grey(col), 0.5) })
	default:
		house := newCanvas(s.w, s.h)
		house.paste(m.house(s, h, state == "guttering"), 0, 0, false)
		if state == "guttering" {
			ridgeFire(house, qosRidge(hearthFrame(house, s), h))
		}
		if state == "ghost" {
			house.mapPixels(func(col color.NRGBA) color.NRGBA {
				return withAlpha(mix(grey(col), rgb(170, 200, 230), 0.4), min(col.A, 130))
			})
		}
		c.paste(house, 0, 0, false)
	}
}

func qosHouse(s size, h hut, burning bool) *canvas {
	return hutRough(s, h.look(), func(fr iso) {
		switch h.qos {
		case "vagrant":
			paintVagrant(fr, h, !burning)
		case "landed":
			paintLanded(fr, h, !burning)
		default:
			paintFreeman(fr, h, !burning)
		}
	})
}

type ridge struct {
	a, b pt
}

func qosRidge(s iso, h hut) ridge {
	switch h.qos {
	case "vagrant":
		return shackRidge(s)
	case "landed":
		return ridge{s.p(manorX0, manorYM, manorRoof+manorRise), s.p(manorX1, manorYM, manorRoof+manorRise)}
	}
	return ridge{s.p(cotX0, cotYM, cotRoof+cotRise), s.p(cotX1, cotYM, cotRoof+cotRise)}
}

func ridgeFire(c *canvas, r ridge) {
	for i := 0; i < 4; i++ {
		t := 0.15 + float64(i)*0.23
		x, y := r.a.x+(r.b.x-r.a.x)*t, r.a.y+(r.b.y-r.a.y)*t+2
		h := float64(5 + (i*3)%4)
		c.shadePoly([]pt{{x - 2.5, y}, {x, y - h}, {x + 2.5, y}}, flat(fireR[1]))
		c.dot(int(x), int(y)-2, fireR[3])
		c.dot(int(x), int(y)-1, fireR[2])
	}
	for i := 0; i < 5; i++ {
		x := r.a.x + (r.b.x-r.a.x)*float64(i)/4
		c.dot(int(x), int(r.a.y+(r.b.y-r.a.y)*float64(i)/4)-9-i%3, withAlpha(greyR[0], 200))
	}
}

func tierPennant(s iso, gx, gy, z float64, r ramp) {
	base, top := s.p(gx, gy, z), s.p(gx, gy, z+7)
	s.c.line(int(base.x), int(base.y), int(top.x), int(top.y), woodR[0])
	tx, ty := int(top.x), int(top.y)
	s.c.rect(tx+1, ty, 4, 3, r[2])
	s.c.rect(tx+1, ty, 4, 1, r[3])
	s.c.dot(tx+5, ty+1, r[1])
	s.c.dot(tx, ty-1, goldR[3])
}

func (s iso) tierDoorL(y1, u, w, z, h float64, r ramp) {
	s.onLeft(y1, u-0.04, u+w+0.04, z, z+h+2, flat(ink))
	s.onLeft(y1, u, u+w, z, z+h, func(x, y int) color.NRGBA {
		if x%2 == 0 {
			return r[1]
		}
		return r[2]
	})
	p := s.p(u+w*0.8, y1, z+h/2)
	s.c.dot(int(p.x), int(p.y), goldR[3])
}
