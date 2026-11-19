package art

import (
	"fmt"
	"image/color"
	"math"
)

const (
	WispFrames  = 6
	CartFrames  = 4
	CartLoads   = 3
	ScrollPiles = 4
	StallLooks  = 3
	stallW      = 34
	stallH      = 42
	stallAX     = 17
	stallAY     = 40
	cartW       = 30
	cartH       = 24
	cartPad     = 2
	wispPad     = 1
)

func AmbientName(kind string, variant int) string {
	return fmt.Sprintf("ambient/%s/%d", kind, variant)
}

func CartName(load int, dir string, frame int) string {
	return fmt.Sprintf("ambient/cart/%d/%s/%d", load, dir, frame)
}

type propJob struct {
	kind     string
	variants int
	w, h     int
	ax, ay   int
	pad      int
	paint    func(c *canvas, v int)
}

var ambientProps = []propJob{
	{kind: "lamp", variants: 2, w: 7, h: 22, ax: 3, ay: 21, pad: 1, paint: paintLampPost},
	{kind: "ward", variants: 1, w: 22, h: 26, ax: 11, ay: 24, pad: 1, paint: paintWard},
	{kind: "idle", variants: 1, w: 15, h: 13, ax: 7, ay: 12, pad: 2, paint: paintIdleBubble},
	{kind: "bundle", variants: 1, w: 8, h: 7, ax: 4, ay: 6, pad: 2, paint: paintBundle},
	{kind: "scroll", variants: 1, w: 8, h: 5, ax: 4, ay: 4, pad: 2, paint: paintLooseScroll},
	{kind: "scrolls", variants: ScrollPiles, w: 16, h: 13, ax: 8, ay: 12, pad: 3, paint: paintScrollPile},
	{kind: "stall", variants: StallLooks, w: stallW, h: stallH, ax: stallAX, ay: stallAY, paint: paintStall},
	{kind: "crate", variants: 1, w: 8, h: 8, ax: 4, ay: 7, pad: 2, paint: paintCrate},
	{kind: "chimney", variants: 1, w: 7, h: 12, ax: 3, ay: 11, pad: 2, paint: paintChimneyStack},
}

func addAmbience(b *atlasBuilder) {
	for f := 0; f < WispFrames; f++ {
		frame := f
		s := paddedSize(size{14, 24, 7, 23}, wispPad)
		b.add(effectName("wisp", f), s.w, s.h, s.ax, s.ay, framed(wispPad, func(c *canvas) { paintWisp(c, frame) }))
	}
	for _, j := range ambientProps {
		for v := 0; v < j.variants; v++ {
			job, variant := j, v
			s := paddedSize(size{j.w, j.h, j.ax, j.ay}, j.pad)
			b.add(AmbientName(j.kind, v), s.w, s.h, s.ax, s.ay, framed(j.pad, func(c *canvas) { job.paint(c, variant) }))
		}
	}
	for load := 0; load < CartLoads; load++ {
		for f := 0; f < CartFrames; f++ {
			addCart(b, load, f)
		}
	}
}

func addCart(b *atlasBuilder, load, frame int) {
	s := paddedSize(size{cartW, cartH, cartW / 2, cartH - 3}, cartPad)
	src := newFramedCanvas(s.w, s.h, cartPad, cartPad)
	paintCart(src, load, frame)
	b.add(CartName(load, "se", frame), s.w, s.h, s.ax, s.ay, func(c *canvas) { c.paste(src, 0, 0, false) })
	b.add(CartName(load, "sw", frame), s.w, s.h, s.ax, s.ay, func(c *canvas) { c.paste(src, 0, 0, true) })
}

var smokeR = ramp{rgb(150, 148, 150), rgb(192, 190, 188), rgb(222, 220, 214), rgb(244, 242, 236)}

func paintWisp(c *canvas, f int) {
	for k := 3; k >= 0; k-- {
		phase := math.Mod(float64(f)/WispFrames+float64(k)/4, 1)
		y := 21 - phase*18
		x := 7 + math.Sin(phase*5+float64(k))*1.8*phase
		r := 1.3 + phase*2.1
		a := uint8(245 - 150*phase*phase)
		c.disc(x, y, r, withAlpha(smokeR[1], a))
		c.disc(x-r*0.3, y-r*0.3, r*0.7, withAlpha(smokeR[2], a))
		c.disc(x-r*0.45, y-r*0.45, r*0.35, withAlpha(smokeR[3], a))
	}
	c.outline(withAlpha(smokeR[0], 140))
}

