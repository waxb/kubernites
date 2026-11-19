package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
)

const (
	incidentFPS = 6.0
	markerDX    = -13.0
	markerDY    = 3.0
	ringDY      = -2.0
	ringAlpha   = .88
	ringScale   = .9
	badgeDX     = 11.0
	badgeDY     = -12.0
)

type IncidentMark struct {
	Pod   string
	Crash bool
}

type incidentSprite struct {
	name   string
	sx, sy float64
	alpha  float32
	scale  float64
}

func IncidentFrame(t float64, reduce bool) int {
	if reduce {
		return 0
	}
	return int(t*incidentFPS) % art.IncidentFrames
}

func BackoffStep(restarts int32) int {
	return min(max(int(restarts), 0), len(art.BackoffWaits))
}

func restartsOf(sc Scene, pod string) int32 {
	name := podName(pod)
	for _, p := range sc.Pods {
		if p.Name == name {
			return p.Restarts
		}
	}
	return 0
}

func (r *Renderer) incidentSprites(cam render.Camera, sc Scene) []incidentSprite {
	out := []incidentSprite{}
	zoom := cam.Scale()
	for _, m := range sc.Incidents {
		home, ok := r.Homes[podName(m.Pod)]
		if !ok {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, Vec{home.X + .5, home.Y + .5})
		if !onScreen(sx, sy, zoom) {
			continue
		}
		out = append(out, incidentSprite{name: art.IncidentMarkerName(IncidentFrame(sc.Time, sc.Reduce)), sx: sx + markerDX*zoom, sy: sy + markerDY*zoom})
		if !m.Crash {
			continue
		}
		restarts := restartsOf(sc, m.Pod)
		out = append(out, incidentSprite{name: art.BackoffRingName(BackoffStep(restarts)), sx: sx, sy: sy + ringDY*zoom, alpha: ringAlpha, scale: ringScale})
		if restarts > 0 {
			out = append(out, incidentSprite{name: art.RestartBadgeName(int(restarts)), sx: sx + badgeDX*zoom, sy: sy + badgeDY*zoom})
		}
	}
	return out
}

func (r *Renderer) drawIncidents(dst *ebiten.Image, cam render.Camera, sc Scene) {
	if r.Lib == nil {
		return
	}
	zoom := cam.Scale()
	for _, s := range r.incidentSprites(cam, sc) {
		r.Lib.Draw(dst, s.name, s.sx, s.sy, zoom, render.SpriteOptions{Alpha: s.alpha, Scale: s.scale})
	}
}
