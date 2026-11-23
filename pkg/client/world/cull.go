package world

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const reachPad = 4

func (r *Renderer) clipTo(b image.Rectangle) func() {
	prev, was := r.clip, r.clipping
	r.clip, r.clipping = b, true
	return func() { r.clip, r.clipping = prev, was }
}

func (r *Renderer) reaches(box ui.Rect) bool {
	if !r.clipping {
		return true
	}
	return box.X < r.clip.Max.X && r.clip.Min.X < box.X+box.W && box.Y < r.clip.Max.Y && r.clip.Min.Y < box.Y+box.H
}

func spanRect(x0, y0, x1, y1 float64) ui.Rect {
	l, t := int(math.Floor(x0)), int(math.Floor(y0))
	return ui.Rect{X: l, Y: t, W: int(math.Ceil(x1)) - l, H: int(math.Ceil(y1)) - t}
}

func (r *Renderer) structureReach(cam render.Camera, site view.Site, p Placed) ui.Rect {
	zoom := cam.Scale()
	out := p.Rect
	for _, m := range r.Lib.Flags(r.ShownName(p.names)) {
		x, y := p.sx+float64(m.X)*zoom, p.sy+float64(m.Y)*zoom
		out = union(out, spanRect(x, y, x+art.FlagClothW*zoom, y+art.FlagClothH*zoom))
	}
	lo, hi := Footprint(p.S)
	hw, hh := render.TileWidth/2*zoom, render.TileHeight/2*zoom
	for _, c := range [4]Point{lo, {hi.X, lo.Y}, {lo.X, hi.Y}, hi} {
		x, y := cam.GridToScreen(float64(c.X), float64(c.Y))
		out = union(out, spanRect(x-hw, y-hh, x+hw, y+hh))
	}
	ax, ay := ScreenPointOf(cam, site, Anchor(p.S))
	w, h := float64(p.S.W*12)*zoom, float64(10+6*p.S.W)*zoom
	out = union(out, spanRect(ax-w/2, ay-h-8*zoom, ax+w/2, ay-8*zoom))
	return out.Inset(-int(reachPad * max(zoom, 1)))
}

func (r *Renderer) structureWarm(sc Scene, it Placed) func(*ebiten.Image) {
	name := r.ShownName(it.names)
	if !r.Lib.Has(name) {
		return nil
	}
	ivy, flags := r.ivyFor(it.S), r.Lib.Flags(name)
	cold := ivy > 0 && !r.ivyPainted(name, ivy)
	for i, m := range flags {
		cold = cold || !gfx.HasFlagCloth(m, FlagFrame(sc.Time, it.S, i))
	}
	if !cold {
		return nil
	}
	now, s := sc.Time, it.S
	return func(*ebiten.Image) {
		if ivy > 0 {
			r.ivyImage(name, ivy)
		}
		for i, m := range flags {
			gfx.MakeFlagCloth(m, FlagFrame(now, s, i))
		}
	}
}

func (r *Renderer) structureDraw(cam render.Camera, sc Scene, it Placed, alpha float32) func(*ebiten.Image) {
	zoom := cam.Scale()
	selected := it.S.ID == sc.Selected
	ivy := r.ivyFor(it.S)
	opts := render.SpriteOptions{Alpha: alpha}
	if tint, ok := sc.heraldTint(it.S.Object.Namespace); ok && it.S.Kind != view.StructKeep {
		opts.Tint = tint
	}
	site, now := sc.Site, sc.Time
	return func(dst *ebiten.Image) {
		if selected {
			r.outlineSprite(dst, it.names, it.sx, it.sy, zoom)
		}
		name, ok := r.Lib.DrawFirst(dst, it.names, it.sx, it.sy, zoom, opts)
		if !ok {
			fallbackBlock(dst, cam, site, it.S)
			return
		}
		r.drawIvy(dst, name, ivy, it.sx, it.sy, zoom, alpha)
		r.drawFlags(dst, name, it.S, it.sx, it.sy, zoom, now, alpha)
	}
}
