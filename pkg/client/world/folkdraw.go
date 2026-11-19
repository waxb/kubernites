package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	stallFront  = .6
	merchantHz  = .25
	nightShare  = 3
	folkFaded   = .92
	stallsDepth = -.01
)

var (
	stallNames = func() [gfx.StallLooks]string {
		var out [gfx.StallLooks]string
		for i := range out {
			out[i] = gfx.AmbientName("stall", i)
		}
		return out
	}()
	folkTint = [3]float32{folkFaded, folkFaded, folkFaded}
)

func VisibleFolk(total int, night bool) int {
	if night {
		return total / nightShare
	}
	return total
}

func (r *Renderer) folkItems(out []Drawable, cam render.Camera, sc Scene, l *life) []Drawable {
	zoom := cam.Scale()
	night := Darkness(sc.DayPhase) > restingDark
	c := l.people.of(sc)
	n := min(VisibleFolk(FolkCount(c.declared, c.rate, readyKeeps(sc.Site)), night), len(l.folk))
	budget := FolkBudget(zoom)
	for i := 0; i < n && budget > 0; i++ {
		s := l.folk[i]
		if night && s.role == "child" {
			continue
		}
		pose := s.At(sc.Time)
		sx, sy := ScreenPointOf(cam, sc.Site, pose.Pos)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		budget--
		name := l.folkName(s.role, s.look, pose)
		out = append(out, Drawable{person: s.role, Depth: render.DepthKey(pose.Pos.X, pose.Pos.Y) + .05, stand: stand(pose.Pos, sx, sy), Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{Tint: folkTint})
		}})
	}
	return out
}

func (r *Renderer) stallItems(out []Drawable, cam render.Camera, sc Scene, l *life) []Drawable {
	zoom := cam.Scale()
	open := Darkness(sc.DayPhase) <= restingDark
	for i := 0; i < min(StallCount(l.people.of(sc).rate), len(l.stalls)); i++ {
		spot := l.stalls[i]
		sx, sy := ScreenPointOf(cam, sc.Site, spot)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		name := stallNames[i%len(stallNames)]
		out = append(out, Drawable{Depth: render.DepthKey(spot.X, spot.Y) + stallsDepth, Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{})
		}})
		if open && drawFiguresWithoutPods {
			out = r.merchant(out, cam, sc, l, spot, i)
		}
	}
	return out
}

func (r *Renderer) merchant(out []Drawable, cam render.Camera, sc Scene, l *life, spot Vec, i int) []Drawable {
	zoom := cam.Scale()
	at := Vec{spot.X + stallFront, spot.Y + stallFront}
	pose := Pose{Pos: at, Action: "idle", Dir: "sw", Frame: Frame(sc.Time+float64(i), 2)}
	if int(sc.Time*merchantHz+float64(i)*.5)%2 == 0 {
		pose.Action, pose.Frame = "work", Frame(sc.Time, 3)
	}
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	name := l.folkName("merchant", i, pose)
	return append(out, Drawable{person: "merchant", Depth: render.DepthKey(at.X, at.Y) + .05, stand: stand(at, sx, sy), Draw: func(dst *ebiten.Image) {
		r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{Tint: folkTint})
	}})
}
