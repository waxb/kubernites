package art

import "image/color"

func (s iso) layer(draw func(s iso)) {
	tmp := newCanvas(s.c.w, s.c.h)
	tmp.enableTags()
	l := s
	l.c = tmp
	draw(l)
	contourUnder(s.c, tmp)
	for y := 0; y < tmp.h; y++ {
		for x := 0; x < tmp.w; x++ {
			if col := tmp.at(x, y); col.A > 0 {
				s.c.copyFrom(tmp, x, y, col)
			}
		}
	}
	s.c.flags = append(s.c.flags, tmp.flags...)
}

func (c *canvas) copyFrom(src *canvas, x, y int, col color.NRGBA) {
	c.img.SetNRGBA(x, y, col)
	if c.tags != nil && src.tags != nil {
		i := y*c.w + x
		c.tags[i], c.lift[i] = src.tags[i], src.lift[i]
	}
}

func contourUnder(dst, layer *canvas) {
	for y := 0; y < layer.h; y++ {
		for x := 0; x < layer.w; x++ {
			if layer.at(x, y).A > 0 || !touchesSolid(layer, x, y) {
				continue
			}
			if col := dst.at(x, y); col.A > 0 {
				dst.img.SetNRGBA(x, y, darken(col, 0.5))
			}
		}
	}
}

func touchesSolid(c *canvas, x, y int) bool {
	return c.at(x-1, y).A > 96 || c.at(x+1, y).A > 96 || c.at(x, y-1).A > 96 || c.at(x, y+1).A > 96
}

func (s iso) aloft(z float64, draw func()) {
	s.c.pen, s.c.penZ = matTrim, liftByte(z)
	draw()
	s.c.pen, s.c.penZ = matNone, 0
}
