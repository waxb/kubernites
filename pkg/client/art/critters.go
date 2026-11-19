package art

import "image/color"

func paintDoe(c *canvas, action string, frame int)  { paintDeer(c, action, frame, false) }
func paintStag(c *canvas, action string, frame int) { paintDeer(c, action, frame, true) }

func paintDeer(c *canvas, action string, frame int, antlers bool) {
	groundBlot(c, 4.6)
	s := strideFor(action, frame)
	legs := legSet{hips: [4]float64{4.5, 5.5, 10.5, 11.5}, top: 14, ground: 21, near: deerR[1], far: deerR[0], hoof: ink}
	legs.drawFar(c, s)
	by := 13.5 + float64(s.bob)
	c.disc(3.4, by-1.2, 1, cream2)
	furBody(c, 7.8, by, 4.6, 2.1, deerR, cream2)
	legs.drawNear(c, s)
	hx, hy := 13.0, by-5
	switch {
	case action == "work":
		hx, hy = 13.5, by+[]float64{5.5, 6.5, 6.5, 5.5}[frame]
	case action == "idle" && frame == 2:
		hy--
	}
	deerHead(c, 10.5, by-0.5, hx, hy, antlers && action != "work", action == "idle" && frame == 1)
}

func deerHead(c *canvas, sx, sy, hx, hy float64, antlers, flick bool) {
	c.shadePoly([]pt{{sx - 1, sy + 1}, {sx + 1.5, sy - 1}, {hx + 0.5, hy}, {hx - 1.5, hy + 0.5}}, flat(deerR[1]))
	c.line(int(sx)+1, int(sy)-1, int(hx), int(hy), deerR[2])
	c.ellipse(hx, hy, 1.7, 1.3, deerR[2])
	c.rect(int(hx)+1, int(hy), 2, 1, deerR[2])
	c.dot(int(hx)+2, int(hy)+1, deerR[1])
	c.dot(int(hx)+2, int(hy), ink)
	c.dot(int(hx), int(hy)-1, ink)
	ear := 0
	if flick {
		ear = -1
	}
	c.dot(int(hx)-2, int(hy)-1+ear, deerR[1])
	c.dot(int(hx)-1, int(hy)-2+ear, deerR[0])
	if antlers {
		ty := int(hy) - 2
		for _, p := range [][4]int{{0, 0, -1, -3}, {-1, -3, -2, -4}, {-1, -2, 1, -4}, {1, 0, 2, -3}} {
			c.line(int(hx)+p[0], ty+p[1], int(hx)+p[2], ty+p[3], rgb(214, 196, 150))
		}
	}
}

func paintFox(c *canvas, action string, frame int) {
	groundBlot(c, 4)
	s := strideFor(action, frame)
	crouch := 0
	if action == "work" {
		crouch = []int{0, 1, -2, 0}[frame]
	}
	legs := legSet{hips: [4]float64{4.5, 5.5, 9.5, 10.5}, top: 18 + crouch, ground: 21, near: foxR[0], far: inkSft, hoof: ink}
	legs.drawFar(c, s)
	by := 17.0 + float64(s.bob+crouch)
	tail := []float64{0, -1, 0, 1}[frame]
	c.shadePoly([]pt{{4, by - 1}, {0.5, by + 0.5 + tail}, {1, by + 2 + tail}, {4.5, by + 1}}, flat(foxR[1]))
	c.dot(0, int(by+1+tail), white)
	c.dot(1, int(by+1+tail), white)
	furBody(c, 7.2, by, 3.9, 1.8, foxR, cream2)
	legs.drawNear(c, s)
	hx, hy := 11.5, by-1.5
	if action == "idle" && frame == 2 {
		hy--
	}
	c.shadePoly([]pt{{hx - 1.5, hy - 1.5}, {hx + 1.5, hy - 1.5}, {hx + 3.5, hy + 0.8}, {hx - 0.5, hy + 1.5}}, flat(foxR[2]))
	c.rect(int(hx)-1, int(hy)+1, 3, 1, cream2)
	c.dot(int(hx)+3, int(hy), ink)
	c.dot(int(hx)+1, int(hy)-1, ink)
	c.shadePoly([]pt{{hx - 1.5, hy - 1.5}, {hx - 0.8, hy - 4}, {hx + 0.5, hy - 1.5}}, flat(foxR[1]))
	c.dot(int(hx)-1, int(hy)-3, ink)
}

