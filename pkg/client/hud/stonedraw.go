package hud

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	stoneHalfH      = 180
	stoneMinH       = 120
	maxCandidates   = 6
	stoneHeaderH    = 26
	stonePromptH    = 16
	stoneCandidateH = 12
)

var consoleInk = color.RGBA{10, 14, 16, 255}

var stoneEdits = map[ebiten.Key]func(*Stone){
	ebiten.KeyBackspace:  (*Stone).Backspace,
	ebiten.KeyDelete:     (*Stone).Delete,
	ebiten.KeyArrowLeft:  func(s *Stone) { s.Move(-1) },
	ebiten.KeyArrowRight: func(s *Stone) { s.Move(1) },
	ebiten.KeyHome:       func(s *Stone) { s.Move(-len(s.Input)) },
	ebiten.KeyEnd:        func(s *Stone) { s.Move(len(s.Input)) },
	ebiten.KeyArrowUp:    func(s *Stone) { s.Recall(-1) },
	ebiten.KeyArrowDown:  func(s *Stone) { s.Recall(1) },
}

func (h *HUD) StoneRect() ui.Rect {
	if h.Stone.Tall {
		return ui.Rect{X: 6, Y: Overlay.Y, W: Width - 12, H: Overlay.H}
	}
	top := ConsoleY - 6 - stoneHalfH
	if c := h.Yard.place; h.teacherShown() && c.W > 0 {
		top = min(max(top, c.Y+h.TeacherLayout().Height+ShadowH+TipClear), ConsoleY-6-stoneMinH)
	}
	return ui.Rect{X: 6, Y: top, W: Width - 12, H: ConsoleY - 6 - top}
}

func (h *HUD) stoneOutput() ui.Rect {
	r := h.StoneRect()
	return ui.Rect{X: r.X + 8, Y: r.Y + stoneHeaderH, W: r.W - 16, H: r.H - stoneHeaderH - stonePromptH - stoneCandidateH - 10}
}

func (h *HUD) StoneVisible() int {
	return ui.VisibleLines(h.stoneOutput().H-6, 1)
}

func (h *HUD) StoneKeys(in Input) []Intent {
	if h.kube != nil {
		return h.kubectlKeys(in)
	}
	for _, r := range in.Chars {
		h.Stone.Insert(r)
	}
	var out []Intent
	for _, k := range in.Keys {
		if edit, ok := stoneEdits[k]; ok {
			edit(&h.Stone)
			continue
		}
		switch k {
		case ebiten.KeyTab:
			h.stoneTab()
		case ebiten.KeyPageUp:
			h.PageStone(-1)
		case ebiten.KeyPageDown:
			h.PageStone(1)
		case ebiten.KeyEscape:
			h.OverlayID = OverlayNone
		case ebiten.KeyEnter, ebiten.KeyNumpadEnter:
			out = append(out, h.Speak()...)
		}
	}
	return out
}

func (h *HUD) Speak() []Intent {
	if h.kube != nil {
		return nil
	}
	cmd := h.typedCommand(h.Stone.Submit())
	if cmd == "" {
		return nil
	}
	if cmd == "clear" {
		h.Stone.Output, h.Stone.Scroll = nil, ui.Scroll{}
		return nil
	}
	h.EchoTyped(cmd)
	if cmd == unfoldWord {
		return h.unfold()
	}
	if h.startKubectl(cmd) {
		return []Intent{{Kind: IntentKubectl, Command: cmd}}
	}
	if h.answerReach(cmd) {
		return []Intent{{Kind: IntentKubectl, Command: cmd}, {Kind: IntentSound, Command: "error"}}
	}
	res := h.kubectl(cmd)
	kind, sound := LineOutput, "click"
	if !res.OK {
		kind, sound = LineError, "error"
	}
	if res.Summary != "" && ui.StoryNaming() {
		h.Stone.Append(LineHint, res.Summary)
	}
	answer := StoryAnswer(res.Output, res.OK)
	if res.OK && ui.StoryNaming() {
		answer, h.Stone.folded = foldSheet(answer)
	}
	written := h.Stone.Written
	h.Stone.Append(kind, answer...)
	h.hintStone(cmd, res)
	h.pointStone(written)
	return []Intent{{Kind: IntentKubectl, Command: cmd}, {Kind: IntentSound, Command: sound}}
}

func (h *HUD) OpenStone(prefill string) {
	h.OverlayID, h.Stone.away = OverlayStone, false
	if cmd := strings.TrimPrefix(strings.TrimSpace(prefill), "kubectl "); cmd != "" {
		h.Stone.SetInput(h.typedInput(cmd))
		h.Stone.Fresh = true
	}
}

func (h *HUD) RunStone(cmd string, run bool) bool {
	cmd = strings.TrimPrefix(strings.TrimSpace(cmd), "kubectl ")
	h.OverlayID, h.Stone.away = OverlayStone, false
	if !run || h.kube != nil || h.backend.Start == nil {
		h.Stone.SetInput(h.typedInput(cmd))
		h.Stone.Fresh = true
		return false
	}
	h.Stone.SetInput("")
	h.EchoTyped(cmd)
	job, ok := h.backend.Start(cmd)
	if !ok {
		return false
	}
	h.kube = &stoneRun{job: job, command: cmd}
	return true
}

