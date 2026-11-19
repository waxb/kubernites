package hud

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	gaugeBarY  = 11
	gaugeBarH  = 6
	railH      = 6
	knobW      = 16
	knobH      = 14
	handleW    = 5
	tickMax    = 30
	tickMajor  = 5
	pieceGap   = 8
	landLabelW = 34
	landGap    = 6
	landMinBar = 80
	memBarH    = 2
	lostShown  = 0.005
)

var (
	railDark   = color.RGBA{24, 30, 32, 255}
	bandFill   = color.RGBA{60, 104, 128, 255}
	lostFill   = ui.Palette.Red
	unmetFill  = color.RGBA{120, 96, 40, 255}
	previewAdd = ui.Palette.GoldBright
)

func (h *HUD) drawCrewBoard(dst *ebiten.Image, v BoardView, f Frame) {
	ui.DarkPanel(dst, v.box)
	h.drawBoardTitle(dst, v, f)
	h.drawChips(dst, v)
	if h.live != nil {
		h.drawPieces(dst, liveImagePieces(v.Chain), v.box.X+cbPad, v.box.Y+cbGaugeY, v.box.X+v.box.W-cbPad)
	} else {
		h.drawFlowGauge(dst, v)
	}
	h.drawYard(dst, v, f)
	h.drawSlider(dst, v)
	if h.live != nil {
		h.drawPieces(dst, liveCountPieces(v.Chain), v.box.X+cbPad, v.box.Y+cbDeltaY, v.box.X+v.box.W-cbPad)
	} else {
		h.drawBoardDeltas(dst, v, f)
	}
	h.drawBoardUse(dst, v)
	h.drawLandStrip(dst, v)
	for _, group := range [][]Button{v.Steps, v.Buy} {
		for _, b := range group {
			h.drawButton(dst, b)
		}
	}
	h.drawButton(dst, closeButtonAs(v.box, CrewPrefix+":close", ""))
}

func (h *HUD) drawBoardTitle(dst *ebiten.Image, v BoardView, f Frame) {
	title, kind := boardTitle(v, f)
	x, y := v.box.X+cbPad, v.box.Y+cbTitleY
	h.Font.DrawShadow(dst, title, x, y, 1, ui.Palette.GoldBright)
	x += h.Font.Width(title, 1) + ui.Advance
	h.Font.Draw(dst, kind, x, y, 1, ui.Palette.Blue)
	if kind != "" {
		x += h.Font.Width(kind, 1) + ui.Advance
	}
	count := fmt.Sprintf(ui.Bridge("WANTS %d  READY %d", "REPLICAS %d  READY %d"), v.Chain.Desired, v.Chain.Ready)
	right := v.box.X + v.box.W - cbCloseW
	if w := h.Font.Width(count, 1); x+w <= right {
		h.Font.Draw(dst, count, right-w, y, 1, readyTone(v.Chain.Ready, v.Chain.Desired))
	}
}

func boardTitle(v BoardView, f Frame) (string, string) {
	if v.stall == "" {
		return chainTitle(v.Chain, f)
	}
	return ui.Bridge(v.stall+" depot -> crew "+v.Chain.Name, v.stall+" Service -> "+v.Chain.Name+" Deployment"), ""
}

func (h *HUD) drawChips(dst *ebiten.Image, v BoardView) {
	for _, b := range v.chips {
		name, hist := strings.CutPrefix(b.ID, CrewPrefix+":pick:hist:")
		if hist {
			fm, _ := v.Chain.ForemanNamed(name)
			drawFlag(dst, b.Rect.X, b.Rect.Y+1, ui.Tint(model.RevisionColor(fm.Revision), 140), ui.Palette.Muted)
			continue
		}
		fm, _ := v.Chain.ForemanNamed(strings.TrimPrefix(b.ID, CrewPrefix+":pick:rs:"))
		switch {
		case b.Selected:
			ui.Fill(dst, b.Rect, ui.Palette.GreenDark)
		case h.rowHot(b.ID):
			ui.Fill(dst, b.Rect, rowHover)
		}
		tone := model.RevisionColor(fm.Revision)
		drawFlag(dst, b.Rect.X+2, b.Rect.Y+1, tone, tone)
		text := ui.Clip(foremanChipText(fm), ui.MaxChars(b.Rect.W-cbFlagW-2, 1))
		h.Font.Draw(dst, text, b.Rect.X+cbFlagW+2, b.Rect.Y+2, 1, TallyInk(fm))
	}
}

