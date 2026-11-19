package art

import (
	"image/color"
	"math"
)

type portMat struct {
	wood, stone, metal ramp
}

var (
	driftR  = ramp{rgb(72, 60, 52), rgb(104, 88, 72), rgb(136, 116, 92), rgb(166, 146, 116)}
	tarR    = ramp{rgb(44, 34, 32), rgb(70, 54, 44), rgb(98, 76, 58), rgb(132, 106, 80)}
	limedR  = ramp{rgb(148, 126, 100), rgb(194, 170, 134), rgb(224, 202, 162), rgb(244, 230, 194)}
	rubbleR = ramp{rgb(96, 88, 84), rgb(134, 124, 112), rgb(170, 158, 138), rgb(202, 190, 164)}
	ironR   = ramp{rgb(40, 38, 46), rgb(62, 60, 70), rgb(96, 94, 104), rgb(140, 138, 146)}
)

var portMats = [AgeTiers]portMat{
	{wood: driftR, stone: rockR, metal: woodR},
	{wood: pierR, stone: rubbleR, metal: ironR},
	{wood: tarR, stone: stoneR, metal: ironR},
	{wood: limedR, stone: marbleR, metal: goldR},
}

func portMatOf(age int) portMat { return portMats[clampAge(age)] }

func (l PortLook) seed() int { return l.Level*131 + l.Age*17 + l.Variant*7 }

func boardTop(l PortLook, gx, gy float64) color.NRGBA {
	m := portMatOf(l.Age)
	rows := 5.0
	gap := 0.2
	if l.Level == 1 {
		rows, gap = 4, 0.3
	}
	row := int(math.Floor(gy * rows))
	f := gy*rows - math.Floor(gy*rows)
	if f < gap {
		return gapColor(l, m)
	}
	u := gx*2 + float64(hash(row, l.Variant, 77)%3)/3
	fu := u - math.Floor(u)
	switch {
	case fu < 0.06:
		return m.wood[0]
	case l.Age >= ageCastle && fu < 0.12 && f > 0.5 && f < 0.72:
		return m.metal[2]
	case l.Age == ageDark && hash(row, int(math.Floor(u*5)), 29)%9 == 0:
		return m.wood[0]
	}
	return plankTone(m.wood, row, int(math.Floor(u)), l.Variant)
}

func gapColor(l PortLook, m portMat) color.NRGBA {
	if l.Level == 1 {
		return darken(sandR[0], 0.35)
	}
	return m.wood[0]
}

func plankTone(r ramp, row, col, v int) color.NRGBA {
	switch hash(row, col, 53+v) % 5 {
	case 0:
		return r[3]
	case 1:
		return r[1]
	}
	return r[2]
}

func flagTop(l PortLook, gx, gy float64) color.NRGBA {
	m := portMatOf(l.Age)
	if l.Level >= PortLevels {
		if c, ok := curbBand(l, m, gx, gy); ok {
			return c
		}
	}
	rows, w := 3.0, 0.28+float64(hash(int(math.Floor(gy*3)), l.Variant, 41)%3)*0.08
	if l.Age == ageDark {
		rows, w = 5, 0.16+float64(hash(int(math.Floor(gy*5)), l.Variant, 41)%3)*0.04
	}
	row := int(math.Floor(gy * rows))
	fv := gy*rows - math.Floor(gy*rows)
	u := gx/w + float64(hash(row, 3, 43)%5)/5
	fu := u - math.Floor(u)
	if fv < 0.12 || fu < 0.08*rows/3 {
		return flagMortar(l, m)
	}
	j := (float64(hash(row, int(math.Floor(u)), 47+l.Variant)%3) - 1) * 0.6
	return rampAt(m.stone, 2+j)
}

func flagMortar(l PortLook, m portMat) color.NRGBA {
	if l.Age == ageDark {
		return dirtR[0]
	}
	return mix(m.stone[0], m.stone[1], 0.4)
}

func curbBand(l PortLook, m portMat, gx, gy float64) (color.NRGBA, bool) {
	edge := min(gx, gy)
	switch {
	case l.Age == ageEmpire && edge < 0.04:
		return goldR[2], true
	case edge < 0.1:
		return m.stone[1], true
	}
	return color.NRGBA{}, false
}

func deckTop(l PortLook, gx, gy float64) color.NRGBA {
	if l.stone() {
		return flagTop(l, gx, gy)
	}
	return boardTop(l, gx, gy)
}

func portFace(l PortLook, lit bool) shader {
	m := portMatOf(l.Age)
	return func(x, y int) color.NRGBA {
		if l.stone() {
			return stoneFace(m.stone, x, y, lit)
		}
		base, line := m.wood[1], m.wood[0]
		if !lit {
			base, line = m.wood[0], darken(m.wood[0], 0.25)
		}
		if y%3 == 0 {
			return line
		}
		return base
	}
}

func stoneFace(r ramp, x, y int, lit bool) color.NRGBA {
	base, mortar := r[1], r[0]
	if lit {
		base, mortar = r[2], r[1]
	}
	if y%4 == 0 || (x+(y/4)%2*3)%6 == 0 {
		return mortar
	}
	return base
}

func paintPortDeck(c *canvas, l PortLook) {
	s := tileIso(c)
	top := func(x, y int) color.NRGBA {
		gx, gy := s.ground(x, y, HavenDeck)
		return deckTop(l, gx, gy)
	}
	s.box(0, 0, 1, 1, 0, HavenDeck, faces{top: top, left: portFace(l, true), right: portFace(l, false)})
	if l.Level == 1 && l.Variant == 1 {
		p := s.p(0.62, 0.38, HavenDeck)
		s.c.ellipse(p.x, p.y, 3, 1.4, darken(sandR[1], 0.2))
	}
	c.outline(ink)
	trimFarJoin(c, s, func(gx, gy float64) bool { return gx < 0 || gy < 0 })
}
