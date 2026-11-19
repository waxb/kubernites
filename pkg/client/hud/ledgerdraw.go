package hud

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	LedgerNodeW = 190
	ledgerNodeH = 50
	ledgerColX  = 254
	ledgerRowY  = 82
	ledgerTop   = 34
	ledgerLeft  = 12
	arrowHead   = 5
)

func LedgerNodeRect(n LedgerNode) ui.Rect {
	return ui.Rect{X: Overlay.X + ledgerLeft + n.Col*ledgerColX, Y: Overlay.Y + ledgerTop + n.Row*ledgerRowY, W: LedgerNodeW, H: ledgerNodeH}
}

func (h *HUD) toggleLedger() []Intent {
	h.toggleOverlay(OverlayLedger)
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) ledgerButtons(Frame) []Button {
	return []Button{closeButton(Overlay)}
}

const ledgerTopW = 16

func ledgerTopButton(f Frame, x int) (Button, bool) {
	if !f.Shown(SurfaceLedger) && !f.Shown(MeterGold) {
		return Button{}, false
	}
	return Button{ID: LedgerButton, Rect: ui.Rect{X: x, Y: 2, W: ledgerTopW, H: 14}, Label: "L", Tip: "The Ledger: how the realm works, what feeds what [F1 or L]"}, true
}

func (h *HUD) drawLedger(dst *ebiten.Image, f Frame) {
	ui.DarkPanel(dst, Overlay)
	h.header(dst, Overlay, ledgerTitleText)
	nodes := LedgerNodes(f)
	rects := map[string]ui.Rect{}
	for _, n := range nodes {
		rects[n.ID] = LedgerNodeRect(n)
	}
	for _, e := range LedgerEdges(nodes) {
		h.drawEdge(dst, rects[e.From], rects[e.To], e.Label)
	}
	for _, n := range nodes {
		h.drawLedgerNode(dst, rects[n.ID], n)
	}
	h.DrawOverlayButtons(dst, h.ledgerButtons(f))
}

func (h *HUD) drawLedgerNode(dst *ebiten.Image, r ui.Rect, n LedgerNode) {
	ui.Fill(dst, r.Offset(1, 2), ui.Palette.Shadow)
	ui.Fill(dst, r, consoleInk)
	ui.Stroke(dst, r, ui.Palette.Gold)
	h.Font.Draw(dst, n.Title, r.X+6, r.Y+5, 1, ui.Palette.GoldBright)
	for i, l := range n.Lines {
		drawClipped(dst, l, r.X+6, r.Y+18+i*ui.LineHeight, r.W-12, ui.Palette.Cream)
	}
}

func EdgePoints(from, to ui.Rect) (float32, float32, float32, float32) {
	fx, fy := centre(from)
	tx, ty := centre(to)
	switch {
	case from.Y == to.Y && from.X < to.X:
		return float32(from.X + from.W), fy, float32(to.X), ty
	case from.Y == to.Y:
		return float32(from.X), fy, float32(to.X + to.W), ty
	case from.X == to.X && from.Y < to.Y:
		return fx, float32(from.Y + from.H), tx, float32(to.Y)
	case from.X == to.X:
		return fx, float32(from.Y), tx, float32(to.Y + to.H)
	}
	return edgeCorner(from, to)
}

func edgeCorner(from, to ui.Rect) (float32, float32, float32, float32) {
	fx, fy := float32(from.X), float32(from.Y+from.H)
	tx, ty := float32(to.X+to.W), float32(to.Y)
	if from.Y > to.Y {
		fy, ty = float32(from.Y), float32(to.Y+to.H)
	}
	if from.X < to.X {
		fx, tx = float32(from.X+from.W), float32(to.X)
	}
	return fx, fy, tx, ty
}

func centre(r ui.Rect) (float32, float32) {
	return float32(r.X) + float32(r.W)/2, float32(r.Y) + float32(r.H)/2
}

func (h *HUD) drawEdge(dst *ebiten.Image, from, to ui.Rect, label string) {
	x0, y0, x1, y1 := EdgePoints(from, to)
	clr := ui.Palette.ParchmentDark
	vector.StrokeLine(dst, x0, y0, x1, y1, 1, clr, false)
	a := math.Atan2(float64(y1-y0), float64(x1-x0))
	var p vector.Path
	p.MoveTo(x1, y1)
	p.LineTo(x1-arrowHead*float32(math.Cos(a-.5)), y1-arrowHead*float32(math.Sin(a-.5)))
	p.LineTo(x1-arrowHead*float32(math.Cos(a+.5)), y1-arrowHead*float32(math.Sin(a+.5)))
	p.Close()
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, &p, &vector.FillOptions{}, op)
	w, _ := h.Font.Measure(label, 1)
	mx, my := int((x0+x1)/2), int((y0+y1)/2)
	box := ui.Rect{X: mx - w/2 - 2, Y: my - 5, W: w + 4, H: 10}
	ui.Fill(dst, box, ui.Palette.Console)
	h.Font.Draw(dst, label, box.X+2, box.Y+2, 1, ui.Palette.TextDim)
}
