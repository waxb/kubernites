package art

import (
	"fmt"
	"image/color"
	"math"
	"sort"
)

const (
	MiddenLevels   = 4
	MiddenVariants = 2
)

var middenSize = size{w: 44, h: 51, ax: 22, ay: 47}

var (
	rubbishR = ramp{rgb(56, 50, 46), rgb(84, 76, 66), rgb(114, 104, 88), rgb(150, 138, 116)}
	rotGreen = rgb(104, 118, 52)
	debris   = []color.NRGBA{tkParchment, greyR[2], rotGreen, woodR[2], rgb(122, 64, 54), rgb(70, 96, 120)}
)

type heapItem struct {
	kind      byte
	gx, gy, z float64
}

type moundSpec struct {
	rx, h float64
}

var middenMounds = [MiddenLevels]moundSpec{{0, 0}, {8, 3}, {13, 8}, {18, 17}}

var middenLayouts = [MiddenLevels][]heapItem{
	{{'c', 0.3, 0.28, 0}, {'b', 0.66, 0.36, 0}, {'s', 0.46, 0.7, 0}},
	{{'c', 0.22, 0.3, 0}, {'c', 0.56, 0.14, 0}, {'b', 0.74, 0.46, 0}, {'s', 0.38, 0.74, 0}, {'s', 0.64, 0.8, 0}, {'r', 0.34, 0.56, 1}},
	{{'c', 0.16, 0.3, 0}, {'c', 0.52, 0.1, 0}, {'c', 0.3, 0.36, 6}, {'b', 0.78, 0.5, 0}, {'t', 0.3, 0.86, 0}, {'s', 0.5, 0.5, 7},
		{'s', 0.7, 0.82, 0}, {'s', 0.14, 0.66, 1}, {'r', 0.6, 0.34, 5}, {'r', 0.46, 0.7, 3}},
	{{'c', 0.12, 0.2, 0}, {'c', 0.5, 0.02, 0}, {'c', 0.26, 0.3, 10}, {'c', 0.4, 0.3, 16}, {'b', 0.86, 0.46, 0}, {'t', 0.3, 0.9, 0},
		{'t', 0.62, 0.52, 11}, {'s', 0.52, 0.56, 15}, {'s', 0.1, 0.66, 2}, {'s', 0.76, 0.86, 1}, {'s', 1.02, 0.72, 0}, {'s', 0.22, 0.52, 9},
		{'r', 0.64, 0.24, 10}, {'r', 0.3, 0.64, 8}, {'r', 0.9, 0.8, 0}, {'f', 0.46, 0.4, 19}},
}

func middenName(level, v int) string { return fmt.Sprintf("midden/%d/%d", level, v) }

func addMidden(b *atlasBuilder) {
	s := middenSize
	for l := 0; l < MiddenLevels; l++ {
		for v := 0; v < MiddenVariants; v++ {
			level, variant := l, v
			b.add(middenName(l, v), s.w, s.h, s.ax, s.ay, func(c *canvas) { paintMidden(c, level, variant) })
		}
	}
}

func middenItems(level, v int) []heapItem {
	out := make([]heapItem, 0, len(middenLayouts[level]))
	for _, it := range middenLayouts[level] {
		if v == 1 {
			it.gx, it.gy = it.gy, it.gx
		}
		out = append(out, it)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].gx+out[i].gy+out[i].z/20 < out[j].gx+out[j].gy+out[j].z/20
	})
	return out
}

func paintMidden(c *canvas, level, v int) {
	s := isoFrame(c, 1, middenSize.ax, middenSize.ay)
	centre := s.p(0.5, 0.5, 0)
	floorShadow(c, centre.x, centre.y+1, 10+float64(level)*2.4)
	if level == 3 {
		ooze(c, centre.x, centre.y+2, v)
	}
	if m := middenMounds[level]; m.h > 0 {
		heap := newCanvas(c.w, c.h)
		mound(heap, centre.x, centre.y+1, m.rx, m.h, 90+v, level == 3)
		heap.outline(ink)
		c.paste(heap, 0, 0, false)
	}
	for _, it := range middenItems(level, v) {
		item := newCanvas(c.w, c.h)
		paintHeapItem(iso{c: item, ox: s.ox, oy: s.oy}, it)
		item.outline(ink)
		c.paste(item, 0, 0, false)
	}
	middenTufts(c, centre, 10+float64(level)*2.4, level*3+v)
	if level == 3 {
		stench(c, s, v)
	}
}

func middenTufts(c *canvas, centre pt, rx float64, seed int) {
	for i := 0; i < 9; i++ {
		a := math.Pi * (0.08 + 0.84*float64(i)/8)
		x, y := int(centre.x+math.Cos(a)*rx*1.05), int(centre.y+1+math.Sin(a)*rx*0.44)
		switch h := hash(i, seed, 151); {
		case h%3 == 0:
			c.dot(x, y, rockR[2])
			c.dot(x+1, y, rockR[1])
		case h%3 == 1:
			tuft(c, x, y, dryR, h)
		}
	}
}

