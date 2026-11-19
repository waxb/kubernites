package art

import "image/color"

func propIso(c *canvas, ax, ay int) iso {
	return iso{c: c, ox: float64(ax), oy: float64(ay) - 8}
}

func propShadow(c *canvas, cx, cy, rx float64) {
	for y := int(cy - 2); y <= int(cy+2); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/(rx/2.4)
			if dx*dx+dy*dy <= 1 && c.at(x, y).A == 0 {
				c.dot(x, y, shadow)
			}
		}
	}
}

func paintBollard(c *canvas, _ int) {
	iron := ramp{rgb(40, 38, 46), rgb(62, 60, 70), rgb(96, 94, 104), rgb(140, 138, 146)}
	c.rect(3, 4, 3, 6, iron[1])
	c.rect(3, 4, 1, 6, iron[2])
	c.rect(5, 4, 1, 6, iron[0])
	c.rect(2, 2, 5, 2, iron[2])
	c.rect(2, 2, 5, 1, iron[3])
	c.rect(3, 7, 3, 1, ropeR[2])
	c.dot(6, 8, ropeR[1])
	c.dot(7, 9, ropeR[1])
	c.outline(ink)
	propShadow(c, 4.5, 10, 3.5)
}

func paintCrates(c *canvas, v int) {
	s := propIso(c, 12, 19)
	switch v {
	case 1:
		s.barrel(0.3, 0.55)
		s.barrel(0.62, 0.42)
		s.barrel(0.62, 0.78)
	case 2:
		s.crateAt(0.2, 0.2, 0.32, 0)
		sack(c, s.p(0.62, 0.72, 0), rgb(196, 170, 120))
		sack(c, s.p(0.42, 0.76, 0), rgb(176, 150, 104))
	default:
		s.crateAt(0.12, 0.3, 0.36, 0)
		s.crateAt(0.56, 0.56, 0.3, 0)
		s.crateAt(0.18, 0.36, 0.26, 6.84)
	}
	c.outline(ink)
	propShadow(c, 12, 20, 8)
}

var crateR = ramp{rgb(104, 70, 42), rgb(160, 114, 66), rgb(204, 158, 96), rgb(236, 204, 142)}

func crateFace(f, u float64, lit bool) color.NRGBA {
	switch {
	case absf(u-0.5) < 0.1:
		return crateR[0]
	case lit && f > 0.8:
		return crateR[3]
	case lit:
		return crateR[2]
	}
	return crateR[1]
}

func (s iso) crateAt(gx, gy, sz, z float64) {
	z1 := z + sz*19
	left := func(x, y int) color.NRGBA {
		u, zz := s.leftUV(x, y, gy+sz)
		return crateFace((zz-z)/(z1-z), (u-gx)/sz, true)
	}
	right := func(x, y int) color.NRGBA {
		u, zz := s.rightUV(x, y, gx+sz)
		return crateFace((zz-z)/(z1-z), (u-gy)/sz, false)
	}
	s.box(gx, gy, gx+sz, gy+sz, z, z1, faces{top: flat(crateR[3]), left: left, right: right})
}

func sack(c *canvas, p pt, col color.NRGBA) {
	r := ramp{darken(col, 0.35), darken(col, 0.15), col, mix(col, white, 0.35)}
	c.ellipse(p.x, p.y-3, 3.5, 3, r[1])
	c.ellipse(p.x-0.8, p.y-3.6, 2.4, 2, r[2])
	c.dot(int(p.x)-1, int(p.y)-5, r[3])
	c.rect(int(p.x)-1, int(p.y)-7, 2, 1, r[0])
	c.dot(int(p.x), int(p.y)-8, r[2])
}

func paintHarborSign(c *canvas, _ int) {
	body := newCanvas(17, 28)
	signBody(body)
	c.paste(body, 2, 1, false)
	c.outline(ink)
	propShadow(c, 6, 27, 4)
}

func signBody(c *canvas) {
	c.rect(3, 3, 2, 23, woodR[1])
	c.rect(3, 3, 1, 23, woodR[2])
	c.rect(2, 2, 4, 1, woodR[3])
	c.rect(3, 5, 12, 1, woodR[1])
	c.rect(14, 5, 1, 2, woodR[0])
	c.dot(7, 6, greyR[2])
	c.dot(13, 6, greyR[2])
	c.rect(5, 7, 11, 8, woodR[0])
	c.rect(6, 8, 9, 6, plasterR[3])
	signEmblem(c)
}

func signEmblem(c *canvas) {
	c.rect(7, 12, 7, 1, woodR[1])
	c.rect(8, 13, 5, 1, woodR[0])
	c.rect(10, 8, 1, 4, woodR[0])
	c.shadePoly([]pt{{11, 8.5}, {13.5, 11.5}, {11, 11.5}}, flat(redR[2]))
	c.shadePoly([]pt{{10, 9}, {10, 11.5}, {8, 11.5}}, flat(white))
	c.dot(10, 8, goldR[2])
}
