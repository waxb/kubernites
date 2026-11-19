package art

import (
	"image/color"
	"math"
)

var (
	vgBubble  = rgb(250, 246, 232)
	vgSweat   = rgb(150, 210, 240)
	vgSeal    = rgb(196, 40, 44)
	vgBolt    = rgb(255, 250, 200)
	vgFly     = rgb(40, 34, 30)
	vgStink   = color.NRGBA{150, 190, 70, 150}
	vgChalk   = rgb(236, 90, 70)
	vgSack    = ramp{rgb(112, 84, 52), rgb(152, 116, 72), rgb(186, 150, 98), rgb(214, 184, 132)}
	vgWither  = ramp{rgb(92, 70, 44), rgb(128, 96, 56), rgb(160, 124, 70), rgb(190, 156, 96)}
	vgGlyphQ  = []string{"XXX", "..X", ".XX", "...", ".X."}
	vgGlyphEx = []string{".X.", ".X.", ".X.", "...", ".X."}
	vgGlyphNo = []string{"X.X", ".X.", "X.X", "...", "..."}
)

func vgGlyph(c *canvas, rows []string, x, y int, col color.NRGBA) {
	for j, row := range rows {
		for i, ch := range row {
			if ch == 'X' {
				c.dot(x+i, y+j, col)
			}
		}
	}
}

func vgBubbleArt(c *canvas, x, y int, rows []string, col color.NRGBA) {
	c.rect(x-4, y-5, 9, 11, vgInk)
	c.rect(x-5, y-4, 11, 9, vgInk)
	c.rect(x-3, y-4, 7, 9, vgBubble)
	c.rect(x-4, y-3, 9, 7, vgBubble)
	c.rect(x-1, y+5, 3, 2, vgInk)
	c.dot(x, y+5, vgBubble)
	c.dot(x-1, y+7, vgInk)
	vgGlyph(c, rows, x-1, y-2, col)
}

func (p *vgPaint) bubbleAt(x, y int, rows []string, col color.NRGBA, scale int) {
	s := p.stamp(13, 14, 6, 13, func(c *canvas) { vgBubbleArt(c, 6, 5, rows, col) })
	p.putSprite(s, x, y+p.bob(1, float64(x)), vgPut{scale: scale})
}

func (p *vgPaint) sweat(x, y int) {
	dy := p.cyc(6) / 2
	p.c.dot(x, y+dy, vgCloud)
	p.c.rect(x, y+dy+1, 2, 2, vgSweat)
	p.c.dot(x+1, y+dy, vgSweat)
}

func (p *vgPaint) stars(x, y int) {
	for k := 0; k < 3; k++ {
		a := p.phase() + float64(k)*2*math.Pi/3
		sx := x + int(math.Round(6*math.Cos(a)))
		sy := y + int(math.Round(2*math.Sin(a)))
		col := goldR[3]
		if math.Sin(a) > 0.2 {
			col = goldR[2]
		}
		p.c.dot(sx, sy, col)
		p.c.dot(sx-1, sy, col)
		p.c.dot(sx+1, sy, col)
		p.c.dot(sx, sy-1, col)
		p.c.dot(sx, sy+1, col)
	}
}

func vgGem(c *canvas, x, y int, r ramp) {
	c.dot(x, y-1, r[3])
	c.rect(x-1, y, 3, 1, r[2])
	c.dot(x, y, r[3])
	c.dot(x, y+1, r[1])
	c.dot(x-1, y+1, r[0])
	c.dot(x+1, y+1, r[0])
}

func vgSpillArt(c *canvas, x, y, n int) {
	for i := 0; i < n; i++ {
		r := blueR
		if i%3 == 1 {
			r = voidR
		}
		vgGem(c, x+i*4+int(hash(i, 3, 41)%3), y-int(hash(i, 5, 41)%4), r)
	}
}

func (p *vgPaint) spill(x, y, n, scale int) {
	w := n*4 + 6
	s := p.stamp(w, 8, 1, 6, func(c *canvas) { vgSpillArt(c, 2, 6, n) })
	p.putSprite(s, x, y, vgPut{scale: scale})
	i := p.cyc(n)
	gx := x + scale*(1+i*4+int(hash(i, 3, 41)%3))
	gy := y - scale*(int(hash(i, 5, 41)%4)+2)
	p.c.rect(gx, gy, scale, scale, white)
}

