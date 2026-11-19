package hud

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	linePrefix = "line"
	StoneDo    = "stone:doit"
	ObjDo      = "obj:doit"
	ObjSpeak   = "obj:speak"
	doLabel    = "DO IT"
	doW        = 40
	lineGap    = 3
	SpeakOp    = "speak"
	UnwiredOp  = "unwired"
	unwiredInk = 110
	unwiredTip = "Read-only: this is the command a person would run. The realm only watches the cluster, so it is not sent."
	DoOp       = "do"
	EnterHint  = "press Enter to speak it"
	speakTip   = "Opens the Speaking Stone with this command. Press Enter to speak it."
	cardLineX  = 69
	ButtonPad  = 6
)

type opsState struct {
	prefill  CommandLine
	Calamity CalamityState `json:"-"`
	live     map[string]view.CalamityView
	abandon  retireArm
	Chooser  chooserState `json:"-"`
	Raise    raiseState   `json:"-"`
	shelved  map[string]bool
}

func doTip(l CommandLine) string {
	tip := "No typing needed: the button does the same.\n" + l.Label
	if OpensPlacement(l.Action) {
		tip = l.Label + "\n" + placeNote
	}
	if l.Payoff != "" {
		return tip + "\n" + l.Payoff
	}
	return tip
}

func chipWidth(cmd string) int {
	return len(ChipLabel(cmd))*ui.Advance + chipPad
}

func LineRects(row ui.Rect, l CommandLine) (ui.Rect, ui.Rect) {
	if l.Command == "" {
		if l.Action == nil {
			return ui.Rect{}, ui.Rect{}
		}
		return ui.Rect{}, row
	}
	room := row.W
	if l.Action != nil {
		room -= doW + lineGap
	}
	text := l.Command
	if l.Shown != "" {
		text = l.Shown
	}
	chip := ui.Rect{X: row.X, Y: row.Y, W: min(chipWidth(text), room), H: row.H}
	if l.Action == nil {
		return chip, ui.Rect{}
	}
	return chip, ui.Rect{X: chip.X + chip.W + lineGap, Y: row.Y, W: doW, H: row.H}
}

func LineButtons(id string, row ui.Rect, l CommandLine) []Button {
	chip, do := LineRects(row, l)
	out := []Button{}
	if chip.W > 0 && l.Order != "" {
		label := l.Command
		if l.Shown != "" {
			label = l.Shown
		}
		return []Button{{ID: LiveOrderPrefix + ":" + l.Order, Rect: chip, Label: label, Tip: l.Label}}
	}
	if chip.W > 0 && l.Shown != "" && !l.Unwired && l.Blocked == "" {
		return append([]Button{{ID: id + ":" + SpeakOp, Rect: chip, Label: l.Shown, Tip: speakTip, Speaks: l.Command}}, out...)
	}
	if chip.W > 0 && l.Unwired {
		return []Button{{ID: id + ":" + UnwiredOp, Rect: chip, Label: l.Command, Tip: unwiredTip, Disabled: true}}
	}
	if chip.W > 0 && l.Blocked == "" {
		out = append(out, Button{ID: id + ":" + SpeakOp, Rect: chip, Label: l.Command, Tip: speakTip})
	}
	if do.W > 0 {
		label, tip := doLabel, doTip(l)
		if l.Command == "" {
			label = strings.ToUpper(l.Label)
		}
		if l.Show {
			label, tip = showLabel, ShowTip(*l.Action)
		}
		b := Button{ID: id + ":" + DoOp, Rect: do, Label: label, Tip: tip}
		if l.Blocked != "" {
			b.Disabled, b.Tip = true, CapitalFirst(l.Blocked)+".\n"+tip
		}
		out = append(out, b)
	}
	return out
}

func (h *HUD) drawCommandChip(dst *ebiten.Image, r ui.Rect, cmd string, hot bool) {
	fill := consoleInk
	if hot {
		fill = ui.Palette.BlueDark
	}
	ui.Fill(dst, r, fill)
	ui.Stroke(dst, r, ui.Palette.ChipBorder)
	h.Font.Draw(dst, ">", r.X+4, r.Y+3, 1, ui.Palette.Gold)
	Literal.Draw(dst, ChipFit(cmd, ui.MaxChars(r.W-20, 1)), r.X+14, r.Y+3, 1, ui.Palette.Text)
}

