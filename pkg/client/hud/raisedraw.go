package hud

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	raiseHeadRows = 2
	raiseValueW   = 96
	raiseLabelW   = 70
	raiseBtnH     = 16
)

func RaiseRect() ui.Rect {
	height := raiseHead + raiseMaxRows*raiseRowStep + 4 + raiseDetailRows*ui.LineHeight + 2*raiseRowStep + raiseBtnH + 2*raisePad
	return ui.Rect{X: Width - raiseW - 6, Y: ConsoleY - height - 4, W: raiseW, H: height}
}

func raiseRowRect(box ui.Rect, i int) ui.Rect {
	return ui.Rect{X: box.X + raisePad, Y: box.Y + raiseHead + i*raiseRowStep, W: box.W - 2*raisePad, H: raiseRowH}
}

func raiseDetailY(box ui.Rect) int {
	return box.Y + raiseHead + raiseMaxRows*raiseRowStep + 4
}

func stepperY(box ui.Rect, i int) int {
	return raiseDetailY(box) + (raiseDetailRows-2)*ui.LineHeight + i*raiseRowStep
}

func stepperButtons(box ui.Rect, prefix string, i int) []Button {
	y := stepperY(box, i)
	x := box.X + raisePad + raiseLabelW
	down := Button{ID: prefix + ":down", Rect: ui.Rect{X: x, Y: y, W: raiseBtnW, H: raiseRowH}, Label: "-"}
	up := Button{ID: prefix + ":up", Rect: ui.Rect{X: x + raiseBtnW + raiseValueW, Y: y, W: raiseBtnW, H: raiseRowH}, Label: "+"}
	return []Button{down, up}
}

func confirmRect(box ui.Rect) ui.Rect {
	return ui.Rect{X: box.X + box.W - raisePad - raiseConfirmW, Y: box.Y + box.H - raisePad - raiseBtnH, W: raiseConfirmW, H: raiseBtnH}
}

func RaiseRowText(r RaiseRow) string {
	status := "no crew yet"
	switch {
	case r.Foreman && !r.Standing && len(r.Images) > 0:
		status = "nobody keeps them yet"
	case len(r.Images) == 0:
		status = "endow a class first"
	}
	return fmt.Sprintf("%s: %s, %s", strings.ToUpper(r.Crew), r.Name, status)
}

func pickButtons(box ui.Rect, rows []RaiseRow, pick string) []Button {
	out := []Button{}
	for i, r := range rows {
		tip := fmt.Sprintf("Choose the %s: the crew for %s", r.Crew, r.Name)
		out = append(out, Button{ID: raiseKey + ":pick:" + r.Crew, Rect: raiseRowRect(box, i), Label: RaiseRowText(r), Tip: tip, Selected: r.Crew == pick})
	}
	return out
}

func (h *HUD) raiseButtons(f Frame) []Button {
	box := RaiseRect()
	rows := creatableRows(RaiseRows(f))
	s := h.Ops.Raise
	out := append(pickButtons(box, rows, s.Pick), closeButton(box))
	r, ok := s.row(rows)
	if !ok {
		return out
	}
	if len(r.Images) > 1 {
		out = append(out, labelled(stepperButtons(box, raiseKey+":image", 0), "<", ">", "The previous class of the trade", "The next class of the trade")...)
	}
	out = append(out, labelled(stepperButtons(box, raiseKey+":workers", 1), "-", "+", "One hut fewer", "One hut more")...)
	return append(out, raiseConfirm(box, r, s.imageOf(r), s.workersOf(r)))
}

func labelled(buttons []Button, down, up, downTip, upTip string) []Button {
	out := []Button{buttons[0], buttons[1]}
	out[0].Label, out[0].Tip = down, downTip
	out[1].Label, out[1].Tip = up, upTip
	return out
}

func raiseConfirm(box ui.Rect, r RaiseRow, image string, workers int) Button {
	_, can := RaiseAction(r, image, workers)
	b := Button{ID: RaiseConfirm, Rect: confirmRect(box), Label: ui.Bridge("RAISE", "CREATE"), Disabled: !can}
	switch {
	case can && r.Foreman && !r.Standing:
		b.Tip = foremanTip(r, workers)
	case can:
		b.Tip = "Runs " + RaiseCommand(r, image, workers) + "\nThe console echoes it."
	default:
		b.Tip = fmt.Sprintf("The Academy trains no class of %s for you yet: endow one first (ACADEMY).", r.Crew)
	}
	return b
}

