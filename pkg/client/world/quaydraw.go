package world

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	HitPile    = "pile"
	FlyTime    = .45
	flyArc     = 14.0
	levelStep  = .001
	boxAbove   = .02
	heapBack   = .2
	heapBehind = 1.0
	fxAbove    = .3
	hitPad     = 3
	pileGrow   = 3.0
	ghostAlpha = .5
	PileIDHead = "pile:"
	urgentHz   = 1.6
)

type boxKey struct {
	size, variant, level     int
	kind                     string
	sealed, crowned, spoiled bool
}

func PileID(kind int) string { return PileIDHead + KindName(kind) }

func PileKind(id string) (int, bool) {
	name, ok := strings.CutPrefix(id, PileIDHead)
	if !ok {
		return 0, false
	}
	for i, k := range FreightKinds {
		if k == name {
			return i, true
		}
	}
	return 0, false
}

func (h *Harbor) layers(look gfx.BoxLook) []string {
	k := boxKey{size: look.Size, variant: look.Variant, level: gfx.FreshLevel(look.Fresh), kind: look.Kind,
		sealed: look.Sealed, crowned: look.Priority, spoiled: look.Spoiled}
	if names, ok := h.names[k]; ok {
		return names
	}
	if h.names == nil {
		h.names = map[boxKey][]string{}
	}
	names := gfx.BoxLayers(look)
	h.names[k] = names
	return names
}

func boxPoint(p HavenPlan, b QuayBox) Vec {
	return StackSpot(p, b.Kind, b.Column)
}

func (r *Renderer) quayItems(out []Drawable, cam render.Camera, sc Scene) []Drawable {
	h := r.HarborOf()
	for _, y := range h.Yards {
		if y.Haven < len(h.Plans) {
			out = r.YardItems(out, cam, sc, y)
		}
	}
	for _, g := range h.Book.Ghosts {
		out = r.GhostItems(out, cam, sc, g)
	}
	return out
}

type pileMark struct {
	Rect   ui.Rect `json:"-"`
	Shapes []Shape `json:"-"`
}

func (r *Renderer) YardItems(out []Drawable, cam render.Camera, sc Scene, y Yard) []Drawable {
	h := r.HarborOf()
	p := h.Plans[y.Haven]
	for _, b := range y.Boxes {
		if d, mark, ok := r.BoxItem(cam, sc, p, b); ok {
			out = append(out, pileHit(d, mark, y))
		}
	}
	if y.Overflow > 0 {
		d, mark := r.heapItem(cam, sc, p, y)
		out = append(out, pileHit(d, mark, y))
	}
	return out
}

func pileHit(d Drawable, mark pileMark, y Yard) Drawable {
	next := d
	next.Hit = &Hit{Kind: HitPile, ID: PileID(y.Kind), Rect: mark.Rect.Inset(-hitPad), Serial: y.Haven, Shapes: mark.Shapes, Grow: pileGrow}
	return next
}

func layerShapes(names []string, sx, sy, zoom float64) []Shape {
	out := make([]Shape, 0, len(names))
	for _, n := range names {
		out = append(out, Shape{Name: n, X: sx, Y: sy, Zoom: zoom})
	}
	return out
}

func (r *Renderer) boxMark(names []string, sx, sy, zoom float64) pileMark {
	return pileMark{Rect: SpriteRect(r.Lib, names[0], sx, sy, zoom, 8, 8), Shapes: layerShapes(names, sx, sy, zoom)}
}

func (r *Renderer) BoxItem(cam render.Camera, sc Scene, p HavenPlan, b QuayBox) (Drawable, pileMark, bool) {
	h := r.HarborOf()
	zoom := cam.Scale()
	pos := boxPoint(p, b)
	sx, sy := ScreenPointOf(cam, sc.Site, pos)
	if !onScreen(sx, sy, zoom) {
		return Drawable{}, pileMark{}, false
	}
	sy -= (h.Lift(pos) + float64(b.Lift)) * zoom
	depth := render.DepthKey(pos.X, pos.Y) + boxAbove + float64(b.Level)*levelStep
	if at, landing := h.RevealAt(b); landing && h.Now < at {
		return r.flyingBox(cam, sc, b, at, sx, sy, depth)
	}
	names := h.layers(b.Look)
	tint := UrgentTint(b.Look, sc.Time, sc.Reduce)
	return Drawable{Depth: depth, Draw: func(dst *ebiten.Image) { r.drawBox(dst, names, sx, sy, zoom, b.Look, tint) }}, r.boxMark(names, sx, sy, zoom), true
}

