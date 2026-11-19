package main

import (
	"image"
	"image/color"
	"image/draw"
	"sort"
	"strings"

	"github.com/waxb/kubernites/pkg/client/art"
)

const stripGap = 2

func stripSheet(atlases map[string]art.Atlas, prefixes string, k int) *image.NRGBA {
	rows := [][]stripCell{}
	w, h := 1, 0
	for _, p := range strings.Split(prefixes, ",") {
		row := stripRow(atlases, p)
		rw, rh := 0, 0
		for _, cell := range row {
			rw += (cell.r.W + stripGap) * k
			rh = max(rh, cell.r.H*k)
		}
		rows = append(rows, row)
		w, h = max(w, rw+stripGap*k), h+rh+stripGap*k
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, max(h, 1)))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.NRGBA{96, 140, 72, 255}), image.Point{}, draw.Src)
	y := stripGap * k
	for _, row := range rows {
		x, rh := stripGap*k, 0
		for _, cell := range row {
			blitScaled(img, cell.a.Image.SubImage(image.Rect(cell.r.X, cell.r.Y, cell.r.X+cell.r.W, cell.r.Y+cell.r.H)), x, y, k)
			x += (cell.r.W + stripGap) * k
			rh = max(rh, cell.r.H*k)
		}
		y += rh + stripGap*k
	}
	return img
}

type stripCell struct {
	a art.Atlas
	r art.Region
}

func stripRow(atlases map[string]art.Atlas, prefix string) []stripCell {
	names := []string{}
	owner := map[string]art.Atlas{}
	for _, a := range atlases {
		for n := range a.Regions {
			if strings.HasPrefix(n, prefix) {
				names = append(names, n)
				owner[n] = a
			}
		}
	}
	sort.Strings(names)
	out := make([]stripCell, 0, len(names))
	for _, n := range names {
		out = append(out, stripCell{a: owner[n], r: owner[n].Regions[n]})
	}
	return out
}