func (h *HUD) drawUnwiredChip(dst *ebiten.Image, r ui.Rect, cmd string) {
	ui.Fill(dst, r, consoleInk)
	ui.Stroke(dst, r, ui.Tint(ui.Palette.ChipBorder, unwiredInk))
	h.Font.Draw(dst, ">", r.X+4, r.Y+3, 1, ui.Palette.TextDim)
	Literal.Draw(dst, ChipFit(cmd, ui.MaxChars(r.W-20, 1)), r.X+14, r.Y+3, 1, ui.Palette.TextDim)
}

func (h *HUD) drawLineButton(dst *ebiten.Image, b Button) bool {
	if !strings.HasPrefix(b.ID, linePrefix+":") {
		return false
	}
	if strings.HasSuffix(b.ID, ":"+SpeakOp) && b.Speaks != "" {
		h.drawButton(dst, b)
		return true
	}
	if strings.HasSuffix(b.ID, ":"+SpeakOp) {
		h.drawCommandChip(dst, b.Rect, b.Label, h.Hovering && h.Hover.ID == b.ID)
		return true
	}
	if strings.HasSuffix(b.ID, ":"+UnwiredOp) {
		h.drawUnwiredChip(dst, b.Rect, b.Label)
		return true
	}
	lines := h.buttonLines(b)
	if len(lines) > 1 {
		h.drawWrappedButton(dst, b, lines)
		return true
	}
	face := b
	face.Label = lines[0]
	h.drawButton(dst, face)
	return true
}

func (h *HUD) buttonLines(b Button) []string {
	if b.Rect.H >= ChipH+ui.LineHeight {
		return h.Font.WrapLines(b.Label, b.Rect.W-ButtonPad, 1, RemedyMaxLines)
	}
	return []string{ui.Clip(b.Label, ui.MaxChars(b.Rect.W-4, 1))}
}

func (h *HUD) drawWrappedButton(dst *ebiten.Image, b Button, lines []string) {
	b = h.liveGate(b)
	state := ui.ButtonState{Hovered: h.Hovering && h.Hover.ID == b.ID, Disabled: b.Disabled, Selected: b.Selected}
	clr := ui.ButtonFace(dst, b.Rect, state)
	top := b.Rect.Y + (b.Rect.H-(len(lines)-1)*ui.LineHeight-ui.CapHeight)/2
	for i, l := range lines {
		h.Font.DrawShadow(dst, l, b.Rect.X+(b.Rect.W-h.Font.Width(l, 1))/2, top+i*ui.LineHeight, 1, clr)
	}
}

func (h *HUD) lineSlots(f Frame) map[string]CommandLine {
	out := map[string]CommandLine{}
	for i, l := range CardLines(f) {
		out[fmt.Sprintf("card:%d", i)] = l
	}
	for i, l := range h.calamityLines(f) {
		out[fmt.Sprintf("cal:%d", i)] = l
	}
	return out
}

func (h *HUD) PressLine(rest string, f Frame) []Intent {
	cut := strings.LastIndex(rest, ":")
	if cut < 0 {
		return nil
	}
	l, ok := h.lineSlots(f)[rest[:cut]]
	if !ok {
		return nil
	}
	from := CardSource(f)
	if strings.HasPrefix(rest, "cal:") {
		from = FromCalamityCard
	}
	if rest[cut+1:] == DoOp && l.Show {
		return h.showFix(*l.Action, f)
	}
	if rest[cut+1:] == DoOp {
		if from == FromCalamityCard {
			h.troubleActed()
		}
		return h.doLine(l, from)
	}
	return h.UseLine(l, from)
}

func (h *HUD) UseLine(l CommandLine, from string) []Intent {
	if l.Command == "" {
		return h.doLine(l, from)
	}
	h.OpenStone(l.Command)
	h.Ops.prefill = l
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) doLine(l CommandLine, from string) []Intent {
	if l.Action == nil {
		return nil
	}
	if h.prefillLive() {
		h.Stone.SetInput("")
	}
	if h.OverlayID == OverlayStone {
		h.OverlayID = OverlayNone
	}
	h.Ops.prefill = CommandLine{}
	return []Intent{ActFrom(*l.Action, from)}
}

