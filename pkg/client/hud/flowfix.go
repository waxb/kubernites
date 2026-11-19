package hud

import (
	"cmp"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	neckLabel   = "BOTTLENECK:"
	NeckNone    = "NONE"
	NoNeckLine  = "Nothing holds it back now."
	fixTextGap  = 4
	NoLossesRow = "No boxes lost: every one delivered"
)

type fixPrice struct {
	text   string
	reason string
	closed bool
}

func priceOf(f Frame, o view.CommandOffer) fixPrice {
	if o.Action.Action != "buyKeep" {
		return fixPrice{}
	}
	if k, ok := buyOffer(f, o.Action); ok {
		return keepPrice(k.KeepOffer)
	}
	kind := cmp.Or(o.Action.KeepType, view.KeepTypeStone)
	for _, k := range f.State.KeepShop {
		if k.KeepType == kind {
			return keepPrice(k)
		}
	}
	return fixPrice{}
}

func keepPrice(k view.KeepOffer) fixPrice {
	return fixPrice{text: " " + Gold(k.Cost), reason: k.Reason, closed: !k.Available}
}

func neckWidth(e view.EconomyView) int {
	name := NeckName(e.Bottleneck)
	if name == "" {
		return 0
	}
	return NamedFont.Width(neckLabel, 1) + ui.Advance + NamedFont.Width(name, 1) + fixTextGap
}

func BoardRowRect(r ui.Rect, i int) ui.Rect {
	return ui.Rect{X: r.X, Y: r.Y + BoardPad + i*ui.LineHeight - 1, W: r.W, H: ui.LineHeight}
}

func FixButton(f Frame, row ui.Rect) (Button, bool) {
	e := EconomyOf(f.State)
	if e.Fix == nil || FixFade(f, *e.Fix) == view.FixNames {
		return Button{}, false
	}
	if FixFade(f, *e.Fix) == view.FixShows {
		return fixButtonAt(e, row, ShowLabel(*e.Fix, fixRoom(e, row)), ShowTip(e.Fix.Action)), true
	}
	price := priceOf(f, *e.Fix)
	label := ShortFixLabel(*e.Fix) + price.text
	if full := FixLabel(*e.Fix) + price.text; NamedFont.Width(full, 1)+FlowbarFixPad <= fixRoom(e, row) {
		label = full
	}
	b := fixButtonAt(e, row, label, fixTip(*e.Fix, buyPayoff(f, &e.Fix.Action)+scaleCharter(f, e.Fix.Action)))
	b.Disabled = price.closed
	if price.closed && price.reason != "" {
		b.Tip = price.reason
	}
	return b, true
}

func fixRoom(e view.EconomyView, row ui.Rect) int {
	return row.W - 2*BoardPad - neckWidth(e)
}

func fixButtonAt(e view.EconomyView, row ui.Rect, label, tip string) Button {
	room := fixRoom(e, row)
	w := min(max(NamedFont.Width(label, 1)+FlowbarFixPad, flowbarFixMinW), max(room, flowbarFixMinW))
	rect := ui.Rect{X: row.X + row.W - BoardPad - w, Y: row.Y, W: w, H: row.H}
	return Button{ID: FlowbarFix, Rect: rect, Label: wholeWords(label, ui.MaxChars(w-FlowbarFixPad, 1)), Tip: tip}
}

func FixTip(o view.CommandOffer) string {
	return fixTip(o, "")
}

func fixTip(o view.CommandOffer, payoff string) string {
	lines := []string{o.Label}
	if payoff != "" {
		lines = append(lines, payoff)
	}
	if o.Command != "" {
		lines = append(lines, "$ "+o.Command)
	}
	if OpensPlacement(&o.Action) {
		return strings.Join(append(lines, placeNote), "\n")
	}
	return strings.Join(append(lines, ActsNote), "\n")
}

func neckTipOf(f Frame) string {
	e := EconomyOf(f.State)
	lines := StateLossLines(f.State)
	if NeckName(e.Bottleneck) != "" || e.Fix != nil || len(lines) == 0 {
		return NeckTip(e)
	}
	return "The bottleneck: what holds your income back now.\n" + lines[0].Text
}

