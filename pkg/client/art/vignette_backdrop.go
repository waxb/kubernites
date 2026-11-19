package art

import (
	"image/color"
	"math"
)

var (
	vgInk      = rgb(18, 14, 22)
	vgNoonTop  = rgb(92, 150, 200)
	vgNoonLow  = rgb(196, 224, 226)
	vgDuskTop  = rgb(52, 42, 92)
	vgDuskLow  = rgb(226, 150, 110)
	vgNightTop = rgb(16, 22, 48)
	vgNightLow = rgb(58, 70, 112)
	vgStormTop = rgb(34, 38, 52)
	vgStormLow = rgb(86, 96, 108)
	vgFogTop   = rgb(128, 140, 150)
	vgFogLow   = rgb(186, 196, 198)
	vgDawnTop  = rgb(44, 58, 118)
	vgDawnLow  = rgb(250, 176, 110)
	vgHazeTop  = rgb(214, 170, 96)
	vgHazeLow  = rgb(246, 222, 158)
	vgCloud    = rgb(244, 240, 232)
	vgCloudSh  = rgb(196, 200, 214)
	vgStormCl  = rgb(70, 74, 90)
	vgStormSh  = rgb(46, 48, 62)
	vgFarGreen = rgb(96, 134, 110)
	vgFarBlue  = rgb(92, 112, 140)
)

func (p *vgPaint) sky(top, bot color.NRGBA, y1 int) {
	for y := 0; y < min(y1+6, p.h); y++ {
		t := min(float64(y)/float64(max(y1-1, 1)), 1)
		for x := 0; x < p.w; x++ {
			q := math.Floor(t*6+dither(x, y)) / 6
			p.c.dot(x, y, mix(top, bot, min(q, 1)))
		}
	}
}

func (p *vgPaint) clouds(n, y0, y1 int, light, shade color.NRGBA) {
	for i := 0; i < n; i++ {
		x := float64(p.bgIntn(p.w))
		y := float64(y0 + p.bgIntn(max(y1-y0, 1)))
		r := float64(4 + p.bgIntn(5))
		x += float64(p.bob(1.4, float64(i)*1.7))
		p.puff(x, y, r, light, shade)
	}
}

func (p *vgPaint) puff(x, y, r float64, light, shade color.NRGBA) {
	p.c.ellipse(x, y+r*0.35, r*1.9, r*0.45, shade)
	p.c.ellipse(x-r*0.7, y, r*0.9, r*0.6, light)
	p.c.ellipse(x+r*0.3, y-r*0.35, r*1.0, r*0.8, light)
	p.c.ellipse(x+r*1.1, y+r*0.05, r*0.7, r*0.5, light)
	p.c.ellipse(x, y+r*0.3, r*1.6, r*0.3, light)
}

func (p *vgPaint) sun(x, y, r float64, core, glow color.NRGBA) {
	for i := 3; i >= 1; i-- {
		g := glow
		g.A = uint8(40 * i)
		p.glowDisc(x, y, r+float64(4-i)*3, g)
	}
	p.c.disc(x, y, r, core)
}

func (p *vgPaint) softGlow(x, y, rx, ry int, col color.NRGBA) {
	for yy := y - ry; yy <= y+ry; yy++ {
		for xx := x - rx; xx <= x+rx; xx++ {
			dx, dy := float64(xx-x)/float64(max(rx, 1)), float64(yy-y)/float64(max(ry, 1))
			d := dx*dx + dy*dy
			if d >= 1 {
				continue
			}
			c := col
			c.A = uint8(float64(col.A) * (1 - d) * (1 - d))
			p.c.blend(xx, yy, c)
		}
	}
}

func (p *vgPaint) glowDisc(x, y, r float64, col color.NRGBA) {
	for yy := int(y - r - 1); yy <= int(y+r+1); yy++ {
		for xx := int(x - r - 1); xx <= int(x+r+1); xx++ {
			if math.Hypot(float64(xx)+0.5-x, float64(yy)+0.5-y) <= r {
				p.c.blend(xx, yy, col)
			}
		}
	}
}

func (p *vgPaint) hills(y, amp int, col color.NRGBA) {
	a, b := float64(p.bgIntn(100))/10, float64(p.bgIntn(100))/10
	rim := mix(col, vgCloud, 0.25)
	for x := 0; x < p.w; x++ {
		top := y - int(math.Round(float64(amp)*(0.55+0.3*math.Sin(float64(x)*0.045+a)+0.15*math.Sin(float64(x)*0.13+b))))
		for yy := top; yy < p.h; yy++ {
			p.c.dot(x, yy, col)
		}
		p.c.dot(x, top, rim)
	}
}

