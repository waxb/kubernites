package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	hourglassLift = 30.0
	knockDX       = 10.0
	knockLift     = 24.0
	bellHz        = 4.0
)

type watchIndex struct {
	huts  map[string]view.HutWatchView
	crews map[string]view.CrewWatchView
}

func indexWatch(w *view.WatchView) watchIndex {
	out := watchIndex{huts: map[string]view.HutWatchView{}, crews: map[string]view.CrewWatchView{}}
	if w == nil {
		return out
	}
	for _, c := range w.Crews {
		out.crews[c.Crew] = c
	}
	for _, h := range w.Huts {
		out.huts[h.Pod] = h
	}
	return out
}

func watchLook(qos string) string {
	if look := model.QoSLook(qos); look != "" {
		return look
	}
	return art.HearthQoS[0]
}

func (r *Renderer) watchSprites(p model.PodView, w watchIndex, age int, sx, sy, zoom, t float64, reduce bool) []incidentSprite {
	hut, ok := w.huts[p.Name]
	if !ok {
		return nil
	}
	crew, look := w.crews[hut.Crew], watchLook(p.QoS)
	age = min(max(age, 0), art.AgeTiers-1)
	out := []incidentSprite{}
	if crew.Readiness != nil {
		out = append(out, incidentSprite{name: art.ShutterName(look, age, !hut.Shut), sx: sx, sy: sy})
	}
	if crew.Liveness != nil {
		frame := 0
		if hut.Knocks > 0 && !reduce {
			frame = int(t * bellHz)
		}
		out = append(out, incidentSprite{name: art.BellName(look, age, frame), sx: sx, sy: sy})
	}
	if hut.Warming {
		out = append(out, incidentSprite{name: art.HourglassName(IncidentFrame(t, reduce)), sx: sx, sy: sy - hourglassLift*zoom})
	}
	switch {
	case hut.Knocks > 0:
		out = append(out, incidentSprite{name: art.KnockName(hut.Knocks), sx: sx + knockDX*zoom, sy: sy - knockLift*zoom})
	case hut.Frozen && crew.Liveness == nil:
		out = append(out, incidentSprite{name: art.FrostName(), sx: sx + knockDX*zoom, sy: sy - knockLift*zoom})
	}
	return out
}

func (r *Renderer) drawWatch(dst *ebiten.Image, list []incidentSprite, zoom float64, opts render.SpriteOptions) {
	if r.Lib == nil {
		return
	}
	for _, s := range list {
		r.Lib.Draw(dst, s.name, s.sx, s.sy, zoom, opts)
	}
}
