package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	pileAway  = 2.2
	pileSide  = .8
	haulPhase = 1.3
)

var crateName = gfx.AmbientName("crate", 0)

func PileSpot(s view.Structure) Vec {
	door := Door(s)
	return Vec{door.X + pileAway, door.Y + pileSide}
}

func (r *Renderer) haulerItems(out []Drawable, cam render.Camera, sc Scene) []Drawable {
	zoom := cam.Scale()
	for _, s := range sc.Site.Structures {
		if s.State != view.StateConstruction {
			continue
		}
		pile, door := PileSpot(s), Door(s)
		px, py := ScreenPointOf(cam, sc.Site, pile)
		if !onScreen(px, py, zoom) {
			continue
		}
		out = append(out, Drawable{Depth: render.DepthKey(pile.X, pile.Y), Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, crateName, px, py, zoom, render.SpriteOptions{})
			r.Lib.Draw(dst, crateName, px+4*zoom, py-3*zoom, zoom, render.SpriteOptions{})
		}})
		if !drawFiguresWithoutPods {
			continue
		}
		pose := DockShuttle(sc.Time, haulPhase*float64(len(s.ID)), pile, door)
		sx, sy := ScreenPointOf(cam, sc.Site, pose.Pos)
		name := r.lifeOf().folkName("dockhand", len(s.ID), pose)
		out = append(out, Drawable{person: "dockhand", Depth: render.DepthKey(pose.Pos.X, pose.Pos.Y) + .05, stand: stand(pose.Pos, sx, sy), Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{Tint: folkTint})
		}})
	}
	return out
}
