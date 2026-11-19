package art

import (
	"image/color"
	"math"
)

const (
	WakeWidth     = 64
	WakeHeight    = 32
	RequestWidth  = 11
	RequestHeight = 12
)

func paintWake(c *canvas, f int) {
	cx, cy := float64(WakeWidth)/2, float64(WakeHeight)/2
	for k := 0; k < 2; k++ {
		ph := math.Mod(float64(f)/EffectFrames+float64(k)/2, 1)
		rx := 7 + ph*23
		a := uint8(190 * (1 - ph) * (1 - ph*0.3))
		wakeRing(c, cx, cy, rx, a, f+k*7)
	}
}

func wakeRing(c *canvas, cx, cy, rx float64, a uint8, seed int) {
	if a < 12 {
		return
	}
	n := int(rx * 7)
	for i := 0; i < n; i++ {
		th := 2 * math.Pi * float64(i) / float64(n)
		if (i/3+seed)%4 == 0 {
			continue
		}
		x, y := cx+math.Cos(th)*rx, cy+math.Sin(th)*rx/2
		col := withAlpha(waterR[3], a)
		if math.Sin(th) > 0.2 {
			col = withAlpha(white, a)
		}
		c.dot(int(math.Floor(x)), int(math.Floor(y)), col)
	}
}

func paintRequest(c *canvas, f int) {
	by := []int{2, 2, 1, 1, 1, 2}[f]
	glow := []int{0, 1, 2, 2, 1, 0}[f]
	c.rect(2, by, 7, 1, white)
	c.rect(1, by+1, 9, 6, white)
	c.rect(2, by+7, 7, 1, white)
	c.dot(5, by+8, white)
	c.outline(ink)
	requestLantern(c, by, glow)
}

func requestLantern(c *canvas, by, glow int) {
	halo := []color.NRGBA{rgb(250, 246, 226), rgb(255, 240, 196), rgb(255, 230, 160)}[glow]
	c.rect(3, by+1, 5, 6, halo)
	c.rect(2, by+2, 7, 4, halo)
	c.dot(5, by+1, woodR[0])
	c.rect(4, by+2, 3, 1, woodR[1])
	c.rect(4, by+3, 3, 2, lampLit)
	c.rect(4, by+5, 3, 1, woodR[1])
	if glow > 0 {
		c.dot(5, by+3, white)
	}
	if glow > 1 {
		c.dot(5, by+4, goldR[3])
	}
}