func floorShadow(c *canvas, cx, cy, rx float64) {
	c.ellipse(cx, cy, rx, rx*0.42, withAlpha(shadow, 80))
}

func ooze(c *canvas, cx, cy float64, v int) {
	for i := 0; i < 4; i++ {
		h := hash(i, v, 101)
		c.ellipse(cx-14+float64(h%28), cy+float64(h/28%4)-1, 4+float64(h/112%3), 1.6, rgb(74, 82, 40))
	}
}

func paintHeapItem(s iso, it heapItem) {
	p := s.p(it.gx, it.gy, it.z)
	switch it.kind {
	case 'c':
		s.crateAt(it.gx, it.gy, 0.3, it.z)
	case 'b':
		barrelAt(s.c, p)
	case 't':
		tippedBarrel(s.c, p)
	case 's':
		scrollBundle(s.c, p)
	case 'r':
		sack(s.c, p, rgb(118, 104, 84))
	case 'f':
		warningFlag(s.c, p)
	}
}

func mound(c *canvas, cx, cy, rx, h float64, seed int, rot bool) {
	ry := rx * 0.5
	for y := int(cy - h - 1); y <= int(cy+ry+1); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			dx, dy := (float64(x)+0.5-cx)/rx, float64(y)+0.5-cy
			vy := dy / ry
			if dy < 0 {
				vy = dy / h
			}
			edge := 1 - noise(x, y, seed)*0.15
			if dx*dx+vy*vy > edge {
				continue
			}
			c.dot(x, y, moundColour(x, y, -dx-vy, seed, rot))
		}
	}
}

func moundColour(x, y int, light float64, seed int, rot bool) color.NRGBA {
	n := noise(x, y, seed+1)
	var col color.NRGBA
	switch {
	case n > 0.9:
		col = debris[hash(x, y, seed+2)%uint32(len(debris))]
	case light > 0.7:
		col = rubbishR[3]
	case light > 0.1:
		col = rubbishR[2]
	case light < -0.7:
		col = rubbishR[0]
	default:
		col = rubbishR[1]
	}
	if rot && noise(x/2, y/2, seed+3) > 0.6 {
		return mix(col, rotGreen, 0.45)
	}
	return col
}

func barrelAt(c *canvas, p pt) {
	c.rect(int(p.x)-3, int(p.y)-8, 6, 8, woodR[1])
	c.rect(int(p.x)-3, int(p.y)-8, 2, 8, woodR[2])
	c.rect(int(p.x)-3, int(p.y)-6, 6, 1, greyR[1])
	c.rect(int(p.x)-3, int(p.y)-3, 6, 1, greyR[1])
	c.ellipse(p.x, p.y-8, 3, 1.3, woodR[3])
}

func tippedBarrel(c *canvas, p pt) {
	c.rect(int(p.x)-5, int(p.y)-6, 8, 6, woodR[1])
	c.rect(int(p.x)-5, int(p.y)-6, 8, 2, woodR[2])
	c.rect(int(p.x)-3, int(p.y)-6, 1, 6, greyR[1])
	c.rect(int(p.x), int(p.y)-6, 1, 6, greyR[1])
	c.ellipse(p.x+3.5, p.y-3, 1.4, 3, woodR[3])
	c.dot(int(p.x)+3, int(p.y)-3, woodR[0])
}

func scrollBundle(c *canvas, p pt) {
	for i, o := range [][2]int{{-4, -2}, {1, -2}, {-2, -4}} {
		x, y := int(p.x)+o[0], int(p.y)+o[1]
		c.rect(x, y-1, 5, 2, tkParchment)
		c.rect(x, y, 5, 1, tkParchmentDark)
		c.dot(x+4, y-1, cream)
		c.dot(x+2, y-1+i%2, redR[2])
	}
}

func warningFlag(c *canvas, p pt) {
	c.line(int(p.x), int(p.y), int(p.x), int(p.y)-14, woodR[0])
	c.shadePoly([]pt{{p.x + 1, p.y - 14}, {p.x + 8, p.y - 11.5}, {p.x + 1, p.y - 8.5}}, flat(tkRed))
	c.dot(int(p.x)+2, int(p.y)-12, rgb(250, 150, 120))
	c.dot(int(p.x), int(p.y)-15, goldR[3])
}

func stench(c *canvas, s iso, v int) {
	top := s.p(0.5, 0.5, 20)
	for i := 0; i < 3; i++ {
		x0 := top.x - 9 + float64(i)*8 + float64(v)
		for k := 0; k < 9; k++ {
			x := x0 + math.Sin(float64(k)*0.9+float64(i+v))*1.5
			c.dot(int(x), int(top.y)-2-k, withAlpha(rgb(170, 204, 76), uint8(240-k*18)))
		}
	}
	for i := 0; i < 12; i++ {
		h := hash(i, v, 97)
		c.dot(int(top.x)-15+int(h%30), int(top.y)-2-int(h/30%14), ink)
	}
}
