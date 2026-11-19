package art

import (
	"image/color"
	"math"
)

var craneSize = size{46, 70, 26, 68}

const (
	wheelCX, wheelCZ   = 0.58, 15.0
	wheelR, wheelRZ    = 0.42, 10.0
	wheelFar, wheelNea = 0.5, 0.76
)

func paintCrane(c *canvas, _ int) {
	s := isoFrame(c, 1, craneSize.ax, craneSize.ay)
	s.box(0.08, 0.08, 0.92, 0.92, 0, 3, faces{top: plazaQuay(s), left: quayFace(true), right: quayFace(false)})
	craneFrame(s, woodR)
	c.outline(ink)
	craneRope(s)
}

func craneFrame(s iso, w ramp) {
	craneSupport(s, wheelFar-0.06, w)
	craneMast(s, w)
	treadwheel(s, w)
	craneSupport(s, wheelNea+0.06, w)
}

func plazaQuay(s iso) shader {
	return func(x, y int) color.NRGBA {
		gx, gy := s.ground(x, y, 3)
		return quayPlank(gx, gy, 0)
	}
}

func (s iso) wheelPt(theta, gy float64) pt {
	return s.p(wheelCX+wheelR*math.Cos(theta), gy, wheelCZ+wheelRZ*math.Sin(theta))
}

func treadwheel(s iso, w ramp) {
	n := 20
	for i := 0; i < n; i++ {
		a, b := 2*math.Pi*float64(i)/float64(n), 2*math.Pi*float64(i+1)/float64(n)
		col := w[1]
		if math.Sin((a+b)/2) > 0.3 {
			col = w[2]
		}
		if i%2 == 0 {
			col = w[0]
		}
		s.c.shadePoly([]pt{s.wheelPt(a, wheelFar), s.wheelPt(b, wheelFar), s.wheelPt(b, wheelNea), s.wheelPt(a, wheelNea)}, flat(col))
	}
	face := make([]pt, 0, n)
	for i := 0; i < n; i++ {
		face = append(face, s.wheelPt(2*math.Pi*float64(i)/float64(n), wheelNea))
	}
	s.c.shadePoly(face, flat(darken(w[0], 0.2)))
	hub := s.p(wheelCX, wheelNea, wheelCZ)
	for k := 0; k < 6; k++ {
		e := s.wheelPt(math.Pi*float64(k)/3+0.3, wheelNea)
		s.c.line(int(hub.x), int(hub.y), int(e.x), int(e.y), w[2])
	}
	for i := 0; i < n; i++ {
		a, b := face[i], face[(i+1)%n]
		s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), w[3])
	}
	s.c.rect(int(hub.x)-1, int(hub.y)-1, 2, 2, greyR[1])
}

func craneSupport(s iso, gy float64, w ramp) {
	top := s.p(wheelCX, gy, wheelCZ)
	for _, gx := range []float64{0.2, 0.96} {
		foot := s.p(gx, gy, 3)
		s.c.line(int(foot.x), int(foot.y), int(top.x), int(top.y), w[1])
		s.c.line(int(foot.x)+1, int(foot.y), int(top.x)+1, int(top.y), w[2])
	}
}

func craneMast(s iso, w ramp) {
	post := faces{top: flat(w[3]), left: flat(w[2]), right: flat(w[0])}
	s.box(0.2, 0.22, 0.32, 0.34, 3, 38, post)
	root, tip := s.p(0.26, 0.28, 34), s.p(-0.95, 0.28, 42)
	for d := 0; d < 2; d++ {
		s.c.line(int(root.x), int(root.y)+d, int(tip.x), int(tip.y)+d, w[1+d])
	}
	brace := s.p(0.26, 0.28, 22)
	mid := s.p(-0.35, 0.28, 38)
	s.c.line(int(brace.x), int(brace.y), int(mid.x), int(mid.y), w[1])
	s.c.rect(int(tip.x)-1, int(tip.y)-1, 3, 2, w[0])
}

func craneRope(s iso) {
	tip := s.p(-0.95, 0.28, 42)
	end := s.p(-0.95, 0.28, 17)
	s.c.line(int(tip.x), int(tip.y)+1, int(end.x), int(end.y), ropeR[1])
	axle := s.p(wheelCX, wheelNea, wheelCZ)
	head := s.p(0.26, 0.28, 34)
	s.c.blendLine(int(axle.x), int(axle.y), int(head.x), int(head.y), withAlpha(ropeR[1], 170))
	x, y := int(end.x), int(end.y)
	c := s.c
	c.rect(x-3, y+1, 7, 5, ink)
	c.rect(x-2, y+2, 5, 3, rgb(196, 170, 120))
	c.rect(x-2, y+2, 5, 1, rgb(226, 204, 156))
	c.dot(x, y+3, rgb(150, 124, 84))
}
