package art

import (
	"image/color"
	"math"
)

var effectPainters = map[string]func(c *canvas, f int){
	"fire": paintFire, "smoke": paintSmoke, "dust": paintDust, "void": paintVoid, "sparkle": paintSparkle,
	"coin": paintCoin, "raven": paintRaven, "explosion": paintExplosion, "heal": paintHeal,
	"boat": paintBoat, "banner": paintBanner, "reweave": paintReweave, "horn": paintHornFx,
	"fish": paintFish, "ley": paintLeyGlow, "charter": paintCharter, "wake": paintWake, "request": paintRequest,
	"alarm": paintAlarm, "spoil": paintSpoil,
}

var effectPads = map[string]int{
	"fire": 2, "smoke": 3, "dust": 3, "void": 2, "sparkle": 2, "explosion": 5, "heal": 2, "raven": 2, "boat": 2,
	"banner": 2, "reweave": 2, "horn": 3, "ley": 2, "charter": 2, "request": 1, "alarm": 1,
}

func paddedSize(s size, pad int) size {
	return size{s.w + 2*pad, s.h + 2*pad, s.ax + pad, s.ay + pad}
}

func framed(pad int, paint func(c *canvas)) func(c *canvas) {
	return func(c *canvas) {
		c.ox, c.oy = pad, pad
		paint(c)
	}
}

func buildEffects() Atlas {
	b := &atlasBuilder{}
	for _, k := range append(append([]string{}, EffectKinds...), ExtraEffects...) {
		pad := effectPads[k]
		s := paddedSize(effectSize(k), pad)
		for f := 0; f < EffectFrameCount(k); f++ {
			kind, frame := k, f
			b.add(effectName(k, f), s.w, s.h, s.ax, s.ay, framed(pad, func(c *canvas) { effectPainters[kind](c, frame) }))
		}
	}
	addShips(b)
	addAmbience(b)
	return b.build()
}

func flame(c *canvas, x, base, h, w, sway float64) {
	for i, col := range fireR {
		k := 1 - float64(i)*0.23
		hw, hh := w*k, h*k
		c.shadePoly([]pt{{x - hw, base}, {x - hw*0.7, base - hh*0.45}, {x + sway*k, base - hh}, {x + hw*0.7, base - hh*0.45}, {x + hw, base}}, flat(col))
	}
}

func puff(c *canvas, x, y, r float64, a uint8, tone ramp) {
	c.disc(x, y, r, withAlpha(tone[1], a))
	c.disc(x-r*0.3, y-r*0.3, r*0.65, withAlpha(tone[2], a))
	c.disc(x-r*0.45, y-r*0.45, r*0.3, withAlpha(tone[3], a))
}

func twinkle(c *canvas, x, y, s int, col color.NRGBA) {
	if s <= 0 {
		return
	}
	for d := 1; d <= s; d++ {
		c.dot(x+d, y, col)
		c.dot(x-d, y, col)
		c.dot(x, y+d, col)
		c.dot(x, y-d, col)
	}
	if s >= 2 {
		for _, o := range [][2]int{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
			c.dot(x+o[0], y+o[1], withAlpha(col, 150))
		}
	}
	c.dot(x, y, white)
}

func paintFire(c *canvas, f int) {
	ph := float64(f) * math.Pi / 3
	c.ellipse(8, 14.5, 5.5, 1.5, withAlpha(fireR[0], 110))
	for i, x := range []float64{4.5, 11.5, 8} {
		h := []float64{7, 8, 12}[i] + math.Sin(ph+float64(i)*2.1)*1.8
		flame(c, x, 15, h, []float64{2.6, 2.6, 3.6}[i], math.Sin(ph+float64(i))*1.5)
	}
	c.outline(darken(fireR[0], 0.5))
	for i := 0; i < 3; i++ {
		c.dot(3+i*5+(f+i)%2, 11-(f*2+i*5)%11, fireR[3-i%2])
	}
}