func (h *HUD) StoneButtons() []Button {
	r := h.StoneRect()
	grow := Button{ID: "stone:grow", Rect: ui.Rect{X: r.X + r.W - 44, Y: r.Y + 5, W: 18, H: 14}, Label: "^", Tip: "Taller Stone"}
	if h.Stone.Tall {
		grow.Label, grow.Tip = "v", "Shorter Stone"
	}
	return append(append([]Button{grow, closeButton(r)}, h.prefillButtons()...), h.stoneAskButtons()...)
}

func (h *HUD) pressStone(rest string, _ Frame) []Intent {
	if rest == "grow" {
		h.Stone.Tall = !h.Stone.Tall
		h.Stone.Scroll = ui.Scroll{}
	}
	if rest == "doit" {
		return h.PressPrefill()
	}
	if rest == "confirm" || rest == "cancel" {
		return h.answerStone(rest == "confirm")
	}
	return nil
}

func (h *HUD) drawStone(dst *ebiten.Image) {
	r := h.StoneRect()
	ui.DarkPanel(dst, r)
	h.header(dst, r, "THE SPEAKING STONE")
	if sc := h.kube.screen(); sc != nil {
		h.drawTerminal(dst, sc)
		h.DrawOverlayButtons(dst, h.StoneButtons())
		return
	}
	h.Font.Draw(dst, "Tab completes  PgUp/PgDn scroll  Up/Down history", r.X+250, r.Y+9, 1, ui.Palette.TextDim)
	out := h.stoneOutput()
	ui.Fill(dst, out, consoleInk)
	ui.Stroke(dst, out, ui.Palette.BlueDark)
	visible, rows := h.stoneShownRows(), h.StoneRows()
	end := len(rows) - h.Stone.Scroll.Offset
	start := max(end-visible, 0)
	width := h.StoneWidth()
	y := out.Y + 3
	for _, r := range rows[start:max(end, start)] {
		Literal.Draw(dst, Literal.Clip(r.Text, width), out.X+4, y, 1, r.clr)
		y += ui.LineHeight
	}
	h.drawScrollbar(dst, out, len(rows), visible, max(len(rows)-visible-h.Stone.Scroll.Offset, 0))
	c := h.stoneCompletion()
	h.drawCandidates(dst, c, ui.Rect{X: out.X, Y: out.Y + out.H + 2, W: out.W, H: stoneCandidateH})
	h.drawPrompt(dst, ui.Rect{X: out.X, Y: out.Y + out.H + 4 + stoneCandidateH, W: out.W, H: stonePromptH}, c.Ghost)
	h.drawPrefillHint(dst)
	h.drawStoneRun(dst)
	h.DrawOverlayButtons(dst, h.StoneButtons())
}

func (h *HUD) drawCandidates(dst *ebiten.Image, c Completion, r ui.Rect) {
	list := c.Candidates
	if n := len(h.Stone.cycle.list); n > 0 {
		list = h.Stone.cycle.list
	}
	if len(list) < 2 {
		return
	}
	x := r.X + 4
	for i, word := range list[:min(len(list), maxCandidates)] {
		clr := ui.Palette.Blue
		if len(h.Stone.cycle.list) > 0 && i == h.Stone.cycle.index {
			clr = ui.Palette.GoldBright
		}
		Literal.Draw(dst, word, x, r.Y+2, 1, clr)
		x += Literal.Span(word, 2)
		if x > r.X+r.W-40 {
			break
		}
	}
	if len(list) > maxCandidates {
		h.Font.Draw(dst, "+more", x, r.Y+2, 1, ui.Palette.TextDim)
	}
}

func (h *HUD) drawPrompt(dst *ebiten.Image, r ui.Rect, ghost string) {
	ui.Fill(dst, r, consoleInk)
	ui.Stroke(dst, r, ui.Palette.Gold)
	prompt := h.stonePrompt()
	width := ui.MaxChars(r.W-8, 1) - len(prompt) - 1
	offset := max(h.Stone.Cursor-width, 0)
	visible := h.Stone.Input[offset:min(len(h.Stone.Input), offset+width)]
	x := r.X + 4
	Literal.Draw(dst, prompt, x, r.Y+4, 1, ui.Palette.GoldBright)
	x += Literal.Span(prompt, 0)
	Literal.Draw(dst, string(visible), x, r.Y+4, 1, ui.Palette.Cream)
	if room := width - len(visible); ghost != "" && room > 0 {
		Literal.Draw(dst, Literal.Clip(ghost, room), x+len(visible)*ui.Advance, r.Y+4, 1, ui.Palette.TextDim)
	}
	if h.Reduce || int(h.Now*2)%2 == 0 {
		cx := x + (h.Stone.Cursor-offset)*ui.Advance
		ui.Fill(dst, ui.Rect{X: cx, Y: r.Y + 3, W: 1, H: 9}, ui.Palette.GoldBright)
	}
}
