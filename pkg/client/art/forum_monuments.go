package art

import (
	"fmt"
	"image/color"
	"math"
)

const (
	ForumColonnades = 2
	colonnadeW      = 26
	colonnadeH      = 48
	colonnadeAX     = 13
	colonnadeAY     = 40
	columnTop       = 24.0
	rostraW         = 46
	rostraH         = 34
	rostraAX        = 23
	rostraAY        = 24
	archW           = 46
	archH           = 80
	archAX          = 23
	archAY          = 66
	archBody        = 32.0
	archAttic       = 42.0
	archSpan        = 1.0
	archDepth       = 0.15
	archOpen        = 0.38
	archSpring      = 20.0
)

func ForumColonnadeName(axis string, v int) string {
	return fmt.Sprintf("forum/colonnade/%s/%d", axis, v)
}
func ForumRostraName(axis string) string { return "forum/rostra/" + axis }
func ForumArchName(axis string) string   { return "forum/arch/" + axis }

func addForumMonuments(b *atlasBuilder) {
	for age := ageCastle; age < AgeTiers; age++ {
		aa := age
		b.add(ForumFountainName(age), fountainW, fountainH, fountainAX, fountainAY, func(c *canvas) { paintFountain(c, aa) })
	}
	b.add(ForumPlinthName(ageCastle), statueW, statueH, statueAX, statueAY, func(c *canvas) { paintStatue(c, 0, false) })
	for v := 0; v < ForumStatues; v++ {
		vv := v
		b.add(ForumStatueName(v), statueW, statueH, statueAX, statueAY, func(c *canvas) { paintStatue(c, vv, true) })
	}
	for _, axis := range []string{"x", "y"} {
		ax := axis == "x"
		for v := 0; v < ForumColonnades; v++ {
			vv := v
			b.add(ForumColonnadeName(axis, v), colonnadeW, colonnadeH, colonnadeAX, colonnadeAY, func(c *canvas) { paintColonnade(c, ax, vv) })
		}
		b.add(ForumRostraName(axis), rostraW, rostraH, rostraAX, rostraAY, func(c *canvas) { paintRostra(c, ax) })
		b.add(ForumArchName(axis), archW, archH, archAX, archAY, func(c *canvas) { paintArch(c, ax) })
	}
}

type axisFrame struct {
	s     iso
	along bool
}

func (f axisFrame) at(u, v float64) (float64, float64) {
	if f.along {
		return u, v
	}
	return v, u
}

func (f axisFrame) box(u0, v0, u1, v1, z0, z1 float64, fc faces) {
	x0, y0 := f.at(u0, v0)
	x1, y1 := f.at(u1, v1)
	f.s.box(min(x0, x1), min(y0, y1), max(x0, x1), max(y0, y1), z0, z1, fc)
}

func marbleFaces(r ramp) faces {
	return faces{top: flat(r[3]), left: flat(r[2]), right: flat(r[1])}
}

func paintColonnade(c *canvas, alongX bool, v int) {
	s := isoFrame(c, 0, colonnadeAX, colonnadeAY)
	f := axisFrame{s: s, along: alongX}
	r := marbleR
	if v == 1 {
		r = mixRamp(marbleR, plasterR, 0.35)
	}
	f.box(-0.5, -0.1, 0.5, 0.1, 0, 2, marbleFaces(r))
	for _, u := range []float64{-0.25, 0.25} {
		x, y := f.at(u, 0)
		s.turn(lathe{cx: x, cy: y, z: 2, h: columnTop - 2, r: 0.055}, ribbedMat(r, 211+v), matStone)
		f.box(u-0.08, -0.08, u+0.08, 0.08, columnTop, columnTop+2, marbleFaces(r))
	}
	f.box(-0.5, -0.07, 0.5, 0.07, columnTop+2, columnTop+5, marbleFaces(r))
	f.box(-0.5, -0.1, 0.5, 0.1, columnTop+5, columnTop+7, faces{top: flat(r[3]), left: flat(goldR[2]), right: flat(goldR[1])})
	c.outline(ink)
}

