package art

import (
	"image"
	"slices"
)

func PaintIcon(name string) (*image.NRGBA, bool) {
	paint, ok := iconPainters[name]
	if !ok {
		paint, ok = caesarIconPainters[name]
	}
	if !ok {
		return nil, false
	}
	c := newCanvas(IconSize, IconSize)
	paint(c)
	c.outline(ink)
	return c.img, true
}

func iconFood(c *canvas) {
	c.disc(9.6, 6.2, 4.6, woodR[1])
	c.disc(9.0, 5.6, 4.0, woodR[2])
	c.disc(8.0, 4.4, 1.8, woodR[3])
	c.dot(7, 3, cream)
	c.line(6, 9, 3, 12, plasterR[3])
	c.line(6, 10, 4, 12, plasterR[2])
	c.disc(2.4, 11.0, 1.3, cream)
	c.disc(3.6, 12.6, 1.3, plasterR[3])
}

func iconHousing(c *canvas) {
	c.shadePoly([]pt{{1, 8.5}, {8, 1.5}, {15, 8.5}}, litByX(redR, 8))
	c.rect(3, 8, 10, 6, plasterR[2])
	c.rect(9, 8, 4, 6, plasterR[1])
	c.rect(4, 9, 2, 2, ink)
	c.rect(9, 10, 2, 4, woodR[0])
	c.rect(3, 8, 10, 1, redR[0])
}

func iconWheat(c *canvas) {
	for _, x := range []int{4, 8, 12} {
		c.rect(x, 7, 1, 8, woodR[2])
		c.disc(float64(x)+0.5, 4, 1.6, goldR[2])
		c.dot(x, 3, goldR[3])
	}
	c.rect(3, 10, 11, 1, woodR[1])
}

func PaintBoxIcon(kind string, sealed bool) (*image.NRGBA, bool) {
	if !slices.Contains(BoxKinds, kind) {
		return nil, false
	}
	sz, d := boxCanvas(0), worldBoxes[0]
	c := newCanvas(sz.w, sz.h)
	paintBox(c, sz, d, kind, 0, false)
	if sealed {
		paintSealMark(c, sz, d, 0)
	}
	return c.img, true
}
