package world

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	campSpeed   = 1.3
	marchSpeed  = 1.6
	campNoClerk = .6
	campStale   = 120.0
	campStand   = .45
	campFront   = .6
	bundleSide  = 5.0
	bundleUp    = 9.0
	campWindow  = 7
	levyOwner   = "ReplicaSet/"
)

var bundleName = gfx.AmbientName("bundle", 0)

type camp struct {
	at   Vec
	seen float64
}

type folkKey struct {
	role, action, dir string
	look, frame       int
}

func (l *life) folkName(role string, look int, p Pose) string {
	k := folkKey{role: role, action: p.Action, dir: p.Dir, look: look, frame: p.Frame}
	if n, ok := l.names[k]; ok {
		return n
	}
	if l.names == nil {
		l.names = map[folkKey]string{}
	}
	n := gfx.FolkName(role, look, p.Action, p.Dir, p.Frame)
	l.names[k] = n
	return n
}

func CampWalk(from, to Vec, elapsed, wait float64) Pose {
	if elapsed < wait {
		return Pose{Pos: from, Action: "idle", Dir: Facing(from, to), Frame: Frame(elapsed, 2)}
	}
	d := (elapsed - wait) * campSpeed
	if length := Dist(from, to); d < length {
		return Pose{Pos: Lerp(from, to, d/length), Action: "walk", Dir: Facing(from, to), Frame: Frame(elapsed, 7)}
	}
	return Pose{Pos: to, Action: "work", Dir: "se", Frame: Frame(elapsed, 5)}
}

func CampSpot(home Vec) Vec {
	return Vec{home.X + campStand, home.Y + campFront}
}

func (l *life) noteCamps(sc Scene, homes map[string]Vec) {
	for _, p := range sc.Pods {
		if p.State == model.HearthPending && !p.System {
			if home, ok := homes[p.Name]; ok {
				l.camps[p.Name] = camp{at: CampSpot(home), seen: sc.Time}
			}
		}
	}
	for name, c := range l.camps {
		if sc.Time-c.seen > campStale || sc.Time < c.seen {
			delete(l.camps, name)
		}
	}
}

func (r *Renderer) Camp(pod string) (Vec, bool) {
	if r.life == nil {
		return Vec{}, false
	}
	c, ok := r.life.camps[pod]
	return c.at, ok
}

func clerkArrival(sc Scene, pod string) (float64, bool) {
	for _, e := range sc.Errands {
		if e.Kind == ErrandSchedule && e.Pod == pod && e.Life > 0 {
			act := min(ErrandActs[e.Kind], e.Life)
			return e.Start + (e.Life-act)/2 + act, true
		}
	}
	return 0, false
}

func levyOrigin(sc Scene, p model.PodView) (Vec, bool) {
	if !strings.HasPrefix(p.Owner, levyOwner) {
		return Vec{}, false
	}
	return doorOf(sc.Site, view.StructWarHorn)
}

func (r *Renderer) arrivalItems(cam render.Camera, sc Scene, p model.PodView, home Vec, elapsed float64, i int) ([]Drawable, bool) {
	target := offset(Vec{home.X + .35, home.Y + .35}, PersonAside(i), 1)
	if c, ok := r.lifeOf().camps[p.Name]; ok {
		wait := campNoClerk
		if at, found := clerkArrival(sc, p.Name); found {
			wait = at - (sc.Time - elapsed - float64(i)*arriveGap)
		}
		return r.arrivalWalker(cam, sc, p, CampWalk(c.at, target, elapsed, wait), true), true
	}
	if from, ok := levyOrigin(sc, p); ok {
		pose := CampWalk(from, target, elapsed*marchSpeed/campSpeed, 0)
		return r.arrivalWalker(cam, sc, p, pose, false), true
	}
	return nil, false
}

func (r *Renderer) arrivalWalker(cam render.Camera, sc Scene, p model.PodView, pose Pose, bundle bool) []Drawable {
	d := r.walkerDrawable(cam, sc, p.Sprite, pose, &p, true)
	if !bundle || pose.Action == "work" {
		return []Drawable{d}
	}
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, pose.Pos)
	inner := d.Draw
	d.Draw = func(dst *ebiten.Image) {
		inner(dst)
		r.Lib.Draw(dst, bundleName, sx+bundleSide*zoom, sy-bundleUp*zoom, zoom, render.SpriteOptions{})
	}
	return []Drawable{d}
}
