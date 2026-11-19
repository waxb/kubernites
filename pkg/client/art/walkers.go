package art

import "sync"

func buildWalkers() Atlas {
	b := &atlasBuilder{}
	for _, r := range append(append([]string{}, WalkerRoles...), CritterRoles...) {
		for _, a := range WalkerActions {
			for f := 0; f < WalkerFrames; f++ {
				role, action, frame := r, a, f
				addWalker(b, func(dir string) string { return walkerName(role, action, dir, frame) }, func(c *canvas) {
					paintWalker(c, role, action, frame)
				})
			}
		}
	}
	addLooks(b)
	addTradeHolds(b)
	addCarry(b)
	addSealedCarry(b)
	addFolk(b)
	trimWalkers(b)
	return b.build()
}

func addWalker(b *atlasBuilder, name func(dir string) string, paint func(c *canvas)) {
	var once sync.Once
	var se *canvas
	painted := func() *canvas {
		once.Do(func() {
			se = newCanvas(walkerFrameW, walkerFrameH)
			paint(se)
		})
		return se
	}
	b.add(name("se"), walkerFrameW, walkerFrameH, walkerFrameAX, walkerFrameAY, func(c *canvas) { c.paste(painted(), 0, 0, false) })
	b.add(name("sw"), walkerFrameW, walkerFrameH, walkerFrameW-walkerFrameAX, walkerFrameAY, func(c *canvas) { c.paste(painted(), 0, 0, true) })
}

func addLooks(b *atlasBuilder) {
	for _, r := range VariedRoles {
		for tier := 1; tier <= WalkerTiers(r); tier++ {
			for v := 0; v < WalkerLooks; v++ {
				if tier == 1 && v == 0 {
					continue
				}
				spec := lookSpec(r, v, tier)
				for _, a := range WalkerActions {
					for f := 0; f < WalkerFrames; f++ {
						role, tt, vv, action, frame := r, tier, v, a, f
						addWalker(b, func(dir string) string { return walkerLookName(role, tt, vv, action, dir, frame) }, func(c *canvas) {
							paintPerson(c, spec, action, frame)
						})
					}
				}
			}
		}
	}
}

func paintPerson(c *canvas, spec roleSpec, action string, frame int) {
	if spec.trade != "" {
		paintTrade(c, spec, action, frame)
		return
	}
	body := newWalkerBody()
	p := poseFor(action, frame)
	if spec.emblem && action == "work" {
		p = servePose(frame)
	}
	drawPerson(body, spec, p)
	finishWalker(c, body)
}

func paintWalker(c *canvas, role, action string, frame int) {
	body := newWalkerBody()
	switch role {
	case "imp":
		paintImp(body, action, frame)
	case "sheep":
		paintSheep(body, action, frame)
	default:
		if paint, ok := beastPainters[role]; ok {
			paint(body, action, frame)
			break
		}
		paintPerson(c, roleSpecs[role], action, frame)
		return
	}
	finishWalker(c, body)
}

var furR = ramp{rgb(76, 46, 30), rgb(118, 74, 44), rgb(156, 104, 62), rgb(196, 148, 100)}

func paintImp(c *canvas, action string, frame int) {
	p := poseFor(action, frame)
	hop := []int{0, -2, -3, -1}[frame]
	if action == "idle" {
		hop = p.bob
	}
	y := 14 + hop
	c.ellipse(8, 22, 3, 1.1, shadow)
	c.line(3, y+5, 1, y+1, furR[1])
	c.dot(1, y, furR[2])
	c.ellipse(7, float64(y)+3.5, 3, 3, furR[1])
	c.ellipse(8, float64(y)+0.5, 3, 2.6, furR[2])
	c.rect(5, y-2, 2, 2, furR[1])
	c.rect(10, y-2, 2, 2, furR[1])
	c.rect(7, y, 3, 2, skinR[3])
	c.dot(8, y-1, ink)
	c.dot(10, y-1, ink)
	c.dot(8, y+1, redR[1])
	c.rect(5, y+6, 2, 22-(y+6), furR[0])
	c.rect(9, y+6, 2, 22-(y+6), furR[0])
	if action == "work" || action == "carry" {
		c.rect(10, y+2-frame%2, 3, 1, furR[2])
		c.dot(13, y+1-frame%2, goldR[3])
	}
}

func paintSheep(c *canvas, action string, frame int) {
	p := poseFor(action, frame)
	c.ellipse(8, 22, 4.4, 1.2, shadow)
	legs := []int{0, 1, 0, -1}[frame]
	if action == "idle" || action == "work" {
		legs = 0
	}
	for i, x := range []int{5, 7, 9, 11} {
		shift := legs
		if i%2 == 1 {
			shift = -legs
		}
		c.rect(x+shift, 19, 1, 3, ink)
	}
	wool := ramp{rgb(170, 166, 156), rgb(214, 210, 200), rgb(236, 232, 222), white}
	c.ellipse(8, 16.5+float64(p.bob), 5, 3.4, wool[1])
	c.ellipse(7, 15.6+float64(p.bob), 3.4, 2.1, wool[2])
	c.dot(6, 14+p.bob, wool[3])
	headY := 14 + p.bob
	if action == "work" {
		headY = 17 + frame%2
	}
	c.rect(12, headY, 3, 3, inkSft)
	c.dot(14, headY+1, white)
	c.dot(12, headY-1, inkSft)
}
