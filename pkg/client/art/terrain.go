package art

import (
	"image/color"
	"math"
)

func diamondCoords(x, y int) (float64, float64) {
	return (float64(x) + 0.5 - 16) / 16, (float64(y) + 0.5 - 8) / 8
}

func plazaShader(seed int) shader {
	return func(x, y int) color.NRGBA {
		a, b := diamondCoords(x, y)
		gx, gy := (a+b+1)/2*3, (b-a+1)/2*3
		fx, fy := gx-math.Floor(gx), gy-math.Floor(gy)
		if fx < 0.14 || fy < 0.16 {
			return plazaR[0]
		}
		stone := int(math.Floor(gx))*7 + int(math.Floor(gy))*3 + seed
		if fx > 0.8 || fy > 0.84 {
			return plazaR[1]
		}
		if stone%3 == 0 {
			return plazaR[3]
		}
		return plazaR[2]
	}
}

var soilR = ramp{rgb(70, 46, 32), rgb(98, 66, 42), rgb(126, 88, 54), rgb(158, 116, 72)}

func buildTerrain() Atlas {
	b := &atlasBuilder{}
	addDecor(b)
	addSea(b)
	addBlots(b)
	addGroundTints(b)
	return b.build()
}
