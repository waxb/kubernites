package world

import (
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/soulstone"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	debutWalk  = 7.0
	debutAside = 1.1
	debutFront = .55
	debutFPS   = 8
	debutIdle  = 2
	debutPace  = 1.6
)

type Debut struct {
	Kind     string
	Role     string
	Stall    Vec
	HasStall bool
}

func DebutsOf(reveals []string, fv view.FreightView) []Debut {
	out := []Debut{}
	for _, d := range soulstone.Depots() {
		if d.Kind == soulstone.KindParcel || !slices.Contains(reveals, d.Crew) || !debutDue(fv, d.Kind) {
			continue
		}
		if p, ok := soulstone.Lookup(d.Image); ok {
			out = append(out, Debut{Kind: d.Kind, Role: p.Sprite})
		}
	}
	return out
}

func debutDue(fv view.FreightView, kind string) bool {
	if kind == soulstone.KindSealed {
		return fv.Seals.Crew == 0 && (fv.Seals.ArrivalRate > 0 || fv.Seals.Waiting > 0)
	}
	for _, k := range fv.Kinds {
		if k.Kind == kind {
			return k.Crew == 0 && k.Open
		}
	}
	return false
}

func (r *Renderer) debutStart(kind string, now float64) float64 {
	if r.debuts == nil {
		r.debuts = map[string]float64{}
	}
	start, ok := r.debuts[kind]
	if !ok || now < start {
		start = now
		r.debuts[kind] = now
	}
	return start
}

func (r *Renderer) debutPath(sc Scene, d Debut) ([]Vec, Vec, bool) {
	h := r.HarborOf()
	i, ok := KindIndex(d.Kind)
	if !ok {
		i = 0
	}
	p := h.Plan(h.pileHaven(i))
	if len(p.Berths) == 0 {
		return nil, Vec{}, false
	}
	across, depth := YardCentre(i)
	pile := QuayPoint(p, across, depth)
	stand := Vec{pile.X + debutFront, pile.Y + debutFront}
	plank := p.Berths[0].Plank
	path := h.route(sc.Site, plank, stand)
	if len(path) < 2 {
		path = []Vec{plank, stand}
	}
	return path, pile, true
}

func (r *Renderer) DebutPose(sc Scene, d Debut) (Pose, bool) {
	path, pile, ok := r.debutPath(sc, d)
	if !ok {
		return Pose{}, false
	}
	elapsed := sc.Time - r.debutStart(d.Kind, sc.Time)
	if elapsed < debutWalk {
		return walkPose(path, elapsed/debutWalk, elapsed), true
	}
	stand := path[len(path)-1]
	if !d.HasStall {
		return Pose{Pos: stand, Action: "idle", Dir: Facing(stand, pile), Frame: Frame(sc.Time, debutIdle)}, true
	}
	leg := r.HarborOf().route(sc.Site, stand, d.Stall)
	if len(leg) < 2 {
		leg = []Vec{stand, d.Stall}
	}
	walk := PathLength(leg) / (debutPace * TradeGait(d.Role))
	if on := elapsed - debutWalk; on < walk {
		return walkPose(leg, on/walk, elapsed), true
	}
	end := leg[len(leg)-1]
	return Pose{Pos: end, Action: "idle", Dir: Facing(end, Vec{X: end.X + 1, Y: end.Y + 1}), Frame: Frame(sc.Time, debutIdle)}, true
}

func walkPose(path []Vec, frac, elapsed float64) Pose {
	pos, dir := PathAt(path, frac)
	return Pose{Pos: pos, Action: "walk", Dir: Facing(Vec{}, dir), Frame: Frame(elapsed, debutFPS)}
}

func (r *Renderer) debutItems(cam render.Camera, sc Scene) []Drawable {
	out := []Drawable{}
	zoom := cam.Scale()
	for _, d := range sc.Debuts {
		pose, ok := r.DebutPose(sc, d)
		if !ok {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, pose.Pos)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		frame := pose.Frame
		if pose.Action != "walk" {
			frame = 0
		}
		names := r.LookNames(NameKey{Role: d.Role, Action: gfx.HoldAction, Dir: pose.Dir, frame: frame, Tier: 1})
		i, _ := KindIndex(d.Kind)
		load := &boxLoad{Kind: LoadKind(d.Role, i)}
		out = append(out, Drawable{person: debutPerson, Depth: render.DepthKey(pose.Pos.X, pose.Pos.Y) + .05, stand: stand(pose.Pos, sx, sy), Draw: func(dst *ebiten.Image) {
			if _, ok := r.Lib.DrawFirst(dst, names, sx, sy, zoom, render.SpriteOptions{}); !ok {
				fallbackFigure(dst, sx, sy, zoom, 10, look{})
			}
			r.drawLoad(dst, load, pose, sx, sy, zoom)
		}})
	}
	return out
}
