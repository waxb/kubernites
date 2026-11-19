package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	ErrandDecree = "decree"
	ErrandUnward = "unward"

	decreeAct  = .5
	unwardAct  = 1.2
	scrollSide = 4.0
	scrollUp   = 11.0
	wardRise   = 12.0
)

var scrollName = gfx.AmbientName("scroll", 0)

func init() {
	ErrandActs[ErrandDecree] = decreeAct
	ErrandActs[ErrandUnward] = unwardAct
	errandFallback["scribe"] = "clerk"
}

func (r *Renderer) decreeItems(out []Drawable, cam render.Camera, sc Scene) []Drawable {
	zoom := cam.Scale()
	for _, e := range sc.Errands {
		if e.Kind != ErrandDecree {
			continue
		}
		step, ok := ErrandAt(e, e.To, sc.Time)
		if !ok || step.Phase != PhaseOut {
			continue
		}
		step = r.RoutedStep(sc, e, e.To, step)
		sx, sy := ScreenPointOf(cam, sc.Site, step.Pose.Pos)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		alpha := float32(step.Alpha)
		out = append(out, Drawable{Depth: render.DepthKey(step.Pose.Pos.X, step.Pose.Pos.Y) + .06, Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, scrollName, sx+scrollSide*zoom, sy-scrollUp*zoom, zoom, render.SpriteOptions{Alpha: alpha})
		}})
	}
	return out
}

func WardLift(sc Scene, porch Vec) (float64, bool) {
	for _, e := range sc.Errands {
		if e.Kind != ErrandUnward || Dist(e.To, porch) > 1.5 {
			continue
		}
		step, ok := ErrandAt(e, e.To, sc.Time)
		switch {
		case !ok || step.Phase == PhaseBack:
		case step.Phase == PhaseAct:
			return step.Progress, true
		default:
			return 0, true
		}
	}
	return 0, false
}
