package art

import "image/color"

var decorPainters = map[string]func(c *canvas, s size, v int){
	"tree-oak": paintOak, "tree-pine": paintPine, "tree-birch": paintBirch, "tree-maple": paintMaple,
	"tree-spruce": paintSpruce, "tree-old": paintOld, "tree-willow": paintWillow, "bush": paintBush,
	"fern": paintFern, "rock": paintRock, "boulder": paintBoulder, "reeds": paintReeds, "flowers": paintFlowers,
	"mushrooms": paintMushrooms, "tallgrass": paintTallgrass, "lily": paintLily, "stump": paintStump,
	"log": paintLog, "haystack": paintHaystack, "fence": paintFence, "ley-stone": paintLeyStone,
	"outcrop": paintOutcrop, "quarry": paintQuarry, "sapling": paintSapling,
}

var flatDecor = map[string]bool{"flowers": true, "tallgrass": true, "lily": true, "fern": true}

var softOutline = map[string]color.NRGBA{"reeds": rgb(30, 58, 36)}

type margin struct{ l, t, r, b int }

const decorPad = 2

var decorMargins = map[string]margin{"quarry": {decorPad, decorPad, 4, decorPad}, "fence": {decorPad, decorPad, decorPad, 6}}

func decorMargin(kind string) margin {
	if m, ok := decorMargins[kind]; ok {
		return m
	}
	return margin{decorPad, decorPad, decorPad, decorPad}
}

func (m margin) grow(s size) size {
	return size{w: s.w + m.l + m.r, h: s.h + m.t + m.b, ax: s.ax + m.l, ay: s.ay + m.t}
}

func paintInset(c *canvas, w, h int, m margin, paint func(t *canvas)) {
	t := newCanvas(w+m.r, h+m.b)
	paint(t)
	c.paste(t, m.l, m.t, false)
}

func addDecor(b *atlasBuilder) {
	for _, k := range DecorKinds {
		s, kind := decorSizes[k], k
		g := decorMargin(k).grow(s)
		for v := 0; v < DecorVariants; v++ {
			vv := v
			b.add(decorName(k, v), g.w, g.h, g.ax, g.ay, func(c *canvas) { paintDecor(c, kind, s, vv) })
		}
	}
}

func paintDecor(c *canvas, kind string, s size, v int) {
	m := decorMargin(kind)
	paintInset(c, s.w, s.h, m, func(t *canvas) { decorPainters[kind](t, s, v) })
	if flatDecor[kind] {
		return
	}
	col, ok := softOutline[kind]
	if !ok {
		col = ink
	}
	c.outline(col)
	g := m.grow(s)
	groundShadow(c, size{w: s.w, h: s.h, ax: g.ax, ay: g.ay})
}

func groundShadow(c *canvas, s size) {
	rx := float64(s.w) / 2.6
	for y := s.ay - 2; y <= s.ay+1; y++ {
		for x := 0; x < c.w; x++ {
			dx, dy := (float64(x)+0.5-float64(s.ax))/rx, (float64(y)+0.5-float64(s.ay))/2.2
			if dx*dx+dy*dy <= 1 && c.at(x, y).A == 0 {
				c.dot(x, y, shadow)
			}
		}
	}
}

func blob(c *canvas, cx, cy, r float64, rr ramp, seed int) {
	for y := int(cy - r - 1); y <= int(cy+r+1); y++ {
		for x := int(cx - r - 1); x <= int(cx+r+1); x++ {
			dx, dy := float64(x)+0.5-cx, float64(y)+0.5-cy
			if dx*dx+dy*dy > r*r {
				continue
			}
			light := -(dx + dy) / (r * 1.4)
			n := noise(x, y, seed)
			switch {
			case light > 0.45 && n > 0.3:
				c.dot(x, y, rr[3])
			case light > 0.05:
				c.dot(x, y, rr[2])
			case light < -0.5 || n < 0.1:
				c.dot(x, y, rr[0])
			default:
				c.dot(x, y, rr[1])
			}
		}
	}
}