func UrgentTint(look gfx.BoxLook, t float64, reduce bool) [3]float32 {
	if look.Spoiled || gfx.FreshLevel(look.Fresh) < gfx.FreshLevels-1 {
		return [3]float32{}
	}
	p := .5
	if !reduce {
		p = .5 + .5*math.Sin(t*2*math.Pi*urgentHz)
	}
	return [3]float32{float32(1 + .3*p), float32(1 - .25*p), float32(1 - .3*p)}
}

func (r *Renderer) drawBox(dst *ebiten.Image, names []string, sx, sy, zoom float64, look gfx.BoxLook, tint [3]float32) {
	drawn := false
	opts := render.SpriteOptions{Tint: tint}
	for i, n := range names {
		if i > 0 {
			opts = render.SpriteOptions{}
		}
		drawn = r.Lib.Draw(dst, n, sx, sy, zoom, opts) || drawn
	}
	if !drawn {
		c := boxFallback[look.Kind]
		block(dst, sx, sy, zoom, 6+2*float64(look.Size), 4+2*float64(look.Size), c)
	}
}

func (r *Renderer) flyingBox(cam render.Camera, sc Scene, b QuayBox, at, sx, sy, depth float64) (Drawable, pileMark, bool) {
	h := r.HarborOf()
	left := at - h.Now
	if left > FlyTime {
		return Drawable{}, pileMark{}, false
	}
	_, s, ok := h.shipOfLanding(b.At)
	p := h.Plan(b.Haven)
	if !ok || len(p.Berths) == 0 {
		return Drawable{}, pileMark{}, false
	}
	zoom := cam.Scale()
	plank := p.Berths[s.Berth%len(p.Berths)].Plank
	fx, fy := ScreenPointOf(cam, sc.Site, plank)
	fy -= h.Lift(plank) * zoom
	t := 1 - left/FlyTime
	x, y := fx+(sx-fx)*t, fy+(sy-fy)*t-flyArc*zoom*4*t*(1-t)
	names := h.layers(b.Look)
	return Drawable{Depth: depth + fxAbove, Draw: func(dst *ebiten.Image) { r.drawBox(dst, names, x, y, zoom, b.Look, [3]float32{}) }}, r.boxMark(names, x, y, zoom), true
}

func (r *Renderer) heapItem(cam render.Camera, sc Scene, p HavenPlan, y Yard) (Drawable, pileMark) {
	h := r.HarborOf()
	zoom := cam.Scale()
	pos := HeapPoint(p, y.Kind)
	sx, sy := ScreenPointOf(cam, sc.Site, pos)
	sy -= h.Lift(pos) * zoom
	name := gfx.BoxPileName(KindName(y.Kind), gfx.BoxPileLevel(y.Overflow))
	mark := pileMark{Rect: SpriteRect(r.Lib, name, sx, sy, zoom, 20, 16), Shapes: ShapeAt(name, sx, sy, zoom)}
	return Drawable{Depth: render.DepthKey(pos.X, pos.Y) - heapBehind, Draw: func(dst *ebiten.Image) {
		if !r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{}) {
			block(dst, sx, sy, zoom, 18, 12, boxFallback[KindName(y.Kind)])
		}
	}}, mark
}

