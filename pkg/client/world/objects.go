package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var componentStructures = map[string]string{
	"etcd": view.StructCodex, "kube-apiserver": view.StructChancery, "kube-scheduler": view.StructQuartermaster,
	"kube-controller-manager": view.StructStewards, "coredns": view.StructHerald, "kube-proxy": view.StructWayfinder,
	"metrics-server": view.StructObservatory,
}

func ComponentStructure(pod model.PodView) string {
	for prefix, kind := range componentStructures {
		if len(pod.Name) >= len(prefix) && pod.Name[:len(prefix)] == prefix {
			return kind
		}
	}
	return ""
}

func (r *Renderer) collect(cam render.Camera, sc Scene) []Drawable {
	r.agesOf().track(sc.Age, sc.Time)
	items := []Drawable{}
	items = append(items, r.sceneryItems(cam, sc)...)
	items = append(items, r.wildItems(cam, sc)...)
	items = append(items, r.charterItems(cam, sc)...)
	items = append(items, r.structureItems(cam, sc)...)
	items = append(items, r.storageItems(cam, sc)...)
	items = append(items, r.middenItems(cam, sc)...)
	items = append(items, r.codexItems(cam, sc)...)
	items = append(items, r.HearthItems(cam, sc)...)
	items = append(items, r.WalkerItems(cam, sc)...)
	items = append(items, r.TransientItems(cam, sc)...)
	items = append(items, r.OmenItems(cam, sc)...)
	return items
}

type Placed struct {
	S     view.Structure `json:"-"`
	names []string
	sx    float64
	sy    float64
	Rect  ui.Rect `json:"-"`
	Depth float64 `json:"-"`
	busy  bool
}

func (r *Renderer) placeStructures(cam render.Camera, sc Scene) []Placed {
	zoom := cam.Scale()
	out := make([]Placed, 0, len(sc.Site.Structures))
	for _, s := range sc.Site.Structures {
		a := structureAnchor(s)
		sx, sy := ScreenPointOf(cam, sc.Site, Vec{a.X - .01, a.Y - .01})
		names, busy := r.structureLook(s, sc)
		rect := SpriteRect(r.Lib, r.ShownName(names), sx, sy, zoom, s.W*24, s.H*20)
		out = append(out, Placed{S: s, names: names, sx: sx, sy: sy, Rect: rect, Depth: render.DepthKey(a.X, a.Y), busy: busy})
	}
	return out
}

