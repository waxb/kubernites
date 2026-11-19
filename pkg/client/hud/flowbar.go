package hud

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	flowbarBarH    = 2
	flowbarBarY    = 8
	FlowbarMinGap  = 6
	flowbarFixMinW = 40
	FlowbarFixPad  = 8
	FlowbarLossMax = 2
	FlowbarFix     = FreightTip + "fix"
	FlowbarNeck    = FreightTip + "neck"
	FlowbarMoney   = FreightTip + "money"
	flowbarLosses  = FreightTip + "losses"
)

var NamedFont ui.Font

type Segment struct {
	Text string
	Clr  color.RGBA
}

func FitSegments(ladder [][]Segment, width int) []Segment {
	return ladder[fitIndex(ladder, width)]
}

func fitIndex(ladder [][]Segment, width int) int {
	for i, segs := range ladder {
		total := FlowbarMinGap * (len(segs) - 1)
		for _, s := range segs {
			total += NamedFont.Width(s.Text, 1)
		}
		if total <= width {
			return i
		}
	}
	return len(ladder) - 1
}

const widestRate = "0.00"

func FitSteady(ladderOf func(rate func(float64) string) [][]Segment, width int) []Segment {
	widest := ladderOf(func(float64) string { return widestRate })
	return ladderOf(Rate)[fitIndex(widest, width)]
}

func FlowSegments(e view.EconomyView, width int) []Segment {
	lostClr := ui.Palette.TextDim
	if e.Lost >= minLoss {
		lostClr = ui.Palette.RedText
	}
	carry, drop := ui.Bridge("CARRIED ", "SERVED "), ui.Bridge("LOST ", "DROPPED ")
	seg := func(a, b, c string) []Segment {
		return []Segment{{a, ui.Palette.Cream}, {b, ui.Palette.Green}, {c, lostClr}}
	}
	return FitSteady(func(rate func(float64) string) [][]Segment {
		in, carried, lost := rate(e.Arriving), rate(e.Carried), rate(e.Lost)
		return [][]Segment{
			seg("ARRIVING "+in+"/S", carry+carried+"/S", drop+lost+"/S"),
			seg("IN "+in+"/S", carry+carried+"/S", drop+lost+"/S"),
			seg("IN "+in, carry+carried, drop+lost+"/S"),
		}
	}, width)
}

func moneyAmount(v float64) string {
	amount := goldAmount(v)
	return ui.Bridge(amount, "$"+amount)
}

func SignedMoney(v float64) string {
	sign, amount := signedGold(v)
	return sign + ui.Bridge(amount, "$"+amount)
}

func MoneySegments(e view.EconomyView, width int) []Segment {
	income, upkeep := moneyAmount(e.Income), moneyAmount(e.Upkeep)
	seg := func(a, b string) []Segment {
		return []Segment{{a, ui.Palette.GoldBright}, {b, ui.Palette.Cream}}
	}
	if e.Upkeep == 0 {
		return FitSegments([][]Segment{
			{{"INCOME " + income + "/S", ui.Palette.GoldBright}},
			{{"INCOME " + income, ui.Palette.GoldBright}},
			{{"INC " + income, ui.Palette.GoldBright}},
		}, width)
	}
	return FitSegments([][]Segment{
		seg("INCOME "+income+"/S", "UPKEEP "+upkeep+"/S"),
		seg("INCOME "+income, "UPKEEP "+upkeep),
		seg("INC "+income, "COST "+upkeep),
	}, width)
}

func (h *HUD) drawSegments(dst *ebiten.Image, segs []Segment, x, y, width int) {
	widths, used := make([]int, len(segs)), 0
	for i, s := range segs {
		widths[i] = h.Font.Width(s.Text, 1)
		used += widths[i]
	}
	gap := FlowbarMinGap
	if len(segs) > 1 {
		gap = max((width-used)/(len(segs)-1), FlowbarMinGap)
	}
	for i, s := range segs {
		h.Font.Draw(dst, s.Text, x, y, 1, s.Clr)
		x += widths[i] + gap
	}
}

func (h *HUD) drawFlowRow(dst *ebiten.Image, f Frame, x, y, width int) {
	e := EconomyOf(f.State)
	h.drawSegments(dst, FlowSegments(e, width), x, y, width)
	bar := ui.Rect{X: x, Y: y + flowbarBarY, W: width, H: flowbarBarH}
	ui.Fill(dst, bar, ui.Tint(ui.Palette.Ink, barShade))
	if e.Arriving <= 0 {
		return
	}
	carried := int(float64(width) * min(max(e.Carried/e.Arriving, 0), 1))
	lost := int(float64(width) * min(max(e.Lost/e.Arriving, 0), 1))
	ui.Fill(dst, ui.Rect{X: x, Y: bar.Y, W: carried, H: flowbarBarH}, ui.Palette.Green)
	if lost > 0 {
		ui.Fill(dst, ui.Rect{X: x + width - lost, Y: bar.Y, W: lost, H: flowbarBarH}, ui.Palette.RedText)
	}
}

func (h *HUD) drawMoneyRow(dst *ebiten.Image, f Frame, x, y, width int) {
	h.drawSegments(dst, MoneySegments(EconomyOf(f.State), width), x, y, width)
}

func FlowbarLossRows(f Frame) int {
	if !LossesShown(f) {
		return 0
	}
	return min(max(len(StateLossLines(f.State)), 1), FlowbarLossMax)
}

func flowbarRows(f Frame) int {
	return FlowbarLossRows(f) + 1
}

func FlowbarFirst(f Frame) int { return boardTopRows(f) + boardExtra(f) }

func lossesFirst(f Frame) int {
	return FlowbarFirst(f) + 1
}
