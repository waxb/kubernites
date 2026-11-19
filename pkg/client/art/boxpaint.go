package art

import (
	"image/color"
	"math"
)

var (
	kraftR   = ramp{rgb(104, 74, 48), rgb(150, 110, 72), rgb(188, 148, 100), rgb(216, 182, 132)}
	leatherR = ramp{rgb(58, 32, 30), rgb(90, 50, 40), rgb(122, 72, 52), rgb(156, 102, 70)}
	twineR   = ramp{rgb(150, 132, 100), rgb(196, 180, 146), rgb(232, 220, 186), rgb(248, 242, 222)}
	tapeCol  = rgb(214, 186, 120)
	amberR   = ramp{rgb(150, 76, 20), rgb(206, 112, 30), rgb(240, 156, 52), rgb(254, 206, 110)}
)

type boxCtx struct {
	s      iso
	d      boxDims
	half   float64
	ox, oy float64
}

type facePix struct {
	face     int
	col, row int
	w, h     int
	a, b     float64
}

type boxMat func(p facePix, x, y int) color.NRGBA

func boxAt(c *canvas, ox, oy float64, d boxDims) boxCtx {
	return boxCtx{s: iso{c: c, ox: ox, oy: oy}, d: d, half: d.s / 2, ox: ox, oy: oy}
}

func (bx boxCtx) width() int  { return int(math.Round(bx.d.hw())) }
func (bx boxCtx) height() int { return int(bx.d.h) }

func (bx boxCtx) fill(m boxMat) {
	w, h, ix := bx.width(), bx.height(), int(math.Floor(bx.ox))
	row := func(z float64) int { return min(max(int(math.Floor(z)), 0), h-1) }
	top := func(x, y int) color.NRGBA {
		gx, gy := bx.s.ground(x, y, bx.d.h)
		return m(facePix{face: 0, a: (gx + bx.half) / bx.d.s, b: (gy + bx.half) / bx.d.s, w: w, h: h}, x, y)
	}
	left := func(x, y int) color.NRGBA {
		_, z := bx.s.leftUV(x, y, bx.half)
		return m(facePix{face: 1, col: min(max(ix-1-x, 0), w-1), row: row(z), w: w, h: h}, x, y)
	}
	right := func(x, y int) color.NRGBA {
		_, z := bx.s.rightUV(x, y, bx.half)
		return m(facePix{face: 2, col: min(max(x-ix, 0), w-1), row: row(z), w: w, h: h}, x, y)
	}
	bx.s.box(-bx.half, -bx.half, bx.half, bx.half, 0, bx.d.h, faces{top: top, left: left, right: right})
}

func (bx boxCtx) T(a, b float64) pt {
	return bx.s.p(-bx.half+a*bx.d.s, -bx.half+b*bx.d.s, bx.d.h)
}

func (bx boxCtx) line(p, q pt, col color.NRGBA) {
	bx.s.c.line(int(math.Floor(p.x)), int(math.Floor(p.y)), int(math.Floor(q.x)), int(math.Floor(q.y)), col)
}

func (bx boxCtx) topCross(at float64, col color.NRGBA) {
	bx.line(bx.T(at, 0.04), bx.T(at, 0.96), col)
	bx.line(bx.T(0.04, at), bx.T(0.96, at), col)
}

func (bx boxCtx) dotTop(a, b float64, col color.NRGBA) {
	p := bx.T(a, b)
	bx.s.c.dot(int(math.Floor(p.x)), int(math.Floor(p.y)), col)
}

func sideTone(r ramp, lit bool) color.NRGBA {
	if lit {
		return r[2]
	}
	return r[1]
}

func sideHi(r ramp, lit bool) color.NRGBA {
	if lit {
		return r[3]
	}
	return r[2]
}

func inRect(p facePix, c0, r0, w, h int) bool {
	return p.col >= c0 && p.col < c0+w && p.row >= r0 && p.row < r0+h
}

func labelRect(p facePix) (int, int, int, int) {
	if p.w < 6 {
		return 1, 1, 2, 2
	}
	mid := p.w / 2
	return mid + 1, max(p.h/2-1, 1), max(2, p.w-mid-2), min(2+p.h/10, max(p.h-3, 1))
}

func onLabel(p facePix) bool {
	c0, r0, w, h := labelRect(p)
	return inRect(p, c0, r0, w, h)
}

func labelColor(p facePix, r ramp, lit bool) color.NRGBA {
	_, r0, _, h := labelRect(p)
	if h > 1 && p.row == r0+h-1 {
		return sideHi(r, lit)
	}
	return sideTone(r, lit)
}

var boxMaterials = map[string]func(v int) boxMat{
	"parcel": parcelMat, "ledger": ledgerMat, "crate": crateMat, "paper": paperMat,
}

var boxDecals = map[string]func(bx boxCtx, v int){
	"parcel": parcelDecals, "ledger": ledgerDecals, "crate": crateDecals, "paper": paperDecals,
}
