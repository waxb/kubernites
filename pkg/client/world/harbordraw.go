package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	shipMargin = 90.0
	wakeBehind = .3
	flatLift   = .5
	detailZoom = .75
	pieceKinds = 3
	shipAbove  = .01
	cheerFor   = 5.0
	cheerLift  = 26.0
)

func flatDepth(v Vec) float64 {
	return render.DepthKey(v.X-flatLift, v.Y-flatLift) - flatLift
}

func pierKind(out Vec) string {
	if out.X != 0 {
		return "pier-x"
	}
	return "pier-y"
}

func shipOnScreen(sx, sy, zoom float64) bool {
	m := shipMargin * max(zoom, 1)
	return sx >= -m && sx <= ViewWidth+m && sy >= -m && sy <= ViewHeight+2*m
}

func (r *Renderer) HarborItems(cam render.Camera, sc Scene) []Drawable {
	h := r.HarborOf()
	cargo := r.quayItems(nil, cam, sc)
	out := []Drawable{}
	for i, p := range h.Plans {
		out = r.postItems(r.pieceItems(out, cam, sc, p, cargo), cam, sc, p)
		if i < len(h.Fleets) {
			out = r.shipItems(out, cam, sc, p, h.Fleets[i])
			out = r.TurnedItems(out, cam, sc, i)
		}
	}
	return append(append(out, cargo...), r.goldenItems(cam, sc)...)
}

func (r *Renderer) pieceItems(out []Drawable, cam render.Camera, sc Scene, p HavenPlan, cargo []Drawable) []Drawable {
	zoom := cam.Scale()
	a := r.portArtOf(sc, p)
	kind := pierKind(p.Out)
	for i, v := range p.Pier {
		if sx, sy := ScreenPointOf(cam, sc.Site, v); onScreen(sx, sy, zoom) {
			piece := HavenProp{Kind: kind, Variant: i % pieceKinds}
			names := r.HarborOf().pieceNames(piece, a.level, a.age)
			out = append(out, Drawable{Depth: flatDepth(v), Draw: func(dst *ebiten.Image) {
				r.drawPortPiece(dst, names, piece, sx, sy, zoom)
			}})
		}
	}
	return r.propItems(out, cam, sc, p, a, cargo)
}

func (r *Renderer) propItems(out []Drawable, cam render.Camera, sc Scene, p HavenPlan, a portArt, cargo []Drawable) []Drawable {
	zoom := cam.Scale()
	fresh := Fresh(p, sc.RealmTime) && !sc.Reduce
	for _, prop := range p.Props {
		sx, sy := ScreenPointOf(cam, sc.Site, prop.At)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		if prop.Deck {
			sy -= art.HavenDeck * zoom
		}
		out = append(out, r.portPropItem(p, prop, a, sx, sy, zoom, cargo))
		if fresh && prop.Tier == p.Tier {
			out = append(out, r.buildDust(sx, sy, zoom, sc.RealmTime-p.Since, prop.At))
		}
		if a.busy && harborBuildings[prop.Kind] && !sc.Reduce {
			out = append(out, r.ageBurst(cam, sc, prop.At, structBurst, p.ID)...)
		}
	}
	return out
}

func (r *Renderer) portPropItem(p HavenPlan, prop HavenProp, a portArt, sx, sy, zoom float64, cargo []Drawable) Drawable {
	names := r.HarborOf().pieceNames(prop, a.level, a.age)
	outline := a.selected && harborBuildings[prop.Kind]
	depth := render.DepthKey(prop.At.X, prop.At.Y)
	see := r.hidesCargo(r.propShape(prop, names, sx, sy, zoom), depth, cargo)
	alpha := float32(1)
	if see {
		alpha = seeThroughAlpha
	}
	return Drawable{Depth: depth, Hit: r.propHitNamed(p, prop, names, sx, sy, zoom), SeeThrough: see, Draw: func(dst *ebiten.Image) {
		if outline {
			r.outlineNames(dst, names, sx, sy, zoom, prop.Flip)
		}
		r.drawPortPieceAt(dst, names, prop, sx, sy, zoom, alpha)
	}}
}

func (r *Renderer) buildDust(sx, sy, zoom, age float64, at Vec) Drawable {
	name := gfx.EffectName("dust", int(age*6))
	return Drawable{Depth: render.DepthKey(at.X, at.Y) + .2, Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{Alpha: float32(1 - age/FreshSeconds)})
	}}
}

func pieceVariant(kind string, variant int) int {
	n := max(art.HavenVariants(kind), 1)
	return ((variant % n) + n) % n
}