func flowPieces(fl flowFacts) []deltaPiece {
	if !fl.Known {
		return []deltaPiece{{ui.Bridge("No boxes of its kind land yet.", "No requests of its kind arrive yet."), ui.Palette.TextDim}}
	}
	lost := ui.Palette.TextDim
	if fl.Lost >= lostShown {
		lost = ui.Palette.RedText
	}
	return []deltaPiece{
		{strings.ToUpper(fl.Name) + ":", ui.Palette.Cream},
		{fmt.Sprintf(ui.Bridge("%s/S CARRIED OF %s ARRIVING", "%s/S SERVED OF %s ARRIVING"), Amount(fl.Carried), Amount(fl.Arriving)), ShareColor(CarriedShare(fl.Arriving, fl.Carried, fl.Waiting))},
		{fmt.Sprintf(ui.Bridge("%s/S LOST", "%s/S DROPPED"), Amount(fl.Lost)), lost},
		{fmt.Sprintf(ui.Bridge("%d WAIT", "%d QUEUED"), int(math.Round(fl.Waiting))), ui.Palette.TextDim},
	}
}

func (h *HUD) drawPieces(dst *ebiten.Image, pieces []deltaPiece, x, y, right int) {
	for _, p := range h.fitPieces(pieces, right-x) {
		h.Font.Draw(dst, p.Text, x, y, 1, p.clr)
		x += h.Font.Width(p.Text, 1) + pieceGap
	}
}

func (h *HUD) fitPieces(pieces []deltaPiece, width int) []deltaPiece {
	span := func(list []deltaPiece) int {
		total := 0
		for _, p := range list {
			total += h.Font.Width(p.Text, 1) + pieceGap
		}
		return total - pieceGap
	}
	out := append([]deltaPiece(nil), pieces...)
	for span(out) > width && len(out) > 1 {
		drop := -1
		for i := len(out) - 1; i > 0; i-- {
			if out[i].clr != ui.Palette.RedText {
				drop = i
				break
			}
		}
		if drop < 0 {
			drop = len(out) - 1
		}
		out = append(out[:drop], out[drop+1:]...)
	}
	return out
}

func (h *HUD) drawFlowGauge(dst *ebiten.Image, v BoardView) {
	fl := v.Facts.Flow
	x, y, w := v.box.X+cbPad, v.box.Y+cbGaugeY, v.box.W-2*cbPad
	h.drawPieces(dst, flowPieces(fl), x, y, x+w)
	bar := ui.Rect{X: x, Y: y + gaugeBarY, W: w, H: gaugeBarH}
	ui.Fill(dst, bar, railDark)
	if !fl.Known || fl.Arriving <= 0 {
		return
	}
	carried := shareWidth(fl.Carried, fl.Arriving, bar.W)
	lost := shareWidth(fl.Lost, fl.Arriving, bar.W-carried)
	ui.Fill(dst, ui.Rect{X: bar.X, Y: bar.Y, W: bar.W, H: bar.H}, unmetFill)
	ui.Fill(dst, ui.Rect{X: bar.X, Y: bar.Y, W: carried, H: bar.H}, ui.Palette.Green)
	ui.Fill(dst, ui.Rect{X: bar.X + bar.W - lost, Y: bar.Y, W: lost, H: bar.H}, lostFill)
	ui.Stroke(dst, bar, ui.Palette.Ink)
}

func shareWidth(part, whole float64, w int) int {
	if whole <= 0 || w <= 0 {
		return 0
	}
	return int(math.Round(min(max(part/whole, 0), 1) * float64(w)))
}

func (h *HUD) drawSlider(dst *ebiten.Image, v BoardView) {
	t := v.Track
	rail := ui.Rect{X: t.X, Y: t.Y + (t.H-railH)/2, W: t.W, H: railH}
	ui.Fill(dst, rail, railDark)
	if wall := v.Wall(); wall < v.Top {
		x := SliderX(t, v.Top, float64(wall))
		ui.Fill(dst, ui.Rect{X: x, Y: rail.Y, W: rail.X + rail.W - x, H: rail.H}, noLandGround)
		ui.Fill(dst, ui.Rect{X: x - 1, Y: t.Y, W: 2, H: t.H}, ui.Palette.RedText)
		if mark, ok := h.noRoomRect(v); ok {
			h.Font.Draw(dst, noRoomLabel(), mark.X, mark.Y, 1, ui.Palette.RedText)
		}
	}
	h.drawTicks(dst, v, rail)
	if v.Band {
		h.drawBand(dst, v, rail)
		return
	}
	h.drawKnobFill(dst, v, rail)
	x := SliderX(t, v.Top, h.Board.shown)
	knob := ui.Rect{X: x - knobW/2, Y: t.Y + (t.H-knobH)/2, W: knobW, H: knobH}
	ui.Fill(dst, knob, ui.Palette.Gold)
	ui.Stroke(dst, knob, ui.Palette.Ink)
	h.Font.DrawCentered(dst, fmt.Sprint(int(math.Round(h.Board.shown))), knob, 1, ui.Palette.Ink)
}

func noRoomLabel() string { return ui.Bridge("NO ROOM", "PENDING") }

