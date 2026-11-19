package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/waxb/kubernites/pkg/client/art"
)

func variantRows() [][]string {
	rows := [][]string{}
	for _, role := range art.VariedRoles {
		for tier := 1; tier <= art.WalkerTiers(role); tier++ {
			row := []string{}
			for v := 0; v < art.WalkerLooks; v++ {
				name := fmt.Sprintf("walker/%s/t%dv%d/walk/se/1", role, tier, v)
				if tier == 1 && v == 0 {
					name = "walker/" + role + "/walk/se/1"
				}
				row = append(row, name)
			}
			rows = append(rows, row)
		}
	}
	return rows
}

func variantSheet(a art.Atlas, k int) *image.NRGBA {
	rows := variantRows()
	cw, ch := (art.WalkerWidth+12)*k, (art.WalkerHeight+6)*k
	img := image.NewNRGBA(image.Rect(0, 0, cw*art.WalkerLooks+4*k, ch*len(rows)+2*k))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{96, 140, 72, 255}), image.Point{}, draw.Src)
	for y, row := range rows {
		for x, name := range row {
			r, ok := a.Regions[name]
			if !ok {
				continue
			}
			px, py := 2*k+x*cw+cw/2-r.AnchorX*k, k+(y+1)*ch-2*k-r.AnchorY*k
			blitScaled(img, a.Image.SubImage(image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)), px, py, k)
		}
	}
	return img
}
