package art

import "image/color"

type side struct {
	left bool
	at   float64
}

func sideL(y1 float64) side { return side{left: true, at: y1} }

func sideR(x1 float64) side { return side{at: x1} }

func (s iso) on(f side, u0, u1, z0, z1 float64, sh shader) {
	if f.left {
		s.onLeft(f.at, u0, u1, z0, z1, sh)
		return
	}
	s.onRight(f.at, u0, u1, z0, z1, sh)
}

func (s iso) paint(f side, tag uint8, u0, u1, z0, z1 float64, col color.NRGBA) {
	s.pen(tag, func() { s.on(f, u0, u1, z0, z1, flat(col)) })
}

func emberGlass(lit bool) shader {
	return func(x, y int) color.NRGBA {
		if !lit {
			return inkSft
		}
		if (x+y)%3 == 0 {
			return fireR[2]
		}
		return fireR[1]
	}
}

func paneGlass(lit bool) shader {
	return func(x, y int) color.NRGBA {
		switch {
		case !lit && (x+y)%4 == 0:
			return glassR[2]
		case !lit:
			return glassR[1]
		case (x+2*y)%6 == 0:
			return glassR[3]
		}
		return lampLit
	}
}

func (s iso) litWindow(u, z float64) bool { return s.lk.roll(int(u*97)+int(z)*13+500) > 0.2 }

func (s iso) window(f side, u, z float64) {
	lit := s.litWindow(u, z)
	switch s.lk.age {
	case ageDark:
		s.paint(f, matTrim, u-0.02, u+0.13, z-1, z+5, woodR[0])
		s.pen(matGlass, func() { s.on(f, u+0.01, u+0.1, z, z+4, emberGlass(lit)) })
		s.paint(f, matTrim, u+0.13, u+0.18, z-1, z+5, woodR[2])
	case ageFeudal:
		s.paint(f, matTrim, u-0.07, u-0.02, z-1, z+6, s.lk.id[1])
		s.paint(f, matTrim, u+0.15, u+0.2, z-1, z+6, s.lk.id[2])
		s.paint(f, matTrim, u-0.02, u+0.15, z-1, z+6, ink)
		s.pen(matGlass, func() { s.on(f, u+0.02, u+0.11, z, z+5, paneGlass(lit)) })
	case ageCastle:
		s.paint(f, matTrim, u-0.04, u+0.16, z-2, z-1, stoneR[3])
		s.paint(f, matTrim, u-0.02, u+0.14, z-1, z+8, ink)
		s.paint(f, matTrim, u+0.02, u+0.1, z+8, z+10, ink)
		s.pen(matGlass, func() { s.on(f, u+0.02, u+0.1, z, z+8, paneGlass(lit)) })
	default:
		s.paint(f, matTrim, u-0.04, u+0.18, z-2, z+11, goldR[1])
		s.paint(f, matTrim, u-0.01, u+0.15, z-1, z+10, ink)
		s.pen(matGlass, func() { s.on(f, u+0.02, u+0.12, z, z+9, paneGlass(lit)) })
		s.paint(f, matTrim, u+0.06, u+0.08, z, z+9, goldR[2])
		s.paint(f, matTrim, u+0.05, u+0.09, z+11, z+12, goldR[3])
	}
}

func (s iso) doorway(f side, u, w, h float64) {
	h = h * s.kit().scale
	switch s.lk.age {
	case ageDark:
		s.paint(f, matTrim, u-0.03, u+w+0.03, 0, h+1, woodR[0])
		s.pen(matTrim, func() { s.on(f, u, u+w, 0, h, s.plankDoor(woodR)) })
	case ageFeudal:
		s.paint(f, matTrim, u-0.06, u+w+0.06, h+1, h+3, stoneR[3])
		s.paint(f, matTrim, u-0.03, u+w+0.03, 0, h+1, ink)
		s.pen(matTrim, func() { s.on(f, u, u+w, 0, h, s.plankDoor(woodR)) })
		s.paint(f, matTrim, u+0.02, u+0.06, h*0.3, h*0.3+1, greyR[1])
		s.paint(f, matTrim, u+0.02, u+0.06, h*0.75, h*0.75+1, greyR[1])
	case ageCastle:
		s.paint(f, matTrim, u-0.05, u+w+0.05, 0, h+2, stoneR[3])
		s.paint(f, matTrim, u-0.02, u+w+0.02, 0, h+1, ink)
		s.paint(f, matTrim, u+w*0.2, u+w*0.8, h+1, h+3, ink)
		s.pen(matTrim, func() { s.on(f, u, u+w, 0, h, s.plankDoor(woodR)) })
	default:
		s.paint(f, matTrim, u-0.06, u+w+0.06, 0, h+3, goldR[1])
		s.paint(f, matTrim, u-0.03, u+w+0.03, 0, h+2, ink)
		s.pen(matTrim, func() { s.on(f, u, u+w, 0, h, s.plankDoor(s.lk.id)) })
		s.paint(f, matTrim, u+w/2-0.01, u+w/2+0.01, 0, h, goldR[2])
	}
	s.knob(f, u+w*0.8, h/2)
}

func (s iso) plankDoor(r ramp) shader {
	return func(x, y int) color.NRGBA {
		if x%2 == 0 {
			return r[0]
		}
		return r[1]
	}
}

func (s iso) knob(f side, u, z float64) {
	var p pt
	if f.left {
		p = s.p(u, f.at, z)
	} else {
		p = s.p(f.at, u, z)
	}
	s.c.dot(int(p.x), int(p.y), goldR[2])
}

func (s iso) chimney(gx, gy, zBase, zTop float64) {
	k := s.kit()
	w := 0.1
	body := k.base
	if s.lk.age >= ageCastle {
		body = k.wall
	}
	s.block(gx, gy, gx+w, gy+w, zBase, zTop, body, flatMat(ink), matStone)
	switch s.lk.age {
	case ageCastle:
		s.block(gx-0.02, gy-0.02, gx+w+0.02, gy+w+0.02, zTop, zTop+2, flatMat(stoneR[3]), flatMat(ink), matStone)
	case ageEmpire:
		s.block(gx-0.02, gy-0.02, gx+w+0.02, gy+w+0.02, zTop, zTop+2, flatMat(goldR[2]), flatMat(ink), matTrim)
	}
}

func (s iso) dormer(gx, yd, zd, w, h float64) {
	k := s.kit()
	x0, x1, y0 := gx-w/2, gx+w/2, yd-0.3
	cheek := toneMat(mixRamp(s.roofRamp(), ramp{ink, inkSft, s.roofRamp()[0], s.roofRamp()[1]}, 0.3))
	s.pen(k.wallTag, func() {
		s.box(x0, y0, x1, yd, zd, zd+h, faces{top: s.topMat(zd+h, cheek), left: s.leftMat(yd, k.wall), right: s.rightMat(x1, cheek)})
	})
	s.pen(matGlass, func() { s.onLeft(yd, gx-0.05, gx+0.05, zd+1, zd+h-1, paneGlass(s.litWindow(gx, zd))) })
	s.roofGableY(roofBox{x0, y0, x1, yd, zd + h, w * 20, 0.03}, s.roofMat(), k.wall)
}
