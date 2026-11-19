package world

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	TurnedMax    = 5
	turnedHop    = 1.4
	crossLift    = 12.0
	labelRise    = 46.0
	ringHz       = 1.5
	QuayFullText = "QUAY FULL: ships turn boxes away"
)

func TurnedAt(age float64, k, n int) float64 {
	return wrap(age/turnedHop+float64(k)/float64(max(n, 1)), 1)
}

func (r *Renderer) TurnedItems(out []Drawable, cam render.Camera, sc Scene, haven int) []Drawable {
	h := r.HarborOf()
	p := h.Plan(haven)
	if haven >= len(h.Fleets) || len(p.Berths) == 0 {
		return out
	}
	if !h.Full {
		return out
	}
	for _, s := range h.Fleets[haven].Ships {
		if !s.Counted || h.Now < s.Dock || h.Now >= s.Leave() {
			continue
		}
		if turned := s.Cargo - s.Accepted; turned >= 1 {
			out = r.turnedBoxes(out, cam, sc, p, s, min(turned, TurnedMax))
		}
	}
	return out
}

func (r *Renderer) turnedBoxes(out []Drawable, cam render.Camera, sc Scene, p HavenPlan, s Ship, n int) []Drawable {
	h := r.HarborOf()
	zoom := cam.Scale()
	plank := p.Berths[s.Berth%len(p.Berths)].Plank
	age := h.Now - s.Dock
	for k := 0; k < n; k++ {
		pos := Lerp(p.Quay, plank, TurnedAt(age, k, n))
		sx, sy := ScreenPointOf(cam, sc.Site, pos)
		if !onScreen(sx, sy, zoom) {
			continue
		}
		sy -= h.Lift(pos) * zoom
		look := gfx.BoxLook{Kind: KindName(0), Variant: k, Fresh: 1}
		names := h.layers(look)
		out = append(out, Drawable{Depth: render.DepthKey(pos.X, pos.Y) + fxAbove, Draw: func(dst *ebiten.Image) {
			r.drawBox(dst, names, sx, sy, zoom, look, [3]float32{})
			r.props().draw(dst, "cross", sx, sy-crossLift*zoom, zoom, render.SpriteOptions{})
		}})
	}
	return out
}

func (r *Renderer) drawQuayOverlay(dst *ebiten.Image, cam render.Camera, sc Scene) {
	h := r.HarborOf()
	for i, p := range h.Plans {
		if !p.Open {
			continue
		}
		if quayFullShown(sc, h.Full) {
			r.drawQuayFull(dst, cam, sc, p)
		}
		if !strings.HasPrefix(sc.Hover, PileIDHead) {
			continue
		}
		for _, y := range h.Yards {
			if y.Haven == i && PileID(y.Kind) == sc.Hover {
				drawYardRing(dst, cam, sc, p, y.Kind)
			}
		}
	}
}

func quayFullShown(sc Scene, full bool) bool { return full && !sc.Chapter }

func (r *Renderer) drawQuayFull(dst *ebiten.Image, cam render.Camera, sc Scene, p HavenPlan) {
	zoom := cam.Scale()
	at := QuayPoint(p, YardRight, yardShore+YardDeep)
	sx, sy := ScreenPointOf(cam, sc.Site, at)
	if !onScreen(sx, sy, zoom) {
		return
	}
	box, clear := r.settleLabel(labelBox(QuayFullText, sx, sy-labelRise*zoom), int(sx), sc.Panels)
	if !clear {
		return
	}
	ui.Fill(dst, box.Offset(1, 1), ui.Palette.Shadow)
	ui.Fill(dst, box, ui.Tint(ui.Palette.ConsoleEdge, 235))
	ui.Stroke(dst, box, ui.Palette.RedText)
	var font ui.Font
	font.Draw(dst, QuayFullText, box.X+4, box.Y+3, 1, ui.Palette.RedText)
}

func YardCorners(kind int) [4][2]float64 {
	a0, d0 := YardOrigin(kind)
	a1, d1 := a0+YardWide, d0+YardDeep
	return [4][2]float64{{a0, d0}, {a1, d0}, {a1, d1}, {a0, d1}}
}

func drawYardRing(dst *ebiten.Image, cam render.Camera, sc Scene, p HavenPlan, kind int) {
	zoom := cam.Scale()
	pulse := 1.0
	if !sc.Reduce {
		pulse = .6 + .4*math.Sin(sc.RealTime()*2*math.Pi*ringHz)
	}
	c := ui.Tint(ui.Palette.GoldBright, uint8(255*pulse))
	pts := [4][2]float32{}
	for i, k := range YardCorners(kind) {
		v := QuayPoint(p, k[0], k[1])
		x, y := ScreenPointOf(cam, sc.Site, v)
		pts[i] = [2]float32{float32(x), float32(y - art.HavenDeck*zoom)}
	}
	w := float32(max(zoom, 1))
	for i := range pts {
		a, b := pts[i], pts[(i+1)%len(pts)]
		vector.StrokeLine(dst, a[0], a[1], b[0], b[1], w, c, false)
	}
}
