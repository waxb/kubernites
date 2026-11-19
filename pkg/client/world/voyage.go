package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	VoyageSeconds = 9.0
	sailShare     = 0.45
	sailDistance  = 5.0
)

func EastLanding(site view.Site, y int) (Vec, bool) {
	y = min(max(y, 0), site.Height-1)
	for x := site.Width - 1; x >= 0; x-- {
		if !Wet(site.Tile(x, y)) {
			return Vec{float64(x) + .5, float64(y) + .5}, true
		}
	}
	return Vec{}, false
}

type Voyage struct {
	Sailing bool
	Boat    Vec
	Walker  Pose
	Arrived bool
}

func VoyageAt(site view.Site, home Vec, elapsed float64) Voyage {
	landing, ok := EastLanding(site, int(home.Y))
	progress := elapsed / VoyageSeconds
	if !ok || progress >= 1 {
		return Voyage{Arrived: true, Walker: Pose{Pos: home, Action: "idle", Dir: "sw"}}
	}
	sea := Vec{min(landing.X+sailDistance, float64(site.Width)-.5), landing.Y}
	if progress < sailShare {
		boat := Lerp(sea, landing, progress/sailShare)
		return Voyage{Sailing: true, Boat: boat, Walker: Pose{Pos: boat, Action: "idle", Dir: "sw"}}
	}
	t := (progress - sailShare) / (1 - sailShare)
	pos := Lerp(landing, home, t)
	return Voyage{Walker: Pose{Pos: pos, Action: "walk", Dir: Facing(landing, home), Frame: Frame(elapsed, 8)}}
}

func (r *Renderer) kindleElapsed(name string, now float64, seen map[string]float64) float64 {
	start, ok := r.kindled[name]
	if !ok {
		start = now
	}
	seen[name] = start
	return now - start
}

func (r *Renderer) voyageItems(cam render.Camera, sc Scene, p model.PodView, home Vec, seen map[string]float64) []Drawable {
	elapsed := r.kindleElapsed(p.Name, sc.Time, seen)
	var out []Drawable
	for i := range p.Persons() {
		out = append(out, r.arrivalOf(cam, sc, p.Person(i), home, elapsed-float64(i)*arriveGap, i)...)
	}
	return out
}

func (r *Renderer) arrivalOf(cam render.Camera, sc Scene, p model.PodView, home Vec, elapsed float64, i int) []Drawable {
	if items, ok := r.arrivalItems(cam, sc, p, home, elapsed, i); ok {
		return items
	}
	if elapsed < 0 {
		return nil
	}
	v := VoyageAt(sc.Site, offset(Vec{home.X + .5, home.Y + .5}, PersonAside(i), 1), elapsed)
	if !v.Sailing {
		return []Drawable{r.walkerDrawable(cam, sc, p.Sprite, v.Walker, &p, true)}
	}
	if i > 0 {
		return nil
	}
	sx, sy := ScreenPointOf(cam, sc.Site, v.Boat)
	name := gfx.EffectName("boat", Frame(sc.Time, 4))
	return []Drawable{{Depth: render.DepthKey(v.Boat.X, v.Boat.Y), Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, name, sx, sy, cam.Scale(), render.SpriteOptions{})
	}}}
}
