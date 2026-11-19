package art

import "image/color"

var uiPainters = map[string]func(c *canvas){
	"panel":              func(c *canvas) { paintPanel(c, plasterR[3], plasterR[2]) },
	"panel-dark":         func(c *canvas) { paintPanel(c, rgb(52, 38, 36), rgb(34, 25, 26)) },
	"button":             func(c *canvas) { paintButton(c, 0) },
	"button-hover":       func(c *canvas) { paintButton(c, 1) },
	"button-pressed":     func(c *canvas) { paintButton(c, 2) },
	"button-disabled":    func(c *canvas) { paintButton(c, 3) },
	"tab":                func(c *canvas) { paintTab(c, false) },
	"tab-active":         func(c *canvas) { paintTab(c, true) },
	"scroll":             paintScroll,
	"tooltip":            paintTooltip,
	"bar-frame":          paintBarFrame,
	"bar-fill-good":      func(c *canvas) { paintBarFill(c, greenR) },
	"bar-fill-bad":       func(c *canvas) { paintBarFill(c, redR) },
	"minimap-frame":      paintMinimapFrame,
	"woven-strip":        paintWovenStrip,
	"chip":               paintChip,
	"bar-mini":           paintBarMini,
	"kmap-frame":         paintKMapFrame,
	"kmap-ribbon":        paintKMapRibbon,
	"plaque-locked":      func(c *canvas) { paintPlaque(c, "locked") },
	"plaque-available":   func(c *canvas) { paintPlaque(c, "available") },
	"plaque-researching": func(c *canvas) { paintPlaque(c, "researching") },
	"plaque-learned":     func(c *canvas) { paintPlaque(c, "learned") },
}

func buildUI() Atlas {
	b := &atlasBuilder{}
	for _, n := range UINames {
		s := uiSizes[n]
		b.addSliced(uiName(n), s[0], s[1], s[2], uiPainters[n])
	}
	return b.build()
}

func ring(c *canvas, i int, lite, dark color.NRGBA) {
	w, h := c.w-2*i, c.h-2*i
	c.rect(i, i, w, 1, lite)
	c.rect(i, i, 1, h, lite)
	c.rect(i, c.h-1-i, w, 1, dark)
	c.rect(c.w-1-i, i, 1, h, dark)
}

func woodBorder(c *canvas, inset, width int) {
	lites := []color.NRGBA{woodR[3], woodR[2], woodR[2], woodR[1], woodR[2]}
	darks := []color.NRGBA{woodR[1], woodR[1], woodR[0], woodR[0], woodR[1]}
	for i := 0; i < width; i++ {
		ring(c, inset+i, lites[i%5], darks[i%5])
	}
}

func roundCorners(c *canvas, bottom bool) {
	erase(c, 0, 0)
	erase(c, c.w-1, 0)
	c.dot(1, 1, ink)
	c.dot(c.w-2, 1, ink)
	if bottom {
		erase(c, 0, c.h-1)
		erase(c, c.w-1, c.h-1)
		c.dot(1, c.h-2, ink)
		c.dot(c.w-2, c.h-2, ink)
	}
}

func bracket(c *canvas, fx, fy bool) {
	px := func(v int) int {
		if fx {
			return c.w - 2 - v
		}
		return 1 + v
	}
	py := func(v int) int {
		if fy {
			return c.h - 2 - v
		}
		return 1 + v
	}
	for i := 0; i < 8; i++ {
		for t, col := range []color.NRGBA{goldR[2], goldR[1]} {
			c.dot(px(i), py(t), col)
			c.dot(px(t), py(i), col)
		}
	}
	c.dot(px(0), py(0), goldR[3])
	c.dot(px(3), py(3), goldR[3])
	c.dot(px(4), py(4), goldR[0])
}

func paintPanel(c *canvas, fill, edge color.NRGBA) {
	c.rect(0, 0, c.w, c.h, fill)
	ring(c, 0, ink, ink)
	woodBorder(c, 1, 4)
	ring(c, 5, ink, ink)
	ring(c, 6, edge, mix(fill, edge, 0.35))
	for _, f := range [][2]bool{{false, false}, {true, false}, {false, true}, {true, true}} {
		bracket(c, f[0], f[1])
	}
}

func paintButton(c *canvas, mode int) {
	face, lite, dark, depth := woodR[2], woodR[3], woodR[1], woodR[0]
	switch mode {
	case 1:
		face, lite = mix(woodR[2], woodR[3], 0.5), goldR[3]
	case 2:
		face, lite, dark = woodR[1], woodR[0], woodR[2]
	case 3:
		face, lite, dark, depth = grey(woodR[2]), grey(woodR[3]), grey(woodR[1]), grey(woodR[0])
	}
	c.rect(1, 1, c.w-2, c.h-2, face)
	c.rect(1, 1, c.w-2, 1, lite)
	c.rect(1, 1, 1, c.h-3, lite)
	c.rect(c.w-2, 1, 1, c.h-2, dark)
	if mode == 2 {
		c.rect(1, c.h-2, c.w-2, 1, dark)
	} else {
		c.rect(1, c.h-4, c.w-2, 1, dark)
		c.rect(1, c.h-3, c.w-2, 2, depth)
	}
	ring(c, 0, ink, ink)
	if mode == 1 {
		ring(c, 1, goldR[2], goldR[1])
	}
	roundCorners(c, true)
}

func paintTab(c *canvas, active bool) {
	face, lite, dark := woodR[1], woodR[2], woodR[0]
	if active {
		face, lite, dark = plasterR[3], cream, plasterR[2]
	}
	c.rect(1, 1, c.w-2, c.h-1, face)
	c.rect(1, 1, c.w-2, 1, lite)
	c.rect(1, 1, 1, c.h-1, lite)
	c.rect(c.w-2, 1, 1, c.h-1, dark)
	c.rect(0, 0, c.w, 1, ink)
	c.rect(0, 0, 1, c.h, ink)
	c.rect(c.w-1, 0, 1, c.h, ink)
	if active {
		c.rect(2, 1, c.w-4, 1, goldR[2])
	} else {
		c.rect(0, c.h-1, c.w, 1, ink)
	}
	roundCorners(c, false)
}
