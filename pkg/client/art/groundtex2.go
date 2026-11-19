package art

import (
	"image/color"
	"math"
	"sort"
)

func roadTex(level uint8, margin, side float64, under color.NRGBA, x, y int, gx, gy float64) color.NRGBA {
	switch level {
	case 1:
		return troddenTex(margin, under, x, y)
	case 2:
		return dirtTex(margin, side, x, y)
	}
	return cobbleTex(margin, x, y, gx, gy)
}

func troddenTex(margin float64, under color.NRGBA, x, y int) color.NRGBA {
	if margin < 0.05 {
		return mix(under, dirtR[1], 0.3)
	}
	return []color.NRGBA{dirtR[1], dirtR[1], dirtR[2]}[hash(x, y, 72)%3]
}

func dirtTex(margin, side float64, x, y int) color.NRGBA {
	if margin < 0.04 {
		if pick(x, y, 73) > 0.5 {
			return grassR[1]
		}
		return dirtR[0]
	}
	for _, rut := range []float64{-0.13, 0.13} {
		if math.Abs(side-rut) < 0.03 && pick(x, y, 78) > 0.25 {
			return dirtR[1]
		}
	}
	if pick(x, y, 74) > 0.965 {
		return rockR[2]
	}
	return speckle(dirtR, x, y, 75, 0.12)
}

func cobbleTex(margin float64, x, y int, gx, gy float64) color.NRGBA {
	if margin < 0.05 {
		if pick(x, y, 77) > 0.5 {
			return stoneR[0]
		}
		return grassR[0]
	}
	if p, dx, dy, ok := pebbleIn(x, y, 5, 3, 76, 0.97); ok {
		return stoneShade(stoneR, p, dx, dy)
	}
	return dirtR[0]
}

type offset struct {
	dx, dy int
	d      float64
}

var coastRing = func() []offset {
	out := []offset{}
	for dy := -5; dy <= 5; dy++ {
		for dx := -10; dx <= 10; dx++ {
			d := math.Hypot(float64(dx)/2, float64(dy))
			if d > 0 && d <= 5 {
				out = append(out, offset{dx, dy, d})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].d != out[j].d {
			return out[i].d < out[j].d
		}
		if out[i].dy != out[j].dy {
			return out[i].dy < out[j].dy
		}
		return out[i].dx < out[j].dx
	})
	return out
}()

func shoreDistance(b *groundBuf, x, y int) float64 {
	for _, o := range coastRing {
		if m := b.matAt(x+o.dx, y+o.dy); m != GroundWater && m != voidGround {
			return o.d
		}
	}
	return 99
}

func waterTint(b *groundBuf, x, y, i int) color.NRGBA {
	d := float64(b.depth[i])
	if d > 2.2 {
		return color.NRGBA{}
	}
	shore := shoreDistance(b, x, y)
	n := pick(x, y, 81)
	switch {
	case shore <= 1:
		if n > 0.18 {
			return withAlpha(white, 235)
		}
		return withAlpha(shallowR[3], 220)
	case shore <= 2 && n > 0.45:
		return withAlpha(white, 170)
	case shore > 3.5 && shore <= 4.3 && n > 0.4:
		return withAlpha(shallowR[3], 110)
	}
	shelf := min(max((2.2-d)/1.6, 0), 1)
	if shore <= 3 {
		return withAlpha(mix(shallowR[2], sandR[2], 0.35), uint8(120+60*shelf))
	}
	return withAlpha(shallowR[2], uint8(150*shelf))
}

func faceColor(b *groundBuf, x, y int, r ray) color.NRGBA {
	k := r.level*ElevationStep - r.lift
	k = (k-1)%ElevationStep + 1
	lit := faceLit(b, x, y, r.level)
	top := b.mat[r.i]
	switch {
	case k == 1:
		return lipColor(top, lit)
	case k == ElevationStep:
		if lit > 0 {
			return soilR[1]
		}
		return soilR[0]
	}
	col := soilR[1]
	if lit > 0 {
		col = soilR[2]
	}
	if lit < 0 && k > 2 {
		col = darken(col, 0.12)
	}
	switch n := pick(x, r.lift+r.level*7, 91); {
	case n > 0.86:
		return soilR[min(lit+2, 3)]
	case n < 0.06:
		return rockR[1]
	}
	if b.wild[r.i] {
		return wildTint(col)
	}
	return col
}

func lipColor(top Ground, lit int) color.NRGBA {
	r := grassR
	switch top {
	case GroundSand:
		r = sandR
	case GroundRock:
		r = rockR
	case GroundForest:
		r = forestR
	case GroundMeadow:
		r = meadowR
	}
	if lit > 0 {
		return r[3]
	}
	return r[1]
}

func faceLit(b *groundBuf, x, y, level int) int {
	left, right := 99, 99
	for k := 1; k <= 8; k++ {
		if left == 99 && b.lvlAt(x-k, y) < level {
			left = k
		}
		if right == 99 && b.lvlAt(x+k, y) < level {
			right = k
		}
	}
	switch {
	case left < right:
		return 1
	case right < left:
		return -1
	}
	return 0
}

func leyVein(col color.NRGBA, power float64, x, y int, gx, gy float64) color.NRGBA {
	if power < 1.5 {
		return col
	}
	strength := min((power-1.5)/1.5, 1)
	v := vnoise(gx*2.2, gy*2.2, 500)
	if math.Abs(v-0.5) < 0.012+0.02*strength {
		return mix(col, leyGlow, 0.35+0.4*strength)
	}
	if power >= 2.5 && pick(x, y, 501) > 0.994 {
		return mix(white, leyGlow, 0.5)
	}
	return col
}
