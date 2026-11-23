package world

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	porchStep     = 1.25
	reeveAside    = 1.2
	idleLift      = .35
	idleSide      = .15
	HelmAhead     = 0.6
	helmSide      = 0.9
	serveAnim     = 0.6
	banishedAlpha = .85
)

var stateEffects = map[string][]string{
	model.HearthGuttering: {"fire", "smoke"},
	model.HearthBanished:  {"dust"},
}

func (r *Renderer) hearthDrawable(cam render.Camera, sc Scene, p model.PodView, home Vec, rise float64, seen map[string]float64) []Drawable {
	var out []Drawable
	if p.State == model.HearthKindling {
		out = r.voyageItems(cam, sc, p, home, seen)
	}
	zoom := cam.Scale()
	anchor := Vec{home.X + .5, home.Y + .5}
	sx, sy := ScreenPointOf(cam, sc.Site, Vec{anchor.X - .01, anchor.Y - .01})
	if !onScreen(sx, sy, zoom) {
		return out
	}
	pod, now, reduce := p, sc.Time, sc.Reduce
	name := r.hutName(p)
	opts := render.SpriteOptions{Alpha: 1}
	if p.Terminating || p.State == model.HearthBanished {
		opts.Alpha = banishedAlpha
	}
	if tint, ok := sc.heraldTint(p.Namespace); ok {
		opts.Tint = tint
	}
	hit := &Hit{Kind: HitHearth, ID: p.Namespace + "/" + p.Name, Pod: p, Rect: SpriteRect(r.Lib, name, sx, sy, zoom, 22, 22), Shapes: ShapeAt(name, sx, sy, zoom)}
	tile := Point{int(home.X + .5), int(home.Y + .5)}
	selected := selectedPod(p.Name, sc.Selected) || p.Namespace+"/"+p.Name == sc.Selected
	out = append(out, Drawable{Depth: render.DepthKey(anchor.X, anchor.Y), Hit: hit, box: footprint{on: true, lo: tile, hi: tile, rect: hit.Rect}, Draw: func(dst *ebiten.Image) {
		if selected && rise >= 1 {
			r.outlineSprite(dst, []string{name}, sx, sy, zoom)
		}
		if !r.drawRising(dst, name, sx, sy, zoom, opts, rise) {
			vector.FillRect(dst, float32(sx-6*zoom), float32(sy-12*zoom), float32(12*zoom), float32(10*zoom), color.RGBA{170, 110, 70, 255}, false)
		}
		if rise >= 1 {
			age, _ := r.agesOf().phase(pod.Name)
			r.drawWatch(dst, r.watchSprites(pod, r.watch, age, sx, sy, zoom, now, reduce), zoom, opts)
			r.drawRequestPiles(dst, pod, sx, sy, zoom)
			r.drawRankMark(dst, name, pod.Rank, sx, sy, zoom)
		}
	}})
	if d, ok := r.riseDust(cam, sc, anchor, rise); ok {
		out = append(out, d)
	}
	if _, busy := r.agesOf().phase(p.Name); busy {
		out = append(out, r.ageBurst(cam, sc, anchor, hutBurst, p.Name)...)
	}
	return append(out, r.stateEffectItems(sc, p, anchor, sx, sy, zoom)...)
}

func (r *Renderer) stateEffectItems(sc Scene, p model.PodView, anchor Vec, sx, sy, zoom float64) []Drawable {
	var out []Drawable
	for i, fx := range stateEffects[p.State] {
		fxName := gfx.EffectName(fx, int(sc.Time*8)+i*2)
		fy := sy - float64(14+i*6)*zoom
		out = append(out, Drawable{Depth: render.DepthKey(anchor.X, anchor.Y) + .1, Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, fxName, sx, fy, zoom, render.SpriteOptions{})
		}})
	}
	return out
}

func (r *Renderer) WalkerItems(cam render.Camera, sc Scene) []Drawable {
	rt := RouteOf(sc.Site)
	out := []Drawable{}
	carrying := 0
	figs := r.Crowd(sc, rt)
	if !r.inset {
		r.livingOf().wear(figs, sc)
		r.markFocus(sc, figs)
	}
	for _, f := range MarkNearest(cam, sc.Site, Visible(cam, sc.Site, figs, sc.Selected)) {
		out = append(out, r.figureItems(cam, sc, f, rt)...)
		if f.Load != nil {
			carrying++
		}
	}
	r.HarborOf().carrying = carrying
	out = append(out, r.ferryItems(cam, sc)...)
	out = append(out, r.reeveItems(cam, sc)...)
	out = append(out, r.ClerkItems(cam, sc)...)
	out = append(out, r.ErrandItems(cam, sc)...)
	out = append(out, r.freightItems(cam, sc)...)
	out = append(out, r.debutItems(cam, sc)...)
	return out
}