func paintLampPost(c *canvas, lit int) {
	c.ellipse(3.5, 21, 2.5, 0.9, shadow)
	c.rect(2, 19, 3, 2, stoneR[1])
	c.rect(3, 6, 1, 14, woodR[1])
	c.dot(3, 7, woodR[2])
	c.rect(1, 2, 5, 1, inkSft)
	c.rect(2, 1, 3, 1, inkSft)
	glass := glassR[1]
	if lit == 1 {
		glass = lampLit
	}
	c.rect(2, 3, 3, 3, glass)
	if lit == 1 {
		c.dot(3, 4, white)
	}
	c.outline(ink)
}

func paintWard(c *canvas, _ int) {
	c.ellipse(11, 24, 9, 1.6, shadow)
	for _, x := range []int{3, 18} {
		c.rect(x, 3, 1, 21, woodR[1])
		c.dot(x, 3, woodR[3])
	}
	c.rect(2, 3, 18, 1, woodR[2])
	for x := 2; x <= 19; x++ {
		col := redR[2]
		if (x/3)%2 == 0 {
			col = white
		}
		c.rect(x, 16, 1, 2, col)
	}
	c.line(7, 4, 7, 5, inkSft)
	c.line(14, 4, 14, 5, inkSft)
	c.rect(5, 6, 12, 8, redR[1])
	c.rect(6, 7, 10, 6, redR[2])
	c.rect(7, 9, 8, 2, white)
	c.outline(ink)
}

func paintIdleBubble(c *canvas, _ int) {
	c.ellipse(7.5, 5, 6.5, 4.6, white)
	c.dot(4, 10, white)
	c.dot(5, 10, white)
	c.dot(3, 12, white)
	for _, x := range []int{4, 7, 10} {
		c.rect(x, 5, 2, 1, inkSft)
	}
	c.outline(ink)
}

func paintBundle(c *canvas, _ int) {
	c.ellipse(4, 6, 3.5, 1, shadow)
	c.disc(4, 3.8, 2.6, rgb(150, 124, 88))
	c.disc(3.4, 3.2, 1.4, rgb(186, 160, 118))
	c.rect(3, 1, 2, 1, redR[2])
	c.dot(5, 0, redR[1])
	c.outline(ink)
}

func paintLooseScroll(c *canvas, _ int) {
	c.rect(1, 1, 6, 3, plasterR[3])
	c.rect(1, 3, 6, 1, plasterR[1])
	c.rect(0, 1, 1, 3, plasterR[1])
	c.rect(7, 1, 1, 3, plasterR[1])
	c.dot(4, 2, redR[2])
	c.outline(ink)
}

func paintScrollPile(c *canvas, level int) {
	c.ellipse(8, 12, 7, 1.4, shadow)
	spots := [][2]int{{3, 10}, {9, 10}, {6, 8}, {12, 8}, {4, 6}, {9, 6}, {7, 4}, {11, 3}}
	n := []int{1, 3, 5, 8}[min(max(level, 0), ScrollPiles-1)]
	for i, s := range spots[:n] {
		tone := plasterR[3]
		if i%2 == 1 {
			tone = plasterR[2]
		}
		c.rect(s[0], s[1], 5, 2, tone)
		c.dot(s[0], s[1], plasterR[1])
		c.dot(s[0]+4, s[1]+1, plasterR[1])
		c.dot(s[0]+2, s[1], redR[2])
	}
	c.outline(ink)
}

var stallCloths = []ramp{
	{rgb(120, 58, 52), rgb(166, 86, 72), rgb(200, 118, 96), rgb(226, 162, 136)},
	{rgb(52, 74, 104), rgb(76, 104, 138), rgb(104, 136, 166), rgb(150, 176, 196)},
	{rgb(130, 102, 40), rgb(176, 142, 62), rgb(208, 176, 90), rgb(232, 210, 138)},
}

func paintStall(c *canvas, look int) {
	cloth := stallCloths[look%len(stallCloths)]
	s := isoFrame(c, 1, stallAX, stallAY)
	s.box(0.15, 0.2, 0.85, 0.7, 0, 7, faces{top: flat(woodR[3]), left: flat(woodR[1]), right: flat(woodR[2])})
	for _, g := range [][2]float64{{0.3, 0.35}, {0.5, 0.4}, {0.65, 0.5}, {0.4, 0.55}} {
		p := s.p(g[0], g[1], 8)
		c.disc(p.x, p.y, 1.2, []color.NRGBA{redR[2], goldR[2], greenR[2], orangeR[2]}[int(g[0]*10)%4])
	}
	for _, g := range [][2]float64{{0.15, 0.7}, {0.85, 0.7}, {0.85, 0.2}} {
		a, b := s.p(g[0], g[1], 7), s.p(g[0], g[1], 20)
		c.line(int(a.x), int(a.y), int(b.x), int(b.y), woodR[0])
	}
	awning(s, cloth)
	c.outline(ink)
}

