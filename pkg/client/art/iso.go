package art

import (
	"image/color"
	"math"
)

type shader func(x, y int) color.NRGBA

func flat(col color.NRGBA) shader { return func(int, int) color.NRGBA { return col } }

func hash(x, y, seed int) uint32 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(seed)*2246822519
	h = (h ^ (h >> 13)) * 1274126177
	return h ^ (h >> 16)
}

func noise(x, y, seed int) float64 {
	return float64(hash(x, y, seed)%1000) / 1000
}

type iso struct {
	c      *canvas
	ox, oy float64
	lk     look
	gnd    *canvas
}

func isoFrame(c *canvas, n, ax, ay int) iso {
	return iso{c: c, ox: float64(ax), oy: float64(ay) - float64(n*TileHeight)}
}

func (s iso) p(gx, gy, z float64) pt {
	return pt{s.ox + (gx-gy)*16, s.oy + (gx+gy)*8 - z}
}

func (s iso) quad(a, b, c, d pt, sh shader) {
	s.c.shadePoly([]pt{a, b, c, d}, sh)
}

type faces struct {
	top, left, right shader
}

func (s iso) box(x0, y0, x1, y1, z0, z1 float64, f faces) {
	s.quad(s.p(x0, y1, z1), s.p(x1, y1, z1), s.p(x1, y1, z0), s.p(x0, y1, z0), f.left)
	s.quad(s.p(x1, y0, z1), s.p(x1, y1, z1), s.p(x1, y1, z0), s.p(x1, y0, z0), f.right)
	s.quad(s.p(x0, y0, z1), s.p(x1, y0, z1), s.p(x1, y1, z1), s.p(x0, y1, z1), f.top)
}

func (s iso) leftUV(sx, sy int, y1 float64) (float64, float64) {
	gx := (float64(sx)+0.5-s.ox)/16 + y1
	z := s.oy + (gx+y1)*8 - float64(sy) - 0.5
	return gx, z
}

func (s iso) rightUV(sx, sy int, x1 float64) (float64, float64) {
	gy := x1 - (float64(sx)+0.5-s.ox)/16
	z := s.oy + (x1+gy)*8 - float64(sy) - 0.5
	return gy, z
}

func masonry(r ramp, u, z float64, seed int, lit bool) color.NRGBA {
	base, mortar := r[1], r[0]
	if lit {
		base, mortar = r[2], r[1]
	}
	row := int(math.Floor(z / 4))
	if int(math.Floor(z))%4 == 0 {
		return mortar
	}
	off := float64(row%2) * 3
	if int(math.Floor(u*16+off))%7 == 0 {
		return mortar
	}
	if noise(int(u*16), row, seed) > 0.86 {
		return r[3]
	}
	return base
}

func (s iso) stoneLeft(r ramp, y1 float64, seed int) shader {
	return func(x, y int) color.NRGBA {
		u, z := s.leftUV(x, y, y1)
		return masonry(r, u, z, seed, true)
	}
}

func (s iso) stoneRight(r ramp, x1 float64, seed int) shader {
	return func(x, y int) color.NRGBA {
		u, z := s.rightUV(x, y, x1)
		return masonry(r, u, z, seed, false)
	}
}

func planks(r ramp, u float64, lit bool) color.NRGBA {
	base := r[1]
	if lit {
		base = r[2]
	}
	if int(math.Floor(u*16))%4 == 0 {
		return r[0]
	}
	return base
}

func (s iso) plankLeft(r ramp, y1 float64) shader {
	return func(x, y int) color.NRGBA {
		u, _ := s.leftUV(x, y, y1)
		return planks(r, u, true)
	}
}

func (s iso) plankRight(r ramp, x1 float64) shader {
	return func(x, y int) color.NRGBA {
		u, _ := s.rightUV(x, y, x1)
		return planks(r, u, false)
	}
}

func shingles(r ramp, lit bool) shader {
	return func(x, y int) color.NRGBA {
		base, line := r[1], r[0]
		if lit {
			base, line = r[2], r[1]
		}
		if y%3 == 0 {
			return line
		}
		if (x+(y/3)%2*2)%4 == 0 {
			return line
		}
		if noise(x, y, 7) > 0.9 {
			return r[3]
		}
		return base
	}
}

func (s iso) stoneBox(x0, y0, x1, y1, z0, z1 float64, r ramp, top shader, seed int) {
	s.box(x0, y0, x1, y1, z0, z1, faces{top: top, left: s.stoneLeft(r, y1, seed), right: s.stoneRight(r, x1, seed)})
}

func (s iso) gableX(x0, y0, x1, y1, z, rh, eave float64, roof ramp, gableCol color.NRGBA) {
	ym := (y0 + y1) / 2
	a0, a1 := x0-eave, x1+eave
	s.quad(s.p(a0, y0-eave, z), s.p(a1, y0-eave, z), s.p(a1, ym, z+rh), s.p(a0, ym, z+rh), shingles(roof, false))
	s.c.shadePoly([]pt{s.p(x1, y0, z), s.p(x1, ym, z+rh), s.p(x1, y1, z)}, flat(gableCol))
	s.quad(s.p(a0, ym, z+rh), s.p(a1, ym, z+rh), s.p(a1, y1+eave, z), s.p(a0, y1+eave, z), shingles(roof, true))
}

func (s iso) gableY(x0, y0, x1, y1, z, rh, eave float64, roof ramp, gableCol color.NRGBA) {
	xm := (x0 + x1) / 2
	b0, b1 := y0-eave, y1+eave
	s.quad(s.p(x0-eave, b0, z), s.p(xm, b0, z+rh), s.p(xm, b1, z+rh), s.p(x0-eave, b1, z), shingles(roof, true))
	s.c.shadePoly([]pt{s.p(x0, y1, z), s.p(xm, y1, z+rh), s.p(x1, y1, z)}, flat(gableCol))
	s.quad(s.p(xm, b0, z+rh), s.p(x1+eave, b0, z), s.p(x1+eave, b1, z), s.p(xm, b1, z+rh), shingles(roof, false))
}

func (s iso) pyramid(x0, y0, x1, y1, z, h float64, roof ramp) {
	apex := s.p((x0+x1)/2, (y0+y1)/2, z+h)
	s.c.shadePoly([]pt{s.p(x0, y1, z), s.p(x1, y1, z), apex}, shingles(roof, true))
	s.c.shadePoly([]pt{s.p(x1, y0, z), s.p(x1, y1, z), apex}, shingles(roof, false))
}

func (s iso) crenels(x0, y0, x1, y1, z float64, r ramp) {
	step := 0.5
	for x := x0; x < x1-0.01; x += step * 2 {
		s.box(x, y1-0.22, x+step, y1, z, z+4, faces{top: flat(r[3]), left: flat(r[2]), right: flat(r[1])})
	}
	for y := y0; y < y1-0.01; y += step * 2 {
		s.box(x1-0.22, y, x1, y+step, z, z+4, faces{top: flat(r[3]), left: flat(r[2]), right: flat(r[1])})
	}
}
