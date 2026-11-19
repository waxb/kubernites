package render

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
)

var outlineOffsets = [4][2]float64{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}

func SolidColor(c color.RGBA) colorm.ColorM {
	var cm colorm.ColorM
	cm.Scale(0, 0, 0, float64(c.A)/255)
	cm.Translate(float64(c.R)/255, float64(c.G)/255, float64(c.B)/255, 0)
	return cm
}

func OutlineAt(dst, sheet *ebiten.Image, r Region, sx, sy, camZoom float64, c color.RGBA) {
	OutlineFlipAt(dst, sheet, r, sx, sy, camZoom, c, false)
}

func OutlineFlipAt(dst, sheet *ebiten.Image, r Region, sx, sy, camZoom float64, c color.RGBA, flip bool) {
	if sheet == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	sub := sheet.SubImage(r.Rect()).(*ebiten.Image)
	cm := SolidColor(c)
	width := math.Max(1, math.Floor(camZoom))
	left := SpriteLeft(r, sx, camZoom, flip)
	for _, o := range outlineOffsets {
		op := &colorm.DrawImageOptions{Filter: ebiten.FilterNearest}
		if flip {
			op.GeoM.Scale(-camZoom, camZoom)
			op.GeoM.Translate(float64(r.W)*camZoom, 0)
		} else {
			op.GeoM.Scale(camZoom, camZoom)
		}
		op.GeoM.Translate(left+o[0]*width, math.Round(sy-float64(r.AnchorY)*camZoom)+o[1]*width)
		colorm.DrawImage(dst, sub, cm, op)
	}
}

func SpriteLeft(r Region, sx, zoom float64, flip bool) float64 {
	if flip {
		return math.Round(sx - float64(r.W-r.AnchorX)*zoom)
	}
	return math.Round(sx - float64(r.AnchorX)*zoom)
}
