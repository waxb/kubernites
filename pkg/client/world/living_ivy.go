package world

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	ivyClimb     = .6
	ivyMaxRise   = 34
	ivyDoorGap   = 4
	ivyDoorRise  = 2
	mossRise     = 2
	mossStep     = 2
	ivyFlowerAt  = 5
	ivyEdgeSoft  = 3
	ivyTendrilIn = 5
)

type ivyKey struct {
	name  string
	level int
}

var (
	ivyLeaf   = [4]color.NRGBA{{40, 76, 50, 255}, {58, 110, 62, 255}, {86, 146, 74, 255}, {136, 182, 96, 255}}
	mossLeaf  = [3]color.NRGBA{{84, 120, 70, 255}, {108, 140, 80, 255}, {70, 98, 64, 255}}
	ivyBloom  = color.NRGBA{244, 232, 150, 255}
	ivyBloom2 = color.NRGBA{238, 168, 186, 255}
)

func (r *Renderer) ivyFor(s view.Structure) int {
	if s.Kind != view.StructKeep || s.State != view.StateReady {
		return 0
	}
	return r.livingOf().ivyOf(s.ID)
}

func (l *living) ivyOf(id string) int {
	if f, ok := l.keepFolkOf(id); ok {
		return IvyLevel(f.quiet)
	}
	return 0
}

func (r *Renderer) drawIvy(dst *ebiten.Image, name string, level int, sx, sy, zoom float64, alpha float32) {
	if level <= 0 {
		return
	}
	l := r.livingOf()
	key := ivyKey{name: name, level: level}
	img, ok := l.ivy[key]
	if !ok {
		img = r.paintIvy(name, level)
		l.ivy[key] = img
	}
	if img == nil {
		return
	}
	_, reg, _ := r.Lib.Lookup(name)
	b := img.Bounds()
	render.DrawAt(dst, img, render.Region{W: b.Dx(), H: b.Dy(), AnchorX: reg.AnchorX, AnchorY: reg.AnchorY}, sx, sy, zoom, render.SpriteOptions{Alpha: alpha})
}

func (r *Renderer) paintIvy(name string, level int) *ebiten.Image {
	mask, ok := r.Lib.Mask(name)
	if !ok {
		return nil
	}
	_, reg, _ := r.Lib.Lookup(name)
	return ebiten.NewImageFromImage(IvyOverlay(mask, reg.AnchorX, level))
}

func IvyOverlay(mask gfx.Mask, anchorX, level int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, mask.W, mask.H))
	if level <= 0 {
		return img
	}
	for x := 0; x < mask.W; x++ {
		bottom, top := columnSpan(mask, x)
		if bottom < 0 {
			continue
		}
		rise := ivyRise(x, bottom-top, level, iabs(x-anchorX) <= ivyDoorGap)
		for y := bottom; y > bottom-rise && y >= top; y-- {
			if mask.At(x, y) {
				if col, ok := ivyPixel(x, y, bottom-y, rise, level); ok {
					img.SetNRGBA(x, y, col)
				}
			}
		}
	}
	return img
}

func iabs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func columnSpan(mask gfx.Mask, x int) (bottom, top int) {
	bottom, top = -1, -1
	for y := 0; y < mask.H; y++ {
		if mask.At(x, y) {
			if top < 0 {
				top = y
			}
			bottom = y
		}
	}
	return bottom, top
}

func ivyRise(x, height, level int, door bool) int {
	h := mix32(uint32(x)*2654435761 + 17)
	if level <= 2 {
		rise := mossRise + (level-1)*mossStep
		if h%3 == 0 {
			return 0
		}
		return min(rise+int(h>>8%2), height)
	}
	if door {
		return ivyDoorRise
	}
	grown := float64(level-2) / float64(IvySteps-2)
	rise := int(float64(min(int(float64(height)*ivyClimb), ivyMaxRise)) * grown * (.45 + .55*float64(h>>4%100)/100))
	if h%ivyTendrilIn == 0 {
		rise = rise * 8 / 5
	}
	return min(max(rise, 2), height)
}

func ivyPixel(x, y, up, rise, level int) (color.NRGBA, bool) {
	h := mix32(uint32(x)*374761393 ^ uint32(y)*668265263)
	if level <= 2 {
		return mossLeaf[h%3], h>>6%5 != 0
	}
	if rise-up <= ivyEdgeSoft && int(h>>9%uint32(ivyEdgeSoft+1)) < ivyEdgeSoft-(rise-up) {
		return color.NRGBA{}, false
	}
	switch n := h % 20; {
	case level >= ivyFlowerAt && n == 0 && h>>11%3 == 0:
		return ivyBloom, true
	case level >= ivyFlowerAt && n == 1 && h>>11%3 == 0:
		return ivyBloom2, true
	case n < 11:
		return ivyLeaf[1], true
	case n < 15:
		return ivyLeaf[2], true
	case n < 18:
		return ivyLeaf[0], true
	}
	return ivyLeaf[3], true
}
