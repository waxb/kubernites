package art

import (
	"fmt"
	"image/color"
	"math"
)

const (
	CodexBloatLevels = 4
	bloatSpillX      = 14
	bloatSpillDown   = 8
	bloatSpillUp     = 8
)

var (
	paperR = ramp{tkParchmentDark, rgb(206, 184, 140), tkParchment, cream}
	spines = []color.NRGBA{redR[1], blueR[2], tealR[1], greenR[1], goldR[1], purpleR[2], woodR[1]}
)

type paperMound struct {
	gx, gy, rx, h float64
}

type bookPile struct {
	gx, gy, z float64
	n         int
}

type bloatLayout struct {
	mounds  []paperMound
	stacks  []bookPile
	roof    []bookPile
	loose   [][2]float64
	windows int
	sheets  int
}

var bloatLayouts = [CodexBloatLevels]bloatLayout{
	{stacks: []bookPile{{0.22, 1.5, 3, 3}}},
	{mounds: []paperMound{{0.72, 1.62, 5, 3}}, stacks: []bookPile{{0.2, 1.48, 3, 4}, {1.3, 1.08, 3, 3}}, loose: [][2]float64{{1.1, 1.8}, {1.7, 1.5}}},
	{mounds: []paperMound{{0.72, 1.64, 11, 7}, {1.66, 1.32, 7, 5}}, stacks: []bookPile{{0.18, 1.45, 3, 6}, {1.3, 1.02, 3, 6}, {1.84, 1.76, 2, 4}},
		loose: [][2]float64{{1.1, 1.86}, {0.3, 1.9}, {1.9, 1.2}, {1.4, 1.9}, {0.9, 1.35}}, windows: 2},
	{mounds: []paperMound{{0.8, 1.62, 20, 20}, {1.62, 1.26, 13, 15}, {1.86, 1.86, 9, 7}},
		stacks: []bookPile{{0.14, 1.4, 3, 11}, {1.26, 0.98, 3, 10}, {1.95, 1.5, 4, 5}}, roof: []bookPile{{1.62, 0.62, 26, 3}},
		loose: [][2]float64{{1.1, 2.0}, {0.3, 2.0}, {2.05, 1.2}, {1.5, 2.05}, {0.1, 1.8}, {2.05, 1.7}, {0.6, 2.1}, {1.9, 2.05}}, windows: 3, sheets: 7},
}

func codexBloatName(level int) string { return fmt.Sprintf("codexbloat/%d", level) }

func bloatSize() size {
	b := buildingSize("codex")
	w := b.w + 2*bloatSpillX
	ay := b.ay + bloatSpillUp
	return size{w: w, h: ay + bloatSpillDown, ax: w / 2, ay: ay}
}

func addCodexBloat(b *atlasBuilder) {
	s := bloatSize()
	for l := 0; l < CodexBloatLevels; l++ {
		level := l
		b.add(codexBloatName(l), s.w, s.h, s.ax, s.ay, func(c *canvas) { paintCodexBloat(c, s, level) })
	}
}

func paintCodexBloat(c *canvas, sz size, level int) {
	s := isoFrame(c, Footprints["codex"], sz.ax, sz.ay)
	l := bloatLayouts[level]
	windowScrolls(s, l.windows)
	for i, st := range l.roof {
		p := s.p(st.gx, st.gy, st.z)
		bookStack(c, p.x, p.y, st.n, 7+i)
	}
	for i, m := range l.mounds {
		p := s.p(m.gx, m.gy, 0)
		paperHeap(c, p.x, p.y, m.rx, m.h, 40+i+level*5)
	}
	for i, st := range l.stacks {
		p := s.p(st.gx, st.gy, st.z)
		bookStack(c, p.x, p.y, st.n, 11+i+level*3)
	}
	for i, q := range l.loose {
		p := s.p(q[0], q[1], 0)
		looseScroll(c, p.x, p.y, i)
	}
	c.outline(ink)
	flyingSheets(c, s, l.sheets)
}

