package art

import "image/color"

var (
	ravenR = ramp{rgb(18, 16, 26), rgb(36, 34, 52), rgb(60, 62, 92), rgb(112, 124, 168)}
	houndR = ramp{rgb(58, 44, 36), rgb(100, 78, 58), rgb(142, 112, 80), rgb(188, 156, 114)}
	deerR  = ramp{rgb(88, 56, 34), rgb(142, 94, 56), rgb(182, 130, 80), rgb(222, 184, 130)}
	foxR   = ramp{rgb(116, 48, 20), rgb(186, 88, 34), rgb(224, 128, 54), rgb(246, 178, 104)}
	boarR  = ramp{rgb(40, 32, 30), rgb(70, 56, 48), rgb(100, 82, 68), rgb(138, 118, 98)}
	hareR  = ramp{rgb(86, 70, 58), rgb(134, 112, 92), rgb(172, 150, 124), rgb(212, 194, 166)}
	finchR = ramp{rgb(44, 36, 40), rgb(96, 70, 56), rgb(150, 110, 78), rgb(214, 182, 132)}
	gullR  = ramp{rgb(120, 128, 140), rgb(176, 184, 194), rgb(226, 230, 234), white}
	cream2 = rgb(236, 226, 204)
)

type beastPainter func(c *canvas, action string, frame int)

var beastPainters = map[string]beastPainter{
	"raven": paintRavenWalker, "hound": paintHound, "deer": paintDoe, "stag": paintStag, "rabbit": paintHare,
	"fox": paintFox, "boar": paintBoar, "finch": paintFinch, "gull": paintGull,
}

type stride struct {
	swing [4]int
	lift  [4]int
	bob   int
}

func walkStride(frame int) stride {
	return []stride{
		{swing: [4]int{1, -1, -1, 1}},
		{swing: [4]int{0, 0, 0, 0}, lift: [4]int{1, 0, 0, 1}, bob: -1},
		{swing: [4]int{-1, 1, 1, -1}},
		{swing: [4]int{0, 0, 0, 0}, lift: [4]int{0, 1, 1, 0}, bob: -1},
	}[frame]
}

func runStride(frame int) stride {
	return []stride{
		{swing: [4]int{2, 2, -2, -2}, bob: -1},
		{swing: [4]int{1, 1, -1, -1}, lift: [4]int{2, 2, 2, 2}, bob: -2},
		{swing: [4]int{-1, -1, 1, 1}, lift: [4]int{1, 0, 1, 0}, bob: -1},
		{swing: [4]int{-2, -1, 2, 1}},
	}[frame]
}

func strideFor(action string, frame int) stride {
	switch action {
	case "walk":
		return walkStride(frame)
	case "carry":
		return runStride(frame)
	}
	return stride{}
}

type legSet struct {
	hips   [4]float64
	top    int
	ground int
	near   color.NRGBA
	far    color.NRGBA
	hoof   color.NRGBA
}

func (l legSet) draw(c *canvas, s stride) {
	for i, hx := range l.hips {
		col := l.near
		if i%2 == 1 {
			col = l.far
		}
		foot := l.ground - s.lift[i]
		x0, x1 := int(hx), int(hx)+s.swing[i]
		c.line(x0, l.top+s.bob, x1, foot, col)
		c.dot(x1, foot, l.hoof)
	}
}

func (l legSet) drawFar(c *canvas, s stride) {
	far := l
	far.hips = [4]float64{-99, l.hips[1], -99, l.hips[3]}
	far.draw(c, s)
}

func (l legSet) drawNear(c *canvas, s stride) {
	near := l
	near.hips = [4]float64{l.hips[0], -99, l.hips[2], -99}
	near.draw(c, s)
}

func furBody(c *canvas, cx, cy, rx, ry float64, r ramp, belly color.NRGBA) {
	for y := int(cy - ry - 1); y <= int(cy+ry+1); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry
			if dx*dx+dy*dy > 1 {
				continue
			}
			col := r[1]
			switch {
			case dy < -0.55:
				col = r[2]
			case dy > 0.55 && belly.A > 0:
				col = belly
			case dy > 0.4:
				col = r[0]
			case noise(x, y, 61) > 0.86:
				col = r[2]
			}
			c.dot(x, y, col)
		}
	}
}

func groundBlot(c *canvas, rx float64) {
	c.ellipse(8, 22, rx, 1.2, shadow)
}
