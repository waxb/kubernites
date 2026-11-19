package art

import "image/color"

type kit struct {
	wall, base mat
	baseH      float64
	baseTag    uint8
	wallTag    uint8
	beams      bool
	posts      bool
	quoins     bool
	gilt       bool
	eave       float64
	scale      float64
	rise       float64
}

func (s iso) kit() kit {
	sd := s.lk.seed
	switch s.lk.age {
	case ageDark:
		return kit{wall: wattleMat(sd), base: logMat(sd), baseH: 4, baseTag: matWall, wallTag: matWall, posts: true, eave: 0.16, scale: 0.8, rise: 1.25}
	case ageFeudal:
		return kit{wall: plasterMat(sd), base: rubbleMat(stoneR, sd), baseH: 0.42, baseTag: matStone, wallTag: matWall, beams: true, eave: 0.12, scale: 0.92, rise: 1.05}
	case ageCastle:
		return kit{wall: ashlarMat(stoneR, sd), base: rubbleMat(stoneR, sd+1), baseH: 5, baseTag: matStone, wallTag: matStone, quoins: true, eave: 0.1, scale: 1, rise: 1}
	}
	return kit{wall: marbleMat(sd), base: ashlarMat(stoneR, sd), baseH: 4, baseTag: matStone, wallTag: matStone, gilt: true, eave: 0.1, scale: 1, rise: 0.95}
}

func mixRamp(a, b ramp, t float64) ramp {
	out := ramp{}
	for i := range a {
		out[i] = mix(a[i], b[i], t)
	}
	return out
}

func (s iso) roofRamp() ramp {
	switch s.lk.age {
	case ageDark:
		return mixRamp(strawR, s.lk.id, 0.2)
	case ageCastle:
		return mixRamp(slateR, s.lk.id, 0.68)
	}
	return s.lk.id
}

func (s iso) roofMat() mat {
	r, sd := s.roofRamp(), s.lk.seed+17
	switch s.lk.age {
	case ageDark:
		return thatchMat(r, sd)
	case ageFeudal:
		if s.lk.variant == 1 {
			return tileMat(r, sd)
		}
		return shingleMat(r, sd)
	case ageCastle:
		return slateMat(r, sd)
	}
	return tileMat(r, sd)
}

func (s iso) domeMat() mat {
	if s.lk.variant == 2 {
		return ribbedMat(goldR, s.lk.seed)
	}
	return ribbedMat(copperR, s.lk.seed)
}

func (s iso) baseTop(k kit, z1 float64) float64 {
	if k.baseH < 1 {
		return z1 * k.baseH
	}
	return min(k.baseH, z1)
}

func (s iso) walls(x0, y0, x1, y1, z0, z1 float64) {
	k := s.kit()
	zb := z0 + s.baseTop(k, z1-z0)
	s.block(x0, y0, x1, y1, z0, zb, k.base, k.base, k.baseTag)
	s.block(x0, y0, x1, y1, zb, z1, k.wall, k.wall, k.wallTag)
	switch {
	case k.posts:
		s.cornerPosts(x0, y0, x1, y1, z0, z1)
	case k.beams:
		s.timberFrame(x0, y0, x1, y1, zb, z1)
	case k.quoins:
		s.quoinsAt(x0, y0, x1, y1, z0, z1)
	case k.gilt:
		s.cornice(x0, y0, x1, y1, z1)
	}
}

func (s iso) edge(a, b pt, col color.NRGBA) {
	s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), col)
}

func (s iso) cornerPosts(x0, y0, x1, y1, z0, z1 float64) {
	s.pen(matTrim, func() {
		for i, c := range [][2]float64{{x0, y1}, {x1, y1}, {x1, y0}} {
			lean := s.lk.jit(40+i, 0.6)
			a, b := s.p(c[0], c[1], z0), s.p(c[0], c[1], z1+1)
			s.edge(a, pt{b.x + lean, b.y}, woodR[1+i%2])
		}
		s.edge(s.p(x0, y1, z1), s.p(x1, y1, z1), woodR[0])
		s.edge(s.p(x1, y1, z1), s.p(x1, y0, z1), woodR[0])
	})
}

func (s iso) timberFrame(x0, y0, x1, y1, z0, z1 float64) {
	s.pen(matTrim, func() {
		s.edge(s.p(x0, y1, z0), s.p(x1, y1, z0), woodR[0])
		s.edge(s.p(x1, y1, z0), s.p(x1, y0, z0), woodR[0])
		s.edge(s.p(x0, y1, z1-1), s.p(x1, y1, z1-1), woodR[1])
		s.edge(s.p(x1, y1, z1-1), s.p(x1, y0, z1-1), woodR[0])
		step := 0.34 + s.lk.jit(44, 0.05)
		for u := x0 + 0.02; u < x1; u += step {
			s.edge(s.p(u, y1, z0), s.p(u, y1, z1), woodR[0])
		}
		for v := y0 + 0.02; v < y1; v += step {
			s.edge(s.p(x1, v, z0), s.p(x1, v, z1), woodR[0])
		}
		if s.lk.chance(45, 0.7) && x1-x0 > 0.5 {
			s.edge(s.p(x0+0.04, y1, z0), s.p(x0+step, y1, z1-1), woodR[1])
		}
	})
}

func (s iso) quoinsAt(x0, y0, x1, y1, z0, z1 float64) {
	light := stoneR[3]
	s.pen(matStone, func() {
		for z := z0 + 1; z < z1-1; z += 4 {
			w := 0.08 + 0.05*float64(int(z/4)%2)
			for _, c := range [][3]float64{{x0, y1, 1}, {x1, y1, -1}} {
				s.onLeft(y1, c[0]+min(c[2], 0)*w, c[0]+max(c[2], 0)*w, z, z+3, flat(light))
			}
			s.onRight(x1, y1-w, y1, z, z+3, flat(stoneR[2]))
		}
	})
}

func (s iso) cornice(x0, y0, x1, y1, z1 float64) {
	s.pen(matTrim, func() {
		s.onLeft(y1, x0, x1, z1-2, z1, flat(goldR[2]))
		s.onRight(x1, y0, y1, z1-2, z1, flat(goldR[1]))
		s.onLeft(y1, x0, x1, z1-3, z1-2, flat(marbleR[3]))
	})
}
