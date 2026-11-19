package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	pickupInland = .45
	ServiceKind  = "Service"
	pileAbove    = .01
)

type haul struct {
	Kind   int `json:"-"`
	pickup Vec
	Drop   Vec `json:"-"`
	via    Vec
	gated  bool
}

func gateOf(site view.Site) (view.Structure, bool) {
	for _, s := range site.Structures {
		if s.Kind == view.StructGate && s.State == view.StateReady {
			return s, true
		}
	}
	return view.Structure{}, false
}

type boxLoad struct {
	Kind string `json:"-"`
	size int
}

func TradeKindIndex(role string) (int, bool) {
	if !gfx.IsTrade(role) {
		return 0, false
	}
	kind := gfx.TradeKind(role)
	for i, k := range FreightKinds {
		if k == kind {
			return i, true
		}
	}
	return 0, true
}

func tradeDepot(role string) string {
	kind := gfx.TradeKind(role)
	if kind == "" {
		kind = soulstone.KindSealed
	}
	d, _ := soulstone.DepotFor(kind)
	return d.Service
}

func DepotOf(site view.Site, service string) (view.Structure, bool) {
	for _, s := range site.Structures {
		if s.Object.Kind == ServiceKind && s.Object.Name == service && s.State == view.StateReady {
			return s, true
		}
	}
	return view.Structure{}, false
}

func (r *Renderer) HaulOf(sc Scene, p model.PodView) (haul, bool) {
	kind, ok := TradeKindIndex(p.Sprite)
	h := r.HarborOf()
	if !ok || len(h.Plans) == 0 {
		return haul{}, false
	}
	drop, ok := dropFor(sc.Site, p)
	if !ok {
		return haul{}, false
	}
	plan := h.Plan(HavenFor(int(model.Hash(p.Name)%4096), Weights(h.Plans)))
	across, depth := YardCentre(kind)
	out := haul{Kind: kind, pickup: QuayPoint(plan, across, depth+YardDeep/2+pickupInland), Drop: drop}
	if gate, ok := gateOf(sc.Site); ok {
		if _, depot := DepotOf(sc.Site, tradeDepot(p.Sprite)); depot {
			out.via, out.gated = Door(gate), true
		}
	}
	return out, true
}

func dropFor(site view.Site, p model.PodView) (Vec, bool) {
	if depot, ok := DepotOf(site, tradeDepot(p.Sprite)); ok {
		return Door(depot), true
	}
	if !p.Static {
		return Vec{}, false
	}
	keep, ok := KeepForNode(site, p.Node)
	if !ok || keep.State != view.StateReady {
		return Vec{}, false
	}
	return Porch(keep), true
}

func (r *Renderer) carrierFigure(sc Scene, p model.PodView, h haul) (Figure, bool) {
	home, ok := r.Homes[p.Name]
	if !ok {
		return Figure{}, false
	}
	start := Vec{home.X + .35, home.Y + .35}
	hb := r.HarborOf()
	loop := NewTripLoop(hb.Trip(sc.Site, start, h.pickup), hb.Trip(sc.Site, h.pickup, h.Drop), hb.Trip(sc.Site, h.Drop, start))
	if h.gated {
		loop = NewGateLoop(hb.Trip(sc.Site, start, h.pickup), hb.Trip(sc.Site, h.pickup, h.via), hb.Trip(sc.Site, h.via, h.Drop), hb.Trip(sc.Site, h.Drop, start))
	}
	rate, working, throttled := Workload(sc, p, true)
	f := r.PacedOn(sc, p, start, loop, rate, working, throttled)
	f.home, f.deliver = home, false
	if f.Pose.Carry {
		f.Load = &boxLoad{Kind: LoadKind(p.Sprite, h.Kind), size: TripSize(sc.Freight, h.Kind, personName(p), f.clock, loop.Length())}
	}
	return f, true
}

func TripSize(fv view.FreightView, kind int, pod string, clock, loop float64) int {
	if kind < 0 || kind >= len(fv.Kinds) || loop <= 0 {
		return 0
	}
	trip := int(clock / loop)
	u := float64(mix32(model.Hash(pod)+uint32(trip)*0x9e3779b1)%10000) / 10000
	return SizeFromMix(fv.Kinds[kind].SizeMix, u)
}

func (r *Renderer) drawLoad(dst *ebiten.Image, l *boxLoad, pose Pose, sx, sy, zoom float64) {
	if l == nil {
		return
	}
	r.Lib.Draw(dst, gfx.CarryName(l.size, l.Kind, pose.Dir, pose.Frame), sx, sy, zoom, render.SpriteOptions{})
}

func (r *Renderer) Carrying() int { return r.HarborOf().carrying }

func DepotPileSpot(s view.Structure) Vec {
	if f, ok := stallFrameOf(s); ok {
		return f.pile()
	}
	_, hi := Footprint(s)
	return Vec{X: float64(hi.X), Y: float64(hi.Y)}
}

func DepotWaiting(fv view.FreightView, kind string) float64 {
	if kind == soulstone.KindSealed {
		return fv.Seals.Waiting
	}
	for _, k := range fv.Kinds {
		if k.Kind == kind {
			return k.Waiting
		}
	}
	return 0
}

func depotBoxKind(s view.Structure) (string, bool) {
	if len(s.Kind) <= len(view.DepotStructure("")) || s.Kind[:len(view.DepotStructure(""))] != view.DepotStructure("") {
		return "", false
	}
	return s.Kind[len(view.DepotStructure("")):], true
}

func (r *Renderer) DepotItems(out []Drawable, cam render.Camera, sc Scene) []Drawable {
	zoom := cam.Scale()
	for _, s := range sc.Site.Structures {
		kind, ok := depotBoxKind(s)
		if !ok || s.State != view.StateReady {
			continue
		}
		level := gfx.BoxPileLevel(int(DepotWaiting(sc.Freight, kind) + .5))
		if level < 0 {
			continue
		}
		at := DepotPileSpot(s)
		sx, sy := ScreenPointOf(cam, sc.Site, at)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		name := gfx.BoxPileName(kind, level)
		a := Anchor(s)
		out = append(out, Drawable{Depth: render.DepthKey(a.X, a.Y) + pileAbove, Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{})
		}})
	}
	return out
}

func StructureNames(s view.Structure) []string {
	return gfx.BuildingNames(s)
}
