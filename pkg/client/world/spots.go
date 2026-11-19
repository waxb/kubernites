package world

import (
	"cmp"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	previewAlpha = .6
	spotWidth    = 2
	spotLabelGap = 3
	spotBestPad  = .15
	platePad     = 4
	plateTop     = 3
	plateH       = 12
	plateAlpha   = 220
)

type SpotMark struct {
	X, Y, W, H int
	Class      string
	Machine    string
	Note       string
}

func RingOf(tiles []view.Point) []view.Point {
	in := make(map[view.Point]bool, len(tiles))
	for _, p := range tiles {
		in[p] = true
	}
	out := []view.Point{}
	for _, p := range tiles {
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			if !in[view.Point{X: p.X + d[0], Y: p.Y + d[1]}] {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

func PreviewMarks(site view.Site, tiles []view.Point, center view.Point) []BorderMark {
	if len(tiles) == 0 {
		return nil
	}
	ring := view.Territory{Center: center, Border: RingOf(tiles)}
	return BorderMarks([]view.Territory{ring}, func(x, y int) bool { return FreeTile(site, x, y) })
}

func (r *Renderer) markItems(cam render.Camera, sc Scene, marks []BorderMark, alpha float32) []Drawable {
	zoom := cam.Scale()
	out := make([]Drawable, 0, len(marks))
	for _, m := range marks {
		sx, sy := cam.GridToScreenElevated(float64(m.X), float64(m.Y), float64(sc.Site.ElevationAt(m.X, m.Y)))
		if !onScreen(sx, sy, zoom) || sc.Site.RoadAt(m.X, m.Y) >= roadHides {
			continue
		}
		name := MarkName(m, sc.Time, sc.Reduce)
		out = append(out, Drawable{Depth: render.DepthKey(float64(m.X), float64(m.Y)) - markBias, Draw: func(dst *ebiten.Image) {
			r.Lib.Draw(dst, name, sx, sy, zoom, render.SpriteOptions{Alpha: alpha})
		}})
	}
	return out
}

func (r *Renderer) DrawSpots(dst *ebiten.Image, cam render.Camera, sc Scene) {
	zoom := cam.Scale()
	for _, sp := range sc.Spots {
		lo, hi := Point{sp.X, sp.Y}, Point{sp.X + max(sp.W, 1) - 1, sp.Y + max(sp.H, 1) - 1}
		elev := float64(sc.Site.ElevationAt(hi.X, hi.Y))
		clr := model.ClassColor(sp.Class)
		strokeDiamond(dst, cam, float64(lo.X), float64(lo.Y), float64(hi.X), float64(hi.Y), elev, clr, spotWidth)
		if zoom < 1 || sp.Note != "" {
			continue
		}
		sx, sy := cam.GridToScreenElevated(float64(hi.X)+.5, float64(hi.Y)+.5, elev)
		r.drawPlate(dst, []string{SpotLabel(sp)}, int(sx), int(sy)+spotLabelGap, clr, sc.Panels)
	}
}

func plateBox(lines []string, cx, y int) ui.Rect {
	var font ui.Font
	w := 0
	for _, l := range lines {
		lw, _ := font.Measure(l, 1)
		w = max(w, lw)
	}
	return ui.Rect{X: cx - w/2 - platePad, Y: y, W: w + 2*platePad, H: plateH + (len(lines)-1)*ui.LineHeight}
}

func SpotPlate(lines []string, cx, y int, panels []ui.Rect) (ui.Rect, bool) {
	return SettleLabel(plateBox(lines, cx, y), cx, panels)
}

func (r *Renderer) drawPlate(dst *ebiten.Image, lines []string, cx, y int, clr color.RGBA, panels []ui.Rect) {
	box, ok := r.settleLabel(plateBox(lines, cx, y), cx, panels)
	if !ok {
		return
	}
	var font ui.Font
	ui.Fill(dst, box, ui.Tint(ui.Palette.ConsoleEdge, plateAlpha))
	ui.Stroke(dst, box, clr)
	for i, l := range lines {
		lw, _ := font.Measure(l, 1)
		font.Draw(dst, l, box.X+(box.W-lw)/2, box.Y+plateTop+i*ui.LineHeight, 1, clr)
	}
}

func SpotLabel(sp SpotMark) string {
	return cmp.Or(sp.Machine, sp.Class)
}

func BestLines(sp SpotMark, zoom float64) []string {
	switch {
	case sp.Machine != "" && zoom >= 1:
		return []string{sp.Note, sp.Machine}
	case sp.Class != "":
		return []string{sp.Class + ": " + sp.Note}
	}
	return []string{sp.Note}
}

func (r *Renderer) drawBestSpot(dst *ebiten.Image, cam render.Camera, sc Scene) {
	for _, sp := range sc.Spots {
		if sp.Note == "" {
			continue
		}
		lo, hi := Point{sp.X, sp.Y}, Point{sp.X + max(sp.W, 1) - 1, sp.Y + max(sp.H, 1) - 1}
		elev := float64(sc.Site.ElevationAt(hi.X, hi.Y))
		outlineDiamond(dst, cam, float64(lo.X)-spotBestPad, float64(lo.Y)-spotBestPad, float64(hi.X)+spotBestPad, float64(hi.Y)+spotBestPad, elev, ui.Palette.GoldBright)
		sx, sy := cam.GridToScreenElevated(float64(hi.X)+.5, float64(hi.Y)+.5, elev)
		r.drawPlate(dst, BestLines(sp, cam.Scale()), int(sx), int(sy)+spotLabelGap, ui.Palette.GoldBright, sc.Panels)
	}
}
