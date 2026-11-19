package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	trialPad     = 20
	trialRowH    = 70
	trialRowGap  = 6
	trialListTop = 62
	TrialBtnH    = 18
	TrialBtnPad  = 12
	trialBtnGap  = 6
	trialStartW  = 70
	trialBestW   = 150
	trialHeadGap = 8
	trialColGap  = 16
	trialYAMLW   = 200
)

var trialMedalColors = map[view.Medal]color.RGBA{
	view.MedalGold:   ui.Palette.GoldBright,
	view.MedalSilver: {206, 212, 214, 255},
	view.MedalBronze: {214, 148, 92, 255},
}

func medalColor(m view.Medal) color.RGBA {
	if c, ok := trialMedalColors[m]; ok {
		return c
	}
	return ui.Palette.TextDim
}

func (h *HUD) labelButton(id, label, tip string, x, y int) Button {
	return Button{ID: id, Label: label, Tip: tip, Rect: ui.Rect{X: x, Y: y, W: h.Font.Width(label, 1) + 2*TrialBtnPad, H: TrialBtnH}}
}

func (h *HUD) buttonRow(x, y int, specs [][3]string) []Button {
	out := make([]Button, 0, len(specs))
	for _, s := range specs {
		b := h.labelButton(trialPrefix+":"+s[0], s[1], s[2], x, y)
		out = append(out, b)
		x += b.Rect.W + trialBtnGap
	}
	return out
}

func (h *HUD) TrialsButtons(f Frame) []Button {
	switch h.Trial.PageID {
	case TrialPageBrief:
		return h.briefButtons()
	case TrialPageResult:
		return h.resultButtons()
	}
	return h.listButtons(f)
}

func (h *HUD) TrialsRect() ui.Rect {
	r := Overlay
	switch h.Trial.PageID {
	case TrialPageBrief:
		r.H = h.briefHeight()
	case TrialPageResult:
		r.H = h.resultHeight()
	default:
		r.H = trialListTop + len(view.Content.Trials())*(trialRowH+trialRowGap) + trialPad/2
	}
	return r
}

func (h *HUD) briefHeight() int {
	t := h.trialOf()
	w := Overlay.W - 2*trialPad
	rows := 0
	for _, b := range t.Brief {
		rows += len(h.Font.Wrap(trialText(b), w, 1))
	}
	indent := seeLiteral.Width("MEDALS", 1) + 2*ui.Advance
	rows += len(h.Font.Wrap(trialText(t.Goal), w-indent, 1)) + 2
	return 30 + ui.LineHeight + trialHeadGap + rows*ui.LineHeight + len(t.Brief)*4 + trialHeadGap + 3*4 + 2*ui.LineHeight + TrialBtnH + 24
}

func (h *HUD) DrawTrials(dst *ebiten.Image, f Frame) {
	ui.DarkPanel(dst, h.TrialsRect())
	switch h.Trial.PageID {
	case TrialPageBrief:
		h.drawBrief(dst)
	case TrialPageResult:
		h.drawResult(dst)
	default:
		h.drawList(dst)
	}
	h.drawTrialButtons(dst, h.TrialsButtons(f))
}

func (h *HUD) primaryTrial(id string) bool {
	if h.OverlayID == OverlayTrials && h.Trial.PageID == TrialPageResult {
		return id == h.ResultPrimary()
	}
	return id == trialPrefix+":"+TrialBegin || strings.HasPrefix(id, trialPrefix+":"+TrialStart+":")
}

func (h *HUD) ResultPrimary() string {
	v := h.Trial.view
	if v == nil || v.Status == view.TrialLost {
		return trialPrefix + ":" + TrialRetry
	}
	if next, ok := v.Trial.Next(); ok {
		return trialPrefix + ":" + TrialStart + ":" + next.ID
	}
	return trialPrefix + ":" + TrialBoard + ":" + v.Trial.ID
}

func (h *HUD) drawTrialButtons(dst *ebiten.Image, buttons []Button) {
	for _, b := range buttons {
		switch {
		case b.Label == "":
		case h.primaryTrial(b.ID) && !b.Disabled:
			h.drawPrimary(dst, b)
		default:
			h.drawButton(dst, b)
		}
	}
}

func trialRow(i int) ui.Rect {
	return ui.Rect{X: Overlay.X + trialPad/2, Y: Overlay.Y + trialListTop + i*(trialRowH+trialRowGap), W: Overlay.W - trialPad, H: trialRowH}
}

func (h *HUD) listButtons(Frame) []Button {
	out := []Button{}
	for i, t := range view.Content.Trials() {
		r := trialRow(i)
		out = append(out, Button{ID: trialPrefix + ":pick:" + t.ID, Rect: r, Tip: trialText(t.Idea)})
		out = append(out, Button{ID: trialPrefix + ":" + TrialStart + ":" + t.ID, Label: "START", Tip: "Play " + trialText(t.Title) + ". Your realm waits as you left it.",
			Rect: ui.Rect{X: r.X + r.W - trialStartW - 8, Y: r.Y + r.H/2 - TrialBtnH/2, W: trialStartW, H: TrialBtnH}})
	}
	return append(out, closeButton(h.TrialsRect()))
}

