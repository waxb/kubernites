package art

import (
	"image/color"
	"math"
)

var contactShade = [3]float64{0.3, 0.17, 0.07}

var contactAlpha = [3]uint8{80, 48, 22}

var tuftRate = [AgeTiers]float64{0.3, 0.2, 0.12, 0.05}

func (s iso) groundLayer() *canvas {
	if s.gnd != nil {
		return s.gnd
	}
	return s.c
}

func apronEdge(gx, gy, lo, hi float64, seed int) float64 {
	d := math.Min(math.Min(gx-lo, hi-gx), math.Min(gy-lo, hi-gy))
	cx, cy := math.Max(math.Abs(gx-(lo+hi)/2)-(hi-lo)/2+0.3, 0), math.Max(math.Abs(gy-(lo+hi)/2)-(hi-lo)/2+0.3, 0)
	if cx > 0 && cy > 0 {
		d = 0.3 - math.Hypot(cx, cy)
	}
	return d + (vnoise(gx*3.1, gy*3.1, seed)-0.5)*0.28
}

func (s iso) apron(n int, inset float64) {
	g := s.groundLayer()
	lo, hi, seed := inset, float64(n)-inset, s.lk.seed+101
	top, bot := s.p(lo, lo, 0), s.p(hi, hi, 0)
	for y := int(top.y) - 1; y <= int(bot.y)+1; y++ {
		for x := int(s.p(lo, hi, 0).x) - 1; x <= int(s.p(hi, lo, 0).x)+1; x++ {
			gx, gy := s.ground(x, y, 0)
			d := apronEdge(gx, gy, lo, hi, seed)
			if d < 0 || (d < 0.1 && dither(x, y) > d*10) {
				continue
			}
			g.dot(x, y, s.apronColor(x, y, gx, gy, d))
		}
	}
	s.apronTufts(g, lo, hi, seed)
}

func (s iso) apronColor(x, y int, gx, gy, d float64) color.NRGBA {
	seed := s.lk.seed + 103
	switch s.lk.age {
	case ageDark:
		if vnoise(gx*2.2, gy*2.2, seed) > 0.6 || d < 0.12 {
			return speckle(grassR, x, y, seed, 0.1)
		}
		return speckle(dirtR, x, y, seed, 0.18)
	case ageFeudal:
		if p, dx, dy, ok := pebbleIn(x, y, 7, 4, seed, 0.4); ok && d > 0.15 {
			return stoneShade(plazaR, p, dx, dy)
		}
		return speckle(dirtR, x, y, seed, 0.2)
	case ageCastle:
		if p, dx, dy, ok := pebbleIn(x, y, 5, 3, seed, 0.9); ok && d > 0.08 {
			return stoneShade(stoneR, p, dx, dy)
		}
		return dirtR[1]
	}
	if d < 0.07 {
		return stoneR[1]
	}
	return plazaTex(gx, gy)
}

func (s iso) apronTufts(g *canvas, lo, hi float64, seed int) {
	rate := tuftRate[s.lk.age]
	for i := 0; i < 64; i++ {
		t := float64(i) / 64
		var gx, gy float64
		switch i % 2 {
		case 0:
			gx, gy = lo+(hi-lo)*t, hi+0.02
		default:
			gx, gy = hi+0.02, lo+(hi-lo)*t
		}
		if t < 0.12 || pick(i, 1, seed) > rate*2 {
			continue
		}
		p := s.p(gx, gy, 0)
		tuft(g, int(p.x), int(p.y), grassR, hash(i, 2, seed))
	}
}

func tuft(c *canvas, x, y int, r ramp, h uint32) {
	c.dot(x, y, r[1])
	c.dot(x, y-1, r[3])
	if h%2 == 0 {
		c.dot(x-1, y, r[2])
		c.dot(x, y-2, r[0])
	}
	if h%3 == 0 {
		c.dot(x+1, y, r[2])
		c.dot(x+1, y-1, r[0])
	}
}

func baseRow(body *canvas, x int) (int, bool) {
	for y := body.h - 1; y >= 0; y-- {
		if body.at(x, y).A > 0 {
			return y, true
		}
	}
	return 0, false
}

func grounded(gnd, body *canvas, x, yb int) bool {
	if body.tagAt(x, yb-1) != matNone && body.liftAt(x, yb-1) > 8 {
		return false
	}
	return gnd.at(x, yb+1).A > 0 || gnd.at(x, yb+2).A > 0
}

func contact(gnd, body *canvas) {
	for x := 0; x < body.w; x++ {
		yb, ok := baseRow(body, x)
		if !ok || !grounded(gnd, body, x, yb) {
			continue
		}
		for i := range contactShade {
			y := yb + 1 + i
			if col := gnd.at(x, y); col.A > 0 {
				gnd.img.SetNRGBA(x, y, darken(col, contactShade[i]))
			} else if gnd.in(x, y) {
				gnd.img.SetNRGBA(x, y, color.NRGBA{20, 16, 24, contactAlpha[i]})
			}
		}
	}
}

func footTufts(out, gnd, body *canvas, l look) {
	rate := tuftRate[l.age]
	for x := 0; x < body.w; x++ {
		yb, ok := baseRow(body, x)
		if !ok || !grounded(gnd, body, x, yb) || pick(x, yb, l.seed+107) > rate {
			continue
		}
		h := hash(x, yb, l.seed+109)
		if h%4 == 0 {
			out.dot(x, yb+1, rockR[2])
			if next, ok := baseRow(body, x+1); ok && next == yb {
				out.dot(x+1, yb+1, rockR[1])
			}
			continue
		}
		tuft(out, x, yb, grassR, h)
	}
}
