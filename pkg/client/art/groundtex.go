package art

import (
	"image/color"
	"math"
)

var (
	dryR    = ramp{rgb(74, 108, 54), rgb(106, 142, 66), rgb(138, 168, 78), rgb(182, 196, 104)}
	mossR   = ramp{rgb(58, 90, 48), rgb(78, 112, 56), rgb(104, 136, 64), rgb(138, 160, 80)}
	leyGlow = rgb(196, 170, 255)
)

var bayer4 = [16]float64{0, 8, 2, 10, 12, 4, 14, 6, 3, 11, 1, 9, 15, 7, 13, 5}

func dither(x, y int) float64 {
	return (bayer4[(y&3)*4+(x&3)] + 0.5) / 16
}

func pick(x, y, seed int) float64 {
	return float64(hash(x, y, seed)%1000) / 1000
}

func speckle(r ramp, x, y, seed int, light float64) color.NRGBA {
	n := pick(x, y, seed)
	switch {
	case n < 0.07:
		return r[0]
	case n > 0.97:
		return r[3]
	case n > 0.97-light:
		return r[2]
	}
	return r[1]
}

func groundTex(m Ground, x, y int, gx, gy float64) color.NRGBA {
	switch m {
	case GroundSand:
		return sandTex(x, y, gx, gy)
	case GroundMeadow:
		return meadowTex(x, y, gx, gy)
	case GroundForest:
		return forestTex(x, y, gx, gy)
	case GroundRock:
		return rockTex(x, y, gx, gy)
	case GroundPlaza:
		return plazaTex(gx, gy)
	}
	return grassTex(x, y, gx, gy)
}

func tuftAt(x, y, seed int) bool {
	return pick(x, y, seed) > 0.972
}

func tufted(r ramp, x, y, seed int) (color.NRGBA, bool) {
	switch {
	case tuftAt(x, y, seed):
		return r[0], true
	case tuftAt(x, y+1, seed):
		return r[3], true
	case tuftAt(x-1, y+1, seed) || tuftAt(x+1, y+1, seed):
		return r[2], true
	}
	return color.NRGBA{}, false
}

func calmBase(r ramp, x, y int, gx, gy float64, seed int) color.NRGBA {
	u, v := gx*0.8+gy*0.35, gy*0.8-gx*0.35
	p := 0.6*vnoise(u*0.2, v*0.2, seed) + 0.4*vnoise(u*0.47+7.3, v*0.47+3.1, seed+1) + (dither(x, y)-0.5)*0.12
	switch {
	case p > 0.63:
		return mix(r[1], r[2], 0.45)
	case p < 0.34:
		return mix(r[1], r[0], 0.28)
	}
	return r[1]
}

func grassTex(x, y int, gx, gy float64) color.NRGBA {
	if c, ok := tufted(grassR, x, y, 12); ok {
		return c
	}
	switch n := pick(x, y, 13); {
	case n < 0.035:
		return grassR[0]
	case n > 0.975:
		return grassR[2]
	}
	return calmBase(grassR, x, y, gx, gy, 11)
}

func meadowTex(x, y int, gx, gy float64) color.NRGBA {
	if n := pick(x, y, 22); n > 0.978 {
		blooms := []color.NRGBA{goldR[3], white, redR[3], rgb(176, 156, 230), goldR[2]}
		return blooms[int(vnoise(gx*0.8, gy*0.8, 23)*4.99)]
	}
	if c, ok := tufted(meadowR, x, y, 24); ok {
		return c
	}
	switch n := pick(x, y, 25); {
	case n < 0.03:
		return meadowR[0]
	case n > 0.96:
		return meadowR[2]
	}
	return calmBase(meadowR, x, y, gx, gy, 21)
}

func forestTex(x, y int, gx, gy float64) color.NRGBA {
	if c, ok := tufted(mossR, x, y, 34); ok {
		return c
	}
	switch n := pick(x, y, 32); {
	case n < 0.035:
		return dirtR[0]
	case n < 0.05:
		return soilR[1]
	case n > 0.996:
		return orangeR[1]
	case n > 0.97:
		return forestR[2]
	}
	return calmBase(forestR, x, y, gx, gy, 31)
}