func (h *HUD) drawList(dst *ebiten.Image) {
	h.header(dst, Overlay, "TRIALS")
	seeLiteral.Draw(dst, "Short drills outside your realm, one idea each. Beat the times to win a medal: gold, silver or bronze.", Overlay.X+trialPad/2+2, Overlay.Y+30, 1, ui.Palette.TextDim)
	h.Font.Draw(dst, "Your realm waits as you left it: nothing in a trial touches it.", Overlay.X+trialPad/2+2, Overlay.Y+30+ui.LineHeight, 1, ui.Palette.TextDim)
	for i, t := range view.Content.Trials() {
		r := trialRow(i)
		edge := ui.Palette.BlueDark
		if t.ID == h.Trial.Focus {
			edge = ui.Palette.Gold
		}
		ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, 220))
		ui.Stroke(dst, r, edge)
		textW := r.W - trialStartW - trialBestW - 24
		h.Font.Draw(dst, strings.ToUpper(trialText(t.Title)), r.X+8, r.Y+6, 1, ui.Palette.Gold)
		h.Font.Draw(dst, ui.Clip(trialText(t.Idea), ui.MaxChars(textW, 1)), r.X+8, r.Y+6+ui.LineHeight, 1, ui.Palette.Cream)
		for j, row := range h.Font.WrapLines(trialText(t.Goal), textW, 1, 3) {
			h.Font.Draw(dst, row, r.X+8, r.Y+8+(2+j)*ui.LineHeight, 1, ui.Palette.TextDim)
		}
		h.drawBest(dst, t, r.X+r.W-trialStartW-trialBestW-12, r.Y+14)
	}
}

func (h *HUD) drawBest(dst *ebiten.Image, t view.Trial, x, y int) {
	sc, ok := h.Trial.scores.Of(t.ID)
	if !ok || !sc.Won {
		h.Font.Draw(dst, "NOT WON YET", x, y, 1, ui.Palette.TextDim)
		if ok && sc.Tries > 0 {
			h.Font.Draw(dst, triesText(sc.Tries), x, y+ui.LineHeight, 1, ui.Palette.TextDim)
		}
		return
	}
	m := sc.Earned()
	label := "COMPLETED"
	if m != view.MedalNone {
		label = m.Name() + " MEDAL"
	}
	seeLiteral.Draw(dst, label, x, y, 1, medalColor(m))
	h.Font.Draw(dst, "best "+view.TrialClock(sc.Seconds)+", "+triesText(sc.Tries), x, y+ui.LineHeight, 1, ui.Palette.Text)
}

func triesText(n int) string {
	if n == 1 {
		return "1 try"
	}
	return fmt.Sprintf("%d tries", n)
}

func medalTimes(t view.TrialTimes) string {
	return fmt.Sprintf("GOLD %s   SILVER %s   BRONZE %s", view.TrialClock(t.Gold), view.TrialClock(t.Silver), view.TrialClock(t.Bronze))
}

func (h *HUD) briefBottom() int {
	r := h.TrialsRect()
	return r.Y + r.H - TrialBtnH - 10
}

func (h *HUD) briefButtons() []Button {
	return append(h.buttonRow(Overlay.X+trialPad, h.briefBottom(), [][3]string{
		{TrialBegin, "START", "Start the trial: its clock starts now and does not pause"},
		{TrialLeave, "BACK TO THE REALM", "Leave the trial: your realm waits as you left it"},
	}), closeButtonAs(h.TrialsRect(), trialPrefix+":close", "Look at the trial's land first; START is on its card [Esc]"))
}

func (h *HUD) drawBrief(dst *ebiten.Image) {
	t := h.trialOf()
	x, w := Overlay.X+trialPad, Overlay.W-2*trialPad
	h.header(dst, Overlay, strings.ToUpper(trialText(t.Title)))
	y := Overlay.Y + 30
	h.Font.Draw(dst, trialText(t.Idea), x, y, 1, ui.Palette.Cream)
	y += ui.LineHeight + trialHeadGap
	for _, b := range t.Brief {
		y = h.drawTextLines(dst, []string{trialText(b)}, x, y, w, ui.Palette.Text) + 4
	}
	y += trialHeadGap
	y = h.drawTrialLabel(dst, h.Font, "GOAL", trialText(t.Goal), x, y, w, ui.Palette.Cream)
	y = h.drawTrialLabel(dst, seeLiteral, "MEDALS", medalTimes(t.Times)+"   (finish slower and it still counts)", x, y, w, ui.Palette.Text)
	h.drawTrialLabel(dst, h.Font, "CLOCK", "It starts when you press START, does not pause, and runs at x1. RETRY starts afresh.", x, y, w, ui.Palette.TextDim)
	h.drawBest(dst, t, Overlay.X+Overlay.W-trialBestW-trialPad, h.briefBottom()-2)
}

