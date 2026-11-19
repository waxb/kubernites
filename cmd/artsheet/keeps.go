package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/waxb/kubernites/pkg/client/art"
)

const (
	keepCellW  = 112
	keepCellH  = 196
	keepLabelH = 16
	keepSeed   = 0x5eed
)

var keepAges = []string{"dark", "feudal", "castle", "imperial"}

func keepSheet() *image.NRGBA {
	cols := len(art.KeepClasses) * art.AgeTiers
	img := image.NewNRGBA(image.Rect(0, 0, cols*keepCellW, len(art.KeepSizes)*keepCellH))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{58, 74, 64, 255}), image.Point{}, draw.Src)
	for row, size := range art.KeepSizes {
		for age := range art.AgeTiers {
			for c, class := range art.KeepClasses {
				k := art.KeepLook{Size: size, Class: class, Age: age, Seed: keepSeed + uint32(row*16+age*4+c)}
				sprite, _ := art.KeepSprite(k, "ready")
				x := (age*len(art.KeepClasses) + c) * keepCellW
				y := (row+1)*keepCellH - keepLabelH - sprite.Rect.Dy()
				blitScaled(img, sprite, x+(keepCellW-sprite.Rect.Dx())/2, y, 1)
				label(img, fmt.Sprintf("%s %s", size, class), x+4, (row+1)*keepCellH-4)
				label(img, keepAges[age], x+4, row*keepCellH+12)
			}
		}
	}
	return img
}
