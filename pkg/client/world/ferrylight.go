package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	landingLampX    = -4.5
	landingLampHead = 23.5
)

type harborLight struct {
	At       Vec     `json:"-"`
	dx, Head float64 `json:"-"`
}

func (r *Renderer) HarborLights(sc Scene) []harborLight {
	out := []harborLight{}
	for _, l := range r.HarborOf().Lanes {
		for _, j := range []Jetty{l.Home, l.Away} {
			out = append(out, harborLight{At: j.Landing(), dx: landingLampX, Head: landingLampHead})
		}
		age := r.laneAge(sc, l)
		for k := 0; k < FerryBoats; k++ {
			pose := l.Boat(k, sc.Time)
			if dx, dy, ok := art.FerryLamp(art.FerryLook{Age: age, Dir: pose.Dir, Frame: pose.Frame}); ok {
				out = append(out, harborLight{At: pose.Pos, dx: dx, Head: -dy})
			}
		}
	}
	return out
}

func (r *Renderer) drawHarborGlow(dst *ebiten.Image, cam render.Camera, sc Scene, strength float64) {
	zoom := cam.Scale()
	for _, l := range r.HarborLights(sc) {
		sx, sy := ScreenPointOf(cam, sc.Site, l.At)
		if onScreen(sx, sy, zoom) {
			r.drawGlow(dst, sx+l.dx*zoom, sy-(l.Head+r.HarborOf().Lift(l.At))*zoom, lampGlowR*zoom, strength)
		}
	}
}