func (p *vgPaint) bolt(x0, y0, x1, y1 int, lit bool) {
	pts := []pt{{float64(x0), float64(y0)}}
	steps := 6
	for i := 1; i < steps; i++ {
		t := float64(i) / float64(steps)
		pts = append(pts, pt{float64(x0) + (float64(x1)-float64(x0))*t + float64(p.jit(4)), float64(y0) + (float64(y1)-float64(y0))*t})
	}
	pts = append(pts, pt{float64(x1), float64(y1)})
	if !lit {
		return
	}
	for i := 1; i < len(pts); i++ {
		a, b := pts[i-1], pts[i]
		p.c.line(int(a.x)+1, int(a.y), int(b.x)+1, int(b.y), goldR[2])
		p.c.line(int(a.x), int(a.y), int(b.x), int(b.y), vgBolt)
	}
}

func vgScrollArt(c *canvas, x, y int, seal bool) {
	c.rect(x-6, y-3, 12, 5, plasterR[3])
	c.rect(x-6, y+1, 12, 1, plasterR[1])
	c.rect(x-7, y-4, 2, 7, plasterR[2])
	c.rect(x+5, y-4, 2, 7, plasterR[2])
	c.rect(x-7, y-4, 1, 7, plasterR[0])
	c.rect(x+6, y-4, 1, 7, plasterR[0])
	c.rect(x-3, y-2, 6, 1, plasterR[0])
	if seal {
		c.rect(x+1, y+1, 2, 4, vgSeal)
		c.disc(float64(x)+0.5, float64(y)+0.5, 2.2, vgSeal)
		c.dot(x, y-1, redR[3])
	}
	c.outline(vgInk)
}

func (p *vgPaint) scroll(x, y int, seal bool) {
	vgScrollArt(p.c, x, y, seal)
}

func (p *vgPaint) scrollAt(x, y, scale int) {
	s := p.stamp(18, 12, 9, 5, func(c *canvas) { vgScrollArt(c, 9, 5, true) })
	p.putSprite(s, x, y, vgPut{scale: scale})
}

func (p *vgPaint) rope(x0, y0, x1, y1 int, sag float64) {
	px, py := x0, y0
	for i := 1; i <= 32; i++ {
		t := float64(i) / 32
		x := x0 + int(math.Round(float64(x1-x0)*t))
		y := y0 + int(math.Round(float64(y1-y0)*t+sag*4*t*(1-t)))
		p.c.line(px, py+1, x, y+1, woodR[0])
		p.c.line(px, py, x, y, sandR[2])
		px, py = x, y
	}
}

func vgMarkArt(c *canvas, x, y int, col color.NRGBA) {
	c.rect(x-1, y-2, 3, 5, vgInk)
	c.rect(x-2, y-1, 5, 3, vgInk)
	c.rect(x-1, y-1, 3, 3, col)
	c.dot(x, y-2, col)
	c.dot(x, y+2, col)
	c.dot(x-1, y-1, mix(col, white, 0.5))
	c.outline(vgInk)
}

func (p *vgPaint) markAt(x, y int, col color.NRGBA, scale int) {
	s := p.stamp(9, 9, 4, 4, func(c *canvas) { vgMarkArt(c, 4, 4, col) })
	p.putSprite(s, x, y+p.bob(1, float64(x)*0.3), vgPut{scale: scale})
}

func (p *vgPaint) cracks(x, y, n int) {
	for i := 0; i < n; i++ {
		x0 := x + (i-n/2)*17 + int(hash(i, 1, 13)%5)
		y0 := y + int(hash(i, 2, 13)%6)
		p.c.line(x0, y0, x0+3, y0+1, vgInk)
		p.c.line(x0+3, y0+1, x0+5, y0, vgInk)
		p.c.line(x0+3, y0+1, x0+4, y0+3, vgInk)
		p.c.dot(x0-1, y0-1, vgDryR[3])
	}
}

func vgArmArt(c *canvas, x, y, dir int, sleeve color.NRGBA) {
	for i := 0; i < 5; i++ {
		c.dot(x+dir*i, y-i/3, sleeve)
	}
	c.dot(x+dir*5, y-2, skinR[2])
	c.dot(x+dir*6, y-2, skinR[2])
}

func (p *vgPaint) armAt(x, y, dir int, sleeve color.NRGBA, scale int) {
	s := p.stamp(9, 5, 1, 3, func(c *canvas) { vgArmArt(c, 1+max(-dir, 0)*6, 3, dir, sleeve) })
	if dir < 0 {
		s.ax = 7
	}
	p.putSprite(s, x, y, vgPut{scale: scale})
}

func (p *vgPaint) flies(x, y, n int) {
	for k := 0; k < n; k++ {
		a := p.phase()*2 + float64(k)*2.1
		p.c.dot(x+int(math.Round(5*math.Cos(a))), y+int(math.Round(3*math.Sin(a*1.5))), vgFly)
	}
}

