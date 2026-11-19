package art

import "image/color"

type pierSpec struct {
	near, far, zLow float64
	solid           bool
}

var pierSpecs = [PortLevels]pierSpec{
	{near: 0.32, far: 0.68, zLow: HavenDeck - 1.2},
	{near: 0.2, far: 0.8, zLow: HavenDeck - havenBeamDepth},
	{near: 0.15, far: 0.85, zLow: -1, solid: true},
	{near: 0.08, far: 0.92, zLow: -1, solid: true},
}

func pierSpecOf(level int) pierSpec { return pierSpecs[ClampPortLevel(level)-1] }

func paintPortPier(c *canvas, l PortLook, alongX bool) {
	s, m, sp := tileIso(c), pierAxis(alongX), pierSpecOf(l.Level)
	x0, y0 := m.g(0, sp.near)
	x1, y1 := m.g(1, sp.far)
	top := func(x, y int) color.NRGBA {
		gx, gy := s.ground(x, y, HavenDeck)
		return pierTop(l, m, sp, gx, gy)
	}
	s.box(x0, y0, x1, y1, sp.zLow, HavenDeck, faces{top: top, left: pierFace(l, true), right: pierFace(l, false)})
	if !sp.solid {
		portPierPosts(s, m, l, sp)
	}
	portPierExtras(s, m, l, sp)
	c.outline(ink)
	trimFarJoin(c, s, func(gx, gy float64) bool { return m.along(gx, gy) < 0 })
	portPierRipples(s, m, sp)
}

func pierTop(l PortLook, m pierAxis, sp pierSpec, gx, gy float64) color.NRGBA {
	a := m.along(gx, gy)
	across := m.along(gy, gx)
	if l.stone() {
		if l.Level >= PortLevels && (across < sp.near+0.08 || across > sp.far-0.08) {
			return portMatOf(l.Age).stone[1]
		}
		return flagTop(PortLook{Kind: l.Kind, Level: portStone, Age: l.Age, Variant: l.Variant}, a, across)
	}
	w := portMatOf(l.Age).wood
	if !alongGap(l, a) {
		return plankShade(w, a, 6, l.seed()+boolInt(bool(m)))
	}
	if l.Level == 1 {
		return deepR[1]
	}
	return w[0]
}

func alongGap(l PortLook, a float64) bool {
	switch {
	case l.Variant == 1 && a > 0.52 && a < 0.64:
		return true
	case l.Level == 1:
		f := a*6 - float64(int(a*6))
		return f < 0.3
	}
	return false
}

func pierFace(l PortLook, lit bool) shader {
	if l.stone() {
		r := portMatOf(l.Age).stone
		return func(x, y int) color.NRGBA { return stoneFace(r, x, y, lit) }
	}
	return beamShader(portMatOf(l.Age).wood, lit)
}

func portPierPosts(s iso, m pierAxis, l PortLook, sp pierSpec) {
	w := portMatOf(l.Age).wood
	half := 0.05
	if l.Level == 1 {
		half = 0.035
	}
	for _, a := range []float64{0.15, 0.85} {
		gx, gy := m.g(a, sp.far)
		post := faces{top: flat(w[2]), left: flat(w[1]), right: flat(w[0])}
		s.box(gx-half, gy-half, gx+half, gy+half, -1, sp.zLow, post)
	}
}

func portPierExtras(s iso, m pierAxis, l PortLook, sp pierSpec) {
	if l.stone() {
		stonePierExtras(s, m, l, sp)
		return
	}
	if l.Variant == 0 {
		return
	}
	w := portMatOf(l.Age).wood
	gx, gy := m.g(0.85, sp.far-0.08)
	s.box(gx-0.06, gy-0.06, gx+0.06, gy+0.06, HavenDeck, HavenDeck+5, faces{top: flat(w[3]), left: flat(w[2]), right: flat(w[1])})
	if l.Variant == 2 {
		rx, ry := m.g(0.4, (sp.near+sp.far)/2-0.05)
		p := s.p(rx, ry, HavenDeck)
		s.c.ellipse(p.x, p.y-0.5, 3, 1.6, ropeR[1])
		s.c.ellipse(p.x, p.y-0.8, 2, 1, ropeR[3])
		s.c.ellipse(p.x, p.y-0.8, 1, 0.5, ropeR[0])
	}
}

func stonePierExtras(s iso, m pierAxis, l PortLook, sp pierSpec) {
	mt := portMatOf(l.Age).metal
	switch l.Variant {
	case 1:
		gx, gy := m.g(0.5, sp.far)
		p := s.p(gx, gy, HavenDeck-2)
		s.c.ellipse(p.x, p.y, 1.5, 1.5, mt[1])
		s.c.dot(int(p.x), int(p.y), clear)
	case 2:
		gx, gy := m.g(0.5, sp.far-0.1)
		portLampPost(s, gx, gy, mt, l.Level >= PortLevels)
	}
	if l.Level >= PortLevels {
		fx, fy := m.g(0.2, sp.far)
		w := portMatOf(l.Age).wood
		s.box(fx-0.04, fy-0.03, fx+0.04, fy+0.03, -1, HavenDeck+1, faces{top: flat(w[2]), left: flat(w[1]), right: flat(w[0])})
	}
}

func portLampPost(s iso, gx, gy float64, mt ramp, tall bool) {
	h := 9.0
	if !tall {
		s.box(gx-0.05, gy-0.05, gx+0.05, gy+0.05, HavenDeck, HavenDeck+3, faces{top: flat(mt[2]), left: flat(mt[1]), right: flat(mt[0])})
		return
	}
	base, top := s.p(gx, gy, HavenDeck), s.p(gx, gy, HavenDeck+h)
	s.c.rect(int(base.x), int(top.y), 1, int(base.y-top.y), mt[1])
	s.c.rect(int(top.x)-1, int(top.y)-3, 3, 3, lampLit)
	s.c.rect(int(top.x)-1, int(top.y)-4, 3, 1, mt[0])
}

func portPierRipples(s iso, m pierAxis, sp pierSpec) {
	if sp.solid {
		for _, a := range []float64{0.1, 0.5, 0.9} {
			gx, gy := m.g(a, sp.far+0.04)
			p := s.p(gx, gy, -1)
			s.c.rect(int(p.x)-2, int(p.y), 5, 1, withAlpha(waterR[3], 150))
		}
		return
	}
	for _, a := range []float64{0.15, 0.85} {
		gx, gy := m.g(a, sp.far+0.05)
		p := s.p(gx, gy, 0)
		s.c.rect(int(p.x)-2, int(p.y), 4, 1, withAlpha(waterR[3], 150))
	}
}
