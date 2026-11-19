package art

import (
	"image"
	"image/color"
	"math"
)

type canvas struct {
	img    *image.NRGBA
	w, h   int
	ox, oy int
	tags   []uint8
	lift   []uint8
	pen    uint8
	penZ   uint8
	flags  []FlagMark
}

type pt struct {
	x, y float64
}

func newCanvas(w, h int) *canvas {
	return &canvas{img: image.NewNRGBA(image.Rect(0, 0, w, h)), w: w, h: h}
}

func newFramedCanvas(w, h, ox, oy int) *canvas {
	c := newCanvas(w, h)
	c.ox, c.oy = ox, oy
	return c
}

func (c *canvas) in(x, y int) bool {
	return x >= 0 && y >= 0 && x < c.w && y < c.h
}

func (c *canvas) dot(x, y int, col color.NRGBA) {
	x, y = x+c.ox, y+c.oy
	if c.in(x, y) && col.A > 0 {
		c.img.SetNRGBA(x, y, col)
		if c.tags != nil {
			c.tags[y*c.w+x], c.lift[y*c.w+x] = c.pen, c.penZ
		}
	}
}

func (c *canvas) at(x, y int) color.NRGBA {
	x, y = x+c.ox, y+c.oy
	if !c.in(x, y) {
		return color.NRGBA{}
	}
	return c.img.NRGBAAt(x, y)
}

func (c *canvas) rect(x, y, w, h int, col color.NRGBA) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.dot(xx, yy, col)
		}
	}
}

func (c *canvas) line(x0, y0, x1, y1 int, col color.NRGBA) {
	dx, dy := abs(x1-x0), -abs(y1-y0)
	sx, sy := sign(x1-x0), sign(y1-y0)
	e := dx + dy
	for {
		c.dot(x0, y0, col)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * e
		if e2 >= dy {
			e += dy
			x0 += sx
		}
		if e2 <= dx {
			e += dx
			y0 += sy
		}
	}
}

func (c *canvas) shadePoly(points []pt, shade func(x, y int) color.NRGBA) {
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, p := range points {
		minY, maxY = math.Min(minY, p.y), math.Max(maxY, p.y)
	}
	buf := make([]float64, 0, len(points))
	for y := int(math.Floor(minY)); y <= int(math.Ceil(maxY)); y++ {
		xs := crossings(points, float64(y)+0.5, buf[:0])
		for i := 0; i+1 < len(xs); i += 2 {
			for x := int(math.Ceil(xs[i] - 0.5)); x <= int(math.Floor(xs[i+1]-0.5)); x++ {
				c.dot(x, y, shade(x, y))
			}
		}
	}
}

func crossings(points []pt, y float64, xs []float64) []float64 {
	for i := range points {
		a, b := points[i], points[(i+1)%len(points)]
		if (a.y <= y && b.y > y) || (b.y <= y && a.y > y) {
			xs = append(xs, a.x+(y-a.y)*(b.x-a.x)/(b.y-a.y))
		}
	}
	for i := 1; i < len(xs); i++ {
		for j := i; j > 0 && xs[j] < xs[j-1]; j-- {
			xs[j], xs[j-1] = xs[j-1], xs[j]
		}
	}
	return xs
}

func (c *canvas) disc(cx, cy, r float64, col color.NRGBA) {
	for y := int(cy - r - 1); y <= int(cy+r+1); y++ {
		for x := int(cx - r - 1); x <= int(cx+r+1); x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			if dx*dx+dy*dy <= r*r {
				c.dot(x, y, col)
			}
		}
	}
}

func (c *canvas) ellipse(cx, cy, rx, ry float64, col color.NRGBA) {
	for y := int(cy - ry - 1); y <= int(cy+ry+1); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if dx*dx+dy*dy <= 1 {
				c.dot(x, y, col)
			}
		}
	}
}

func (c *canvas) outline(col color.NRGBA) {
	src := snapshot(c)
	defer release(src)
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if src.NRGBAAt(x, y).A != 0 {
				continue
			}
			if solid(src, x-1, y) || solid(src, x+1, y) || solid(src, x, y-1) || solid(src, x, y+1) {
				c.img.SetNRGBA(x, y, col)
			}
		}
	}
}

func solid(img *image.NRGBA, x, y int) bool {
	if !(image.Point{X: x, Y: y}).In(img.Rect) {
		return false
	}
	return img.NRGBAAt(x, y).A > 96
}

func (c *canvas) mapPixels(fn func(col color.NRGBA) color.NRGBA) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if col := c.img.NRGBAAt(x, y); col.A > 0 {
				c.img.SetNRGBA(x, y, fn(col))
			}
		}
	}
}

func (c *canvas) paste(src *canvas, dx, dy int, flip bool) {
	for y := 0; y < src.h; y++ {
		for x := 0; x < src.w; x++ {
			sx := x
			if flip {
				sx = src.w - 1 - x
			}
			col := src.img.NRGBAAt(sx, y)
			if col.A > 0 {
				c.dot(dx+x, dy+y, col)
			}
		}
	}
	if !flip {
		for _, m := range src.flags {
			c.flags = append(c.flags, FlagMark{X: m.X + dx + c.ox, Y: m.Y + dy + c.oy, Light: m.Light, Dark: m.Dark})
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}
