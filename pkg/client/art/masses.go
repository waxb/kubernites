package art

type hallSpec struct {
	x0, y0, x1, y1 float64
	h, rise        float64
	alongY, hip    bool
}

func (s iso) hall(h hallSpec) float64 {
	k := s.kit()
	top := h.h * k.scale
	s.walls(h.x0, h.y0, h.x1, h.y1, 0, top)
	s.roofOver(roofBox{h.x0, h.y0, h.x1, h.y1, top, h.rise * k.rise, k.eave}, h.alongY, h.hip)
	return top
}

func (s iso) roofOver(b roofBox, alongY, hip bool) {
	m, end := s.roofMat(), s.kit().wall
	hip = hip || (s.lk.age == ageEmpire && s.lk.variant != 1)
	switch {
	case hip:
		s.roofHip(b, m)
	case alongY:
		s.roofGableY(b, m, end)
	default:
		s.roofGableX(b, m, end)
	}
	if s.lk.age == ageDark {
		s.thatchLip(b, alongY)
	}
	if s.lk.age == ageEmpire {
		s.giltRidge(b, alongY, hip)
	}
}

func (s iso) thatchLip(b roofBox, alongY bool) {
	r := s.roofRamp()
	s.pen(matRoof, func() {
		if alongY {
			s.quad(s.p(b.x1+b.eave, b.y0-b.eave, b.z), s.p(b.x1+b.eave, b.y1+b.eave, b.z), s.p(b.x1+b.eave, b.y1+b.eave, b.z-2), s.p(b.x1+b.eave, b.y0-b.eave, b.z-2), flat(r[0]))
			return
		}
		s.quad(s.p(b.x0-b.eave, b.y1+b.eave, b.z), s.p(b.x1+b.eave, b.y1+b.eave, b.z), s.p(b.x1+b.eave, b.y1+b.eave, b.z-2), s.p(b.x0-b.eave, b.y1+b.eave, b.z-2), flat(r[1]))
	})
}

func (s iso) giltRidge(b roofBox, alongY, hip bool) {
	top := b.z + b.rise
	x0, y0, x1, y1 := b.x0, b.y0, b.x1, b.y1
	xm, ym := (x0+x1)/2, (y0+y1)/2
	if hip {
		w, d := x1-x0+2*b.eave, y1-y0+2*b.eave
		alongY = d > w
		x0, x1 = xm-max(w-d, 0)/2, xm+max(w-d, 0)/2
		y0, y1 = ym-max(d-w, 0)/2, ym+max(d-w, 0)/2
	}
	s.pen(matTrim, func() {
		if alongY {
			s.edge(s.p(xm, y0, top), s.p(xm, y1, top), goldR[2])
			return
		}
		s.edge(s.p(x0, ym, top), s.p(x1, ym, top), goldR[2])
	})
	s.finial(xm, ym, top, 4)
}

type towerSpec struct {
	cx, cy, r, z0, h, roofH float64
	crown, round            bool
}

func (s iso) shapeOf(t towerSpec) int {
	if t.round {
		return shapeRound
	}
	return s.towerShape()
}

func (s iso) towerShape() int {
	switch s.lk.age {
	case ageDark, ageFeudal:
		return shapeSquare
	case ageCastle:
		return shapeRound
	}
	if s.lk.variant == 1 {
		return shapeOct
	}
	return shapeRound
}

func (s iso) towerWall() mat {
	k := s.kit()
	if s.lk.age == ageDark {
		return logMat(s.lk.seed + 5)
	}
	if s.lk.age == ageFeudal {
		return k.base
	}
	return k.wall
}

func (s iso) tower(t towerSpec) float64 {
	shape := s.shapeOf(t)
	top := t.z0 + t.h*s.kit().scale
	s.turn(lathe{cx: t.cx, cy: t.cy, z: t.z0, h: top - t.z0, r: t.r, shape: shape}, s.towerWall(), matStone)
	if s.lk.age >= ageCastle {
		s.turn(lathe{cx: t.cx, cy: t.cy, z: top - 3, h: 3, r: t.r + 0.04, shape: shape}, s.kit().wall, matStone)
	}
	if t.crown {
		s.merlons(t.cx, t.cy, t.r+0.04, top, shape)
		return top
	}
	s.towerRoof(t.cx, t.cy, t.r+0.06, top, t.roofH, shape)
	return top
}

func (s iso) towerRoof(cx, cy, r, z, h float64, shape int) {
	switch s.lk.age {
	case ageEmpire:
		s.roofDome(cx, cy, r, z, h*0.8, s.domeMat())
		s.finial(cx, cy, z+h*0.8, 6)
	case ageDark:
		s.roofCone(cx, cy, r+0.05, z, h*1.05, shapeSquare, s.roofMat())
	default:
		s.roofCone(cx, cy, r, z, h, shape, s.roofMat())
	}
}

func (s iso) merlons(cx, cy, r, z float64, shape int) {
	s.turn(lathe{cx: cx, cy: cy, z: z - 1, h: 1, r: r, shape: shape}, flatMat(stoneR[3]), matStone)
	for _, i := range []int{5, 4, 6, 3, 7, 0, 2, 1} {
		if s.lk.chance(60+i, 0.12) {
			continue
		}
		a, b := unitDir(i)
		gx, gy := cx+a*r*0.85, cy+b*r*0.85
		s.block(gx-0.05, gy-0.05, gx+0.05, gy+0.05, z, z+3+float64(s.lk.pick(70+i, 2)), s.kit().wall, flatMat(stoneR[3]), matStone)
	}
}

func unitDir(i int) (float64, float64) {
	dirs := [8][2]float64{{1, 0}, {0.7, 0.7}, {0, 1}, {-0.7, 0.7}, {-1, 0}, {-0.7, -0.7}, {0, -1}, {0.7, -0.7}}
	return dirs[i%8][0], dirs[i%8][1]
}

func (s iso) turret(cx, cy, r, z0, h float64) {
	if s.lk.age < ageCastle {
		s.tower(towerSpec{cx: cx, cy: cy, r: r, z0: z0, h: h, roofH: r * 40})
		return
	}
	top := z0 + h
	s.turn(lathe{cx: cx, cy: cy, z: z0 - 3, h: 3, r: r, profile: func(t float64) float64 { return 0.5 + 0.5*t }}, s.kit().wall, matStone)
	s.turn(lathe{cx: cx, cy: cy, z: z0, h: h, r: r}, s.kit().wall, matStone)
	s.towerRoof(cx, cy, r+0.04, top, r*50, shapeRound)
}

func (s iso) apse(x1, y0, y1, h, roofH float64) {
	r, cy := (y1-y0)/2, (y0+y1)/2
	front := func(a, b float64) bool { return a >= -0.01 }
	top := h * s.kit().scale
	s.turn(lathe{cx: x1, cy: cy, h: top, r: r, clip: front}, s.kit().wall, s.kit().wallTag)
	s.turn(lathe{cx: x1, cy: cy, z: top, h: roofH, r: r + s.kit().eave*0.6, profile: coneProfile, clip: front}, s.roofMat(), matRoof)
}