func NeckTip(e view.EconomyView) string {
	lines := []string{"The bottleneck: what holds your income back now."}
	switch {
	case e.Why != "":
		lines = append(lines, e.Why)
	case NeckLine(e.Bottleneck) != "":
		lines = append(lines, NeckLine(e.Bottleneck))
	case e.Fix == nil:
		lines = append(lines, NoNeckLine)
	}
	if e.Fix != nil {
		lines = append(lines, fixTipLine(e))
	}
	return strings.Join(lines, "\n")
}

func LossesRect(f Frame) ui.Rect {
	board := FreightRect(f)
	top := BoardRowRect(board, lossesFirst(f))
	return ui.Rect{X: board.X, Y: top.Y, W: board.W, H: max(FlowbarLossRows(f), 1) * ui.LineHeight}
}

func (h *HUD) FlowbarButtons(f Frame) []Button {
	r := FreightRect(f)
	row := BoardRowRect(r, FlowbarFirst(f))
	out := []Button{{ID: FlowbarNeck, Rect: row, Tip: neckTipOf(f)}}
	if b, ok := FixButton(f, row); ok {
		out = append(out, b)
	}
	if LossesShown(f) {
		out = append(out, Button{ID: flowbarLosses, Rect: LossesRect(f), Tip: LossTip()})
	}
	return out
}

func (h *HUD) drawFlowbarRows(dst *ebiten.Image, f Frame, r ui.Rect) {
	x := r.X + BoardPad
	h.drawNeckRow(dst, f, r, x)
	if !LossesShown(f) {
		return
	}
	y := BoardRowRect(r, lossesFirst(f)).Y + 1
	lines := StateLossLines(f.State)
	if len(lines) == 0 {
		h.Font.Draw(dst, NoLossesRow, x, y, 1, ui.Palette.Green)
		return
	}
	for i, l := range lines[:min(len(lines), FlowbarLossMax)] {
		h.drawLossLine(dst, l, x, y+i*ui.LineHeight, r.W-2*BoardPad)
	}
}

func (h *HUD) drawNeckRow(dst *ebiten.Image, f Frame, r ui.Rect, x int) {
	row := BoardRowRect(r, FlowbarFirst(f))
	room := r.W - 2*BoardPad
	if b, ok := FixButton(f, row); ok {
		h.drawButton(dst, b)
		room = b.Rect.X - x - fixTextGap
	}
	e := EconomyOf(f.State)
	name, clr := NeckText(f), ui.Palette.RedText
	switch {
	case name == "" && e.Fix != nil:
		return
	case name == "":
		name, clr = NeckNone, ui.Palette.Green
	}
	lw := h.Font.Width(neckLabel, 1) + ui.Advance
	h.Font.Draw(dst, neckLabel, x, row.Y+1, 1, ui.Palette.Gold)
	drawClipped(dst, name, x+lw, row.Y+1, room-lw, clr)
}

func (h *HUD) drawLossLine(dst *ebiten.Image, l LossLine, x, y, width int) {
	head, fix, ok := strings.Cut(l.Text, ": ")
	if !ok {
		drawClipped(dst, l.Text, x, y, width, ui.Palette.Cream)
		return
	}
	head += ":"
	hw := h.Font.Width(head, 1) + ui.Advance
	h.Font.Draw(dst, head, x, y, 1, ui.Palette.RedText)
	drawClipped(dst, fix, x+hw, y, width-hw, ui.Palette.Cream)
}

func (h *HUD) PressFix(f Frame) []Intent {
	fix := EconomyOf(f.State).Fix
	if fix == nil {
		return nil
	}
	switch FixFade(f, *fix) {
	case view.FixShows:
		return h.showFix(fix.Action, f)
	case view.FixNames:
		return nil
	}
	return h.doLine(OfferLine(*fix), FromHarborBoard)
}

func wholeWords(label string, chars int) string {
	named := ui.Name(label)
	if len([]rune(named)) <= chars {
		return label
	}
	words := strings.Fields(label)
	for n := len(words) - 1; n > 0; n-- {
		if cut := strings.Join(words[:n], " "); len([]rune(ui.Name(cut))) <= chars {
			return cut
		}
	}
	return ui.Clip(label, chars)
}