func BareCommandOf(cmd string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(cmd), "kubectl "))
}

func (h *HUD) prefillLive() bool {
	p := h.Ops.prefill
	return h.OverlayID == OverlayStone && p.Action != nil && p.Command != "" && strings.TrimSpace(h.typedCommand(string(h.Stone.Input))) == BareCommandOf(p.Command)
}

func (h *HUD) promptRow() ui.Rect {
	out := h.stoneOutput()
	return ui.Rect{X: out.X, Y: out.Y + out.H + 4 + stoneCandidateH, W: out.W, H: stonePromptH}
}

func (h *HUD) prefillButtons() []Button {
	if !h.prefillLive() {
		return nil
	}
	row := h.promptRow()
	r := ui.Rect{X: row.X + row.W - doW - 2, Y: row.Y + 2, W: doW, H: row.H - 4}
	return []Button{{ID: StoneDo, Rect: r, Label: doLabel, Tip: doTip(h.Ops.prefill)}}
}

func (h *HUD) drawPrefillHint(dst *ebiten.Image) {
	buttons := h.prefillButtons()
	if len(buttons) > 0 {
		r := buttons[0].Rect
		hint := EnterHint + ", or"
		w, _ := h.Font.Measure(hint, 1)
		h.Font.Draw(dst, hint, r.X-w-6, r.Y+2, 1, ui.Palette.GoldBright)
		return
	}
	if h.Stone.Fresh && len(h.Stone.Input) > 0 {
		row := h.promptRow()
		w, _ := h.Font.Measure(EnterHint, 1)
		h.Font.Draw(dst, EnterHint, row.X+row.W-w-6, row.Y+5, 1, ui.Palette.GoldBright)
	}
}

func (h *HUD) PressPrefill() []Intent {
	if !h.prefillLive() {
		return nil
	}
	return h.doLine(h.Ops.prefill, FromStoneButton)
}

func ObjectiveChipRects(card ui.Rect, l CommandLine) (ui.Rect, ui.Rect) {
	chip := ChipRect(card, l.Command)
	if !l.Runnable() {
		chip.W = min(chip.W, card.W-12-orSpeakW)
		chip.X = card.X + card.W - 6 - chip.W
		return chip, ui.Rect{}
	}
	do := ui.Rect{X: card.X + card.W - 6 - doW, Y: chip.Y, W: doW, H: ChipH}
	chip.W = min(chip.W, do.X-lineGap-card.X-6)
	chip.X = do.X - lineGap - chip.W
	return chip, do
}

func (h *HUD) objectiveLineButtons(f Frame) []Button {
	if !h.chipShown(f) {
		return nil
	}
	l, _ := f.ObjectiveLine()
	l.Command = h.shownCommand(f)
	chip, do := ObjectiveChipRects(h.ObjectiveRectOf(f), l)
	out := []Button{{ID: ObjSpeak, Rect: chip, Label: EchoLabel(l.Command), Tip: speakTip}}
	if do.W > 0 {
		out = append(out, Button{ID: ObjDo, Rect: do, Label: doLabel, Tip: doTip(l)})
	}
	return out
}

func (h *HUD) drawObjectiveLine(dst *ebiten.Image, f Frame, card ui.Rect) {
	if !h.chipShown(f) {
		return
	}
	l, _ := f.ObjectiveLine()
	l.Command = h.shownCommand(f)
	chip, do := ObjectiveChipRects(card, l)
	h.drawCommandChip(dst, chip, l.Command, h.Hovering && h.Hover.ID == ObjSpeak)
	if do.W > 0 {
		h.drawButton(dst, Button{ID: ObjDo, Rect: do, Label: doLabel})
		return
	}
	w := h.Font.Span(orSpeak, 1)
	h.Font.Draw(dst, orSpeak, chip.X-w-4, chip.Y+3, 1, ui.Palette.TextDim)
}
