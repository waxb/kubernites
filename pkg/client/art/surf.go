package art

import (
	"image/color"
	"math"
)

type surf struct {
	x, y  int
	u, v  float64
	light float64
	up    float64
}

type mat func(p surf) color.NRGBA

const (
	litLeft  = 0.55
	litRight = -0.45
	litTop   = 1.0
	faceU    = 18.0
)

func tone(r ramp, light float64, x, y int) color.NRGBA {
	f := min(max(1.5+light*1.5, 0), 3)
	i := int(f)
	if i < 3 && f-float64(i) > dither(x, y) {
		i++
	}
	return r[min(i, 3)]
}

func liftByte(z float64) uint8 { return uint8(min(max(z, 0), 250)) }

func (c *canvas) enableTags() {
	c.tags, c.lift = make([]uint8, c.w*c.h), make([]uint8, c.w*c.h)
}

func (c *canvas) tagAt(x, y int) uint8 {
	if c.tags == nil || !c.in(x, y) {
		return matNone
	}
	return c.tags[y*c.w+x]
}

func (c *canvas) liftAt(x, y int) int {
	if c.lift == nil || !c.in(x, y) {
		return 255
	}
	return int(c.lift[y*c.w+x])
}

func (s iso) pen(tag uint8, draw func()) {
	s.c.pen, s.c.penZ = tag, 0
	draw()
	s.c.pen, s.c.penZ = matNone, 0
}

func (s iso) leftMat(y1 float64, m mat) shader {
	return func(x, y int) color.NRGBA {
		gx, z := s.leftUV(x, y, y1)
		s.c.penZ = liftByte(z)
		return m(surf{x: x, y: y, u: gx * faceU, v: z, light: litLeft})
	}
}

func (s iso) rightMat(x1 float64, m mat) shader {
	return func(x, y int) color.NRGBA {
		gy, z := s.rightUV(x, y, x1)
		s.c.penZ = liftByte(z)
		return m(surf{x: x, y: y, u: gy * faceU, v: z, light: litRight})
	}
}

func (s iso) topMat(z float64, m mat) shader {
	return func(x, y int) color.NRGBA {
		gx, gy := s.ground(x, y, z)
		s.c.penZ = liftByte(z)
		return m(surf{x: x, y: y, u: gx * faceU, v: gy * faceU, light: litTop})
	}
}

func (s iso) block(x0, y0, x1, y1, z0, z1 float64, wall, top mat, tag uint8) {
	s.pen(tag, func() {
		s.box(x0, y0, x1, y1, z0, z1, faces{top: s.topMat(z1, top), left: s.leftMat(y1, wall), right: s.rightMat(x1, wall)})
	})
}

func (s iso) planeX(y0, dy, z, dz float64, light float64, m mat) shader {
	den := 16*dy - dz
	slope := math.Hypot(dy*faceU, dz)
	return func(x, y int) color.NRGBA {
		fx, fy := float64(x)+0.5-s.ox, float64(y)+0.5-s.oy
		w := (fy - fx/2 - 16*y0 + z) / den
		gx := fx/16 + y0 + w*dy
		s.c.penZ = liftByte(z + w*dz)
		return m(surf{x: x, y: y, u: gx * faceU, v: w * slope, light: light, up: w})
	}
}

func (s iso) planeY(x0, dx, z, dz float64, light float64, m mat) shader {
	den := 16*dx - dz
	slope := math.Hypot(dx*faceU, dz)
	return func(x, y int) color.NRGBA {
		fx, fy := float64(x)+0.5-s.ox, float64(y)+0.5-s.oy
		w := (fy + fx/2 - 16*x0 + z) / den
		gy := x0 + w*dx - fx/16
		s.c.penZ = liftByte(z + w*dz)
		return m(surf{x: x, y: y, u: gy * faceU, v: w * slope, light: light, up: w})
	}
}
