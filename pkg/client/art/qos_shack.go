package art

import (
	"image/color"
	"math"
)

const (
	shackX0, shackX1 = 0.24, 0.76
	shackY0, shackY1 = 0.42, 0.78
	shackBack        = 15.0
	shackFront       = 10.0
	shackEave        = 0.04
	pipeX0, pipeX1   = 0.32, 0.38
	pipeY0, pipeY1   = 0.52, 0.58
	pipeTop          = 20.0
	doorU0, doorU1   = 0.44, 0.58
	doorH            = 7.5
)

var shackWood = ramp{rgb(82, 60, 44), rgb(140, 108, 74), rgb(184, 146, 98), rgb(214, 180, 128)}

func weathered(u, z float64, lit bool, seed int) color.NRGBA {
	base := shackWood[1]
	if lit {
		base = shackWood[2]
	}
	plank := int(math.Floor(u * 16))
	switch {
	case plank%3 == 0:
		return shackWood[0]
	case noise(plank, int(z), seed) > 0.9:
		return shackWood[3]
	case noise(plank, int(z/3), seed+1) > 0.8:
		return mix(base, greyR[2], 0.35)
	}
	return base
}

var hutShackFooting = [AgeTiers]float64{0, 3, 6, 2}

func hutShackPlanks(age, seed int) mat {
	white, stones, boards := plankMat(plasterR, seed), rubbleMat(stoneR, seed), plankMat(woodR, seed+1)
	return func(p surf) color.NRGBA {
		plank := int(math.Floor(p.u / faceU * 16))
		switch {
		case age >= ageFeudal && p.v < hutShackFooting[age]:
			return stones(p)
		case age == ageEmpire:
			return white(p)
		case age == ageFeudal:
			return boards(p)
		case age == ageCastle && plank%3 != 0 && hash(plank/3, 1, seed)%4 == 0:
			return rampAt(woodR, level(p.light))
		case p.v < 2 && dither(p.x, p.y) < (2-p.v)/3:
			return shackWood[0]
		}
		return weathered(p.u/faceU, p.v, p.light > 0, seed+boolInt(p.light > 0)*2)
	}
}

func shackRoofAt(gx float64) float64 {
	t := (gx - shackX0) / (shackX1 - shackX0)
	return shackBack + (shackFront-shackBack)*t
}

func paintVagrant(s iso, h hut, lit bool) {
	g := s.hutGround()
	if s.lk.age >= ageCastle {
		g.apron(1, 0.16)
	} else {
		trodden(g, 10, 4)
	}
	hutShackGround(g)
	if s.lk.variant == 2 {
		s.hutWashLine([2]float64{0.1, 0.3}, [2]float64{0.1, 0.78})
	}
	walls := hutShackPlanks(s.lk.age, s.lk.seed)
	s.pen(matTrim, func() {
		s.quad(s.p(shackX0, shackY1, 0), s.p(shackX1, shackY1, 0), s.p(shackX1, shackY1, shackFront), s.p(shackX0, shackY1, shackBack), s.leftMat(shackY1, walls))
		s.quad(s.p(shackX1, shackY0, 0), s.p(shackX1, shackY1, 0), s.p(shackX1, shackY1, shackFront), s.p(shackX1, shackY0, shackFront), s.rightMat(shackX1, walls))
	})
	shackDoor(s, lit)
	hutShackWindow(s, lit)
	shackRoof(s, tentRamp(h.tier))
	stovepipe(s)
	hutShackProps(s)
}

func hutShackGround(g iso) {
	switch g.lk.age {
	case ageFeudal:
		for _, p := range [][2]float64{{0.52, 0.86}, {0.6, 0.92}} {
			q := g.p(p[0], p[1], 0)
			g.c.ellipse(q.x, q.y, 2.2, 1.1, plazaR[2])
			g.c.dot(int(q.x)-1, int(q.y)-1, plazaR[3])
		}
	case ageCastle, ageEmpire:
		for i := 0; i < 5; i++ {
			q := g.p(0.46+float64(i)*0.05, 0.84+float64(i%2)*0.05, 0)
			g.c.rect(int(q.x)-1, int(q.y), 3, 1, []color.NRGBA{stoneR[2], plazaR[3]}[g.lk.age-ageCastle])
		}
	}
}

func hutShackProps(s iso) {
	switch s.lk.variant {
	case 0:
		sack(s.c, s.p(0.24, 0.9, 0), rgb(150, 124, 88))
	case 1:
		s.hutWoodpile(0.86, 0.36)
		s.hutBucket(0.22, 0.92)
	default:
		s.hutProp(s.lk.pick(3, 4), 0.86, 0.4)
	}
}

func shackDoor(s iso, lit bool) {
	s.onLeft(shackY1, doorU0-0.03, doorU1+0.03, 0, doorH+1.5, flat(shackWood[0]))
	s.onLeft(shackY1, doorU0, doorU1, 0, doorH, flat(inkSft))
	s.onLeft(shackY1, doorU0, doorU0+0.05, 1.5, doorH, flat(rgb(150, 124, 88)))
	if s.lk.age >= ageCastle {
		s.onLeft(shackY1, doorU1-0.05, doorU1, 0, doorH, flat(s.lk.id[1]))
	}
	if lit {
		g := s.p(doorU0+0.09, shackY1, 1)
		s.c.dot(int(g.x), int(g.y), fireR[1])
		s.c.dot(int(g.x)+1, int(g.y)-1, fireR[2])
	}
}

