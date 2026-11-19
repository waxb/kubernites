package art

import "image/color"

func (s iso) onLeft(y1, u0, u1, z0, z1 float64, sh shader) {
	s.quad(s.p(u0, y1, z1), s.p(u1, y1, z1), s.p(u1, y1, z0), s.p(u0, y1, z0), sh)
}

func (s iso) onRight(x1, v0, v1, z0, z1 float64, sh shader) {
	s.quad(s.p(x1, v0, z1), s.p(x1, v1, z1), s.p(x1, v1, z0), s.p(x1, v0, z0), sh)
}

func glass(lit bool) shader {
	return func(x, y int) color.NRGBA {
		if !lit {
			return glassR[0]
		}
		if (x+y)%5 == 0 {
			return goldR[3]
		}
		return lampLit
	}
}

func (s iso) windowL(y1, u, z float64, lit bool) {
	s.onLeft(y1, u-0.02, u+0.2, z-1, z+7, flat(ink))
	s.onLeft(y1, u+0.03, u+0.16, z, z+5, glass(lit))
}

func (s iso) windowR(x1, v, z float64, lit bool) {
	s.onRight(x1, v-0.02, v+0.2, z-1, z+7, flat(ink))
	s.onRight(x1, v+0.03, v+0.16, z, z+5, glass(lit))
}

func (s iso) doorL(y1, u, w, h float64) {
	s.onLeft(y1, u-0.04, u+w+0.04, 0, h+2, flat(ink))
	s.onLeft(y1, u, u+w, 0, h, func(x, y int) color.NRGBA {
		if x%2 == 0 {
			return woodR[0]
		}
		return woodR[1]
	})
	p := s.p(u+w*0.8, y1, h/2)
	s.c.dot(int(p.x), int(p.y), goldR[2])
}

func (s iso) bannerL(y1, u, z, w, h float64, cloth ramp, emblem color.NRGBA) {
	s.onLeft(y1, u, u+w, z-h, z, func(x, y int) color.NRGBA {
		if x%3 == 0 {
			return cloth[1]
		}
		return cloth[2]
	})
	c := s.p(u+w/2, y1, z-h/2)
	s.c.dot(int(c.x), int(c.y), emblem)
	s.c.dot(int(c.x)+1, int(c.y), emblem)
	tip := s.p(u+w/2, y1, z-h-2)
	s.c.dot(int(tip.x), int(tip.y), cloth[1])
}

func (s iso) flag(gx, gy, z, pole float64, cloth ramp) {
	base, top := s.p(gx, gy, z), s.p(gx, gy, z+pole)
	s.c.line(int(base.x), int(base.y), int(top.x), int(top.y), woodR[0])
	tx, ty := int(top.x), int(top.y)
	s.c.shadePoly([]pt{{float64(tx) + 1, float64(ty)}, {float64(tx) + 9, float64(ty) + 2}, {float64(tx) + 8, float64(ty) + 5}, {float64(tx) + 1, float64(ty) + 5}}, func(x, y int) color.NRGBA {
		if y%2 == 0 {
			return cloth[2]
		}
		return cloth[3]
	})
	s.c.dot(tx, ty-1, goldR[3])
	s.c.flags = append(s.c.flags, FlagMark{X: tx + s.c.ox, Y: ty - 1 + s.c.oy, Light: cloth[2], Dark: cloth[3]})
}

func (s iso) crate(gx, gy, sz float64) {
	s.box(gx, gy, gx+sz, gy+sz, 0, sz*14, faces{top: flat(woodR[3]), left: s.plankLeft(woodR, gy+sz), right: s.plankRight(woodR, gx+sz)})
}

func (s iso) barrel(gx, gy float64) {
	p := s.p(gx, gy, 0)
	s.c.rect(int(p.x)-3, int(p.y)-8, 6, 8, woodR[1])
	s.c.rect(int(p.x)-3, int(p.y)-8, 2, 8, woodR[2])
	s.c.rect(int(p.x)-3, int(p.y)-6, 6, 1, greyR[1])
	s.c.rect(int(p.x)-3, int(p.y)-3, 6, 1, greyR[1])
	s.c.ellipse(p.x, p.y-8, 3, 1.3, woodR[3])
}

func (s iso) foundation(n int, r ramp, inset float64) {
	f := float64(n)
	s.box(inset, inset, f-inset, f-inset, 0, 3, faces{top: flat(r[2]), left: flat(r[1]), right: flat(r[0])})
}

func (s iso) goldPile(gx, gy float64, big bool) {
	p := s.p(gx, gy, 0)
	r := 3.5
	if big {
		r = 5
	}
	s.c.ellipse(p.x, p.y-r/2, r, r*0.7, goldR[1])
	s.c.ellipse(p.x-1, p.y-r/2-1, r*0.6, r*0.4, goldR[2])
	s.c.dot(int(p.x)-1, int(p.y-r)+1, goldR[3])
	s.c.dot(int(p.x)+2, int(p.y-r/2), goldR[3])
}
