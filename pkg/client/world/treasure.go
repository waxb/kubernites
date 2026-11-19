package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	chestLift    = 10.0
	chestGleams  = 3
	chestGleamDX = 7.0
	chestGleamHz = 10.0
)

type ChestMark struct {
	Kind string
	Open bool
	Age  float64
}

func (r *Renderer) chestSprites(cam render.Camera, sc Scene) []incidentSprite {
	out := []incidentSprite{}
	zoom := cam.Scale()
	for _, c := range sc.Chests {
		at, ok := YardFocus(sc.Site, c.Kind)
		if !ok {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, at)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		sy -= chestLift * zoom
		out = append(out, incidentSprite{name: art.TreasureChestName(c.Kind, c.Open), sx: sx, sy: sy})
		if !c.Open || sc.Reduce {
			continue
		}
		for i := range chestGleams {
			dx := float64(i-1) * chestGleamDX * zoom
			out = append(out, incidentSprite{name: gfx.EffectName("sparkle", int(c.Age*chestGleamHz)+i*2), sx: sx + dx, sy: sy - chestLift*zoom})
		}
	}
	return out
}

func (r *Renderer) drawChests(dst *ebiten.Image, cam render.Camera, sc Scene) {
	if r.Lib == nil {
		return
	}
	zoom := cam.Scale()
	for _, s := range r.chestSprites(cam, sc) {
		r.Lib.Draw(dst, s.name, s.sx, s.sy, zoom, render.SpriteOptions{})
	}
}