func paintBoar(c *canvas, action string, frame int) {
	groundBlot(c, 5)
	s := strideFor(action, frame)
	legs := legSet{hips: [4]float64{4.5, 5.5, 10.5, 11.5}, top: 18, ground: 21, near: boarR[1], far: boarR[0], hoof: ink}
	legs.drawFar(c, s)
	by := 16.5 + float64(s.bob)
	c.line(3, int(by)-1, 2, int(by)-2+frame%2, boarR[1])
	furBody(c, 7.5, by, 4.9, 3, boarR, clear)
	for x := 4; x <= 10; x++ {
		c.dot(x, int(by)-3-int(hash(x, 1, 90)%2), boarR[0])
	}
	legs.drawNear(c, s)
	root := 0.0
	if action == "work" {
		root = []float64{0, 1, 2, 1}[frame]
	}
	hx, hy := 12.0, by-0.5+root
	c.shadePoly([]pt{{hx - 1.5, hy - 2}, {hx + 1.5, hy - 1}, {hx + 3, hy + 0.5}, {hx + 2.5, hy + 2}, {hx - 1, hy + 2}}, flat(boarR[2]))
	c.rect(int(hx)+2, int(hy), 2, 2, rgb(164, 120, 110))
	c.dot(int(hx)+3, int(hy), ink)
	c.dot(int(hx)+2, int(hy)+2, white)
	c.dot(int(hx), int(hy)-1, ink)
	c.dot(int(hx)-1, int(hy)-3, boarR[1])
}

func paintHare(c *canvas, action string, frame int) {
	groundBlot(c, 2.4)
	lift := 0.0
	stretch := 0.0
	switch action {
	case "walk":
		lift = []float64{0, 1, 2, 1}[frame]
		stretch = []float64{0, 0.8, 0.6, 0}[frame]
	case "carry":
		lift = []float64{0, 2, 3, 1}[frame]
		stretch = []float64{0.4, 1.2, 1, 0.2}[frame]
	}
	by := 19.5 - lift
	c.disc(5, by, 0.9, white)
	furBody(c, 7.5, by, 2.6+stretch, 1.9-stretch*0.3, hareR, clear)
	c.rect(6, int(by)+1, 2, 1, hareR[0])
	if lift == 0 {
		c.rect(6, int(by)+2, 3, 1, hareR[0])
	}
	hy := by - 2
	if action == "work" {
		hy = by - 0.5 + float64(frame%2)
	}
	hx := 10 + stretch
	c.disc(hx, hy, 1.6, hareR[2])
	c.dot(int(hx)+1, int(hy), ink)
	c.dot(int(hx)+2, int(hy)+1, rgb(170, 120, 120))
	ears := 3
	if action == "idle" && frame == 3 {
		ears = 2
	}
	c.line(int(hx)-1, int(hy)-1, int(hx)-2, int(hy)-1-ears, hareR[1])
	c.line(int(hx), int(hy)-1, int(hx), int(hy)-1-ears, hareR[2])
}

func paintFinch(c *canvas, action string, frame int) {
	if action == "walk" || action == "carry" {
		wings(c, frame, action == "carry", finchR, rgb(196, 90, 60), 4)
		return
	}
	perched(c, action, frame, finchR, rgb(196, 90, 60), 2.2)
}

func paintGull(c *canvas, action string, frame int) {
	if action == "walk" || action == "carry" {
		wings(c, frame, action == "carry", gullR, goldR[2], 6)
		return
	}
	perched(c, action, frame, gullR, goldR[2], 3)
}

func wings(c *canvas, frame int, glide bool, r ramp, beak color.NRGBA, span float64) {
	tip := []float64{-3, -1, 2, 0}[frame]
	if glide {
		tip = []float64{-1, -1, 0, 0}[frame]
	}
	y := 18.0
	mid := tip * 0.4
	for _, side := range []float64{-1, 1} {
		elbow := pt{8 + side*span*0.45, y + mid - 0.5}
		end := pt{8 + side*span, y + tip}
		c.line(8, int(y), int(elbow.x), int(elbow.y), r[2])
		c.line(int(elbow.x), int(elbow.y), int(end.x), int(end.y), r[1])
		c.line(8, int(y)+1, int(elbow.x), int(elbow.y)+1, r[1])
	}
	c.dot(int(8-span), int(y+tip), r[0])
	c.dot(int(8+span), int(y+tip), r[0])
	c.ellipse(8, y+1, 1.6, 1.1, r[3])
	c.dot(10, int(y)+1, beak)
	c.dot(9, int(y), ink)
}

func perched(c *canvas, action string, frame int, r ramp, b color.NRGBA, size float64) {
	c.ellipse(8, 22, size*0.8, 0.7, shadow)
	peck := 0.0
	if action == "work" {
		peck = []float64{0, 1, 2, 1}[frame]
	}
	c.dot(7, 21, inkSft)
	c.dot(9, 21, inkSft)
	c.shadePoly([]pt{{6, 19}, {3.5, 20.5}, {4, 21}, {6.5, 20}}, flat(r[1]))
	c.ellipse(7.5, 19.5, size, size*0.7, r[2])
	c.rect(6, 19, 3, 1, r[3])
	c.disc(9.5, 18+peck, size*0.55, r[2])
	c.dot(10, int(17.5+peck), ink)
	c.dot(11, int(18+peck), b)
}
