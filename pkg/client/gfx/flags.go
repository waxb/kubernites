package gfx

import (
	"image/color"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
)

type clothKey struct {
	light, dark color.NRGBA
	frame       int
}

var cloths sync.Map

func (l *Library) Flags(name string) []art.FlagMark {
	e, ok := l.entry(name)
	if !ok {
		return nil
	}
	return e.region.Flags
}

func clothImage(m art.FlagMark, frame int) *ebiten.Image {
	key := clothKey{light: m.Light, dark: m.Dark, frame: frame % art.FlagFrames}
	if img, ok := cloths.Load(key); ok {
		return img.(*ebiten.Image)
	}
	img := ebiten.NewImageFromImage(art.FlagClothFrame(m.Light, m.Dark, key.frame))
	actual, _ := cloths.LoadOrStore(key, img)
	return actual.(*ebiten.Image)
}

func DrawFlag(dst *ebiten.Image, m art.FlagMark, frame int, sx, sy, zoom float64, o render.SpriteOptions) {
	r := render.Region{W: art.FlagClothW, H: art.FlagClothH, AnchorX: -m.X, AnchorY: -m.Y}
	render.DrawAt(dst, clothImage(m, frame), r, sx, sy, zoom, o)
}
