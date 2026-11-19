package world

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	dockSpeed = .9
	dockDrop  = .8
	dockPick  = .6
	maxDock   = 3
	jibX      = -20.0
	hookY     = -34.0
	hoistMin  = 3.0
	hoistSpan = 18.0
	hoistTime = 3.2
)

var ropeColor = color.RGBA{58, 44, 36, 255}

func HoistDrop(t float64) float64 {
	return hoistMin + hoistSpan*(1-math.Cos(2*math.Pi*t/hoistTime))/2
}

func dockTrip(from, to Vec) float64 {
	return Dist(from, to) / dockSpeed
}

func DockShuttle(t, phase float64, from, to Vec) Pose {
	trip := dockTrip(from, to)
	cycle := 2*trip + dockDrop + dockPick
	u := wrap(t+phase, max(cycle, .001))
	switch {
	case u < trip:
		return Pose{Pos: Lerp(from, to, u/max(trip, .001)), Action: "carry", Dir: Facing(from, to), Frame: Frame(u, 6)}
	case u < trip+dockDrop:
		return Pose{Pos: to, Action: "work", Dir: Facing(from, to), Frame: Frame(u, 6)}
	case u < 2*trip+dockDrop:
		return Pose{Pos: Lerp(to, from, (u-trip-dockDrop)/max(trip, .001)), Action: "walk", Dir: Facing(to, from), Frame: Frame(u, 6)}
	}
	return Pose{Pos: from, Action: "work", Dir: Facing(to, from), Frame: Frame(u, 6)}
}

func crateSpot(p HavenPlan) Vec {
	for _, prop := range p.Props {
		if prop.Kind == "crates" && prop.Deck {
			return offset(prop.At, p.Side, .6)
		}
	}
	return offset(p.Quay, p.Side, -DeckHalf+1)
}

func unloadingBerth(p HavenPlan, f *fleet, now float64) (Berth, bool) {
	if len(p.Berths) == 0 {
		return Berth{}, false
	}
	for _, s := range f.Ships {
		b := p.Berths[s.Berth%len(p.Berths)]
		if pose, ok := ShipAt(s, b, p.Sea, now); ok && pose.Unloading {
			return b, true
		}
	}
	return Berth{}, false
}

func (r *Renderer) dockItems(out []Drawable, cam render.Camera, sc Scene) []Drawable {
	h := r.HarborOf()
	for i, p := range h.Plans {
		if !p.Open || i >= len(h.Fleets) {
			continue
		}
		crates := crateSpot(p)
		b, busy := unloadingBerth(p, h.Fleets[i], h.Now)
		if !busy {
			if p.Tier >= 1 && drawFiguresWithoutPods {
				pose := Pose{Pos: crates, Action: "idle", Dir: Facing(crates, p.Head), Frame: Frame(sc.Time, 2)}
				out = r.dockhand(out, cam, sc, pose, i)
			}
			continue
		}
		out = r.craneItems(out, cam, sc, p)
		if !drawFiguresWithoutPods {
			continue
		}
		n := min(1+p.Tier, maxDock)
		cycle := 2*dockTrip(b.Plank, crates) + dockDrop + dockPick
		for k := 0; k < n; k++ {
			out = r.dockhand(out, cam, sc, DockShuttle(sc.Time, cycle*float64(k)/float64(n), b.Plank, crates), i+k)
		}
	}
	return out
}

func (r *Renderer) dockhand(out []Drawable, cam render.Camera, sc Scene, pose Pose, look int) []Drawable {
	zoom := cam.Scale()
	sx, sy := ScreenPointOf(cam, sc.Site, pose.Pos)
	if !onScreen(sx, sy, zoom) {
		return out
	}
	sy -= r.HarborOf().Lift(pose.Pos) * zoom
	name := r.lifeOf().folkName("dockhand", look, pose)
	return append(out, Drawable{person: "dockhand", Depth: render.DepthKey(pose.Pos.X, pose.Pos.Y) + .05, stand: stand(pose.Pos, sx, sy), Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{})
	}})
}

func (r *Renderer) craneItems(out []Drawable, cam render.Camera, sc Scene, p HavenPlan) []Drawable {
	zoom := cam.Scale()
	for _, prop := range p.Props {
		if prop.Kind != "crane" {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, prop.At)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		dx := jibX
		if prop.Flip {
			dx = -jibX
		}
		x, top := sx+dx*zoom, sy-(art.HavenDeck-hookY)*zoom
		t := sc.Time
		if sc.Reduce {
			t = 0
		}
		drop := HoistDrop(t) * zoom
		out = append(out, Drawable{Depth: render.DepthKey(prop.At.X, prop.At.Y) + .01, Draw: func(dst *ebiten.Image) {
			vector.StrokeLine(dst, float32(x), float32(top), float32(x), float32(top+drop), float32(max(zoom, 1)), ropeColor, false)
			r.Lib.Draw(dst, crateName, x, top+drop+6*zoom, zoom, render.SpriteOptions{})
		}})
	}
	return out
}
