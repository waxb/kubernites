package world

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	lampsPerKeep  = 2
	lampsPerGuild = 2
	lampDusk      = .1
	lampOut       = 1.1
	lampIn        = .6
	lampHead      = 19.0
	lampGlowR     = 17.0
	lampGlowMax   = .95
	quayLampSide  = DeckHalf + .6
	quayLampIn    = .3
)

var lampNames = [2]string{gfx.AmbientName("lamp", 0), gfx.AmbientName("lamp", 1)}

type lampSpot struct {
	at  Vec
	lit bool
}

func LampsLit(phase float64) bool {
	return Darkness(phase) > lampDusk
}

func flankingLamps(s view.Structure, lit bool) []lampSpot {
	if f, ok := stallFrameOf(s); ok {
		return []lampSpot{{at: f.lamp(), lit: lit}}
	}
	_, hi := Footprint(s)
	return []lampSpot{
		{at: Vec{float64(hi.X) + lampOut, float64(hi.Y) - lampIn}, lit: lit},
		{at: Vec{float64(hi.X) - lampIn, float64(hi.Y) + lampOut}, lit: lit},
	}
}

func LampSpots(site view.Site) []lampSpot {
	out := []lampSpot{}
	for _, k := range Keeps(site) {
		out = append(out, flankingLamps(k, k.State == view.StateReady)...)
	}
	if g, ok := FirstOfKind(site, view.StructGuildhall, false); ok {
		out = append(out, flankingLamps(g, g.State == view.StateReady)...)
	}
	for _, h := range SiteHavens(site) {
		if !h.Open {
			continue
		}
		q, side := VecOf(h.Quay), VecOf(h.Along())
		for _, s := range []float64{-quayLampSide, quayLampSide} {
			if at := offset(offset(q, side, s), VecOf(h.Out), -quayLampIn); !WetAt(site, at) {
				out = append(out, lampSpot{at: at, lit: true})
			}
		}
	}
	return out
}

func (r *Renderer) lampItems(out []Drawable, cam render.Camera, sc Scene) []Drawable {
	zoom := cam.Scale()
	night := LampsLit(sc.DayPhase)
	lamps := append(LampSpots(sc.Site), r.berthLamps()...)
	r.lifeOf().lamps = lamps
	for _, l := range lamps {
		sx, sy := ScreenPointOf(cam, sc.Site, l.at)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		sy -= r.HarborOf().Lift(l.at) * zoom
		name := lampNames[0]
		if l.lit && night {
			name = lampNames[1]
		}
		out = append(out, Drawable{Depth: render.DepthKey(l.at.X, l.at.Y), Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{})
		}})
	}
	return out
}

func (r *Renderer) drawLampGlow(dst *ebiten.Image, cam render.Camera, sc Scene) {
	dark := Darkness(sc.DayPhase)
	if dark <= lampDusk {
		return
	}
	strength := min(dark/.55, 1) * lampGlowMax
	zoom := cam.Scale()
	for _, l := range r.lifeOf().lamps {
		if !l.lit {
			continue
		}
		sx, sy := ScreenPointOf(cam, sc.Site, l.at)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		sy -= r.HarborOf().Lift(l.at) * zoom
		r.drawGlow(dst, sx, sy-lampHead*zoom, lampGlowR*zoom, strength)
	}
	r.drawHarborGlow(dst, cam, sc, strength)
}