func RaiseHeadline(rows []RaiseRow) string {
	if len(rows) == 0 {
		return "No kind of box has landed yet."
	}
	return foremanWord(rows, "Pick a trade to raise its crew.", "Pick a trade to raise its shift.")
}

func (h *HUD) drawRaise(dst *ebiten.Image, f Frame) {
	box := RaiseRect()
	rows := creatableRows(RaiseRows(f))
	drawPanel(dst, box)
	h.Font.DrawShadow(dst, raiseTitle(f), box.X+raisePad, box.Y+7, 1, ui.Palette.GoldBright)
	h.Font.Draw(dst, ui.Clip(RaiseHeadline(rows), ui.MaxChars(box.W-2*raisePad-20, 1)), box.X+raisePad, box.Y+7+ui.LineHeight, 1, ui.Palette.TextDim)
	for _, b := range h.raiseButtons(f) {
		if strings.Contains(b.ID, ":pick:") {
			h.drawRaisePick(dst, b)
			continue
		}
		h.drawButton(dst, b)
	}
	if r, ok := h.Ops.Raise.row(rows); ok {
		h.drawRaiseDetail(dst, f, box, r)
	}
}

func (h *HUD) drawRaiseDetail(dst *ebiten.Image, f Frame, box ui.Rect, r RaiseRow) {
	s := h.Ops.Raise
	x, y, w := box.X+raisePad, raiseDetailY(box), box.W-2*raisePad
	ui.Fill(dst, ui.Rect{X: x, Y: y - 3, W: w, H: 1}, ui.Palette.Muted)
	text, cmd := CrewChooserLine(r, s.workersOf(r)), RaiseCommand(r, s.imageOf(r), s.workersOf(r))
	for i, row := range h.Font.WrapLines(text, w, 1, raiseDetailRows-3) {
		h.Font.Draw(dst, row, x, y+i*ui.LineHeight, 1, ui.Palette.Text)
	}
	if cmd != "" {
		Literal.Draw(dst, "$ "+ChipFit(cmd, ui.MaxChars(w, 1)-2), x, y+(raiseDetailRows-3)*ui.LineHeight, 1, ui.Palette.GoldBright)
	}
	h.drawStepperValue(dst, box, 0, ui.Bridge("CLASS", "IMAGE"), s.imageOf(r))
	h.drawStepperValue(dst, box, 1, ui.Bridge("COUNT", "REPLICAS"), fmt.Sprint(s.workersOf(r)))
}

func (h *HUD) drawRaisePick(dst *ebiten.Image, b Button) {
	b = h.liveGate(b)
	state := ui.ButtonState{Hovered: h.Hovering && h.Hover.ID == b.ID, Disabled: b.Disabled, Selected: b.Selected}
	text := ui.ButtonFace(dst, b.Rect, state)
	if b.Selected {
		ui.Stroke(dst, b.Rect.Inset(-1), ui.Palette.GoldBright)
	}
	h.Font.Draw(dst, ui.Clip(b.Label, ui.MaxChars(b.Rect.W-8, 1)), b.Rect.X+4, b.Rect.Y+4, 1, text)
}

func (h *HUD) drawStepperValue(dst *ebiten.Image, box ui.Rect, i int, label, value string) {
	y := stepperY(box, i)
	h.Font.Draw(dst, label, box.X+raisePad, y+3, 1, ui.Palette.Cream)
	cell := ui.Rect{X: box.X + raisePad + raiseLabelW + raiseBtnW, Y: y, W: raiseValueW, H: raiseRowH}
	h.Font.DrawCentered(dst, value, cell, 1, ui.Palette.GoldBright)
}

func menuMarkRects(r ui.Rect) []ui.Rect {
	return []ui.Rect{
		{X: r.X + 4, Y: r.Y + 4, W: 5, H: 1},
		{X: r.X + 5, Y: r.Y + 5, W: 3, H: 1},
		{X: r.X + 6, Y: r.Y + 6, W: 1, H: 1},
	}
}

func drawMenuMark(dst *ebiten.Image, r ui.Rect) {
	for _, m := range menuMarkRects(r) {
		ui.Fill(dst, m, ui.Palette.Gold)
	}
}