func doorOf(site view.Site, kind string) (Vec, bool) {
	s, ok := FirstOfKind(site, kind, true)
	if !ok {
		return Vec{}, false
	}
	return Porch(s), true
}

func Porch(s view.Structure) Vec {
	if f, ok := stallFrameOf(s); ok {
		return f.door()
	}
	_, hi := Footprint(s)
	return Vec{float64(hi.X) + porchStep, float64(hi.Y) + porchStep}
}

func (r *Renderer) reeveItems(cam render.Camera, sc Scene) []Drawable {
	out := []Drawable{}
	for _, k := range Keeps(sc.Site) {
		if k.State != view.StateReady {
			continue
		}
		porch := Porch(k)
		pose := IdlePose(sc.Time, model.Hash(k.ID), Vec{porch.X - reeveAside, porch.Y - idleLift})
		out = append(out, r.walkerDrawable(cam, sc, "reeve", pose, nil, false))
	}
	return out
}

func GatewayOf(site view.Site) Gateway {
	door := Vec{float64(site.Gate.X) + .5, float64(site.Gate.Y) + .5}
	if s, ok := FirstOfKind(site, view.StructGate, false); ok {
		door = Door(s)
	}
	in := Vec{float64(site.Plaza.X) - door.X, float64(site.Plaza.Y) - door.Y}
	n := max(math.Hypot(in.X, in.Y), 1e-9)
	return Gateway{Door: door, Inward: Vec{in.X / n, in.Y / n}}
}

func (r *Renderer) freightItems(cam render.Camera, sc Scene) []Drawable {
	out := r.DepotItems(r.HarborItems(cam, sc), cam, sc)
	if sc.GateOpen {
		out = append(out, r.Helmbearer(cam, sc, GatewayOf(sc.Site)))
	}
	return out
}

func (r *Renderer) Helmbearer(cam render.Camera, sc Scene, g Gateway) Drawable {
	side := g.Side()
	pos := Vec{g.Door.X + g.Inward.X*HelmAhead + side.X*helmSide, g.Door.Y + g.Inward.Y*HelmAhead + side.Y*helmSide}
	facing := offset(g.Door, g.Inward, -1)
	if front, _, ok := r.HarborOf().front(); ok {
		facing = front
	}
	pose := Pose{Pos: pos, Action: "idle", Dir: Facing(pos, facing), Frame: Frame(sc.Time, 2)}
	if age := sc.Time - sc.ServedAt; age >= 0 && age < serveAnim {
		pose.Action, pose.Frame = "work", Frame(age, 10)
	}
	return r.walkerDrawable(cam, sc, "helmbearer", pose, nil, false)
}

func (r *Renderer) walkerDrawable(cam render.Camera, sc Scene, role string, pose Pose, pod *model.PodView, hittable bool) Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, pose.Pos)
	key := NameKey{Role: role, Action: pose.Action, Dir: pose.Dir, frame: pose.Frame, Tier: 1}
	var air gfx.Air
	if pod != nil {
		key.variant, key.Tier = LookOf(*pod)
		air = AirOf(*pod, role)
	}
	names := r.LookNames(key)
	var hit *Hit
	if hittable && pod != nil {
		shown := r.ShownName(names)
		hit = &Hit{Kind: HitWalker, ID: pod.Namespace + "/" + pod.Name, Pod: *pod, Rect: SpriteRect(r.Lib, shown, sx, sy, zoom, 12, 20), Shapes: ShapeAt(shown, sx, sy, zoom)}
	}
	carry, now := pose.Carry, sc.Time
	person := role
	if pod != nil {
		person = personWorker
	}
	return Drawable{person: person, Depth: render.DepthKey(pose.Pos.X, pose.Pos.Y) + .05, Hit: hit, stand: stand(pose.Pos, sx, sy), Draw: func(dst *ebiten.Image) {
		drawn, ok := r.Lib.DrawFirstAir(dst, names, sx, sy, zoom, render.SpriteOptions{}, air)
		if !ok {
			vector.FillRect(dst, float32(sx-2*zoom), float32(sy-10*zoom), float32(4*zoom), float32(10*zoom), color.RGBA{60, 90, 160, 255}, false)
		}
		if carry && role != ScribeRole {
			r.Lib.Draw(dst, gfx.EffectName("coin", int(now*8)), sx, sy-r.headLift(drawn, zoom), zoom, render.SpriteOptions{})
		}
	}}
}