func (p *vgPaint) sea(y0, y1 int, far, near color.NRGBA) {
	ph := math.Sin(p.phase())
	for y := y0; y < min(y1, p.h); y++ {
		t := float64(y-y0) / float64(max(y1-y0-1, 1))
		for x := 0; x < p.w; x++ {
			q := math.Floor(t*4+dither(x, y)) / 4
			col := mix(far, near, min(q, 1))
			s := math.Sin(float64(x)*(0.19+0.02*float64(y%3)) + float64(y)*1.9 + ph*1.3)
			switch {
			case s > 0.94 && hash(x/4, y, 17)%3 != 0:
				col = mix(col, vgCloud, 0.55)
			case s < -0.95:
				col = darken(col, 0.18)
			}
			p.c.dot(x, y, col)
		}
	}
}

func (p *vgPaint) groundLine(x, y0 int, a, b float64) int {
	return y0 + int(math.Round(1.4*math.Sin(float64(x)*0.05+a)+0.9*math.Sin(float64(x)*0.14+b)))
}

func (p *vgPaint) land(y0 int, r ramp, haze color.NRGBA) {
	a, b := float64(p.bgIntn(100))/10, float64(p.bgIntn(100))/10
	for x := 0; x < p.w; x++ {
		top := p.groundLine(x, y0, a, b)
		for y := top; y < p.h; y++ {
			col := speckle(r, x, y, 31, 0.08)
			if y == top {
				col = r[2]
			}
			depth := 1 - float64(y-top)/12
			if depth > 0 {
				col = mix(col, haze, 0.35*depth)
			}
			p.c.dot(x, y, col)
		}
	}
}

func vgRows(y0, y1 int) []int {
	out := []int{}
	for y, h := y0, 3; y < y1; y, h = y+h, min(h+1, 7) {
		out = append(out, y)
	}
	return out
}

func vgRowOf(rows []int, y int) (int, int, int) {
	for i := len(rows) - 1; i >= 0; i-- {
		if y >= rows[i] {
			next := rows[i] + 7
			if i+1 < len(rows) {
				next = rows[i+1]
			}
			return i, y - rows[i], next - rows[i]
		}
	}
	return 0, 0, 1
}

func (p *vgPaint) planks(y0, y1 int) {
	rows := vgRows(y0, y1)
	for y := y0; y < min(y1, p.h); y++ {
		row, dy, hgt := vgRowOf(rows, y)
		for x := 0; x < p.w; x++ {
			col := woodR[2]
			switch {
			case dy == hgt-1:
				col = woodR[0]
			case (x+row*13)%(20+row*4) == 0:
				col = woodR[1]
			case dy == 0 && hash(x, y, 5)%3 == 0:
				col = woodR[3]
			case hash(x, y, 9)%11 == 0:
				col = woodR[1]
			}
			p.c.dot(x, y, col)
		}
	}
	p.c.rect(0, y0, p.w, 1, woodR[3])
	p.c.rect(0, y0-1, p.w, 1, vgInk)
}

func (p *vgPaint) plaza(y0, y1 int) {
	rows := vgRows(y0, y1)
	for y := y0; y < min(y1, p.h); y++ {
		row, dy, hgt := vgRowOf(rows, y)
		stone := 6 + row*2
		for x := 0; x < p.w; x++ {
			sx := (x + row*5) % stone
			col := plazaR[2]
			switch {
			case dy == hgt-1 || sx == stone-1:
				col = plazaR[0]
			case dy == 0 || sx == 0:
				col = plazaR[3]
			case hash((x+row*5)/stone, row, 3)%4 == 0:
				col = plazaR[1]
			}
			p.c.dot(x, y, col)
		}
	}
}

func (p *vgPaint) rain(col color.NRGBA, n int) {
	for i := 0; i < n; i++ {
		x0 := int(hash(i, 1, 77) % uint32(p.w+30))
		y0 := int(hash(i, 2, 77)%uint32(p.h)) + p.frame*7
		for k := 0; k < 5; k++ {
			y := (y0 + k) % p.h
			p.c.blend(x0-y/3, y, col)
		}
	}
}

func (p *vgPaint) fog(y0, y1 int, col color.NRGBA, peak float64) {
	ph := math.Sin(p.phase())
	mid, half := float64(y0+y1)/2, float64(y1-y0)/2
	for y := y0; y < y1; y++ {
		a := peak * (1 - math.Abs(float64(y)-mid)/half)
		for x := 0; x < p.w; x++ {
			n := 0.55 + 0.45*math.Sin(float64(x)*0.06+float64(y)*0.35+ph*0.9)
			c := col
			c.A = uint8(255 * min(max(a*n, 0), 1))
			p.c.blend(x, y, c)
		}
	}
}

func (p *vgPaint) edges() {
	for y := 0; y < p.h; y++ {
		for x := 0; x < p.w; x++ {
			d := min(min(x, p.w-1-x), min(y, p.h-1-y))
			if d >= 7 {
				continue
			}
			t := float64(7-d) / 7 * 0.5
			if d == 0 {
				t = 1
			}
			p.c.dot(x, y, mix(p.c.at(x, y), vgInk, t))
		}
	}
}

func (p *vgPaint) tint(col color.NRGBA, t float64) {
	for y := 0; y < p.h; y++ {
		for x := 0; x < p.w; x++ {
			p.c.dot(x, y, mix(p.c.at(x, y), col, t))
		}
	}
}