func (p *vgPaint) stink(x, y int) {
	for k := 0; k < 3; k++ {
		yy := y - (p.frame+k*4)%12
		xx := x + k*3 - 3 + int(math.Round(1.5*math.Sin(float64(yy)*0.6)))
		p.c.blend(xx, yy, vgStink)
		p.c.blend(xx, yy-1, vgStink)
	}
}

func vgHourglassArt(c *canvas, x, y, step int) {
	c.rect(x-3, y-12, 7, 1, woodR[1])
	c.rect(x-3, y, 7, 1, woodR[1])
	c.rect(x-3, y-11, 1, 11, woodR[0])
	c.rect(x+3, y-11, 1, 11, woodR[0])
	for i := 0; i < 5; i++ {
		w := 5 - i
		c.rect(x-w/2, y-11+i, w, 1, glassR[3])
		c.rect(x-w/2, y-1-i, w, 1, glassR[3])
	}
	left := 3 - step/4
	for i := 0; i < left; i++ {
		c.rect(x-(left-i)/2, y-8+i, left-i, 1, sandR[2])
	}
	c.rect(x-1, y-2, 3, 2, sandR[2])
	if step%2 == 0 {
		c.dot(x, y-5, sandR[3])
	}
	c.outline(vgInk)
}

func (p *vgPaint) hourglassAt(x, y, scale int) {
	step := p.cyc(12)
	s := p.stamp(11, 16, 5, 14, func(c *canvas) { vgHourglassArt(c, 5, 14, step) })
	p.putSprite(s, x, y, vgPut{scale: scale})
}

func (p *vgPaint) tally(x, y int) {
	p.c.rect(x-12, y-14, 25, 11, woodR[2])
	p.c.rect(x-12, y-14, 25, 1, woodR[3])
	p.c.rect(x-12, y-4, 25, 1, woodR[0])
	p.c.rect(x-1, y-3, 2, 4, woodR[1])
	marks := []int{-10, -8, -4, 3}
	for _, m := range marks {
		p.c.rect(x+m, y-12, 1, 6, vgInk)
	}
	p.c.line(x+6, y-12, x+10, y-7, vgSeal)
}

func vgSignArt(c *canvas, x, y int) {
	c.rect(x, y-18, 2, 18, woodR[1])
	c.rect(x-6, y-18, 14, 7, woodR[2])
	c.rect(x-6, y-18, 14, 1, woodR[3])
	c.rect(x-6, y-12, 14, 1, woodR[0])
	vgGlyph(c, vgGlyphQ, x-1, y-17, vgInk)
	c.outline(vgInk)
}

func (p *vgPaint) signAt(x, y, scale int) {
	s := p.stamp(18, 21, 8, 19, func(c *canvas) { vgSignArt(c, 8, 19) })
	p.putSprite(s, x, y, vgPut{scale: scale})
}

func vgPennantArt(c *canvas, x, y, height, sway int, col color.NRGBA) {
	c.rect(x, y-height, 1, height, woodR[0])
	c.rect(x+1, y-height, 1, height, woodR[2])
	for i := 0; i < 5; i++ {
		c.rect(x+2, y-height+1+i, 9-i*2+sway, 1, col)
	}
	c.rect(x+2, y-height+1, 7+sway, 1, mix(col, white, 0.35))
	c.dot(x, y-height-1, goldR[3])
}

func (p *vgPaint) pennantAt(x, y int, col color.NRGBA, scale int) {
	sway := p.bob(1, 0) + 1
	s := p.stamp(14, 32, 1, 31, func(c *canvas) { vgPennantArt(c, 1, 31, 30, sway, col) })
	p.putSprite(s, x, y, vgPut{scale: scale})
}

func vgTorchArt(c *canvas, x, y, step int) {
	c.line(x, y, x+3, y-7, woodR[1])
	c.rect(x+3, y-10, 2, 3, fireR[step%3+1])
	c.dot(x+3+step%2, y-11, fireR[3])
	c.dot(x+4, y-8, fireR[0])
}

func (p *vgPaint) torchAt(x, y, scale int) {
	step := p.cyc(3)
	s := p.stamp(8, 13, 1, 12, func(c *canvas) { vgTorchArt(c, 1, 12, step) })
	p.putSprite(s, x, y, vgPut{scale: scale})
}

func (p *vgPaint) motes(x, y int, r ramp) {
	for k := 0; k < 4; k++ {
		yy := y - (p.frame*2+k*6)%22
		xx := x + int(math.Round(4*math.Sin(float64(yy)*0.3+float64(k))))
		p.c.dot(xx, yy, r[3])
		p.c.dot(xx+1, yy, r[2])
	}
}