func hutShackWindow(s iso, lit bool) {
	if s.lk.age == ageEmpire {
		s.paint(sideR(shackX1), matTrim, 0.49, 0.64, 4, 9, plasterR[3])
		s.pen(matGlass, func() { s.onRight(shackX1, 0.51, 0.62, 5, 8, paneGlass(lit)) })
		s.paint(sideR(shackX1), matNone, 0.48, 0.65, 3, 4, woodR[1])
		return
	}
	s.pen(matGlass, func() { s.onRight(shackX1, 0.5, 0.62, 5, 8, flat(ink)) })
	if s.lk.age == ageCastle {
		s.paint(sideR(shackX1), matTrim, 0.62, 0.67, 4, 9, woodR[2])
	}
	if lit {
		w := s.p(shackX1, 0.56, 6)
		s.c.dot(int(w.x), int(w.y), fireR[2])
	}
}

func shackRoof(s iso, r ramp) {
	x0, x1 := shackX0-shackEave, shackX1+shackEave
	y0, y1 := shackY0-shackEave, shackY1+shackEave
	s.roofShed(x0, y0, x1, y1, shackFront-1, shackBack+1, false, hutRoofMat(s.lk.age, r, s.lk.seed+17))
	s.pen(matRoof, func() {
		s.quad(s.p(x0, y1, shackBack+1), s.p(x1, y1, shackFront-1), s.p(x1, y1, shackFront-2), s.p(x0, y1, shackBack), flat(r[0]))
	})
	switch s.lk.age {
	case ageDark, ageFeudal:
		for _, gx := range []float64{shackX0 + 0.14, shackX0 + 0.3} {
			a, b := s.p(gx, y0+0.02, shackRoofAt(gx)+1.5), s.p(gx, y1-0.02, shackRoofAt(gx)+1.5)
			s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), shackWood[0])
		}
	case ageCastle:
		for i, p := range [][2]float64{{0.5, 0.5}, {0.62, 0.7}} {
			q := s.p(p[0], p[1], shackRoofAt(p[0])+1.5)
			s.c.rect(int(q.x)-1, int(q.y)-1, 2, 2, greyR[1+i%2])
		}
	}
}

func stovepipe(s iso) {
	base := shackRoofAt(pipeX1) - 1
	s.box(pipeX0, pipeY0, pipeX1, pipeY1, base, pipeTop, faces{top: flat(ink), left: flat(greyR[2]), right: flat(greyR[1])})
	rim := s.p((pipeX0+pipeX1)/2, (pipeY0+pipeY1)/2, pipeTop)
	cap := greyR[3]
	if s.lk.age == ageEmpire {
		cap = goldR[2]
	}
	s.c.rect(int(rim.x)-2, int(rim.y)-1, 4, 1, cap)
}

func shackRidge(s iso) ridge {
	return ridge{s.p(shackX0+0.04, shackY0, shackBack+1), s.p(shackX0+0.04, shackY1, shackBack+1)}
}

func shackRubble(s iso, h hut) {
	trodden(s.hutGround(), 13, 5.5)
	roofSlab(s, tentRamp(h.tier), 0.28, 0.34)
	for i, l := range [][6]float64{{0.22, 0.78, 1, 0.7, 0.86, 2}, {0.3, 0.3, 1, 0.12, 0.62, 6}, {0.74, 0.34, 1, 0.9, 0.7, 5}} {
		a, b := s.p(l[0], l[1], l[2]), s.p(l[3], l[4], l[5])
		s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), shackWood[1+i%2])
		s.c.line(int(a.x), int(a.y)-1, int(b.x), int(b.y)-1, shackWood[2+i%2])
	}
	p := s.p(0.66, 0.44, 0)
	s.c.rect(int(p.x)-1, int(p.y)-4, 3, 4, greyR[1])
}

func trodden(s iso, rx, ry float64) {
	c := s.p(0.5, 0.5, 0)
	for y := int(c.y - ry - 1); y <= int(c.y+ry+1); y++ {
		for x := int(c.x - rx - 1); x <= int(c.x+rx+1); x++ {
			dx, dy := (float64(x)+0.5-c.x)/rx, (float64(y)+0.5-c.y)/ry
			if dx*dx+dy*dy > 1-noise(x, y, 17)*0.12 {
				continue
			}
			col := dirtR[2]
			switch n := noise(x, y, 23); {
			case n > 0.86:
				col = dirtR[3]
			case n < 0.18:
				col = dirtR[1]
			}
			s.c.dot(x, y, col)
		}
	}
}

func stitch(c *canvas, a, b pt, col color.NRGBA) {
	n := int(max(absf(b.x-a.x), absf(b.y-a.y)))
	for i := 0; i <= n; i += 2 {
		t := float64(i) / float64(max(n, 1))
		c.dot(int(a.x+(b.x-a.x)*t), int(a.y+(b.y-a.y)*t), col)
	}
}