func Overlaps(a, b ui.Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

func focused(p Placed, selected string) bool {
	return p.S.ID == selected || p.S.State == view.StateConstruction
}

func Occluders(items []Placed, selected string, hides func(back, front Placed) bool) map[string]bool {
	out := map[string]bool{}
	for _, back := range items {
		if !focused(back, selected) {
			continue
		}
		for _, front := range items {
			if front.S.ID != back.S.ID && front.Depth > back.Depth && Overlaps(front.Rect, back.Rect) && hides(back, front) {
				out[front.S.ID] = true
			}
		}
	}
	return out
}

func (r *Renderer) structureItems(cam render.Camera, sc Scene) []Drawable {
	zoom := cam.Scale()
	items := r.placeStructures(cam, sc)
	fade := Occluders(items, sc.Selected, r.hides(zoom))
	out := []Drawable{}
	for _, it := range items {
		it := it
		alpha := float32(1)
		if it.S.State == view.StateConstruction {
			alpha = .55 + .45*float32(it.S.Progress)
		}
		if fade[it.S.ID] {
			alpha = min(alpha, .45)
		}
		hit := &Hit{Kind: HitStructure, ID: it.S.ID, Structure: it.S, Rect: it.Rect, Shapes: ShapeAt(r.ShownName(it.names), it.sx, it.sy, zoom)}
		lo, hi := Footprint(it.S)
		selected := it.S.ID == sc.Selected
		ivy := r.ivyFor(it.S)
		opts := render.SpriteOptions{Alpha: alpha}
		if tint, ok := sc.heraldTint(it.S.Object.Namespace); ok && it.S.Kind != view.StructKeep {
			opts.Tint = tint
		}
		out = append(out, Drawable{Depth: it.Depth, Hit: hit, box: footprint{on: true, lo: lo, hi: hi, rect: it.Rect}, Draw: func(dst *ebiten.Image) {
			if selected {
				r.outlineSprite(dst, it.names, it.sx, it.sy, zoom)
			}
			name, ok := r.Lib.DrawFirst(dst, it.names, it.sx, it.sy, zoom, opts)
			if !ok {
				fallbackBlock(dst, cam, sc.Site, it.S)
				return
			}
			r.drawIvy(dst, name, ivy, it.sx, it.sy, zoom, alpha)
			r.drawFlags(dst, name, it.S, it.sx, it.sy, zoom, sc.Time, alpha)
		}})
		if it.S.State == view.StateConstruction {
			out = append(out, r.builderItems(cam, sc, it.S)...)
		}
		if it.busy {
			out = append(out, r.ageBurst(cam, sc, FootprintCentre(it.S), float64(max(it.S.W, it.S.H))*structBurst, it.S.ID)...)
		}
	}
	return out
}

func (r *Renderer) builderItems(cam render.Camera, sc Scene, s view.Structure) []Drawable {
	door := Door(s)
	out := []Drawable{}
	for i := 0; i < 2; i++ {
		pos := Vec{door.X - float64(i)*.9, door.Y + float64(i)*.2}
		frame := Frame(sc.Time+float64(i)*.4, 5)
		out = append(out, r.walkerDrawable(cam, sc, "builder", Pose{Pos: pos, Action: "work", Dir: "sw", Frame: frame}, nil, false))
	}
	dust := gfx.EffectName("dust", int(sc.Time*8))
	sx, sy := ScreenPointOf(cam, sc.Site, door)
	out = append(out, Drawable{Depth: render.DepthKey(door.X, door.Y) + .2, Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, dust, sx, sy, cam.Scale(), render.SpriteOptions{Alpha: .7})
	}})
	return out
}

func (r *Renderer) HearthItems(cam render.Camera, sc Scene) []Drawable {
	r.watch = indexWatch(sc.Watch)
	h := r.hamletOf()
	r.coverHamlet(h, sc.Site)
	h.Pin(sc.Pins)
	h.Place(sc.Site, sc.Pods, sc.Time)
	gates := h.Gates(sc.Site, sc.Pods)
	housed := withGates(h.Homes, sc.Pods, h, gates)
	plan := PlanCamps(sc.Site, sc.Pods, housed)
	homes := withCamps(housed, sc.Pods, plan)
	seen := map[string]float64{}
	out := []Drawable{}
	for _, p := range sc.Pods {
		if p.System {
			continue
		}
		spot, housed := h.SpotList[p.Name]
		rise := 1.0
		if housed {
			rise = RiseOf(spot.Born, sc.Time)
		}
		if housed || plan.Shown[p.Name] {
			out = append(out, r.hearthDrawable(cam, sc, p, homes[p.Name], rise, seen)...)
		}
	}
	if plan.Hidden > 0 {
		out = append(out, r.signpostItem(cam, sc, plan))
	}
	out = r.gateItems(r.hamletItems(out, cam, sc, h), cam, sc, gates)
	r.Homes, r.kindled, r.post, r.gates = homes, seen, plan, gates
	return out
}

func withCamps(housed map[string]Vec, pods []model.PodView, plan CampPlan) map[string]Vec {
	if len(plan.Spots) == 0 {
		return housed
	}
	out := make(map[string]Vec, len(housed)+len(plan.Spots))
	for name, v := range housed {
		out[name] = v
	}
	for _, p := range pods {
		if _, ok := housed[p.Name]; !ok && !p.System {
			out[p.Name] = plan.Spots[p.Name]
		}
	}
	return out
}

func Keeps(site view.Site) []view.Structure {
	out := []view.Structure{}
	for _, s := range site.Structures {
		if s.Kind == view.StructKeep && s.Object.Name != "" {
			out = append(out, s)
		}
	}
	return out
}
