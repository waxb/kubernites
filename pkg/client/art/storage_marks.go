package art

import "image/color"

var (
	tagRedFill  = rgb(104, 44, 43)
	tagRedEdge  = rgb(240, 128, 112)
	tagConsole  = rgb(33, 44, 48)
	tagGoldEdge = rgb(255, 218, 105)
)

func markTag(c *canvas, fill, edge color.NRGBA) {
	bw, bh := tagW-1, tagH-4
	c.rect(1, 1, bw, bh, withAlpha(shadow, 90))
	c.rect(0, 0, bw, bh, fill)
	c.rect(0, 0, bw, 1, edge)
	c.rect(0, bh-1, bw, 1, edge)
	c.rect(0, 0, 1, bh, edge)
	c.rect(bw-1, 0, 1, bh, edge)
	for i := 0; i < 3; i++ {
		c.rect(tagAX-(2-i), bh+i, 2*(2-i)+1, 1, edge)
	}
	c.dot(tagAX, tagAY, edge)
}

func paintStrandedTag(c *canvas) {
	markTag(c, tagRedFill, tagRedEdge)
	c.rect(3, 4, 6, 4, dirtR[2])
	c.rect(3, 4, 6, 1, dirtR[3])
	c.rect(3, 7, 6, 1, dirtR[1])
	c.dot(5, 5, goldR[2])
	c.line(4, 4, 6, 2, ropeR[2])
	c.line(6, 2, 8, 4, ropeR[2])
	for i := 0; i < 3; i++ {
		c.dot(10+i, 3+i, cream)
		c.dot(12-i, 3+i, cream)
	}
}

func paintDeedTag(c *canvas) {
	markTag(c, tagConsole, tagGoldEdge)
	c.rect(3, 3, 8, 4, tkParchment)
	c.rect(2, 2, 1, 6, plasterR[2])
	c.rect(11, 2, 1, 6, plasterR[2])
	c.rect(4, 4, 5, 1, inkSft)
	c.rect(9, 6, 2, 2, waxRed)
	c.dot(9, 6, redR[2])
}

func paintDeedScroll(c *canvas, sealed bool) {
	c.rect(2, 2, 10, 5, tkParchment)
	c.rect(2, 6, 10, 1, tkParchmentDark)
	for _, x := range []int{1, 12} {
		c.rect(x, 1, 1, 7, plasterR[2])
		c.dot(x, 1, cream)
		c.dot(x, 7, plasterR[1])
	}
	c.rect(4, 3, 5, 1, inkSft)
	c.rect(4, 5, 3, 1, inkSft)
	if !sealed {
		c.rect(10, 7, 2, 1, tkParchment)
		c.dot(11, 7, tkParchmentDark)
		c.outline(ink)
		return
	}
	c.rect(8, 4, 3, 3, waxRed)
	c.dot(9, 4, redR[2])
	c.dot(8, 7, redR[1])
	c.outline(ink)
}

func paintChainLink(c *canvas, red bool, phase int) {
	tone := chainTone(red)
	if phase == 0 {
		for _, d := range [][2]int{{1, 2}, {2, 1}, {3, 1}, {4, 1}, {5, 2}, {5, 4}, {4, 5}, {3, 5}, {2, 5}, {1, 4}, {1, 3}, {5, 3}} {
			c.dot(d[0], d[1], tone[1])
		}
		c.dot(2, 1, tone[3])
		c.dot(5, 4, tone[0])
		return
	}
	c.rect(3, 1, 1, 5, tone[1])
	c.dot(3, 1, tone[3])
	c.dot(3, 5, tone[0])
	c.dot(2, 3, tone[2])
	c.dot(4, 3, tone[2])
}

func paintMountDot(c *canvas) {
	c.dot(1, 0, ropeR[2])
	c.dot(0, 1, ropeR[1])
	c.dot(1, 1, ropeR[3])
	c.dot(2, 1, ropeR[1])
	c.dot(1, 2, ropeR[0])
}

func paintLoosePages(c *canvas, variant int) {
	c.ellipse(float64(pagesAX), float64(pagesAY)+1, 13, 3.5, withAlpha(shadow, 60))
	for i := 0; i < 7; i++ {
		h := hash(i, variant, 331)
		x := 2 + int(h%24)
		y := 3 + int(h/24%9)
		page(c, x, y, int(h/7%4))
	}
	fly := hash(variant, 9, 337)
	page(c, 5+int(fly%18), 0, 3)
}

func page(c *canvas, x, y, kind int) {
	switch kind {
	case 0:
		c.rect(x, y, 4, 3, cream)
		c.rect(x+1, y+1, 2, 1, inkSft)
		c.dot(x+3, y+2, tkParchmentDark)
	case 1:
		c.rect(x, y, 5, 2, tkParchment)
		c.rect(x, y+1, 5, 1, tkParchmentDark)
		c.dot(x+1, y, inkSft)
	case 2:
		c.rect(x, y, 3, 3, cream)
		c.dot(x+1, y+1, inkSft)
		c.dot(x+2, y, tkParchmentDark)
	default:
		c.line(x, y+1, x+3, y, cream)
		c.line(x, y+2, x+3, y+1, tkParchment)
		c.dot(x+1, y+1, inkSft)
	}
}
