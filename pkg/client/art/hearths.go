package art

import "image/color"

var strawR = ramp{rgb(132, 96, 40), rgb(180, 138, 58), rgb(214, 176, 84), rgb(240, 214, 128)}

func addHearths(b *atlasBuilder) {
	s := hearthSize()
	for t := 1; t <= 3; t++ {
		for _, st := range HearthStates {
			tier, state := t, st
			b.add(hearthName(t, st), s.w, s.h, s.ax, s.ay, func(c *canvas) { paintHearth(c, s, tier, state) })
		}
	}
}

func hearthFrame(c *canvas, s size) iso {
	return isoFrame(c, 1, s.ax, s.ay)
}

func paintHearth(c *canvas, s size, tier int, state string) {
	switch state {
	case "pending":
		paintPendingTent(hearthFrame(c, s))
		c.outline(ink)
		return
	case "kindling":
		paintKindling(c, s, tier)
		return
	case "banished":
		paintRubble(hearthFrame(c, s), tier)
		c.outline(ink)
		c.mapPixels(func(col color.NRGBA) color.NRGBA { return mix(col, grey(col), 0.5) })
		return
	}
	house := newCanvas(s.w, s.h)
	paintHouse(hearthFrame(house, s), tier, state == "guttering")
	house.outline(ink)
	if state == "guttering" {
		roofFire(house, tier)
	}
	if state == "ghost" {
		house.mapPixels(func(col color.NRGBA) color.NRGBA {
			return withAlpha(mix(grey(col), rgb(170, 200, 230), 0.4), 130)
		})
	}
	c.paste(house, 0, 0, false)
}

func paintHouse(s iso, tier int, burning bool) {
	lit := !burning
	switch tier {
	case 1:
		s.foundation(1, dirtR, 0.12)
		s.box(0.22, 0.28, 0.8, 0.8, 2, 13, faces{top: flat(plasterR[3]), left: s.plankLeft(woodR, 0.8), right: s.plankRight(woodR, 0.8)})
		s.gableX(0.22, 0.28, 0.8, 0.8, 13, 9, 0.08, strawR, woodR[2])
		s.doorL(0.8, 0.45, 0.14, 8)
		s.windowR(0.8, 0.42, 6, lit)
		chimney(s, 0.35, 0.4, 17, 6)
	case 2:
		s.foundation(1, stoneR, 0.08)
		timberUpper(s, 0.14, 0.2, 0.86, 0.84, 2, 16)
		s.gableX(0.14, 0.2, 0.86, 0.84, 16, 11, 0.08, redR, plasterR[1])
		s.doorL(0.84, 0.44, 0.14, 9)
		s.windowL(0.84, 0.2, 8, lit)
		s.windowR(0.86, 0.45, 8, lit)
		chimney(s, 0.7, 0.35, 21, 7)
	default:
		s.foundation(1, stoneR, 0.05)
		s.stoneBox(0.1, 0.15, 0.9, 0.88, 2, 12, stoneR, stoneTop(), 30)
		timberUpper(s, 0.1, 0.15, 0.9, 0.88, 12, 24)
		s.gableY(0.1, 0.15, 0.9, 0.88, 24, 12, 0.08, blueR, plasterR[1])
		s.doorL(0.88, 0.2, 0.14, 9)
		s.windowL(0.88, 0.55, 15, lit)
		s.windowR(0.9, 0.3, 15, lit)
		s.windowR(0.9, 0.6, 4, lit)
		chimney(s, 0.25, 0.35, 30, 7)
	}
}

func chimney(s iso, gx, gy, z, h float64) {
	s.stoneBox(gx, gy, gx+0.12, gy+0.12, z-4, z+h, stoneR, flat(inkSft), 31)
	top := s.p(gx+0.06, gy+0.06, z+h)
	s.c.dot(int(top.x), int(top.y)-3, withAlpha(greyR[3], 200))
	s.c.dot(int(top.x)+1, int(top.y)-5, withAlpha(greyR[2], 170))
}

func roofFire(c *canvas, tier int) {
	top := []int{0, 21, 16, 8}[tier]
	for i, x := range []int{12, 16, 20, 24} {
		h := 5 + (i*3)%4
		base := top + 6 + i%2
		c.shadePoly([]pt{{float64(x) - 2.5, float64(base)}, {float64(x), float64(base - h)}, {float64(x) + 2.5, float64(base)}}, flat(fireR[1]))
		c.dot(x, base-2, fireR[3])
		c.dot(x, base-1, fireR[2])
	}
	for i := 0; i < 5; i++ {
		c.dot(14+i*3, top-2-i%3, withAlpha(greyR[0], 200))
	}
}

func paintKindling(c *canvas, s size, tier int) {
	house := newCanvas(s.w, s.h)
	paintHouse(hearthFrame(house, s), tier, false)
	house.outline(ink)
	cut := s.ay - 14 - tier*2
	for y := cut; y < s.h; y++ {
		for x := 0; x < s.w; x++ {
			if col := house.at(x, y); col.A > 0 {
				c.dot(x, y, col)
			}
		}
	}
	fr := hearthFrame(c, s)
	scaffold(fr, 1, float64(20+tier*4))
	for i := 0; i < 6; i++ {
		c.dot(6+i*5, s.ay-3-i%2*2, withAlpha(plasterR[3], 170))
	}
}

func paintRubble(s iso, tier int) {
	s.foundation(1, dirtR, 0.12)
	for i, p := range [][2]float64{{0.25, 0.6}, {0.5, 0.35}, {0.7, 0.7}, {0.4, 0.8}, {0.62, 0.5}} {
		h := float64(3 + (i+tier)%4*2)
		s.stoneBox(p[0], p[1], p[0]+0.14, p[1]+0.14, 2, 2+h, stoneR, stoneTop(), 40+i)
	}
	a, b := s.p(0.2, 0.3, 4), s.p(0.8, 0.75, 10)
	s.c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[0])
}

func paintPendingTent(s iso) {
	s.foundation(1, dirtR, 0.2)
	apex := s.p(0.45, 0.45, 18)
	s.c.shadePoly([]pt{s.p(0.2, 0.75, 2), s.p(0.75, 0.75, 2), apex}, flat(plasterR[2]))
	s.c.shadePoly([]pt{s.p(0.75, 0.2, 2), s.p(0.75, 0.75, 2), apex}, flat(plasterR[1]))
	door := s.p(0.48, 0.75, 2)
	s.c.shadePoly([]pt{{door.x - 3, door.y}, {door.x + 2, door.y + 1}, {door.x, door.y - 9}}, flat(inkSft))
	fire := s.p(0.9, 0.95, 2)
	s.c.ellipse(fire.x, fire.y, 3, 1.5, rockR[1])
	s.c.shadePoly([]pt{{fire.x - 2, fire.y}, {fire.x, fire.y - 6}, {fire.x + 2, fire.y}}, flat(fireR[1]))
	s.c.dot(int(fire.x), int(fire.y)-2, fireR[3])
}
