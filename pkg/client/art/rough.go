package art

import (
	"image"
	"image/color"
	"sync"
)

const (
	scratchKeep = 8
	mossReach   = 10
)

var (
	chipRate = [AgeTiers]float64{0.12, 0.08, 0.06, 0.02}
	bumpRate = [AgeTiers]float64{0.1, 0.06, 0.03, 0.01}
	dripRate = [AgeTiers]float64{0.3, 0.1, 0.05, 0}
	mossRate = [AgeTiers]float64{0.7, 0.76, 0.8, 0.95}
	roofMoss = [AgeTiers]float64{0.78, 0.88, 1, 1}
)

func massTag(t uint8) bool { return t == matRoof || t == matWall || t == matStone }

var scratch = struct {
	sync.Mutex
	free map[image.Rectangle][]*image.NRGBA
}{free: map[image.Rectangle][]*image.NRGBA{}}

func snapshot(c *canvas) *image.NRGBA {
	scratch.Lock()
	pool := scratch.free[c.img.Rect]
	var src *image.NRGBA
	if n := len(pool); n > 0 {
		src, scratch.free[c.img.Rect] = pool[n-1], pool[:n-1]
	}
	scratch.Unlock()
	if src == nil {
		src = image.NewNRGBA(c.img.Rect)
	}
	copy(src.Pix, c.img.Pix)
	return src
}

func release(img *image.NRGBA) {
	scratch.Lock()
	if len(scratch.free[img.Rect]) < scratchKeep {
		scratch.free[img.Rect] = append(scratch.free[img.Rect], img)
	}
	scratch.Unlock()
}

func opaque(img *image.NRGBA, x, y int) bool {
	return (image.Point{X: x, Y: y}).In(img.Rect) && img.NRGBAAt(x, y).A > 0
}

func (c *canvas) erase(x, y int) {
	if c.in(x, y) {
		c.img.SetNRGBA(x, y, color.NRGBA{})
		if c.tags != nil {
			c.tags[y*c.w+x] = matNone
		}
	}
}

func (c *canvas) copyPixel(x, y, sx, sy int, col color.NRGBA) {
	if !c.in(x, y) || !c.in(sx, sy) {
		return
	}
	c.img.SetNRGBA(x, y, col)
	if c.tags != nil {
		c.tags[y*c.w+x], c.lift[y*c.w+x] = c.tags[sy*c.w+sx], c.lift[sy*c.w+sx]
	}
}

func breakup(c *canvas, l look) {
	src := snapshot(c)
	defer release(src)
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			if tag := c.tagAt(x, y); opaque(src, x, y) && massTag(tag) {
				breakPixel(c, src, x, y, tag, l)
			}
		}
	}
}

func breakPixel(c *canvas, src *image.NRGBA, x, y int, tag uint8, l look) {
	col, r := src.NRGBAAt(x, y), pick(x, y, l.seed+201)
	top, below := !opaque(src, x, y-1), opaque(src, x, y+1)
	row := opaque(src, x-1, y) && opaque(src, x+1, y)
	switch {
	case top && below && r < chipRate[l.age] && opaque(src, x-1, y+1) && opaque(src, x+1, y+1):
		c.erase(x, y)
	case top && row && !opaque(src, x-1, y-1) && !opaque(src, x+1, y-1) && r > 1-bumpRate[l.age]:
		c.copyPixel(x, y-1, x, y, col)
	case top && tag == matRoof && l.age == ageDark && r > 1-bumpRate[l.age] && (opaque(src, x-1, y) || opaque(src, x+1, y)):
		c.copyPixel(x, y-1, x, y, darken(col, 0.1))
	case tag == matRoof && !below && row && r < dripRate[l.age]:
		c.copyPixel(x, y+1, x, y, darken(col, 0.25))
	case tag != matRoof && (!opaque(src, x-1, y) || !opaque(src, x+1, y)) && !top && below && r < chipRate[l.age]*0.4:
		c.erase(x, y)
	}
}

func roofSpan(c *canvas) (int, int) {
	lo, hi := 255, 0
	for i, t := range c.tags {
		if t == matRoof {
			lo, hi = min(lo, int(c.lift[i])), max(hi, int(c.lift[i]))
		}
	}
	return lo, hi
}

func weather(c *canvas, l look) {
	lo, hi := roofSpan(c)
	stains(c, l)
	defer ivy(c, l)
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			col := c.at(x, y)
			if col.A == 0 {
				continue
			}
			switch c.tagAt(x, y) {
			case matWall, matStone:
				c.img.SetNRGBA(x, y, weatherWall(col, x, y, c.liftAt(x, y), l))
			case matRoof:
				c.img.SetNRGBA(x, y, weatherRoof(col, x, y, c.liftAt(x, y), lo, hi, l))
			}
		}
	}
}

func weatherWall(col color.NRGBA, x, y, lift int, l look) color.NRGBA {
	if lift < 6 && dither(x, y) < float64(6-lift)/7 {
		col = darken(col, 0.2)
	}
	if lift < mossReach && vnoise(float64(x)/4, float64(y)/3, l.seed+211) > mossRate[l.age]+float64(lift)*0.02 && dither(x, y) < 0.55 {
		col = mix(col, mossR[2], 0.35)
	}
	return col
}

func weatherRoof(col color.NRGBA, x, y, lift, lo, hi int, l look) color.NRGBA {
	if hi > lo && float64(lift-lo)/float64(hi-lo) > 0.66 && dither(x, y) < 0.3 {
		col = mix(col, plasterR[3], 0.16)
	}
	if l.age <= ageFeudal && vnoise(float64(x)/5, float64(y)/4, l.seed+221) > roofMoss[l.age] {
		col = mix(col, mossR[2], 0.35)
	}
	return col
}

func stains(c *canvas, l look) {
	for y := c.h - 2; y >= 0; y-- {
		for x := 0; x < c.w; x++ {
			if c.tagAt(x, y) != matGlass || c.tagAt(x, y+1) == matGlass {
				continue
			}
			for k := 2; k <= 5; k++ {
				if t := c.tagAt(x, y+k); massTag(t) && pick(x, y+k, l.seed+231) < 0.75-0.12*float64(k) {
					c.img.SetNRGBA(x, y+k, darken(c.at(x, y+k), 0.12))
				}
			}
		}
	}
}

func finishRough(dst, gnd, body *canvas, l look) {
	breakup(body, l)
	weather(body, l)
	body.outline(ink)
	contact(gnd, body)
	dst.paste(gnd, 0, 0, false)
	dst.paste(body, 0, 0, false)
	footTufts(dst, gnd, body, l)
}