func (h *HUD) noRoomRect(v BoardView) (ui.Rect, bool) {
	t := v.Track
	rail := ui.Rect{X: t.X, Y: t.Y + (t.H-railH)/2, W: t.W, H: railH}
	wall := v.Wall()
	if wall >= v.Top {
		return ui.Rect{}, false
	}
	x, w := SliderX(t, v.Top, float64(wall)), h.Font.Width(noRoomLabel(), 1)
	if rail.X+rail.W-x <= w+4 {
		return ui.Rect{}, false
	}
	mark := ui.Rect{X: x + 3, Y: t.Y - ui.GlyphHeight, W: w, H: ui.GlyphHeight}
	rects, mode := v.plots()
	for i, r := range rects[:min(len(rects), len(v.Slots))] {
		if mode == plotSprites && v.Slots[i].Kind == SlotWaits && Overlaps(mark, pileArea(r)) {
			return ui.Rect{}, false
		}
	}
	return mark, true
}

func (h *HUD) drawKnobFill(dst *ebiten.Image, v BoardView, rail ui.Rect) {
	desired := SliderX(v.Track, v.Top, float64(v.Chain.Desired))
	ui.Fill(dst, ui.Rect{X: rail.X, Y: rail.Y + 1, W: desired - rail.X, H: rail.H - 2}, ui.Palette.Green)
	target := SliderX(v.Track, v.Top, float64(v.Target))
	switch {
	case target > desired:
		ui.Fill(dst, ui.Rect{X: desired, Y: rail.Y + 1, W: target - desired, H: rail.H - 2}, previewAdd)
	case target < desired:
		ui.Fill(dst, ui.Rect{X: target, Y: rail.Y + 1, W: desired - target, H: rail.H - 2}, ui.Palette.Red)
	}
}

func (h *HUD) drawTicks(dst *ebiten.Image, v BoardView, rail ui.Rect) {
	if v.Top > tickMax {
		return
	}
	for i := 0; i <= v.Top; i++ {
		x := SliderX(v.Track, v.Top, float64(i))
		tall := 2
		if i%tickMajor == 0 {
			tall = 4
		}
		ui.Fill(dst, ui.Rect{X: x, Y: rail.Y + rail.H, W: 1, H: tall}, ui.Palette.Muted)
	}
}

func (h *HUD) drawBand(dst *ebiten.Image, v BoardView, rail ui.Rect) {
	p := h.Crew.Horn
	lo, hi := SliderX(v.Track, v.Top, float64(p.Min)), SliderX(v.Track, v.Top, float64(p.Max))
	ui.Fill(dst, ui.Rect{X: lo, Y: rail.Y, W: hi - lo, H: rail.H}, bandFill)
	for _, x := range []int{lo, hi} {
		handle := ui.Rect{X: x - handleW/2, Y: v.Track.Y, W: handleW, H: v.Track.H}
		ui.Fill(dst, handle, ui.Palette.Blue)
		ui.Stroke(dst, handle, ui.Palette.Ink)
	}
	x := SliderX(v.Track, v.Top, h.Board.marker)
	knob := ui.Rect{X: x - knobW/2, Y: v.Track.Y + (v.Track.H-knobH)/2, W: knobW, H: knobH}
	ui.Fill(dst, knob, ui.Palette.GoldBright)
	ui.Stroke(dst, knob, ui.Palette.Ink)
	h.Font.DrawCentered(dst, fmt.Sprint(int(math.Round(h.Board.marker))), knob, 1, ui.Palette.Ink)
}

func liveImagePieces(c model.CrewChain) []deltaPiece {
	if c.Image == "" {
		return []deltaPiece{{ui.Bridge("Its huts' trade is not named.", "Its Pod template names no image."), ui.Palette.TextDim}}
	}
	return []deltaPiece{{ui.Bridge("TRADE", "IMAGE"), ui.Palette.Cream}, {c.Image, ui.Palette.Text}}
}

func liveCountPieces(c model.CrewChain) []deltaPiece {
	out := []deltaPiece{
		{fmt.Sprintf(ui.Bridge("%d OF THE NEWEST SHIFT", "%d UP-TO-DATE"), c.Updated), ui.Palette.Cream},
		{fmt.Sprintf(ui.Bridge("%d AT WORK", "%d AVAILABLE"), c.Available), readyTone(c.Available, c.Desired)},
	}
	if c.Strategy != "" {
		story := fmt.Sprintf("ROLLS %s MORE, %s DOWN", c.MaxSurge, c.MaxUnavailable)
		names := fmt.Sprintf("%s: SURGE %s, UNAVAILABLE %s", strings.ToUpper(c.Strategy), c.MaxSurge, c.MaxUnavailable)
		out = append(out, deltaPiece{ui.Bridge(story, names), ui.Palette.TextDim})
	}
	return out
}