func paperHeap(c *canvas, cx, cy, rx, h float64, seed int) {
	ry := rx * 0.5
	inside := func(x, y int, margin float64) bool {
		dx, dy := (float64(x)+0.5-cx)/rx, float64(y)+0.5-cy
		vy := dy / ry
		if dy < 0 {
			vy = dy / h
		}
		return dx*dx+vy*vy <= margin-noise(x, y, seed)*0.12
	}
	for y := int(cy - h - 1); y <= int(cy+ry+1); y++ {
		for x := int(cx - rx - 1); x <= int(cx+rx+1); x++ {
			if inside(x, y, 1) {
				c.dot(x, y, paperShade(x, y, cx, cy, rx))
			}
		}
	}
	for y := int(cy - h); y <= int(cy+ry); y += 2 {
		for x := int(cx-rx) + (y/2)%2*3; x <= int(cx+rx); x += 5 {
			if inside(x, y, 0.8) && inside(x+4, y+1, 0.8) {
				heapPiece(c, x, y, hash(x, y, seed+5))
			}
		}
	}
}

func paperShade(x, y int, cx, cy, rx float64) color.NRGBA {
	light := -(float64(x)+0.5-cx)/rx - (float64(y)+0.5-cy)/rx
	switch {
	case light > 0.5:
		return paperR[3]
	case light > -0.1:
		return paperR[2]
	case light < -0.6:
		return paperR[0]
	}
	return paperR[1]
}

func heapPiece(c *canvas, x, y int, h uint32) {
	switch h % 10 {
	case 0, 1, 2, 3, 4:
		col := spines[h/10%uint32(len(spines))]
		c.rect(x, y, 5, 1, col)
		c.rect(x+1, y+1, 3, 1, cream)
		c.dot(x, y+1, darken(col, 0.3))
		c.dot(x+4, y+1, darken(col, 0.3))
	case 5, 6:
		c.rect(x, y, 5, 2, tkParchment)
		c.rect(x, y+1, 5, 1, tkParchmentDark)
		c.dot(x+2, y, redR[2])
	}
}

func bookStack(c *canvas, x, y float64, n, seed int) {
	for i := 0; i < n; i++ {
		h := hash(i, seed, 113)
		w := 7 + int(h%3) - 1
		x0 := int(x) - w/2 + int(h/3%3) - 1
		yb := int(y) - 2*(i+1)
		col := spines[h/9%uint32(len(spines))]
		c.rect(x0, yb, w, 2, col)
		c.rect(x0+1, yb+1, w-2, 1, cream)
		c.dot(x0, yb, mix(col, white, 0.3))
	}
}

func looseScroll(c *canvas, x, y float64, i int) {
	x0, y0 := int(x)-2, int(y)-2
	if i%2 == 0 {
		c.rect(x0, y0, 5, 2, tkParchment)
		c.rect(x0, y0+1, 5, 1, tkParchmentDark)
		c.dot(x0+2, y0, redR[2])
		return
	}
	c.rect(x0, y0-1, 4, 3, cream)
	c.rect(x0+1, y0, 2, 1, inkSft)
}

func windowScrolls(s iso, n int) {
	spots := [][3]float64{{0.68, 1.25, 30}, {0.93, 1.25, 40}, {1.2, 0.68, 32}}
	for i := 0; i < n; i++ {
		w := spots[i]
		p := s.p(w[0], w[1], w[2])
		x, y := int(p.x)-1, int(p.y)
		s.c.rect(x, y-1, 4, 2, tkParchment)
		s.c.rect(x, y+1, 4, 6-i, cream)
		for k := 0; k < 2; k++ {
			s.c.rect(x+1, y+2+k*2, 2, 1, inkSft)
		}
		s.c.dot(x+1, y+7-i, tkParchmentDark)
	}
}

func flyingSheets(c *canvas, s iso, n int) {
	top := s.p(0.75, 0.8, 58)
	for i := 0; i < n; i++ {
		a := float64(i)*2.4 + 0.5
		r := 12 + float64(i%3)*5
		x, y := int(top.x+math.Cos(a)*r), int(top.y+math.Sin(a)*r*0.6)
		c.rect(x, y, 3, 2, cream)
		c.dot(x+i%2, y+1, inkSft)
		c.dot(x+3, y+i%2, tkParchmentDark)
	}
}
