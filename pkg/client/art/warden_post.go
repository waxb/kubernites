package art

import "image/color"

var postBoxR = purpleR

func postFrame(c *canvas) iso {
	return isoFrame(c, 1, WardenPostAX, WardenPostAY+8)
}

func inked(c *canvas, paint func(c *canvas)) {
	body := newCanvas(c.w, c.h)
	paint(body)
	body.outline(ink)
	c.paste(body, 0, 0, false)
}

func paintWardenPost(c *canvas, state string) {
	yardSpot(postFrame(c))
	inked(c, func(c *canvas) { paintPostBody(c, state) })
}

func paintPostBody(c *canvas, state string) {
	s := postFrame(c)
	switch state {
	case "absent":
		yardStake(s)
	case "starting":
		postBody(s)
		postLantern(s, false)
		layoutStakes(s)
	default:
		postBody(s)
		postRoof(s)
		postSatchel(s)
		postLantern(s, true)
		postPennant(s)
	}
}

func layoutStakes(s iso) {
	for _, g := range [][2]float64{{0.2, 0.72}, {0.78, 0.26}} {
		p := s.p(g[0], g[1], 0)
		s.c.rect(int(p.x), int(p.y)-4, 1, 4, woodR[2])
		s.c.dot(int(p.x), int(p.y)-4, redR[2])
	}
}

func yardSpot(s iso) {
	g := s.p(0.5, 0.5, 0)
	s.c.ellipse(g.x, g.y, 9, 4, dirtR[1])
	s.c.ellipse(g.x-0.5, g.y-0.5, 7.5, 3, dirtR[2])
	for i := 0; i < 5; i++ {
		h := hash(i, 3, 211)
		x, y := int(g.x)-6+int(h%12), int(g.y)-2+int(h/12%4)
		s.c.dot(x, y, []color.NRGBA{dirtR[3], dirtR[0], sandR[2]}[h%3])
	}
}

func yardStake(s iso) {
	a, b := s.p(0.5, 0.5, 0), s.p(0.5, 0.5, 7)
	s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[2])
	s.c.dot(int(a.x)+1, int(a.y)-1, woodR[0])
	s.c.dot(int(b.x), int(b.y), postBoxR[2])
	s.c.dot(int(b.x)+1, int(b.y), postBoxR[3])
}

func postBody(s iso) {
	s.box(0.46, 0.46, 0.54, 0.54, 0, 11, faces{top: flat(woodR[3]), left: flat(woodR[2]), right: flat(woodR[1])})
	s.box(0.34, 0.36, 0.66, 0.64, 11, 17, faces{top: flat(postBoxR[3]), left: flat(postBoxR[2]), right: flat(postBoxR[1])})
	slot0, slot1 := s.p(0.4, 0.64, 16), s.p(0.56, 0.64, 16)
	s.c.line(int(slot0.x), int(slot0.y), int(slot1.x), int(slot1.y), ink)
	band0, band1 := s.p(0.34, 0.64, 12), s.p(0.66, 0.64, 12)
	s.c.line(int(band0.x), int(band0.y), int(band1.x), int(band1.y), goldR[1])
}

func postRoof(s iso) {
	s.pyramid(0.3, 0.32, 0.7, 0.68, 17, 5, slateR)
}

func postSatchel(s iso) {
	p := s.p(0.3, 0.64, 10)
	x, y := int(p.x), int(p.y)
	s.c.rect(x-3, y, 4, 4, dirtR[1])
	s.c.rect(x-3, y, 4, 1, dirtR[2])
	s.c.dot(x-2, y+1, goldR[2])
	s.c.line(x-3, y, x, y-3, ropeR[1])
}

func postLantern(s iso, lit bool) {
	arm0, arm1 := s.p(0.66, 0.5, 10), s.p(0.84, 0.5, 10)
	s.c.line(int(arm0.x), int(arm0.y), int(arm1.x), int(arm1.y), woodR[1])
	x, y := int(arm1.x), int(arm1.y)
	s.c.dot(x, y+1, greyR[1])
	glow := glassR[1]
	if lit {
		glow = lampLit
		s.c.disc(float64(x)+0.5, float64(y)+4, 3.2, withAlpha(lampLit, 60))
	}
	s.c.rect(x-1, y+2, 3, 3, glow)
	s.c.rect(x-1, y+5, 3, 1, greyR[0])
	s.c.dot(x, y+2, greyR[2])
}

func postPennant(s iso) {
	top, tip := s.p(0.5, 0.5, 22), s.p(0.5, 0.5, 27)
	s.c.line(int(top.x), int(top.y), int(tip.x), int(tip.y), woodR[2])
	x, y := int(tip.x), int(tip.y)
	s.c.rect(x+1, y, 4, 1, postBoxR[2])
	s.c.rect(x+1, y+1, 3, 1, postBoxR[1])
	s.c.dot(x+1, y+2, postBoxR[1])
}
