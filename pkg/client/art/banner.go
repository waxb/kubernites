package art

import "math"

func bannerWave(x, f int) (int, float64) {
	a := float64(x)*2*math.Pi/80 + float64(f)*2*math.Pi/BannerFrames
	return int(math.Round(1 + math.Sin(a))), math.Cos(a)
}

func paintBanner(c *canvas, f int) {
	cloth := newCanvas(c.w, c.h)
	bannerCloth(cloth)
	for x := 0; x < c.w; x++ {
		dy, slope := bannerWave(x, f)
		for y := 0; y < c.h; y++ {
			col := cloth.at(x, y)
			if col.A == 0 {
				continue
			}
			switch {
			case slope > 0.7:
				col = mix(col, white, 0.07)
			case slope < -0.7:
				col = mix(col, ink, 0.12)
			case slope < -0.2:
				col = mix(col, ink, 0.05)
			}
			c.dot(x, y+dy, col)
		}
	}
	for _, x := range []int{24, 64, 120, 176, 215} {
		dy, _ := bannerWave(x, f)
		c.rect(x, 3, 1, 1+dy, tkGold)
	}
	c.outline(ink)
	c.rect(4, 1, 232, 2, woodR[2])
	c.rect(4, 1, 232, 1, woodR[3])
	c.disc(3, 2, 2, tkGoldBright)
	c.disc(236, 2, 2, tkGold)
}

func bannerCloth(c *canvas) {
	c.shadePoly([]pt{{17, 4}, {2, 4}, {9, 18}, {2, 32}, {17, 32}}, flat(blueR[0]))
	c.shadePoly([]pt{{222, 4}, {237, 4}, {230, 18}, {237, 32}, {222, 32}}, flat(blueR[0]))
	c.rect(16, 4, 208, 29, blueR[0])
	c.rect(35, 4, 170, 29, redR[1])
	c.rect(16, 4, 208, 2, tkGold)
	c.rect(16, 4, 208, 1, tkGoldBright)
	c.rect(16, 31, 208, 2, tkGold)
	c.rect(16, 32, 208, 1, goldR[0])
	for _, x := range []int{35, 204} {
		c.rect(x, 4, 1, 29, tkGold)
	}
	c.rect(40, 8, 160, 1, tkGold)
	c.rect(40, 28, 160, 1, tkGold)
	for _, x := range []float64{25.5, 213.5} {
		c.shadePoly([]pt{{x, 11}, {x + 6, 18}, {x, 25}, {x - 6, 18}}, flat(tkGold))
		c.shadePoly([]pt{{x, 14}, {x + 3, 18}, {x, 22}, {x - 3, 18}}, flat(tkRedDark))
		c.dot(int(x), 17, tkGoldBright)
	}
	for x := 20; x <= 220; x += 8 {
		c.rect(x, 33, 1, 2, tkGold)
	}
}

func paintReweave(c *canvas, f int) {
	t := float64(f) / float64(EffectFrames-1)
	if f >= 3 {
		c.disc(12, 12, 2+float64(f-3)*1.6, withAlpha(tkGoldBright, 90))
	}
	for k := 0; k < 3; k++ {
		a0 := float64(k)*2*math.Pi/3 + t*math.Pi*1.5
		outer := 11 * (1 - t*0.8)
		for i := 0; i <= 10; i++ {
			r := outer * (1 - float64(i)*0.06)
			a := a0 + float64(i)*0.32
			col := tkGold
			if i > 7 {
				col = tkGoldBright
			}
			c.dot(int(12+math.Cos(a)*r), int(12+math.Sin(a)*r*0.9), col)
		}
	}
	if f >= 2 {
		c.disc(12, 12, 0.8+float64(f-2)*0.5, tkGoldBright)
	}
	c.outline(darken(tkGold, 0.6))
	if f >= 4 {
		twinkle(c, 12, 12, f-2, white)
	}
}