func (r *Renderer) GhostItems(out []Drawable, cam render.Camera, sc Scene, g spoilGhost) []Drawable {
	h := r.HarborOf()
	zoom := cam.Scale()
	alpha := GhostAlpha(sc.Time - g.start)
	age := sc.Time - g.start
	for _, b := range g.Boxes {
		if b.Haven >= len(h.Plans) {
			continue
		}
		pos := boxPoint(h.Plans[b.Haven], b)
		sx, sy := ScreenPointOf(cam, sc.Site, pos)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		sy -= (h.Lift(pos) + float64(b.Lift)) * zoom
		names := h.layers(b.Look)
		depth := render.DepthKey(pos.X, pos.Y) + boxAbove + float64(b.Level)*levelStep
		fx := gfx.SpoilName(int(age / SpoilLinger * 6))
		out = append(out, Drawable{Depth: depth, Draw: func(dst *ebiten.Image) {
			for _, n := range names {
				r.Lib.Draw(dst, n, sx, sy, zoom, render.SpriteOptions{Alpha: alpha})
			}
			if !sc.Reduce && b.Level == 0 {
				r.Lib.Draw(dst, fx, sx, sy, zoom, render.SpriteOptions{})
			}
		}})
	}
	return out
}

func (h *Harbor) RevealAt(b QuayBox) (float64, bool) {
	i, s, ok := h.shipOfLanding(b.At)
	if !ok || i != b.Haven || s.Cargo <= 0 || h.Now >= s.Dock+s.Unload+FlyTime {
		return 0, false
	}
	offset, accepted := h.shipOffset(s, i, keyOf(b.Lane, b.At))
	each := s.Unload / float64(max(accepted, 1))
	return s.Dock + (float64(offset+b.Reveal)+.5)*each, true
}

func (h *Harbor) shipOffset(s Ship, haven int, key loadKey) (int, int) {
	offset, accepted, found := 0, 0, false
	for _, l := range h.landed {
		if l.Haven != haven || math.Abs(l.At-s.Dock) > sameShip {
			continue
		}
		if keyOf(l.Lane, l.At) == key {
			found = true
		}
		if !found {
			offset += l.Total
		}
		accepted += l.Total
	}
	return offset, accepted
}

func (h *Harbor) accepted(s Ship, haven int) int {
	_, total := h.shipOffset(s, haven, loadKey{at: math.MinInt64})
	return total
}

func GhostAlpha(age float64) float32 {
	return float32(ghostAlpha * min(max(1-age/SpoilLinger, 0), 1))
}

func (h *Harbor) front() (Vec, float64, bool) {
	top, _, ok := h.frontOf(0, -1)
	return top.pos, top.lift, ok
}

type pileTop struct {
	pos  Vec
	lift float64
}

func (h *Harbor) frontOf(kind, haven int) (pileTop, *Load, bool) {
	l, ok := NextToServe(h.Loads, kind, haven)
	if !ok || l.Haven >= len(h.Plans) {
		return pileTop{}, nil, false
	}
	boxes := l.Boxes()
	if len(boxes) == 0 {
		return pileTop{}, nil, false
	}
	top := boxes[0]
	pos := boxPoint(h.Plans[l.Haven], top)
	if top.Column == offYard {
		pos = HeapPoint(h.Plans[l.Haven], top.Kind)
	}
	return pileTop{pos: pos, lift: h.Lift(pos) + float64(top.Lift+gfx.BoxLift(top.Look.Size))}, l, true
}

type PileTop struct {
	X, Y float64
	At   float64
}

func (r *Renderer) PileFront(kind, haven int, cam render.Camera, site view.Site) (PileTop, bool) {
	top, l, ok := r.HarborOf().frontOf(kind, haven)
	if !ok {
		return PileTop{}, false
	}
	sx, sy := ScreenPointOf(cam, site, top.pos)
	return PileTop{X: sx, Y: sy - top.lift*cam.Scale(), At: l.seen}, true
}

func (r *Renderer) FrontBox(cam render.Camera, site view.Site) (float64, float64, bool) {
	pos, lift, ok := r.HarborOf().front()
	if !ok {
		return 0, 0, false
	}
	sx, sy := ScreenPointOf(cam, site, pos)
	return sx, sy - lift*cam.Scale(), true
}

var boxFallback = map[string]color.RGBA{
	"parcel": {196, 160, 110, 255}, "ledger": {92, 120, 150, 255}, "crate": {150, 104, 64, 255}, "dispatch": {120, 90, 70, 255},
}
