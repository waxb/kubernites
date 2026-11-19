package art

import (
	"image/color"
	"math"
)

var windowHeight = [AgeTiers]int{4, 5, 8, 9}

func (t towerSpec) facing(deg float64) (float64, float64, float64) {
	a := deg * math.Pi / 180
	c, s := math.Cos(a), math.Sin(a)
	return c, s, (c + s) / math.Sqrt2
}

func (s iso) towerWindow(t towerSpec, deg, z float64) {
	c, sn, f := t.facing(deg)
	if f <= 0.08 {
		return
	}
	if s.shapeOf(t) == shapeSquare {
		if sn >= c {
			s.window(sideL(t.cy+t.r), clampf(t.cx+t.r*c*1.2-0.07, t.cx-t.r+0.05, t.cx+t.r-0.2), z)
			return
		}
		s.window(sideR(t.cx+t.r), clampf(t.cy+t.r*sn*1.2-0.07, t.cy-t.r+0.05, t.cy+t.r-0.2), z)
		return
	}
	sp, w := s.roundPane(t, deg, z)
	s.slit(sp, w, windowHeight[s.lk.age], s.litWindow(deg, z))
}

func (s iso) roundPane(t towerSpec, deg, z float64) (span, int) {
	o := s.opening(t, deg, z)
	w := 1 + int(math.Round(2*o.facing))
	return o.at(w), w
}

type opening struct {
	x, y, slope, facing float64
}

type span struct {
	x, y  int
	slope float64
	cx    float64
}

func (o opening) at(w int) span {
	x := int(math.Round(o.x)) - w/2
	return span{x: x, y: int(math.Round(o.y)), slope: o.slope, cx: float64(x) + float64(w)/2}
}

func upright(x, y int) span { return span{x: x, y: y} }

func (sp span) drop(x int) int {
	return int(math.Round(sp.slope * (float64(x) + 0.5 - sp.cx)))
}

func (s iso) opening(t towerSpec, deg, z float64) opening {
	a := deg * math.Pi / 180
	dist := t.r * 0.97
	if s.shapeOf(t) == shapeOct {
		a = math.Round(a/(math.Pi/4)) * (math.Pi / 4)
		dist = octApothem(t.r, a) * 0.99
	}
	c, sn := math.Cos(a), math.Sin(a)
	p := s.p(t.cx+dist*c, t.cy+dist*sn, z)
	return opening{x: p.x, y: p.y, slope: wallSlope(a), facing: (c + sn) / math.Sqrt2}
}

func octApothem(r, a float64) float64 {
	if math.Abs(math.Sin(2*a)) > 0.5 {
		return r / 1.08
	}
	return r
}

func wallSlope(a float64) float64 {
	tx, ty := -math.Sin(a), math.Cos(a)
	dx, dy := 16*(tx-ty), 8*(tx+ty)
	if math.Abs(dx) < 1e-6 {
		return 0
	}
	return clampf(dy/dx, -1.2, 1.2)
}

func clampf(v, lo, hi float64) float64 { return min(max(v, lo), hi) }

func (s iso) slit(sp span, w, h int, lit bool) {
	frame, glassSh := ink, paneGlass(lit)
	switch s.lk.age {
	case ageDark:
		frame, glassSh = woodR[0], emberGlass(lit)
	case ageEmpire:
		frame = goldR[1]
	}
	x, y := sp.x, sp.y
	s.pen(matTrim, func() {
		s.column(sp, x-1, x+w+1, y-h-1, h+2, frame)
		if s.lk.age >= ageCastle {
			s.column(sp, x, x+w, y-h-2, 1, frame)
		}
		if s.lk.age == ageFeudal {
			s.column(sp, x-2, x-1, y-h, h, s.lk.id[1])
			s.column(sp, x+w+1, x+w+2, y-h, h, s.lk.id[2])
		}
	})
	s.pen(matGlass, func() {
		for xx := x; xx < x+w; xx++ {
			d := sp.drop(xx)
			for yy := y - h; yy < y; yy++ {
				s.c.dot(xx, yy+d, glassSh(xx, yy))
			}
		}
	})
}

func (s iso) column(sp span, x0, x1, top, h int, clr color.NRGBA) {
	for xx := x0; xx < x1; xx++ {
		s.c.rect(xx, top+sp.drop(xx), 1, h, clr)
	}
}

func (s iso) towerDoor(t towerSpec, deg, h float64) {
	c, sn, _ := t.facing(deg)
	if s.shapeOf(t) == shapeSquare {
		if sn >= c {
			s.doorway(sideL(t.cy+t.r), clampf(t.cx+t.r*c*1.2-0.15, t.cx-t.r+0.05, t.cx+t.r-0.35), 0.3, h)
			return
		}
		s.doorway(sideR(t.cx+t.r), clampf(t.cy+t.r*sn*1.2-0.15, t.cy-t.r+0.05, t.cy+t.r-0.35), 0.3, h)
		return
	}
	o := s.opening(t, deg, 0)
	w, hh := 2+int(math.Round(3*o.facing)), int(h*s.kit().scale)
	sp := o.at(w)
	x, y := sp.x, sp.y
	frame := stoneR[3]
	if s.lk.age == ageEmpire {
		frame = goldR[1]
	}
	s.pen(matTrim, func() {
		s.column(sp, x-1, x+w+1, y-hh-1, hh+1, frame)
		s.column(sp, x, x+w, y-hh-2, 1, frame)
		s.column(sp, x, x+w, y-hh, hh, woodR[1])
		for xx := x; xx < x+w; xx += 2 {
			s.column(sp, xx, xx+1, y-hh, hh, woodR[0])
		}
		s.c.dot(x+w-1, y-hh/2+sp.drop(x+w-1), goldR[2])
	})
}
