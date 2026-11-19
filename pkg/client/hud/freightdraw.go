package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	boardX      = Width - BoardW - railRight - railSize - cardMargin
	BoardW      = 232
	BoardPad    = 5
	boardRows   = 4
	chipRowH    = 16
	chipW       = 56
	chipIconX   = 7
	chipIconY   = 12
	chipTextX   = 16
	boardGap    = 3
	boardBarGap = 6
	BoardMaxH   = 2*BoardPad + (boardRows+4+FlowbarLossMax)*ui.LineHeight + chipRowH + portsRowH
	FreightTip  = "freight:"
)

var freshColors = []color.RGBA{ui.Palette.Green, ui.Palette.GoldBright, ui.Palette.Gold, ui.Palette.RedText}

func BoardShown(f Frame) bool {
	return f.Revealed("freight") && (f.Shown(SurfaceFreight) || f.Teaching(SurfaceLosses)) && len(f.State.Freight.Kinds) > 0
}

func FreightRect(f Frame) ui.Rect {
	h := 2*BoardPad + boardBodyRows(f)*ui.LineHeight
	if !f.BoardCompact && hasChips(f.State.Freight) {
		h += chipRowH
	}
	if PortsRowShown(f) {
		h += portsRowH
	}
	if boardFoldShown(f) {
		h += ui.LineHeight
	}
	return f.placed(PanelHarbor, ui.Rect{X: boardX, Y: TopH + 4, W: BoardW, H: h})
}

func hasChips(fv view.FreightView) bool {
	for _, k := range fv.Kinds {
		if k.Open || k.Waiting >= .5 {
			return true
		}
	}
	return false
}

func LossesShown(f Frame) bool {
	return BoardShown(f) && !f.BoardCompact && f.Revealed("losses") && f.Shown(SurfaceLosses) && (LossTotal(f.State.Freight) >= minLoss || f.Teaching(SurfaceLosses))
}

func FreightRects(f Frame) []ui.Rect {
	if BoardShown(f) {
		return []ui.Rect{FreightRect(f)}
	}
	return []ui.Rect{}
}

func (h *HUD) FreightButtons(f Frame) []Button {
	if !BoardShown(f) || h.OverlayID != OverlayNone {
		return nil
	}
	r := FreightRect(f)
	fv := f.State.Freight
	row := func(i int) ui.Rect { return BoardRowRect(r, i) }
	out := []Button{{ID: FreightTip + "tier", Rect: row(0), Tip: TierTip(fv, PortsRowShown(f))}, {ID: FreightTip + "rate", Rect: row(1), Tip: RateTip(EconomyOf(f.State))}}
	if !f.BoardCompact {
		out = append(out, Button{ID: FlowbarMoney, Rect: row(2), Tip: MoneyTip()}, h.goalButton(f, row(3)))
		out = append(out, h.boardExtraButtons(f, row)...)
	}
	out = append(out, h.FlowbarButtons(f)...)
	if !f.BoardCompact {
		chipY := r.Y + BoardPad + boardBodyRows(f)*ui.LineHeight
		for i, c := range FreightHeadOf(fv, BoxesSpoil(f.State), f.State.Crews...).Chips {
			out = append(out, Button{ID: FreightTip + "kind:" + c.Kind, Rect: ui.Rect{X: r.X + BoardPad + i*chipW, Y: chipY, W: chipW, H: chipRowH}, Tip: c.Tip})
		}
		out = append(out, h.portChips(f, r)...)
	}
	return append(out, h.boardFoldButtons(f, r)...)
}

func TierTip(fv view.FreightView, portsRow bool) string {
	lines := []string{fmt.Sprintf("HARBOR TIER %d, your deepest port's level: %s dock there.", max(fv.Tier, 1), view.HarborTierOf(max(fv.Tier, 1)).Ships)}
	if fv.NextTier > 0 {
		lines = append(lines, fmt.Sprintf("Next upgrade: %s takes %ss, bringing %s, for %s gold.", fv.NextPortName, fv.NextShip, fv.NextOpens, Compact(fv.NextTier)))
	}
	if fv.NextWhy != "" {
		lines = append(lines, fmt.Sprintf("Tier %d waits: %s.", max(fv.Tier, 1)+1, fv.NextWhy))
	}
	way := "click the Gate Harbor, then UPGRADE its port."
	if portsRow {
		way = "click a port on the PORTS row to go there, then upgrade it."
	}
	return strings.Join(append(lines, "Each port's level sets the ships it takes and multiplies its boxes by 4: "+way), "\n")
}

func RateTip(e view.EconomyView) string {
	lines := []string{ui.Bridge("The harbor's flow: IN counts the boxes arriving each second; CARRIED and LOST count every box carried or lost each second, the ones waiting on the quay included.",
		"The flow: IN counts the requests arriving each second; SERVED and DROPPED count every request served or dropped each second, the queued ones included."),
		"The bar fills green with what is carried and red with what is lost.", "Fresher boxes pay up to a quarter more; spoiled ones pay nothing."}
	if lostAboveIn(e) {
		lines = append(lines, fmt.Sprintf(ui.Bridge("Now LOST %s/S runs above IN %s/S: boxes that waited on the quay spoil as well as new ones.",
			"Now DROPPED %s/S runs above IN %s/S: requests that waited in the queue time out as well as new ones."), Rate(e.Lost), Rate(e.Arriving)))
	}
	return tipLines(lines...)
}

