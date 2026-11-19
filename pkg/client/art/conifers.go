package art

import "image/color"

type tier struct {
	top, bottom int
	half        float64
}

func conifer(c *canvas, cx float64, tiers []tier, leaf ramp, seed int) {
	for ti, t := range tiers {
		for y := t.top; y <= t.bottom+1; y++ {
			frac := float64(y-t.top+1) / float64(t.bottom-t.top+1)
			w := t.half * min(frac, 1)
			for x := int(cx - w - 1); x <= int(cx+w+1); x++ {
				dx := float64(x) + 0.5 - cx
				if dx < -w || dx > w {
					continue
				}
				if y > t.bottom && (x+ti)%3 != 0 {
					continue
				}
				c.dot(x, y, needle(leaf, x, y, dx, w, y-t.top, ti, seed))
			}
		}
	}
}

func needle(leaf ramp, x, y int, dx, w float64, depth, ti, seed int) color.NRGBA {
	n := noise(x, y, seed)
	switch {
	case ti > 0 && depth <= 1 && n > 0.3:
		return leaf[0]
	case dx < -w*0.35 && n > 0.55:
		return leaf[3]
	case dx < 0:
		return leaf[2]
	case dx > w*0.55 || n < 0.1:
		return leaf[0]
	}
	return leaf[1]
}

func paintPine(c *canvas, s size, v int) {
	bark(c, 10, 34, s.ay, 3)
	roots(c, 10, s.ay-1, 3)
	shift := float64(v%2) * 0.5
	tiers := []tier{{2, 10, 3.5}, {7, 17, 5.5 + shift}, {13, 24, 7}, {19, 31, 8.5 - shift}, {26, 37, 10}}
	conifer(c, 11.5, tiers, pineLeaf, 60+v)
	if v == 2 {
		for _, p := range [][2]int{{7, 20}, {14, 27}, {9, 33}} {
			c.dot(p[0], p[1], woodR[2])
		}
	}
}

func paintSpruce(c *canvas, s size, v int) {
	bark(c, 7, 34, s.ay, 2)
	tiers := []tier{}
	for i := 0; i < 7; i++ {
		top := 1 + i*5
		tiers = append(tiers, tier{top, top + 6, 1.8 + float64(i)*0.95 + float64(v)*0.15})
	}
	conifer(c, 8, tiers, spruceLeaf, 70+v)
	c.dot(8, 0, spruceLeaf[2])
}