func paintRostra(c *canvas, alongX bool) {
	s := isoFrame(c, 0, rostraAX, rostraAY)
	f := axisFrame{s: s, along: alongX}
	f.box(-0.42, -0.26, 0.42, 0.26, 0, 8, marbleFaces(marbleR))
	for i, z := range []float64{6, 4, 2} {
		d := 0.26 + float64(i+1)*0.06
		f.box(-0.22, d-0.06, 0.22, d, 0, z, marbleFaces(stoneR))
	}
	for _, u := range []float64{-0.34, 0.34} {
		x, y := f.at(u, 0.27)
		p := s.p(x, y, 5)
		c.rect(int(p.x)-1, int(p.y)-1, 3, 2, copperR[2])
		c.dot(int(p.x)+1, int(p.y)+1, copperR[1])
	}
	for u := -0.4; u <= 0.41; u += 0.1 {
		x, y := f.at(u, -0.24)
		p := s.p(x, y, 8)
		c.rect(int(p.x), int(p.y)-3, 1, 3, marbleR[3])
	}
	lx, ly := f.at(0, -0.08)
	f.s.box(lx-0.05, ly-0.05, lx+0.05, ly+0.05, 8, 13, faces{top: flat(woodR[3]), left: flat(woodR[2]), right: flat(woodR[1])})
	c.outline(ink)
	base := s.p(0, 0, 0)
	propShadow(c, base.x+0.5, base.y+2.5, 14)
}

func archOpening(u, z float64) bool {
	if math.Abs(u) < archOpen {
		return z < archSpring+9*math.Sqrt(max(1-(u/archOpen)*(u/archOpen), 0))
	}
	side := math.Abs(u) - 0.72
	return math.Abs(side) < 0.12 && z > 5 && z < 13+3*math.Sqrt(max(1-(side/0.12)*(side/0.12), 0))
}

func archFace(u, z float64, x, y int, lit bool) color.NRGBA {
	r := marbleR
	for _, col := range []float64{0.5, 0.94} {
		if math.Abs(math.Abs(u)-col) < 0.03 && z > 2 && z < archBody-2 {
			return r[3]
		}
	}
	if z > archBody-3 {
		return r[3]
	}
	if lit {
		return tone(r, 0.3, x, y)
	}
	return tone(r, -0.4, x, y)
}

func paintArch(c *canvas, alongX bool) {
	s := isoFrame(c, 0, archAX, archAY)
	f := axisFrame{s: s, along: alongX}
	broad := func(x, y int) color.NRGBA {
		u, z := s.leftUV(x, y, archDepth)
		if !alongX {
			u, z = s.rightUV(x, y, archDepth)
		}
		if archOpening(u, z) {
			return color.NRGBA{}
		}
		return archFace(u, z, x, y, alongX)
	}
	end := flat(marbleR[1])
	fc := faces{top: flat(marbleR[3]), left: broad, right: end}
	if !alongX {
		fc = faces{top: flat(marbleR[3]), left: end, right: broad}
	}
	f.box(-archSpan, -archDepth, archSpan, archDepth, 0, archBody, fc)
	f.box(-archSpan-0.03, -archDepth-0.02, archSpan+0.03, archDepth+0.02, archBody, archAttic, marbleFaces(marbleR))
	for u := -0.5; u <= 0.5; u += 0.08 {
		x, y := f.at(u, archDepth+0.02)
		p := s.p(x, y, archBody+5)
		c.rect(int(p.x), int(p.y), 1, 2, goldR[1+int(u*25)&1])
	}
	qx, qy := f.at(0, 0)
	top := s.p(qx, qy, archAttic)
	c.rect(int(top.x)-5, int(top.y)-5, 10, 4, copperR[1])
	c.rect(int(top.x)-4, int(top.y)-7, 3, 2, copperR[2])
	c.rect(int(top.x)+1, int(top.y)-8, 3, 3, copperR[2])
	c.dot(int(top.x), int(top.y)-10, goldR[3])
	c.outline(ink)
	base := s.p(0, 0, 0)
	propShadow(c, base.x+0.5, base.y+1.5, 18)
}
