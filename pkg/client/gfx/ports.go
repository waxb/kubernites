package gfx

import (
	"image"

	"github.com/waxb/kubernites/pkg/client/art"
)

func portPainter(name string) (func() (*image.NRGBA, art.Region), bool) {
	_, port := art.ParsePortName(name)
	_, ferry := art.ParseFerryName(name)
	if !port && !ferry {
		return nil, false
	}
	return func() (*image.NRGBA, art.Region) {
		img, r, _ := art.PortSpriteByName(name)
		return img, r
	}, true
}
