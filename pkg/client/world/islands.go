package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
)

const charterFPS = 3

func (r *Renderer) charterItems(cam render.Camera, sc Scene) []Drawable {
	zoom := cam.Scale()
	out := []Drawable{}
	for i, is := range sc.Site.Islands {
		if is.Owned {
			continue
		}
		at := Vec{float64(is.Center.X), float64(is.Center.Y)}
		sx, sy := ScreenPointOf(cam, sc.Site, at)
		if r.Land != nil && !inView(r.Land.view, sx, sy, zoom) {
			continue
		}
		name := gfx.EffectName("charter", Frame(sc.Time+float64(i)*.7, charterFPS))
		out = append(out, Drawable{Depth: render.DepthKey(at.X, at.Y), Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{})
		}})
	}
	return out
}