func (r *Renderer) drawPiece(dst *ebiten.Image, kind string, variant int, sx, sy, zoom float64) {
	r.drawProp(dst, HavenProp{Kind: kind, Variant: variant}, sx, sy, zoom)
}

func (r *Renderer) drawProp(dst *ebiten.Image, p HavenProp, sx, sy, zoom float64) {
	if !r.Lib.Draw(dst, gfx.HavenName(p.Kind, pieceVariant(p.Kind, p.Variant)), sx, sy, zoom, render.SpriteOptions{Flip: p.Flip}) {
		drawPieceFallback(dst, p.Kind, p.Variant, sx, sy, zoom)
	}
}

func (r *Renderer) shipItems(out []Drawable, cam render.Camera, sc Scene, p HavenPlan, f *fleet) []Drawable {
	zoom := cam.Scale()
	now := r.HarborOf().Now
	for _, s := range f.Ships {
		if len(p.Berths) == 0 {
			continue
		}
		b := p.Berths[s.Berth%len(p.Berths)]
		pose, ok := ShipAt(s, b, p.Sea, now)
		if !ok {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, pose.Pos)
		if !shipOnScreen(sx, sy, zoom) {
			continue
		}
		out = append(out, r.shipDrawable(cam, sc, s, pose, b))
		if pose.Moving && !sc.Reduce {
			out = append(out, r.wakeDrawable(cam, sc, s, pose))
		}
		if s.Herald && !sc.Reduce && now >= s.Dock && now < s.Dock+cheerFor {
			out = append(out, r.CheerDrawable(sx, sy, zoom, now-s.Dock, pose.Pos))
		}
	}
	return out
}

func (r *Renderer) shipDrawable(cam render.Camera, sc Scene, s Ship, pose ShipPose, b Berth) Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, pose.Pos)
	frame := 0
	if !sc.Reduce {
		frame = Frame(sc.Time+float64(s.Serial)*.37, 3)
	}
	name := gfx.ShipName(s.Class, s.Variant, pose.Dir, frame)
	opts := render.SpriteOptions{Alpha: float32(pose.Alpha)}
	plank := pose.Unloading
	px, py := ScreenPointOf(cam, sc.Site, b.Plank)
	py -= art.HavenDeck * zoom
	return Drawable{Depth: render.DepthKey(pose.Pos.X, pose.Pos.Y), Draw: func(dst *ebiten.Image) {
		if plank {
			drawGangplank(dst, sx, sy, px, py, zoom)
		}
		if !r.Lib.Draw(dst, name, sx, sy, zoom, opts) {
			drawShipFallback(dst, s.Class, s.Variant, pose, sx, sy, zoom)
		}
	}}
}

func (r *Renderer) CheerDrawable(sx, sy, zoom, age float64, at Vec) Drawable {
	return Drawable{Depth: render.DepthKey(at.X, at.Y) + .3, Draw: func(dst *ebiten.Image) {
		for i := 0; i < 3; i++ {
			k := float64(i)
			name := gfx.EffectName("sparkle", int(age*10)+i*2)
			r.Lib.Draw(dst, name, sx+(k-1)*12*zoom, sy-(cheerLift+6*k)*zoom, zoom, render.SpriteOptions{})
		}
	}}
}

func (r *Renderer) wakeDrawable(cam render.Camera, sc Scene, s Ship, pose ShipPose) Drawable {
	back := HeadingVec(pose.Dir)
	at := offset(pose.Pos, back, -wakeBehind)
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	zoom := cam.Scale()
	name := gfx.EffectName("wake", int(sc.Time*8)+s.Serial)
	alpha := float32(pose.Alpha * .9)
	return Drawable{Depth: render.DepthKey(pose.Pos.X, pose.Pos.Y) - shipAbove, Draw: func(dst *ebiten.Image) {
		if !r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{Alpha: alpha}) {
			drawWakeFallback(dst, sx, sy, zoom, sc.Time, alpha)
		}
	}}
}

func HeadingVec(dir string) Vec {
	switch dir {
	case "nw":
		return Vec{-1, 0}
	case "sw":
		return Vec{0, 1}
	case "ne":
		return Vec{0, -1}
	}
	return Vec{1, 0}
}

var shipLengths = map[string]float64{"cog": .9, "caravel": 1.1, "galleon": 1.4, "carrack": 1.7}

func shipLength(class string) float64 {
	if l, ok := shipLengths[class]; ok {
		return l
	}
	return 1
}

func labelBox(text string, sx, sy float64) ui.Rect {
	var font ui.Font
	w, _ := font.Measure(text, 1)
	x := min(max(int(sx)-w/2-4, 4), ViewWidth-w-12)
	return ui.Rect{X: x, Y: int(sy) - 14, W: w + 8, H: 13}
}
