package art

import (
	"fmt"
	"image/color"
)

func PaperBoxName(size, variant int) string { return fmt.Sprintf("box-paper/%d/%d", size, variant) }

func addPaperBoxes(b *atlasBuilder, size int) {
	sz, d := boxCanvas(size), worldBoxes[size]
	for v := 0; v < BoxVariants; v++ {
		vv := v
		b.add(PaperBoxName(size, v), sz.w, sz.h, sz.ax, sz.ay, func(c *canvas) { paintBox(c, sz, d, "paper", vv, false) })
	}
}

func paperMat(v int) boxMat {
	return func(p facePix, x, y int) color.NRGBA {
		if p.face == 0 {
			if hash(x, y, 311+v)%9 == 0 {
				return kraftR[2]
			}
			return kraftR[3]
		}
		lit := p.face == 1
		mid := p.w / 2
		switch {
		case v != 2 && lit && onLabel(p):
			return labelColor(p, blueR, lit)
		case v != 2 && p.w >= 6 && (p.col == mid || (v == 3 && p.w >= 8 && p.col == mid-2)):
			return sideTone(twineR, lit)
		case v == 2 && !lit && inRect(p, 1, max(p.h/2-1, 1), max(mid-1, 2), min(2+p.h/10, max(p.h-3, 1))):
			return sideTone(blueR, lit)
		case v == 2 && p.row == p.h-1:
			return tapeCol
		case p.row == 0:
			return sideTone(kraftR, false)
		}
		return sideTone(kraftR, lit)
	}
}

func paperDecals(bx boxCtx, v int) {
	switch v {
	case 2:
		bx.line(bx.T(0.5, 0.02), bx.T(0.5, 0.98), tapeCol)
		if bx.width() >= 8 {
			bx.line(bx.T(0.42, 0.02), bx.T(0.42, 0.98), tapeCol)
		}
		bx.dotTop(0.25, 0.3, redR[2])
	case 3:
		bx.topCross(0.5, twineR[3])
		if bx.width() >= 8 {
			bx.topCross(0.3, twineR[2])
		}
	default:
		bx.topCross(0.5, twineR[3])
	}
	if v == 1 && bx.width() >= 6 {
		bx.dotTop(0.78, 0.24, redR[2])
		bx.dotTop(0.86, 0.3, redR[1])
		bx.dotTop(0.78, 0.33, white)
	}
	if v != 2 && bx.width() >= 6 {
		bx.dotTop(0.5, 0.5, twineR[1])
	}
}
