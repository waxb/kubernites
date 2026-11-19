package art

import (
	"fmt"
	"image/color"
	"math"
)

const (
	MarketGrowths = 6
	growthSeed    = 5903
)

func MarketGrowthName(v int) string { return fmt.Sprintf("market/growth/%d", v) }

func addMarketGrowth(b *atlasBuilder) {
	for v := 0; v < MarketGrowths; v++ {
		vv := v
		b.add(MarketGrowthName(v), forumPaveW, forumPaveH, forumPaveAX, forumPaveAY, func(c *canvas) { paintTile(c, growthTex(vv)) })
	}
}

func growthTex(v int) tileTex {
	seed := growthSeed + v*61
	flowers := v%3 != 0
	return func(gx, gy float64, x, y int) (color.NRGBA, bool) {
		seam := math.Min(fmod((gx+0.5)*2.6+float64(v)*0.37, 1), fmod((gy+0.5)*2.6+float64(v)*0.21, 1))
		n := vnoise(gx*4+float64(v), gy*4, seed)
		switch {
		case n < 0.42:
			return color.NRGBA{}, false
		case flowers && pick(x, y, seed+3) > 0.94:
			return wildflowers[int(hash(x, y, seed)%uint32(len(wildflowers)))], true
		case seam < 0.18 && pick(x, y, seed+5) < 0.8:
			return grassR[1+int(hash(x, y, seed+7)%3)], true
		case n > 0.7 && pick(x, y, seed+9) < 0.35:
			return mossR[2], true
		}
		return color.NRGBA{}, false
	}
}
