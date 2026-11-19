package world

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	landAlpha = .8
	landFade  = .45
	cropRise  = 4
	landTileW = render.TileWidth
	landTileH = render.TileHeight
)

var (
	landEarth  = color.NRGBA{176, 142, 96, 255}
	landDust   = color.NRGBA{196, 162, 114, 255}
	landPebble = color.NRGBA{140, 108, 72, 255}
	fieldSoil  = color.NRGBA{116, 80, 46, 255}
	fieldRow   = color.NRGBA{92, 62, 36, 255}
	berryLeaf  = color.NRGBA{56, 112, 52, 255}
	berryDark  = color.NRGBA{38, 82, 40, 255}
	berryFruit = color.NRGBA{196, 44, 58, 255}
	wheatStalk = color.NRGBA{214, 168, 64, 255}
	wheatHead  = color.NRGBA{240, 206, 104, 255}
	wheatShade = color.NRGBA{168, 124, 44, 255}
)

type keepLand struct {
	sig     uint64
	homes   int
	patches []KeepPatch
	tile    *ebiten.Image
	crops   map[string]*ebiten.Image
}

func (r *Renderer) keepLandOf(sc Scene) *keepLand {
	if r.fields == nil {
		r.fields = &keepLand{tile: ebiten.NewImageFromImage(PaintLandTile()), crops: map[string]*ebiten.Image{
			"berries": ebiten.NewImageFromImage(PaintField("berries")), "wheat": ebiten.NewImageFromImage(PaintField("wheat")),
		}}
	}
	sig := StructureSig(sc.Site)
	if r.fields.patches == nil || r.fields.sig != sig || r.fields.homes != len(r.Homes) {
		r.fields.patches, r.fields.sig, r.fields.homes = KeepPatches(sc.Site, r.Homes), sig, len(r.Homes)
	}
	return r.fields
}

func (r *Renderer) drawKeepLand(dst *ebiten.Image, cam render.Camera, sc Scene) {
	land := r.keepLandOf(sc)
	x0, y0, x1, y1 := visibleTiles(dst, cam, sc.Site)
	zoom := cam.Scale()
	tileReg := render.Region{W: landTileW, H: landTileH, AnchorX: landTileW / 2, AnchorY: landTileH / 2}
	for _, p := range land.patches {
		for _, t := range p.Land {
			if t.At.X < x0 || t.At.X > x1 || t.At.Y < y0 || t.At.Y > y1 {
				continue
			}
			sx, sy := cam.GridToScreenElevated(float64(t.At.X), float64(t.At.Y), float64(sc.Site.ElevationAt(t.At.X, t.At.Y)))
			alpha := float32(landAlpha * min(max((1-t.Share)/landFade, 0), 1))
			render.DrawAt(dst, land.tile, tileReg, sx, sy, zoom, render.SpriteOptions{Alpha: alpha})
		}
	}
}

func diamondShare(x, y, w, h int) float64 {
	cx, cy := float64(w-1)/2, float64(h-1)/2
	return math.Abs(float64(x)-cx)/(cx+.5) + math.Abs(float64(y)-cy)/(cy+.5)
}

func PaintLandTile() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, landTileW, landTileH))
	for y := range landTileH {
		for x := range landTileW {
			if diamondShare(x, y, landTileW, landTileH) > 1 {
				continue
			}
			clr := landEarth
			switch (x*7 + y*13 + x*y) % 11 {
			case 0:
				clr = landDust
			case 5:
				clr = landPebble
			}
			img.SetNRGBA(x, y, clr)
		}
	}
	return img
}

func PaintField(crop string) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, landTileW, landTileH+cropRise))
	for y := range landTileH {
		for x := range landTileW {
			d := diamondShare(x, y, landTileW, landTileH)
			if d > .92 {
				continue
			}
			clr := fieldSoil
			if (x+2*y)%4 == 0 {
				clr = fieldRow
			}
			img.SetNRGBA(x, y+cropRise, clr)
		}
	}
	for y := 2; y < landTileH-2; y += 3 {
		for x := 4; x < landTileW-4; x += 5 {
			if diamondShare(x, y, landTileW, landTileH) > .7 {
				continue
			}
			plantCrop(img, crop, x+(y%2)*2, y+cropRise)
		}
	}
	return img
}

func plantCrop(img *image.NRGBA, crop string, x, y int) {
	if crop == "berries" {
		img.SetNRGBA(x, y-2, berryLeaf)
		img.SetNRGBA(x-1, y-1, berryLeaf)
		img.SetNRGBA(x+1, y-1, berryDark)
		img.SetNRGBA(x, y-1, berryFruit)
		img.SetNRGBA(x, y, berryDark)
		return
	}
	img.SetNRGBA(x, y-3, wheatHead)
	img.SetNRGBA(x, y-2, wheatStalk)
	img.SetNRGBA(x+1, y-2, wheatHead)
	img.SetNRGBA(x, y-1, wheatStalk)
	img.SetNRGBA(x, y, wheatShade)
}
