package art

import "image/color"

type bounds struct {
	x0, y0, x1, y1 int
}

func opaqueBounds(src *canvas) bounds {
	b := bounds{src.w, src.h, 0, 0}
	for y := 0; y < src.h; y++ {
		for x := 0; x < src.w; x++ {
			if src.at(x, y).A > 0 {
				b = bounds{min(b.x0, x), min(b.y0, y), max(b.x1, x+1), max(b.y1, y+1)}
			}
		}
	}
	return b
}

func fitSprite(src *canvas, box int) *canvas {
	b := opaqueBounds(src)
	w, h := b.x1-b.x0, b.y1-b.y0
	scale := min(1, float64(box)/float64(max(w, h)))
	tw, th := max(1, int(float64(w)*scale)), max(1, int(float64(h)*scale))
	out := newCanvas(tw, th)
	for ty := 0; ty < th; ty++ {
		for tx := 0; tx < tw; tx++ {
			area := bounds{
				b.x0 + int(float64(tx)/scale), b.y0 + int(float64(ty)/scale),
				b.x0 + max(int(float64(tx+1)/scale), int(float64(tx)/scale)+1),
				b.y0 + max(int(float64(ty+1)/scale), int(float64(ty)/scale)+1),
			}
			if col, ok := majority(src, area, scale < 1); ok {
				out.dot(tx, ty, col)
			}
		}
	}
	if scale < 1 {
		out.outline(ink)
	}
	return out
}

func majority(src *canvas, a bounds, skipInk bool) (color.NRGBA, bool) {
	counts := map[color.NRGBA]int{}
	opaque, total := 0, 0
	for y := a.y0; y < a.y1; y++ {
		for x := a.x0; x < a.x1; x++ {
			total++
			col := src.at(x, y)
			if col.A == 0 {
				continue
			}
			opaque++
			if !skipInk || col != ink {
				counts[col]++
			}
		}
	}
	if opaque*2 < total || opaque == 0 {
		return color.NRGBA{}, false
	}
	best, bestN := ink, 0
	for col, n := range counts {
		if n > bestN || (n == bestN && packed(col) < packed(best)) {
			best, bestN = col, n
		}
	}
	return best, true
}

func packed(c color.NRGBA) uint32 {
	return uint32(c.R)<<24 | uint32(c.G)<<16 | uint32(c.B)<<8 | uint32(c.A)
}
