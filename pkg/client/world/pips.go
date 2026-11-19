package world

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	pipRow  = 8
	pipGap  = 2
	pipLift = 6
	pipPad  = 3
	pipMin  = 5
	pipMax  = 11
)

type Pip struct {
	Cap     float64
	Pledged float64
	Used    float64
}

func HandPips(n view.NodeMetrics) []Pip {
	alloc := float64(n.CPUAllocatableMilli) / 1000
	if alloc <= 0 {
		return nil
	}
	req, used := float64(n.CPURequestedMilli)/1000, n.CPUMilli/1000
	out := make([]Pip, int(math.Ceil(alloc-1e-9)))
	for i := range out {
		lo := float64(i)
		c := min(alloc-lo, 1)
		out[i] = Pip{Cap: c, Pledged: min(max(req-lo, 0), c), Used: min(max(used-lo, 0), c)}
	}
	return out
}

func PipSize(zoom float64) int {
	return min(max(int(6*zoom)+1, pipMin), pipMax)
}

func PipsShown(sc Scene) bool {
	_, api := FirstOfKind(sc.Site, view.StructChancery, true)
	return api && len(sc.Metrics.Nodes) > 0
}

func (r *Renderer) DrawKeepPips(dst *ebiten.Image, cam render.Camera, sc Scene) {
	if !PipsShown(sc) {
		return
	}
	for _, it := range r.placeStructures(cam, sc) {
		if it.S.Kind != view.StructKeep || it.S.State != view.StateReady {
			continue
		}
		m, ok := sc.Metrics.Nodes[it.S.Object.Name]
		if !sc.Metrics.Augury {
			m.CPUMilli = 0
		}
		pips := HandPips(m)
		if !ok || len(pips) == 0 || !onScreen(it.sx, it.sy, cam.Scale()) {
			continue
		}
		top := it.sy - 60*cam.Scale()
		if head, ok := r.Lib.Headroom(it.names[0]); ok {
			top = it.sy - float64(head)*cam.Scale()
		}
		drawPips(dst, pips, int(it.sx), int(top)-pipLift, PipSize(cam.Scale()))
	}
}

func drawPips(dst *ebiten.Image, pips []Pip, cx, bottom, size int) {
	rows := (len(pips) + pipRow - 1) / pipRow
	wide := min(len(pips), pipRow)*(size+pipGap) - pipGap
	plaque := ui.Rect{X: cx - wide/2 - pipPad, Y: bottom - rows*(size+pipGap) - pipPad, W: wide + 2*pipPad, H: rows*(size+pipGap) - pipGap + 2*pipPad}
	ui.Fill(dst, plaque.Offset(1, 1), ui.Palette.Shadow)
	ui.Fill(dst, plaque, ui.Tint(ui.Palette.ConsoleEdge, 220))
	ui.Stroke(dst, plaque, ui.Palette.Gold)
	for i, p := range pips {
		row, col := i/pipRow, i%pipRow
		inRow := min(len(pips)-row*pipRow, pipRow)
		width := inRow*(size+pipGap) - pipGap
		x := cx - width/2 + col*(size+pipGap)
		y := bottom - (rows-row)*(size+pipGap)
		drawPip(dst, ui.Rect{X: x, Y: y, W: size, H: size}, p)
	}
}

var (
	pipEmpty   = color.RGBA{36, 44, 46, 255}
	pipPledged = ui.Palette.Gold
	pipUsed    = ui.Palette.Cream
)

func drawPip(dst *ebiten.Image, r ui.Rect, p Pip) {
	ui.Fill(dst, r.Inset(-1), ui.Palette.Ink)
	inner := r
	inner.W = max(int(math.Round(float64(r.W)*p.Cap)), 1)
	ui.Fill(dst, inner, pipEmpty)
	if w := int(math.Round(float64(r.W) * p.Pledged)); w > 0 {
		ui.Fill(dst, ui.Rect{X: r.X, Y: r.Y, W: w, H: r.H}, pipPledged)
	}
	if w := int(math.Round(float64(r.W) * p.Used)); w > 0 {
		h := max(r.H/2, 1)
		ui.Fill(dst, ui.Rect{X: r.X, Y: r.Y + r.H - h, W: w, H: h}, pipUsed)
	}
}