func (h *HUD) drawTrialLabel(dst *ebiten.Image, font ui.Font, label, text string, x, y, w int, clr color.RGBA) int {
	seeLiteral.Draw(dst, label, x, y, 1, ui.Palette.Gold)
	indent := seeLiteral.Width("MEDALS", 1) + 2*ui.Advance
	for _, row := range font.Wrap(text, w-indent, 1) {
		font.Draw(dst, row, x+indent, y, 1, clr)
		y += ui.LineHeight
	}
	return y + 4
}

func (h *HUD) resultButtons() []Button {
	t := h.trialOf()
	specs := [][3]string{{TrialRetry, "RETRY", "Play this trial again from its start"}}
	if next, ok := t.Next(); ok {
		specs = append(specs, [3]string{TrialStart + ":" + next.ID, "NEXT TRIAL", "Play " + trialText(next.Title)})
	}
	specs = append(specs, [3]string{TrialBoard + ":" + t.ID, "ALL TRIALS", "The list of trials and your medals"},
		[3]string{TrialLeave, "BACK TO THE REALM", "Leave the trial: your realm waits as you left it"})
	return append(h.buttonRow(Overlay.X+trialPad, h.briefBottom(), specs), closeButtonAs(Overlay, trialPrefix+":close", "Close and look at the trial's land [Esc]"))
}

func resultTitle(v *TrialView) (string, color.RGBA) {
	switch {
	case v == nil:
		return "", ui.Palette.Text
	case v.Status == view.TrialLost:
		return "THE TRIAL IS LOST", ui.Palette.RedText
	case v.Result.Medal != view.MedalNone:
		return v.Result.Medal.Name() + " MEDAL", medalColor(v.Result.Medal)
	}
	return "TRIAL COMPLETED", ui.Palette.Cream
}

func (h *HUD) resultTop(v *TrialView) int {
	w := Overlay.W - 2*trialPad
	y := Overlay.Y + 28 + 2*ui.LineHeight
	y += len(seeLiteral.Wrap(h.resultLine(v), w, 1))*ui.LineHeight + 2
	closing := v.Trial.Lesson
	if v.Status == view.TrialLost {
		closing = v.Verdict.Why
	}
	return y + len(h.Font.Wrap(trialText(closing), w, 1))*ui.LineHeight + 2 + trialHeadGap
}

func (h *HUD) resultHeight() int {
	v := h.Trial.view
	if v == nil {
		return Overlay.H
	}
	w := Overlay.W - 2*trialPad
	body := h.resultTop(v) - Overlay.Y
	if v.Status == view.TrialLost {
		indent := seeLiteral.Width("MEDALS", 1) + 2*ui.Advance
		body += len(h.Font.Wrap(trialText(v.Trial.Hint), w-indent, 1))*ui.LineHeight + 4
	} else {
		body += seeItHeight(v.Trial.Kube(), w)
	}
	return min(body+TrialBtnH+24, Overlay.H)
}

func (h *HUD) drawResult(dst *ebiten.Image) {
	v := h.Trial.view
	if v == nil {
		return
	}
	t := v.Trial
	x, w := Overlay.X+trialPad, Overlay.W-2*trialPad
	h.header(dst, Overlay, strings.ToUpper(trialText(t.Title)))
	title, clr := resultTitle(v)
	seeLiteral.DrawShadow(dst, title, x, Overlay.Y+28, 2, clr)
	y := Overlay.Y + 28 + 2*ui.LineHeight
	for _, row := range seeLiteral.Wrap(h.resultLine(v), w, 1) {
		seeLiteral.Draw(dst, row, x, y, 1, ui.Palette.Text)
		y += ui.LineHeight
	}
	y += 2
	if v.Status == view.TrialLost {
		y = h.drawTextLines(dst, []string{trialText(v.Verdict.Why)}, x, y, w, ui.Palette.RedText) + 2
		h.drawTrialLabel(dst, h.Font, "HINT", trialText(t.Hint), x, y+trialHeadGap, w, ui.Palette.Text)
		return
	}
	y = h.drawTextLines(dst, []string{trialText(t.Lesson)}, x, y, w, ui.Palette.Cream) + 2
	h.drawSeeIt(dst, t.Kube(), x, y+trialHeadGap, w, h.briefBottom()-6)
}

func (h *HUD) resultLine(v *TrialView) string {
	t := v.Trial
	if v.Status == view.TrialLost {
		return fmt.Sprintf("Lost after %s. RETRY starts the trial afresh.", view.TrialClock(v.Elapsed))
	}
	line := fmt.Sprintf("Done in %s (%s).", view.TrialClock(v.Result.Seconds), medalTimes(t.Times))
	switch {
	case v.NewBest:
		line += " Your best yet."
	case v.HasBest && v.Best.Won:
		line += " Your best: " + view.TrialClock(v.Best.Seconds) + "."
	}
	return line
}