func paintSmoke(c *canvas, f int) {
	for ph := 5; ph >= 0; ph-- {
		if (ph+f)%2 != 0 {
			continue
		}
		id := float64((ph - f + 6) % 6)
		p := float64(ph)
		x := 8 + math.Sin(p*0.9+id*2.1)*2
		puff(c, x, 13-p*1.9, 2.2+p*0.5, uint8(240-p*24), greyR)
	}
	c.outline(withAlpha(greyR[0], 190))
}

func paintDust(c *canvas, f int) {
	t := float64(f+1) / 6
	a := uint8(235 - t*165)
	for _, d := range []float64{-1, 1} {
		puff(c, 8+d*(1+t*5), 13-t*2, 1.6+t*1.8, a, dirtR)
	}
	if f < 4 {
		puff(c, 8, 12-t*4, 2+t*1.5, a, sandR)
	}
	for i := 0; i < 3; i++ {
		c.dot(3+i*5, 14-f-(i%2)*2, withAlpha(dirtR[1], a))
	}
}

func paintVoid(c *canvas, f int) {
	ph := float64(f) * math.Pi / 3
	c.ellipse(8, 15, 5, 1.2, withAlpha(voidR[0], 140))
	c.ellipse(8, 9, 6, 5.5, voidR[0])
	c.ellipse(8, 9, 4.6, 4.2, voidR[1])
	for k := 0; k < 3; k++ {
		for s := 0; s < 6; s++ {
			a := ph + float64(k)*2*math.Pi/3 + float64(s)*0.5
			r := 0.8 + float64(s)*0.8
			col := voidR[2]
			if s > 3 {
				col = voidR[3]
			}
			c.dot(int(8+math.Cos(a)*r), int(9+math.Sin(a)*r*0.9), col)
		}
	}
	c.disc(8, 9, 1.2, ink)
	c.outline(ink)
	for i, x := range []int{1, 14, 3} {
		c.dot(x, 13-((f+i*2)%6)*2, withAlpha(voidR[3], 220))
	}
}

func paintSparkle(c *canvas, f int) {
	for i, p := range [][2]int{{4, 6}, {11, 4}, {9, 11}, {3, 13}} {
		s := []int{0, 1, 2, 3, 2, 1}[(f+[]int{0, 2, 4, 3}[i])%6]
		col := goldR[3]
		if i%2 == 1 {
			col = rgb(200, 230, 255)
		}
		twinkle(c, p[0], p[1], s, col)
	}
}

func paintCoin(c *canvas, f int) {
	w := []float64{3.8, 3, 1.8, 0.8, 1.8, 3}[f]
	y := 8 - []float64{0, 1, 2, 2.5, 2, 1}[f]
	c.ellipse(8, 14.5, 3, 1, shadow)
	if w < 1 {
		c.rect(7, int(y)-3, 2, 8, goldR[1])
		c.dot(7, int(y)-2, goldR[3])
	} else {
		c.ellipse(8, y+0.5, w, 3.8, goldR[0])
		c.ellipse(8-0.3, y+0.2, math.Max(w-0.8, 0.8), 3.2, goldR[2])
		c.dot(7, int(y)-1, goldR[3])
		if w > 2.5 {
			c.rect(8, int(y), 1, 2, goldR[1])
		}
	}
	c.outline(ink)
	if f == 0 || f == 3 {
		twinkle(c, 12, 3, 1, white)
	}
}

func paintRaven(c *canvas, f int) {
	off := []float64{-5, -3, 0, 3, 1, -3}[f]
	body, far, sheen := rgb(40, 36, 52), rgb(26, 22, 34), rgb(74, 70, 100)
	c.shadePoly([]pt{{6.5, 5}, {9.5, 5}, {6.5, 5 + off*0.9}, {3.5, 5 + off*0.7}}, flat(far))
	c.ellipse(8, 6.5, 3.6, 1.9, body)
	c.shadePoly([]pt{{5, 6}, {1.5, 5.5}, {2, 8}}, flat(body))
	c.disc(11.5, 5.5, 1.8, body)
	c.dot(13, 5, goldR[2])
	c.dot(14, 5, goldR[1])
	c.shadePoly([]pt{{7, 6.5}, {10.5, 6.5}, {8.5, 6.5 + off}, {5, 6.5 + off*0.8}}, flat(sheen))
	c.outline(ink)
	c.dot(12, 5, white)
}