func (p *vgPaint) fallingScroll(x, y0, y1, offset int) {
	span := y1 - y0
	y := y0 + (p.frame*span/VignetteFrames+offset)%span
	p.c.rect(x-3, y, 7, 3, plasterR[3])
	p.c.rect(x-3, y+2, 7, 1, plasterR[1])
	p.c.dot(x-4, y+1, plasterR[0])
	p.c.dot(x+4, y+1, plasterR[0])
}

func (p *vgPaint) shadow(x, y, rx int) {
	p.glowEllipse(x, y, rx, 2, shadow)
}

func (p *vgPaint) glowEllipse(x, y, rx, ry int, col color.NRGBA) {
	for yy := y - ry; yy <= y+ry; yy++ {
		for xx := x - rx; xx <= x+rx; xx++ {
			dx, dy := float64(xx-x)/float64(max(rx, 1)), float64(yy-y)/float64(max(ry, 1))
			if dx*dx+dy*dy <= 1 {
				p.c.blend(xx, yy, col)
			}
		}
	}
}

func vgWithered(c color.NRGBA) color.NRGBA {
	if c.G > c.R && c.G > c.B {
		l := (int(c.R) + int(c.G) + int(c.B)) / 3
		switch {
		case l < 70:
			return vgWither[0]
		case l < 105:
			return vgWither[1]
		case l < 140:
			return vgWither[2]
		}
		return vgWither[3]
	}
	return c
}

func vgSilhouette(to color.NRGBA, t float64) func(color.NRGBA) color.NRGBA {
	return func(c color.NRGBA) color.NRGBA {
		m := mix(c, to, t)
		m.A = c.A
		return m
	}
}

func vgCharred(c color.NRGBA) color.NRGBA {
	m := mix(c, rgb(40, 30, 30), 0.55)
	m.A = c.A
	return m
}

func vgSackArt(c *canvas, x, y int) {
	c.ellipse(float64(x), float64(y-5), 8, 6, vgSack[1])
	c.ellipse(float64(x-1), float64(y-7), 6, 4, vgSack[2])
	c.ellipse(float64(x+2), float64(y-3), 5, 2, vgSack[0])
	c.line(x-3, y-11, x+1, y-9, vgSack[0])
	c.line(x+1, y-9, x-1, y-6, vgSack[0])
	c.line(x+4, y-9, x+7, y-6, vgInk)
	c.line(x+5, y-10, x+8, y-8, vgInk)
	c.rect(x-2, y-12, 5, 2, vgSack[3])
	c.line(x-2, y-13, x+2, y-13, woodR[0])
	c.outline(vgInk)
}

func (p *vgPaint) sack(x, y, scale int) {
	s := p.stamp(20, 17, 10, 15, func(c *canvas) { vgSackArt(c, 10, 15) })
	p.putSprite(s, x, y, vgPut{scale: scale})
}

func vgRotten(c color.NRGBA) color.NRGBA {
	m := mix(c, rgb(96, 110, 52), 0.22)
	m.A = c.A
	return m
}

func (p *vgPaint) shimmer(x, y, w int) {
	for i := 0; i < w; i += 9 {
		yy := y + int(hash(i, 1, 55)%10)
		off := p.bob(1, float64(i)*0.7)
		for k := 0; k < 5; k++ {
			p.c.blend(x+i+k, yy+off+(k%2), color.NRGBA{255, 250, 220, 70})
		}
	}
}

func (p *vgPaint) burst(x, y int) {
	col := color.NRGBA{235, 210, 255, 220}
	for k := 0; k < 5; k++ {
		a := -2.4 + float64(k)*0.45
		r0 := 4 + p.cyc(4)
		for r := r0; r < r0+4; r++ {
			p.c.blend(x+int(math.Round(float64(r)*math.Cos(a))), y+int(math.Round(float64(r)*math.Sin(a))), col)
		}
	}
}

func (p *vgPaint) diamond(x, y, rx, ry int, front bool) {
	pts := [][2]int{{x - rx, y}, {x, y - ry}, {x + rx, y}, {x, y + ry}}
	for e := 0; e < 4; e++ {
		a, b := pts[e], pts[(e+1)%4]
		if front != (e >= 2) {
			continue
		}
		n := max(abs(b[0]-a[0]), 1)
		for i := 0; i <= n; i++ {
			if (i/4)%2 == 1 {
				continue
			}
			px := a[0] + (b[0]-a[0])*i/n
			py := a[1] + (b[1]-a[1])*i/n
			p.c.dot(px, py, vgChalk)
			p.c.dot(px, py+1, vgChalk)
		}
	}
}

func (p *vgPaint) fallenPennant(x, y int, col color.NRGBA) {
	s := p.stamp(14, 32, 1, 31, func(c *canvas) { vgPennantArt(c, 1, 31, 30, 1, col) })
	p.putSprite(s, x, y, vgPut{scale: 2, angle: -1.3})
}
