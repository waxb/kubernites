package art

import (
	"fmt"
	"image/color"
	"math"
)

const (
	SeaFrames = 8
	SeaWidth  = 128
	SeaHeight = 64
)

var deepR = ramp{rgb(26, 66, 100), rgb(36, 88, 122), rgb(58, 122, 150), rgb(150, 208, 214)}

func SeaName(frame int) string {
	return fmt.Sprintf("sea/%d", ((frame%SeaFrames)+SeaFrames)%SeaFrames)
}

func seaWave(x, y, frame int) float64 {
	ph := 2 * math.Pi * float64(frame) / SeaFrames
	fx, fy := float64(x)/SeaWidth, float64(y)/SeaHeight
	a := math.Sin(2*math.Pi*(2*fx+4*fy) + ph)
	b := math.Sin(2*math.Pi*(3*fx-2*fy) - ph + 1.3)
	c := math.Sin(2*math.Pi*(fx+6*fy) + 2*ph)
	return 0.5*a + 0.3*b + 0.2*c
}

func seaColor(x, y, frame int) color.NRGBA {
	w := seaWave(x, y, frame)
	n := pick(x, y, 900+frame)
	switch {
	case w > 0.8 && n > 0.2:
		if n > 0.93 {
			return deepR[3]
		}
		return deepR[2]
	case w > 0.62 && n > 0.72:
		return deepR[2]
	case w < -0.78 && n > 0.5:
		return deepR[0]
	}
	return deepR[1]
}

func addSea(b *atlasBuilder) {
	for f := 0; f < SeaFrames; f++ {
		frame := f
		b.add(SeaName(f), SeaWidth, SeaHeight, 0, 0, func(c *canvas) {
			for y := 0; y < SeaHeight; y++ {
				for x := 0; x < SeaWidth; x++ {
					c.dot(x, y, seaColor(x, y, frame))
				}
			}
		})
	}
}
