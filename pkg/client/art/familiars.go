package art

func paintRavenWalker(c *canvas, action string, frame int) {
	switch action {
	case "walk", "carry":
		ravenFlight(c, frame, action == "carry")
	case "work":
		ravenPerch(c, 0, []int{0, 3, 4, 1}[frame], false)
	default:
		ravenPerch(c, []int{0, -1, 0, 0}[frame], 0, frame == 2)
	}
}

func ravenPerch(c *canvas, hop, peck int, blink bool) {
	groundBlot(c, 2.3)
	y := 18.4 + float64(hop)
	c.dot(7, 21+hop, inkSft)
	c.dot(9, 21+hop, inkSft)
	c.dot(7, 20+hop, inkSft)
	c.dot(9, 20+hop, inkSft)
	c.shadePoly([]pt{{5, y - 0.4}, {1.6, y + 2}, {2.4, y + 3}, {5.5, y + 1.3}}, flat(ravenR[1]))
	c.ellipse(7.5, y, 3, 1.9, ravenR[1])
	c.ellipse(7, y-0.5, 2.3, 1.2, ravenR[2])
	c.rect(5, int(y)-1, 3, 1, ravenR[3])
	hx, hy := 10.0, y-2.4+float64(peck)
	c.disc(hx, hy, 1.7, ravenR[1])
	c.dot(int(hx)-1, int(hy)-1, ravenR[2])
	c.shadePoly([]pt{{hx + 1.2, hy - 0.5}, {hx + 3.4, hy + 0.3}, {hx + 1.2, hy + 0.8}}, flat(inkSft))
	eye := goldR[3]
	if blink {
		eye = ravenR[1]
	}
	c.dot(int(hx), int(hy)-1, eye)
}

func ravenFlight(c *canvas, frame int, scroll bool) {
	c.ellipse(8, 22, 2, 0.8, withAlpha(shadow, 60))
	y := 11.0 + float64([]int{0, -1, 0, 1}[frame])
	flap := []float64{-4, -2, 3, 1}[frame]
	c.shadePoly([]pt{{6.5, y}, {9.5, y}, {5.5 - flap*0.2, y + flap}}, flat(ravenR[0]))
	c.shadePoly([]pt{{4.8, y}, {1.4, y - 0.8}, {1.8, y + 1.3}, {4.8, y + 1.3}}, flat(ravenR[1]))
	c.ellipse(8, y+0.5, 3, 1.3, ravenR[1])
	c.rect(6, int(y), 3, 1, ravenR[2])
	c.disc(11.2, y-0.2, 1.5, ravenR[1])
	c.shadePoly([]pt{{12.5, y - 0.7}, {14.6, y}, {12.5, y + 0.5}}, flat(inkSft))
	c.dot(11, int(y)-1, goldR[3])
	c.shadePoly([]pt{{7, y + 0.5}, {10.5, y + 0.5}, {7.5 - flap*0.3, y + flap*1.2 + 0.5}}, flat(ravenR[2]))
	c.line(8, int(y+0.5), int(7.5-flap*0.3), int(y+flap*1.2+0.5), ravenR[3])
	if scroll {
		c.rect(7, int(y)+2, 3, 2, cream)
		c.dot(8, int(y)+3, redR[2])
	}
}

func paintHound(c *canvas, action string, frame int) {
	switch action {
	case "idle":
		houndSit(c, frame)
	case "work":
		houndStand(c, stride{}, 3+[]int{0, 1, 1, 0}[frame], frame%2)
	default:
		houndStand(c, strideFor(action, frame), 0, frame%2)
	}
}

func houndLegs() legSet {
	return legSet{hips: [4]float64{5, 6, 10, 11}, top: 18, ground: 21, near: houndR[1], far: houndR[0], hoof: ink}
}

func houndStand(c *canvas, s stride, nose, wag int) {
	groundBlot(c, 4)
	legs := houndLegs()
	legs.drawFar(c, s)
	by := 16.6 + float64(s.bob)
	c.line(4, int(by)-1, 2, int(by)-3-wag, houndR[1])
	c.dot(2, int(by)-4-wag, houndR[2])
	furBody(c, 7.8, by, 4, 2.1, houndR, houndR[3])
	legs.drawNear(c, s)
	houndHead(c, 11.4, by-2.2+float64(nose))
}

func houndHead(c *canvas, hx, hy float64) {
	c.shadePoly([]pt{{9.5, hy + 1}, {11.5, hy - 1}, {12.5, hy + 1.5}, {11, hy + 3}}, flat(houndR[1]))
	c.ellipse(hx+0.5, hy, 2.3, 1.8, houndR[2])
	c.rect(int(hx)+2, int(hy), 2, 2, houndR[2])
	c.dot(int(hx)+4, int(hy), ink)
	c.dot(int(hx)+1, int(hy)-1, ink)
	c.rect(int(hx)-1, int(hy)-1, 1, 3, houndR[0])
	c.dot(int(hx), int(hy)-2, houndR[0])
	c.rect(int(hx)-2, int(hy)+2, 2, 1, redR[2])
	c.dot(int(hx)-1, int(hy)+3, goldR[3])
}

func houndSit(c *canvas, frame int) {
	groundBlot(c, 3.6)
	wag := []int{0, 1, 0, -1}[frame]
	c.line(4, 21, 1, 20+wag, houndR[1])
	c.shadePoly([]pt{{4, 21}, {5.5, 16.5}, {9, 15}, {11, 17.5}, {10, 21}}, flat(houndR[1]))
	c.shadePoly([]pt{{5.5, 17}, {9, 15.4}, {9.5, 16.6}, {6.2, 18.2}}, flat(houndR[2]))
	c.ellipse(5.8, 19.8, 2.4, 1.7, houndR[1])
	c.rect(5, 21, 3, 1, houndR[0])
	c.line(10, 17, 10, 21, houndR[2])
	c.line(9, 18, 9, 21, houndR[0])
	c.dot(10, 21, ink)
	c.rect(9, 18, 1, 2, houndR[3])
	houndHead(c, 11, 13.2-float64(frame/2%2))
}
