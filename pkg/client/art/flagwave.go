package art

import (
	"image"
	"image/color"
	"math"
	"strings"
)

const (
	FlagClothW = 11
	FlagClothH = 8
	FlagFrames = 8
)

type FlagMark struct {
	X, Y        int
	Light, Dark color.NRGBA
}

func anchored(marks []FlagMark, ax, ay int) []FlagMark {
	if len(marks) == 0 {
		return nil
	}
	out := make([]FlagMark, len(marks))
	for i, m := range marks {
		out[i] = FlagMark{X: m.X - ax, Y: m.Y - ay, Light: m.Light, Dark: m.Dark}
	}
	return out
}

func clothShape() *canvas {
	c := newCanvas(FlagClothW, FlagClothH-2)
	c.shadePoly([]pt{{1, 0}, {9, 2}, {8, 5}, {1, 5}}, func(x, y int) color.NRGBA {
		if y%2 == 0 {
			return rgb(255, 255, 255)
		}
		return rgb(0, 0, 0)
	})
	return c
}

func clothLift(x, frame int) int {
	reach := math.Min(1, float64(max(x-1, 0))/3)
	return int(math.Round(reach * math.Sin(2*math.Pi*float64(frame)/FlagFrames-0.7*float64(x))))
}

func FlagClothFrame(light, dark color.NRGBA, frame int) *image.NRGBA {
	shape := clothShape().img
	out := image.NewNRGBA(image.Rect(0, 0, FlagClothW, FlagClothH))
	for y := 0; y < FlagClothH-2; y++ {
		for x := 0; x < FlagClothW; x++ {
			c := shape.NRGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			col := dark
			if c.R == 255 {
				col = light
			}
			out.SetNRGBA(x, y+1+clothLift(x, frame), col)
		}
	}
	return out
}

func flagsOf(name string, marks []FlagMark, ax, ay int) []FlagMark {
	if strings.Contains(name, "/dark") || strings.Contains(name, "/construction-") {
		return nil
	}
	return anchored(marks, ax, ay)
}
