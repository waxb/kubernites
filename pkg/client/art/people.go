package art

const figureTop = 7

type pose struct {
	bob      int
	legShift int
	armSwing int
	blink    bool
	toolUp   bool
	carry    bool
	reach    bool
	bow      int
}

func poseFor(action string, frame int) pose {
	p := pose{}
	switch action {
	case "walk", "carry":
		p.legShift = []int{0, 1, 0, -1}[frame]
		p.bob = []int{0, -1, 0, -1}[frame]
		p.armSwing = []int{0, 1, 0, -1}[frame]
		p.carry = action == "carry"
	case "work":
		p.toolUp = frame < 2
		p.bob = []int{0, -1, 0, 0}[frame]
	default:
		p.blink = frame == 2
		p.bob = []int{0, 0, 0, -1}[frame]
	}
	return p
}

func servePose(frame int) pose {
	return []pose{{}, {reach: true}, {reach: true, bow: 2}, {reach: true, bow: 1}}[frame]
}

func drawPerson(c *canvas, spec roleSpec, p pose) {
	top := figureTop + p.bob + spec.short
	c.ellipse(8, float64(WalkerHeight-2), 3.5, 1.2, shadow)
	drawLegs(c, spec, p, top)
	drawBody(c, spec, p, top+p.bow/2)
	drawHead(c, spec, p, top+p.bow)
	hx, hy := 11, top+8-p.armSwing
	if p.reach {
		body := top + p.bow/2
		c.rect(11, body+6, 3, 1, spec.tunic[0])
		c.dot(14, body+6, spec.skin[2])
		c.dot(14, body+5, goldR[3])
		drawTool(c, spec.tool, 5, hy, false)
		return
	}
	if p.carry && carryLoad(c, spec.tool, top) {
		return
	}
	if p.carry {
		c.disc(8, float64(top)+7.5, 2.3, goldR[1])
		c.dot(7, top+7, goldR[3])
		c.dot(8, top+5, woodR[0])
		return
	}
	drawTool(c, spec.tool, hx, hy, p.toolUp)
}

func drawLegs(c *canvas, spec roleSpec, p pose, top int) {
	if spec.robe {
		c.rect(6, 20, 2, 2, ink)
		c.rect(9, 20, 2, 2, ink)
		return
	}
	lx, rx := 6+p.legShift, 9-p.legShift
	c.rect(lx, top+11, 2, 21-(top+11), spec.legs)
	c.rect(rx, top+11, 2, 21-(top+11), spec.legs)
	c.rect(lx, 21, 2, 1, ink)
	c.rect(rx, 21, 2, 1, ink)
}

func drawBody(c *canvas, spec roleSpec, p pose, top int) {
	bottom := top + 11
	if spec.robe {
		bottom = 20
	}
	for y := top + 5; y < bottom; y++ {
		widen := 0
		if spec.robe && y > top+9 {
			widen = (y - top - 9) / 2
		}
		c.rect(6-widen, y, 5+widen*2, 1, spec.tunic[1])
		c.dot(6-widen, y, spec.tunic[2])
		c.dot(10+widen, y, spec.tunic[0])
	}
	if !spec.robe {
		c.rect(6, top+8, 5, 1, woodR[0])
		c.dot(8, top+8, spec.trim)
	} else {
		c.rect(8, top+5, 1, bottom-top-5, spec.trim)
	}
	c.rect(5, top+6-p.armSwing, 1, 3, spec.tunic[2])
	c.dot(5, top+9-p.armSwing, spec.skin[2])
	if spec.emblem {
		c.dot(8, top+5, goldR[3])
		c.dot(7, top+6, goldR[2])
		c.dot(9, top+6, goldR[2])
		c.dot(8, top+7, goldR[2])
		c.dot(8, top+6, redR[2])
	}
	drawGarb(c, spec, top, bottom)
	if !p.reach {
		c.rect(11, top+6+p.armSwing, 1, 3, spec.tunic[0])
		c.dot(11, top+9+p.armSwing, spec.skin[1])
	}
}

func drawHead(c *canvas, spec roleSpec, p pose, top int) {
	x := 6 + min(p.bow, 1)
	c.rect(x, top+1, 5, 4, spec.skin[2])
	c.rect(x, top+4, 5, 1, spec.skin[1])
	c.rect(x, top, 5, 1, spec.hair)
	c.rect(x, top+1, 1, 2, spec.hair)
	eye := ink
	if p.blink {
		eye = spec.skin[0]
	}
	c.dot(x+2, top+2, eye)
	c.dot(x+4, top+2, eye)
	c.dot(x+3, top+4, spec.skin[0])
	drawHairStyle(c, spec, x, top)
	drawHat(c, spec, x, top)
}
