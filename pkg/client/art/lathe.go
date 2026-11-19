package art

import "math"

const (
	shapeRound = iota
	shapeOct
	shapeSquare
)

type lathe struct {
	cx, cy, z, h, r float64
	shape           int
	profile         func(t float64) float64
	clip            func(a, b float64) bool
}

func straight(float64) float64 { return 1 }

func coneProfile(t float64) float64 { return 1 - t }

func domeProfile(t float64) float64 { return math.Sqrt(max(1-t*t, 0)) }

func bellProfile(t float64) float64 {
	return max(1-t, 0) * (1 + 0.45*math.Sin(math.Pi*min(t*1.4, 1))) * (1 - 0.2*t)
}

func onionProfile(t float64) float64 {
	return max(0, math.Sin(math.Pi*(0.18+0.82*t))*(1.25-t)*0.95)
}

func flareProfile(t float64) float64 { return 1 - math.Sqrt(t) }

func (l lathe) radiusAt(t float64) float64 {
	p := l.profile
	if p == nil {
		p = straight
	}
	return l.r * p(t)
}

func shapeHas(shape int, a, b, r float64) bool {
	switch shape {
	case shapeSquare:
		return math.Max(math.Abs(a), math.Abs(b)) <= r
	case shapeOct:
		return math.Max(math.Max(math.Abs(a), math.Abs(b)), (math.Abs(a)+math.Abs(b))/math.Sqrt2*1.08) <= r
	}
	return a*a+b*b <= r*r
}

func shapeNormal(shape int, a, b float64) (float64, float64) {
	switch shape {
	case shapeSquare:
		if math.Abs(a) > math.Abs(b) {
			return math.Copysign(1, a), 0
		}
		return 0, math.Copysign(1, b)
	case shapeOct:
		ang := math.Round(math.Atan2(b, a)/(math.Pi/4)) * math.Pi / 4
		return math.Cos(ang), math.Sin(ang)
	}
	d := math.Max(math.Hypot(a, b), 1e-6)
	return a / d, b / d
}

func (s iso) turn(l lathe, m mat, tag uint8) {
	steps := int(math.Ceil(l.h+l.r*8)) + 1
	base := s.p(l.cx, l.cy, 0)
	s.pen(tag, func() {
		for i := 0; i <= steps; i++ {
			t := float64(i) / float64(steps)
			s.ring(l, t, base, m)
		}
	})
}

func (s iso) ring(l lathe, t float64, base pt, m mat) {
	r := l.radiusAt(t)
	if r <= 0.001 && t < 1 {
		return
	}
	z := l.z + l.h*t
	ey := base.y - z
	hw, hh := r*32+1, r*16+1
	for y := int(math.Floor(ey - hh)); y <= int(math.Ceil(ey+hh)); y++ {
		for x := int(math.Floor(base.x - hw)); x <= int(math.Ceil(base.x+hw)); x++ {
			dx, dy := float64(x)+0.5-base.x, float64(y)+0.5-ey
			a, b := (dx/16+dy/8)/2, (dy/8-dx/16)/2
			if !shapeHas(l.shape, a, b, r) || (l.clip != nil && !l.clip(a, b)) {
				continue
			}
			na, nb := shapeNormal(l.shape, a, b)
			light := (nb - na) * 0.72
			s.c.penZ = liftByte(z)
			u := (math.Atan2(b, a) + math.Pi) * l.r * faceU
			s.c.dot(x, y, m(surf{x: x, y: y, u: u, v: l.h * t, light: light, up: t}))
		}
	}
}
