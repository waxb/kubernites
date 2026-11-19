package art

const (
	litBack  = -0.3
	litFront = 0.55
	litEnd   = -0.5
)

type roofBox struct {
	x0, y0, x1, y1, z, rise, eave float64
}

func (s iso) roofGableX(b roofBox, m, end mat) {
	ym := (b.y0 + b.y1) / 2
	a0, a1 := b.x0-b.eave, b.x1+b.eave
	s.pen(matRoof, func() {
		s.quad(s.p(a0, b.y0-b.eave, b.z), s.p(a1, b.y0-b.eave, b.z), s.p(a1, ym, b.z+b.rise), s.p(a0, ym, b.z+b.rise), s.planeX(b.y0-b.eave, ym-b.y0+b.eave, b.z, b.rise, litBack, m))
	})
	s.pen(matWall, func() {
		s.c.shadePoly([]pt{s.p(b.x1, b.y0, b.z), s.p(b.x1, ym, b.z+b.rise), s.p(b.x1, b.y1, b.z)}, s.rightMat(b.x1, end))
	})
	s.pen(matRoof, func() {
		s.quad(s.p(a0, ym, b.z+b.rise), s.p(a1, ym, b.z+b.rise), s.p(a1, b.y1+b.eave, b.z), s.p(a0, b.y1+b.eave, b.z), s.planeX(b.y1+b.eave, ym-b.y1-b.eave, b.z, b.rise, litFront, m))
	})
}

func (s iso) roofGableY(b roofBox, m, end mat) {
	xm := (b.x0 + b.x1) / 2
	c0, c1 := b.y0-b.eave, b.y1+b.eave
	s.pen(matRoof, func() {
		s.quad(s.p(b.x0-b.eave, c0, b.z), s.p(xm, c0, b.z+b.rise), s.p(xm, c1, b.z+b.rise), s.p(b.x0-b.eave, c1, b.z), s.planeY(b.x0-b.eave, xm-b.x0+b.eave, b.z, b.rise, litFront+0.1, m))
	})
	s.pen(matWall, func() {
		s.c.shadePoly([]pt{s.p(b.x0, b.y1, b.z), s.p(xm, b.y1, b.z+b.rise), s.p(b.x1, b.y1, b.z)}, s.leftMat(b.y1, end))
	})
	s.pen(matRoof, func() {
		s.quad(s.p(xm, c0, b.z+b.rise), s.p(b.x1+b.eave, c0, b.z), s.p(b.x1+b.eave, c1, b.z), s.p(xm, c1, b.z+b.rise), s.planeY(b.x1+b.eave, xm-b.x1-b.eave, b.z, b.rise, litEnd, m))
	})
}

func (s iso) roofHip(b roofBox, m mat) {
	x0, y0, x1, y1 := b.x0-b.eave, b.y0-b.eave, b.x1+b.eave, b.y1+b.eave
	xm, ym, top := (x0+x1)/2, (y0+y1)/2, b.z+b.rise
	rx0, rx1, ry0, ry1 := xm, xm, ym, ym
	if x1-x0 >= y1-y0 {
		rx0, rx1 = x0+(y1-y0)/2, x1-(y1-y0)/2
	} else {
		ry0, ry1 = y0+(x1-x0)/2, y1-(x1-x0)/2
	}
	s.pen(matRoof, func() {
		s.quad(s.p(x0, y0, b.z), s.p(x1, y0, b.z), s.p(rx1, ry0, top), s.p(rx0, ry0, top), s.planeX(y0, ry0-y0, b.z, b.rise, litBack, m))
		s.quad(s.p(x0, y0, b.z), s.p(rx0, ry0, top), s.p(rx0, ry1, top), s.p(x0, y1, b.z), s.planeY(x0, rx0-x0, b.z, b.rise, litFront, m))
		s.quad(s.p(x1, y0, b.z), s.p(x1, y1, b.z), s.p(rx1, ry1, top), s.p(rx1, ry0, top), s.planeY(x1, rx1-x1, b.z, b.rise, litEnd, m))
		s.quad(s.p(rx0, ry1, top), s.p(rx1, ry1, top), s.p(x1, y1, b.z), s.p(x0, y1, b.z), s.planeX(y1, ry1-y1, b.z, b.rise, litFront, m))
	})
}

func (s iso) roofPyramid(b roofBox, m mat) {
	x0, y0, x1, y1 := b.x0-b.eave, b.y0-b.eave, b.x1+b.eave, b.y1+b.eave
	xm, ym := (x0+x1)/2, (y0+y1)/2
	apex := s.p(xm, ym, b.z+b.rise)
	s.pen(matRoof, func() {
		s.c.shadePoly([]pt{s.p(x0, y0, b.z), s.p(x1, y0, b.z), apex}, s.planeX(y0, ym-y0, b.z, b.rise, litBack, m))
		s.c.shadePoly([]pt{s.p(x0, y0, b.z), apex, s.p(x0, y1, b.z)}, s.planeY(x0, xm-x0, b.z, b.rise, litFront, m))
		s.c.shadePoly([]pt{s.p(x1, y0, b.z), s.p(x1, y1, b.z), apex}, s.planeY(x1, xm-x1, b.z, b.rise, litEnd, m))
		s.c.shadePoly([]pt{s.p(x0, y1, b.z), s.p(x1, y1, b.z), apex}, s.planeX(y1, ym-y1, b.z, b.rise, litFront, m))
	})
}

func (s iso) roofShed(x0, y0, x1, y1, zLow, zHigh float64, alongX bool, m mat) {
	s.pen(matRoof, func() {
		if alongX {
			s.quad(s.p(x0, y0, zHigh), s.p(x1, y0, zHigh), s.p(x1, y1, zLow), s.p(x0, y1, zLow), s.planeX(y1, y0-y1, zLow, zHigh-zLow, litFront, m))
			return
		}
		s.quad(s.p(x0, y0, zHigh), s.p(x1, y0, zLow), s.p(x1, y1, zLow), s.p(x0, y1, zHigh), s.planeY(x1, x0-x1, zLow, zHigh-zLow, litEnd, m))
	})
}

func (s iso) roofCone(cx, cy, r, z, h float64, shape int, m mat) {
	s.turn(lathe{cx: cx, cy: cy, z: z, h: h, r: r, shape: shape, profile: coneProfile}, m, matRoof)
}

func (s iso) roofDome(cx, cy, r, z, h float64, m mat) {
	s.turn(lathe{cx: cx, cy: cy, z: z, h: h, r: r, profile: domeProfile}, m, matRoof)
}

func (s iso) roofBell(cx, cy, r, z, h float64, shape int, m mat) {
	s.turn(lathe{cx: cx, cy: cy, z: z, h: h, r: r, shape: shape, profile: bellProfile}, m, matRoof)
}

func (s iso) roofOnion(cx, cy, r, z, h float64, m mat) {
	s.turn(lathe{cx: cx, cy: cy, z: z, h: h, r: r, profile: onionProfile}, m, matRoof)
}

func (s iso) finial(gx, gy, z, h float64) {
	a, b := s.p(gx, gy, z), s.p(gx, gy, z+h)
	s.pen(matTrim, func() {
		s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), goldR[1])
		s.c.dot(int(b.x), int(b.y)-1, goldR[3])
		s.c.dot(int(b.x), int(b.y), goldR[2])
	})
}
