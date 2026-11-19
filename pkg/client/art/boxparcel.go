package art

import (
	"image/color"
	"math"
)

var (
	pineR   = ramp{rgb(146, 112, 70), rgb(196, 160, 106), rgb(228, 198, 146), rgb(244, 226, 184)}
	battenR = ramp{rgb(96, 68, 40), rgb(132, 96, 58), rgb(160, 120, 74), rgb(190, 150, 98)}
)

func onBatten(p facePix) bool {
	w, h := float64(p.w-1), float64(p.h-1)
	lim := math.Hypot(w, h) * 0.5
	if p.w >= 8 {
		lim *= 1.7
	}
	d := math.Abs(float64(p.col)*h - float64(p.row)*w)
	e := math.Abs((w-float64(p.col))*h - float64(p.row)*w)
	return d <= lim || e <= lim
}

func parcelNail(p facePix) bool {
	if p.w < 6 {
		return false
	}
	edgeC, edgeR := p.col == 1 || p.col == p.w-2, p.row == 1 || p.row == p.h-2
	return edgeC && edgeR
}

func parcelMat(v int) boxMat {
	return func(p facePix, x, y int) color.NRGBA {
		if p.face == 0 {
			return parcelLid(p, v)
		}
		lit := p.face == 1
		switch {
		case v == 3 && p.h >= 7 && p.row == p.h/2:
			return sideHi(ropeR, lit)
		case v == 1 && parcelNail(p):
			return inkSft
		case onBatten(p) && p.w < 6:
			return sideHi(battenR, lit)
		case onBatten(p):
			return sideTone(battenR, lit)
		case v == 2 && lit && p.h >= 7 && p.row == 1 && abs(p.col-p.w/2) <= max(p.w/6, 1):
			return sideTone(blueR, lit)
		case p.row == 0:
			return sideTone(pineR, false)
		case (p.col+v)%4 == 3 && v != 1:
			return darken(sideTone(pineR, lit), 0.07)
		}
		return sideTone(pineR, lit)
	}
}

func parcelLid(p facePix, v int) color.NRGBA {
	switch {
	case p.b > 0.8:
		return blueR[2]
	case p.b > 0.72:
		return pineR[2]
	}
	f := p.a*3 - math.Floor(p.a*3)
	if f < 0.12 && v != 3 {
		return pineR[2]
	}
	return pineR[3]
}

func parcelDecals(bx boxCtx, v int) {
	if v == 3 {
		bx.topCross(0.5, ropeR[2])
	}
}
