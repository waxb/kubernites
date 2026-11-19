package art

import "image/color"

var (
	landingSize = size{22, 38, 9, 34}
	stakeSize   = size{24, 30, 10, 26}
)

func paintPortSign(c *canvas, l PortLook) {
	body := newCanvas(17, 28)
	signBody(body)
	m := portMatOf(l.Age)
	switch clampAge(l.Age) {
	case ageDark:
		body.mapPixels(func(col color.NRGBA) color.NRGBA { return portWeathered(col) })
	case ageCastle:
		body.rect(5, 7, 11, 1, m.metal[2])
		body.rect(3, 3, 2, 3, m.metal[1])
	case ageEmpire:
		body.rect(5, 7, 11, 1, goldR[2])
		body.rect(5, 14, 11, 1, goldR[1])
		body.rect(2, 2, 4, 1, goldR[3])
		body.dot(10, 8, goldR[3])
	}
	c.paste(body, 2, 1, false)
	c.outline(ink)
	propShadow(c, 6, 27, 4)
}

func portWeathered(col color.NRGBA) color.NRGBA {
	if col.A == 0 {
		return col
	}
	return mix(col, grey(col), 0.45)
}

func paintPortBollard(c *canvas, l PortLook) {
	m := portMatOf(l.Age)
	body := m.metal
	if l.Age <= ageFeudal {
		body = portMatOf(l.Age).wood
	}
	c.rect(3, 4, 3, 6, body[1])
	c.rect(3, 4, 1, 6, body[2])
	c.rect(5, 4, 1, 6, body[0])
	c.rect(2, 2, 5, 2, body[2])
	c.rect(2, 2, 5, 1, body[3])
	switch clampAge(l.Age) {
	case ageFeudal:
		c.rect(3, 6, 3, 1, ironR[1])
	case ageEmpire:
		c.rect(2, 2, 5, 1, goldR[3])
		c.rect(3, 3, 3, 1, goldR[2])
	}
	c.rect(3, 7, 3, 1, ropeR[2])
	c.dot(6, 8, ropeR[1])
	c.dot(7, 9, ropeR[1])
	c.outline(ink)
	propShadow(c, 4.5, 10, 3.5)
}

func paintPortCrane(c *canvas, l PortLook) {
	s := isoFrame(c, 1, craneSize.ax, craneSize.ay)
	deck := PortLook{Kind: "deck", Level: l.Level, Age: l.Age}
	top := func(x, y int) color.NRGBA {
		gx, gy := s.ground(x, y, 3)
		return deckTop(deck, gx, gy)
	}
	s.box(0.08, 0.08, 0.92, 0.92, 0, 3, faces{top: top, left: portFace(deck, true), right: portFace(deck, false)})
	w := woodR
	if l.Age != ageFeudal {
		w = portMatOf(l.Age).wood
	}
	craneFrame(s, w)
	if l.Age >= ageCastle {
		hub := s.p(wheelCX, wheelNea, wheelCZ)
		c.rect(int(hub.x)-1, int(hub.y)-1, 3, 3, portMatOf(l.Age).metal[2])
	}
	c.outline(ink)
	craneRope(s)
	if l.Age >= ageEmpire {
		tip := s.p(-0.95, 0.28, 42)
		c.rect(int(tip.x)-1, int(tip.y)-2, 3, 2, goldR[2])
	}
}

func paintPortLanding(c *canvas, l PortLook) {
	m := portMatOf(l.Age)
	post := m.wood
	if l.Age >= ageCastle {
		post = m.metal
	}
	x, foot := 8, 34
	c.rect(x, 6, 2, foot-6, post[1])
	c.rect(x, 6, 1, foot-6, post[2])
	c.rect(x, 5, 9, 2, post[2])
	c.rect(x, 5, 9, 1, post[3])
	landingBell(c, x+6, 7, l.Age)
	landingLamp(c, x-4, 8, post)
	landingBoard(c, x-5, 16, l.Age)
	if l.Age >= ageEmpire {
		c.rect(x-2, foot-3, 6, 3, marbleR[2])
		c.rect(x-2, foot-3, 6, 1, marbleR[3])
	}
	c.outline(ink)
	propShadow(c, float64(x)+1, float64(foot), 4)
}

func landingBell(c *canvas, x, y, age int) {
	bell := ramp{rgb(120, 72, 30), rgb(176, 118, 46), rgb(216, 164, 70), rgb(244, 214, 128)}
	if age >= ageEmpire {
		bell = goldR
	}
	c.rect(x, y, 1, 2, ropeR[1])
	c.rect(x-2, y+2, 5, 4, bell[1])
	c.rect(x-2, y+2, 2, 4, bell[2])
	c.rect(x-3, y+6, 7, 1, bell[0])
	c.dot(x, y+7, bell[3])
}

func landingLamp(c *canvas, x, y int, post ramp) {
	c.rect(x, y-2, 5, 1, post[1])
	c.rect(x, y-1, 1, 2, post[0])
	c.rect(x-1, y+1, 3, 4, lampLit)
	c.dot(x, y+2, white)
	c.rect(x-1, y+5, 3, 1, post[0])
}

func landingBoard(c *canvas, x, y, age int) {
	board := woodR
	if age >= ageCastle {
		board = blueR
	}
	c.rect(x, y, 9, 6, board[0])
	c.rect(x+1, y+1, 7, 4, plasterR[3])
	c.rect(x+2, y+3, 5, 1, woodR[1])
	c.rect(x+3, y+4, 3, 1, woodR[0])
	c.dot(x+4, y+2, blueR[2])
	c.dot(x+5, y+2, blueR[1])
}

func paintPortStake(c *canvas, l PortLook) {
	w := portMatOf(l.Age).wood
	x, foot := 10, 26
	c.rect(x, 4, 2, foot-4, w[1])
	c.rect(x, 4, 1, foot-4, w[2])
	c.rect(x+2, 4, 8, 4, redR[2])
	c.rect(x+2, 4, 8, 1, redR[3])
	c.rect(x+8, 6, 2, 2, redR[1])
	c.rect(x-7, 11, 9, 7, w[0])
	c.rect(x-6, 12, 7, 5, plasterR[3])
	c.rect(x-4, 13, 3, 3, goldR[2])
	c.dot(x-3, 13, goldR[3])
	c.dot(x-3, 15, goldR[0])
	c.rect(x-5, 10, 1, 1, ropeR[1])
	c.rect(x, 10, 1, 1, ropeR[1])
	for _, dx := range []int{-8, 13} {
		c.rect(x+dx, foot-4, 1, 4, w[2])
		c.dot(x+dx, foot-5, redR[2])
	}
	c.outline(ink)
	propShadow(c, float64(x)+1, float64(foot), 5)
}