func sandTex(x, y int, gx, gy float64) color.NRGBA {
	ripple := int(math.Floor(gx*3.1+gy*1.2+vnoise(gx*0.7, gy*0.7, 41)*2.2)) % 3
	if ripple == 0 && pick(x, y, 42) > 0.45 && (x+y)%2 == 0 {
		return sandR[1]
	}
	if pick(x, y, 43) > 0.993 {
		return rockR[2]
	}
	return speckle(sandR, x, y, 44, 0.24)
}

type pebble struct {
	cx, cy, rx, ry float64
	tone           int
}

func pebbleIn(x, y, cw, ch, seed int, fill float64) (pebble, float64, float64, bool) {
	row := int(math.Floor(float64(y) / float64(ch)))
	shift := (row & 1) * cw / 2
	col := int(math.Floor(float64(x-shift) / float64(cw)))
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			r, c := row+dr, col+dc
			h := hash(c, r, seed)
			if float64(h%100)/100 > fill {
				continue
			}
			p := pebble{
				cx: float64(c*cw+(r&1)*cw/2) + float64(cw)/2 + float64(int(h>>8%3)-1)*0.6,
				cy: float64(r*ch) + float64(ch)/2 + float64(int(h>>12%3)-1)*0.4,
				rx: float64(cw)/2 - 0.3 + float64(h>>16%2)*0.4, ry: float64(ch)/2 - 0.2,
				tone: int(h >> 20 % 3),
			}
			dx, dy := (float64(x)+0.5-p.cx)/p.rx, (float64(y)+0.5-p.cy)/p.ry
			if dx*dx+dy*dy <= 1 {
				return p, dx, dy, true
			}
		}
	}
	return pebble{}, 0, 0, false
}

func stoneShade(r ramp, p pebble, dx, dy float64) color.NRGBA {
	switch light := -(dx*0.7 + dy); {
	case light > 0.75:
		return r[3]
	case light < -0.7:
		return r[0]
	}
	return r[1+p.tone%2]
}

func rockTex(x, y int, gx, gy float64) color.NRGBA {
	if p, dx, dy, ok := pebbleIn(x, y, 7, 4, 51, 0.62); ok {
		return stoneShade(rockR, p, dx, dy)
	}
	if vnoise(gx*1.1, gy*1.1, 52) > 0.55 {
		return speckle(mossR, x, y, 53, 0.18)
	}
	return speckle(ramp{soilR[1], dirtR[1], dirtR[2], rockR[2]}, x, y, 55, 0.2)
}

func plazaTex(gx, gy float64) color.NRGBA {
	u, v := gx*2.6, gy*2.6
	iu, iv := int(math.Floor(u)), int(math.Floor(v))
	fu, fv := u-float64(iu), v-float64(iv)
	switch {
	case fu < 0.12 || fv < 0.14:
		return plazaR[0]
	case fu > 0.84 || fv > 0.86:
		return plazaR[1]
	case fu < 0.26 && fv < 0.3:
		return plazaR[3]
	}
	switch hash(iu, iv, 61) % 7 {
	case 0:
		return plazaR[3]
	case 1:
		return mix(plazaR[2], mossR[2], 0.35)
	}
	return plazaR[2]
}

func rank(m Ground) int {
	switch m {
	case GroundWater:
		return 0
	case GroundSand:
		return 1
	case GroundRock:
		return 3
	case GroundPlaza:
		return 4
	}
	return 2
}

func rimmed(b *groundBuf, m Ground, x, y int, col color.NRGBA) color.NRGBA {
	r := rank(m)
	above := b.matAt(x, y-1)
	if above != voidGround && rank(above) > r && rank(above) > 1 {
		return darken(col, 0.22)
	}
	for _, d := range [4][2]int{{0, 1}, {1, 0}, {-1, 0}, {0, -1}} {
		n := b.matAt(x+d[0], y+d[1])
		if n != voidGround && rank(n) < r && n != GroundWater {
			return darken(col, 0.16)
		}
	}
	return col
}