func lostAboveIn(e view.EconomyView) bool {
	return e.Lost >= minLoss && e.Lost > e.Arriving
}

func LostNote(e view.EconomyView) string {
	if !lostAboveIn(e) {
		return ""
	}
	return ui.Bridge("LOST INCLUDES THE QUAY'S PILE", "DROPPED INCLUDES THE QUEUE")
}

func MoneyTip() string {
	return tipLines("Gold each second: income from boxes delivered, and the upkeep your Keeps cost.",
		"Every ration and room a worker reserves takes Keep land, and Keeps cost upkeep.")
}

func LossTip() string {
	return tipLines("Boxes lost each second, by cause, with the fix for the worst of each.",
		"Spoiled: they waited too long. Quay full: ships turned them away.", "No depot or no crew: nobody may carry that kind.")
}

func (h *HUD) pressFreight(rest string, f Frame) []Intent {
	switch rest {
	case "next":
		return h.PressNext(f)
	case "fix":
		return h.PressFix(f)
	case "demand":
		return h.pressDemand(f)
	}
	if strings.HasPrefix(rest, "port:") {
		return h.pressPortChip(rest)
	}
	if rest == "more" {
		return h.toggleBoard()
	}
	return h.toggleLedger()
}

func (h *HUD) drawFreight(dst *ebiten.Image, f Frame) {
	if BoardShown(f) && h.CardsDrawn(f) {
		h.withAppear(dst, SurfaceFreight, f, func(d *ebiten.Image) { h.DrawBoard(d, f) })
	}
}

func (h *HUD) DrawBoard(dst *ebiten.Image, f Frame) {
	r := FreightRect(f)
	fv := f.State.Freight
	head := FreightHeadOf(fv, BoxesSpoil(f.State))
	if !h.Lib.NineSlice(dst, "panel-dark", r) {
		ui.DarkPanel(dst, r)
	}
	flash := Flashing(fv, f.State.RealmTime)
	if flash {
		ui.Stroke(dst, r.Inset(-1), ui.Tint(ui.Palette.GoldBright, uint8(255*pulse(h.Now*2, h.Reduce))))
	}
	x, y, width := r.X+BoardPad, r.Y+BoardPad, r.W-2*BoardPad
	h.drawTierRow(dst, r, head, flash, x, y)
	h.drawFlowRow(dst, f, x, y+ui.LineHeight, width)
	if !f.BoardCompact {
		h.drawMoneyRow(dst, f, x, y+2*ui.LineHeight, width)
		if !h.drawDemandRow(dst, f, x, y+3*ui.LineHeight, width) {
			drawClipped(dst, head.Goal, x, y+3*ui.LineHeight, width, ui.Palette.TextDim)
		}
		h.drawBoardExtra(dst, f, r, x, y+boardRows*ui.LineHeight)
	}
	h.drawFlowbarRows(dst, f, r)
	if !f.BoardCompact {
		for i, c := range head.Chips {
			h.drawKindChip(dst, c, x+i*chipW, y+boardBodyRows(f)*ui.LineHeight)
		}
		h.drawPortsRow(dst, f, r)
	}
	h.drawBoardFold(dst, f, r)
}

func (h *HUD) drawTierRow(dst *ebiten.Image, r ui.Rect, head FreightHead, flash bool, x, y int) {
	label := fmt.Sprintf("TIER %d", head.Tier)
	clr := ui.Palette.Gold
	if flash {
		clr = ui.Palette.GoldBright
	}
	h.Font.DrawShadow(dst, label, x, y, 1, clr)
	lw, _ := h.Font.Measure(label, 1)
	cw, _ := h.Font.Measure(head.Count, 1)
	bar := ui.Rect{X: x + lw + boardBarGap, Y: y + 1, W: r.W - 2*BoardPad - lw - cw - 2*boardBarGap, H: 6}
	if bar.W > 0 {
		h.drawMiniBar(dst, bar, head.Progress, ui.Palette.Green)
	}
	h.Font.Draw(dst, head.Count, r.X+r.W-BoardPad-cw, y, 1, ui.Palette.Cream)
}

func (h *HUD) drawKindChip(dst *ebiten.Image, c KindChip, x, y int) {
	look := gfx.BoxLook{Kind: c.Kind, Fresh: c.Fresh}
	for _, n := range gfx.BoxLayers(look) {
		h.Lib.Draw(dst, n, float64(x+chipIconX), float64(y+chipIconY), 1, render.SpriteOptions{})
	}
	clr := freshColors[min(max(c.Level, 0), len(freshColors)-1)]
	switch {
	case c.Lost > 0:
		clr = ui.Palette.RedText
	case c.Waiting == 0:
		clr = ui.Palette.TextDim
	}
	h.Font.Draw(dst, KindChipLabel(c), x+chipTextX, y+4, 1, clr)
}

func drawClipped(dst *ebiten.Image, text string, x, y, width int, clr color.Color) {
	ui.Font{Literal: true}.Draw(dst, ui.Clip(text, ui.MaxChars(width, 1)), x, y, 1, clr)
}

func containsAny(rects []ui.Rect, x, y int) bool {
	for _, r := range rects {
		if r.Contains(x, y) {
			return true
		}
	}
	return false
}
