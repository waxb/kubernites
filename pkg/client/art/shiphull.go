package art

import (
	"image/color"
	"math"
)

func (s shipScene) hull() {
	pts := s.sp.plan(-1, 1, 14)
	s.v.deckTop(pts, s.sp.railU, s.sp.bulwark, s.lk.wood[1], s.lk.deck, s.lk.wood)
	s.crew()
	base := func(float64) float64 { return -s.v.lift }
	s.v.walls(prism{pts: pts, low: s.sp.waterPlan(14), base: base, top: s.sp.railU, skin: s.hullSkin})
	s.shields()
	s.gunports()
}

func (s shipScene) hullSkin(idx int, _ bool, rail func(float64) float64) func(float64, float64) color.NRGBA {
	w, trim, lift, stripe := s.lk.wood, s.lk.trim, s.v.lift, s.sp.stripe
	return func(sv, z float64) color.NRGBA {
		d := rail(sv) - z
		switch {
		case d < 1:
			return w[min(idx+1, 3)]
		case d >= 2 && d < 2+stripe:
			return trim[idx]
		case z+lift < 1:
			return darken(w[max(idx-1, 0)], 0.15)
		case int(math.Floor(z))%3 == 0:
			return w[max(idx-1, 0)]
		}
		return w[idx]
	}
}

func (s shipScene) castleSkin(idx int, cross bool, rail func(float64) float64) func(float64, float64) color.NRGBA {
	w, trim := s.lk.wood, s.lk.trim
	return func(sv, z float64) color.NRGBA {
		d := rail(sv) - z
		switch {
		case d < 1:
			return w[min(idx+1, 3)]
		case d >= 2 && d < 3:
			return trim[idx]
		case cross && d >= 4 && d < 6 && int(sv*9)%3 == 1:
			return lampLit
		case cross && d >= 3.5 && d < 6.5 && int(sv*9)%3 == 1:
			return inkSft
		case int(math.Floor(z))%3 == 0:
			return w[max(idx-1, 0)]
		}
		return w[idx]
	}
}

func (s shipScene) castle(cs castleSpec) {
	pts := s.sp.plan(cs.from, cs.to, 6)
	top := func(u float64) float64 { return s.sp.railU(u) + cs.height }
	s.v.deckTop(pts, top, 1.5, s.lk.wood[1], s.lk.deck, s.lk.wood)
	s.v.walls(prism{pts: pts, base: s.sp.railU, top: top, crossDrop: s.sp.bulwark, skin: s.castleSkin})
}

func (s shipScene) castleTop(t float64) float64 {
	top := s.sp.rail(t)
	for _, cs := range s.sp.castles {
		if t >= cs.from-1e-9 && t <= cs.to+1e-9 {
			top = math.Max(top, s.sp.rail(t)+cs.height)
		}
	}
	return top
}

func (s shipScene) nearSign() float64 {
	if s.v.toward(vec3{w: 1}) > 0 {
		return 1
	}
	return -1
}

func (s shipScene) crew() {
	spots := [][2]float64{{-0.3, 0.4}, {0.28, -0.35}, {0.02, 0.1}}
	for i := 0; i < s.lk.crew && i < len(spots); i++ {
		t := spots[i][0]
		p := s.v.p(t*s.sp.half(), s.sp.halfBeam(t)*spots[i][1], s.sp.rail(t)-s.sp.bulwark)
		x, y := int(math.Floor(p.x)), int(math.Floor(p.y))
		tunic := crewTunics[(s.lk.seed+i)%len(crewTunics)]
		s.v.c.dot(x, y-1, tunic)
		s.v.c.dot(x, y-2, tunic)
		s.v.c.dot(x, y-3, skinR[2])
	}
}

func (s shipScene) side(t, z float64) pt {
	hb, u, zb := s.sp.halfBeam(t), t*s.sp.half(), -s.v.lift
	f := clamp01((z - zb) / (s.sp.rail(t) - zb))
	w := hb * (hullFlare + (1-hullFlare)*f)
	return s.v.p(u*(rake(u)+(1-rake(u))*f), s.nearSign()*w, z)
}

func (s shipScene) shields() {
	if !s.lk.shields {
		return
	}
	for i, t := range []float64{-0.55, -0.25, 0.05, 0.35} {
		p := s.side(t, s.sp.rail(t)-1)
		x, y := int(math.Floor(p.x)), int(math.Floor(p.y))
		col := s.lk.trim[2]
		if i%2 == 1 {
			col = s.lk.trim[1]
		}
		s.v.c.rect(x, y, 2, 2, col)
		s.v.c.dot(x+1, y+1, darken(col, 0.3))
	}
}

func (s shipScene) gunports() {
	if !s.sp.gunports {
		return
	}
	for t := -0.62; t <= 0.5; t += 0.2 {
		p := s.side(t, s.sp.rail(t)-2-s.sp.stripe-1.5)
		x, y := int(math.Floor(p.x)), int(math.Floor(p.y))
		s.v.c.rect(x, y, 2, 2, ink)
		s.v.c.dot(x, y-1, s.lk.trim[1])
		s.v.c.dot(x+1, y-1, s.lk.trim[1])
	}
}

func (s shipScene) lantern() {
	if !s.lk.lantern {
		return
	}
	t := -0.96
	p := s.v.p(t*s.sp.half(), 0, s.castleTop(t))
	x, y := int(math.Floor(p.x)), int(math.Floor(p.y))
	s.v.c.dot(x, y-1, s.lk.wood[0])
	s.v.c.rect(x, y-3, 1, 2, lampLit)
}

func (s shipScene) waterline() {
	pts, low := s.sp.plan(-1, 1, 14), s.sp.waterPlan(14)
	for i := range pts {
		j := (i + 1) % len(pts)
		n, ok := outward(pts, pts[i], pts[j])
		if !ok || s.v.toward(n) <= 1e-6 {
			continue
		}
		pa, pb := s.v.p(low[i].u, low[i].w, -s.v.lift), s.v.p(low[j].u, low[j].w, -s.v.lift)
		if pa.x > pb.x {
			pa, pb = pb, pa
		}
		for x := int(math.Floor(pa.x)); x <= int(math.Floor(pb.x)); x++ {
			f := (float64(x) + 0.5 - pa.x) / math.Max(pb.x-pa.x, 1e-9)
			s.foam(x, int(math.Floor(pa.y+math.Min(math.Max(f, 0), 1)*(pb.y-pa.y))))
		}
	}
}

func (s shipScene) foam(x, y int) {
	low := -1
	for yy := y - 2; yy <= y+2; yy++ {
		if s.v.c.at(x, yy).A > 0 {
			low = yy
		}
	}
	if low < 0 || s.v.c.at(x, low) != ink {
		return
	}
	s.v.c.img.SetNRGBA(x, low, withAlpha(waterR[3], 215))
	if (x+s.frame*3)%5 != 0 {
		s.v.c.dot(x, low+1, withAlpha(waterR[3], 70))
	}
}
