package art

import "image/color"

func ledgerMat(v int) boxMat {
	leather := leatherR
	if v == 3 {
		leather = shiftRamp(leatherR, 0.12)
	}
	return func(p facePix, x, y int) color.NRGBA {
		if p.face == 0 {
			return ledgerTop(p, leather)
		}
		lit := p.face == 1
		mid := p.w / 2
		switch {
		case (p.col == 0 || p.col == p.w-1) && (p.row == 0 || p.row >= p.h-2):
			return sideTone(goldR, lit)
		case p.col == 0:
			return sideHi(leather, lit)
		case lit && ledgerPlate(p):
			return labelColor(p, tealR, lit)
		case v == 1 && (p.col == max(p.w/4, 1) || p.col == max(3*p.w/4, 2)):
			if p.row == p.h/2 {
				return goldR[2]
			}
			return leather[0]
		case v == 2 && (p.row == 1 || p.row == p.h-3):
			return sideHi(greyR, lit)
		case p.row == p.h-2:
			return leather[0]
		case p.row == p.h-1:
			return sideHi(leather, lit)
		case p.col == mid && p.row == p.h/2 && v == 2:
			return goldR[3]
		}
		return sideTone(leather, lit)
	}
}

func ledgerPlate(p facePix) bool {
	w := max(p.w/2, 2)
	return inRect(p, (p.w-w)/2+p.w%2, max(p.h/2-2, 1), w, min(2+p.h/10, max(p.h-3, 1)))
}

func ledgerTop(p facePix, leather ramp) color.NRGBA {
	edgeA, edgeB := p.a < 0.18 || p.a > 0.82, p.b < 0.18 || p.b > 0.82
	if edgeA && edgeB {
		return goldR[3]
	}
	if edgeA || edgeB {
		return leather[2]
	}
	return leather[3]
}

func ledgerDecals(bx boxCtx, v int) {
	if v != 3 {
		return
	}
	spines := []color.NRGBA{redR[2], tealR[3], goldR[2], blueR[2], greenR[2]}
	n := min(max(bx.width()/3, 2), len(spines))
	for i := 0; i < n; i++ {
		a := 0.22 + 0.56*float64(i)/float64(max(n-1, 1))
		bx.line(bx.T(a, 0.22), bx.T(a, 0.78), spines[i])
	}
}

func crateMat(v int) boxMat {
	return func(p facePix, x, y int) color.NRGBA {
		if p.face == 0 {
			return crateTop(p, v)
		}
		lit := p.face == 1
		switch {
		case p.col == 0 || p.col == p.w-1 || p.row == p.h-1:
			return sideHi(crateR, lit)
		case v == 1 && (p.row == p.h/2 || p.row == p.h/2-1) && p.h >= 7:
			return sideHi(crateR, lit)
		case v == 3 && p.row == p.h/2:
			return sideHi(ropeR, lit)
		case lit && crateStencil(p, v):
			return sideTone(amberR, lit)
		case v == 2 && p.col%3 == 2:
			return crateR[0]
		case v != 2 && p.row%3 == 2:
			return crateR[0]
		}
		if hash(p.row/3, p.col/3, 91+v)%4 == 0 {
			return darken(sideTone(crateR, lit), 0.08)
		}
		return sideTone(crateR, lit)
	}
}

var stencilArrow = []string{".#.", "###", ".#.", ".#."}

func crateStencil(p facePix, v int) bool {
	side := 2 + p.w/12
	switch {
	case v == 0:
		return p.col > 0 && p.col < p.w-1 && (p.row == p.h/2 || (p.h >= 7 && p.row == p.h/2-1))
	case v != 2 || p.w < 8:
		return inRect(p, (p.w-side)/2+p.w%2, max(p.h/2-side/2, 1), side, side)
	}
	c0, r0 := p.w/2, p.h/2+1
	i, j := p.col-c0, r0-p.row
	if j < 0 || j >= len(stencilArrow) || i < 0 || i >= 3 {
		return false
	}
	return stencilArrow[j][i] == '#'
}

func crateTop(p facePix, v int) color.NRGBA {
	u := p.b
	if v == 2 {
		u = p.a
	}
	n := float64(max(p.w/3, 2))
	f := u*n - float64(int(u*n))
	switch {
	case f < 0.14:
		return crateR[1]
	case p.a < 0.1 || p.a > 0.9 || p.b < 0.1 || p.b > 0.9:
		return crateR[3]
	}
	return crateR[2]
}

func crateDecals(bx boxCtx, v int) {
	if v == 3 {
		bx.topCross(0.5, ropeR[2])
	}
}
