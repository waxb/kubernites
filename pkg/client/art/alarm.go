package art

import (
	"image/color"
	"math"
)

const (
	alarmW, alarmH = 28, 30
	alarmCX        = 14.0
	alarmCY        = 13.0
)

func paintAlarm(c *canvas, f int) {
	for _, k := range []int{f, (f + 3) % EffectFrames} {
		r := 8 + float64(k)*1.1
		a := uint8(250 - k*38)
		alarmRing(c, r, withAlpha(rgb(236, 64, 52), a), withAlpha(fireR[2], a*2/3))
	}
	bob := []float64{0, -1, -1, 0, 0, 0}[f]
	seal := newCanvas(c.w, c.h)
	alarmSeal(seal, alarmCY+bob, f < 2)
	seal.outline(ink)
	c.paste(seal, 0, 0, false)
	if f < 2 {
		for _, d := range [][4]int{{2, 2, 4, 4}, {25, 2, 23, 4}, {0, 13, 2, 13}, {27, 13, 25, 13}} {
			c.line(d[0], d[1], d[2], d[3], goldR[3])
		}
	}
}

func alarmRing(c *canvas, r float64, col, glow color.NRGBA) {
	for y := 0; y < c.h; y++ {
		for x := 0; x < c.w; x++ {
			d := math.Hypot(float64(x)+0.5-alarmCX, (float64(y)+0.5-alarmCY)*1.1)
			switch {
			case math.Abs(d-r) < 0.7:
				c.dot(x, y, col)
			case math.Abs(d-r+1) < 0.5:
				c.dot(x, y, glow)
			}
		}
	}
}

func alarmSeal(c *canvas, cy float64, bright bool) {
	for k := 0; k < 12; k++ {
		a := float64(k) * math.Pi / 6
		c.disc(alarmCX+math.Cos(a)*6, cy+math.Sin(a)*6, 1.6, redR[1])
	}
	c.disc(alarmCX, cy, 6.2, redR[1])
	inner := redR[2]
	if bright {
		inner = mix(redR[2], fireR[3], 0.35)
	}
	c.disc(alarmCX, cy, 4.6, inner)
	c.dot(int(alarmCX)-3, int(cy)-3, redR[3])
	c.dot(int(alarmCX)-2, int(cy)-4, redR[3])
	bang(c, int(alarmCX), int(cy))
	c.shadePoly([]pt{{alarmCX - 3, cy + 6}, {alarmCX - 1, cy + 6}, {alarmCX - 4, cy + 14}, {alarmCX - 6, cy + 12}}, flat(redR[1]))
	c.shadePoly([]pt{{alarmCX + 1, cy + 6}, {alarmCX + 3, cy + 6}, {alarmCX + 6, cy + 12}, {alarmCX + 4, cy + 14}}, flat(redR[0]))
}

func bang(c *canvas, x, y int) {
	c.rect(x-1, y-4, 2, 5, redR[0])
	c.rect(x-1, y+2, 2, 2, redR[0])
	c.rect(x-1, y-4, 1, 5, tkGoldBright)
	c.dot(x-1, y+2, tkGoldBright)
}