func awning(s iso, cloth ramp) {
	s.quad(s.p(0.05, 0.1, 21), s.p(0.95, 0.1, 21), s.p(0.95, 0.85, 16), s.p(0.05, 0.85, 16), func(x, y int) color.NRGBA {
		if (x/3)%2 == 0 {
			return cloth[2]
		}
		return plasterR[3]
	})
	for i := 0; i < 6; i++ {
		p := s.p(0.05+float64(i)*0.18, 0.85, 16)
		s.c.dot(int(p.x), int(p.y)+1, cloth[1])
	}
}

func paintCrate(c *canvas, _ int) {
	c.ellipse(4, 7, 3.5, 0.8, shadow)
	crate(c, 1, 1)
	c.outline(ink)
}

func paintChimneyStack(c *canvas, _ int) {
	c.rect(1, 2, 5, 9, stoneR[1])
	c.rect(1, 2, 2, 9, stoneR[2])
	c.rect(0, 1, 7, 2, stoneR[0])
	c.rect(1, 1, 5, 1, inkSft)
	for y := 4; y < 11; y += 3 {
		c.rect(1, y, 5, 1, stoneR[0])
	}
	c.outline(ink)
}

func paintCart(c *canvas, load, frame int) {
	c.ellipse(15, 21, 13, 1.8, shadow)
	donkey(c, frame)
	c.line(16, 14, 21, 13, woodR[0])
	c.line(16, 16, 21, 15, woodR[0])
	cartBody(c, load)
	wheel(c, 9, 16, frame)
	c.outline(ink)
}

func donkey(c *canvas, frame int) {
	coat := ramp{rgb(84, 74, 70), rgb(116, 104, 96), rgb(150, 138, 124), rgb(186, 176, 160)}
	swing := []int{0, 1, 0, -1}[frame]
	for i, x := range []int{20, 22, 25, 27} {
		shift := swing
		if i%2 == 1 {
			shift = -swing
		}
		c.line(x, 15, x+shift, 20, coat[0])
	}
	c.ellipse(23.5, 13, 4.5, 2.6, coat[1])
	c.ellipse(22.5, 12.2, 3, 1.4, coat[2])
	c.rect(26, 8, 2, 4, coat[1])
	c.rect(27, 9, 3, 2, coat[2])
	c.dot(29, 10, ink)
	c.dot(26, 6, coat[0])
	c.dot(27, 7, coat[0])
	c.dot(28, 9, ink)
	c.line(19, 12, 18, 15, coat[0])
}

func cartBody(c *canvas, load int) {
	c.rect(2, 11, 15, 5, woodR[1])
	c.rect(2, 11, 15, 1, woodR[3])
	c.rect(2, 13, 15, 1, woodR[0])
	switch load {
	case 1:
		c.rect(6, 7, 7, 4, woodR[2])
		c.rect(6, 7, 7, 1, goldR[2])
		c.dot(9, 9, goldR[3])
	case 2:
		c.rect(4, 6, 7, 5, woodR[2])
		c.rect(4, 6, 7, 1, goldR[2])
		c.disc(13, 9, 2.4, strawR[2])
		c.disc(8, 5, 1.6, goldR[2])
		c.dot(7, 4, goldR[3])
		c.dot(14, 7, goldR[3])
	default:
		c.disc(9, 10, 1.6, strawR[1])
	}
}

func wheel(c *canvas, cx, cy, frame int) {
	c.ellipse(float64(cx), float64(cy), 3.2, 3.6, woodR[0])
	c.ellipse(float64(cx), float64(cy), 2.2, 2.6, woodR[2])
	ang := float64(frame) * math.Pi / 4
	for k := 0; k < 2; k++ {
		a := ang + float64(k)*math.Pi/2
		dx, dy := math.Cos(a)*2.4, math.Sin(a)*2.8
		c.line(cx-int(math.Round(dx)), cy-int(math.Round(dy)), cx+int(math.Round(dx)), cy+int(math.Round(dy)), woodR[0])
	}
	c.dot(cx, cy, goldR[1])
}
