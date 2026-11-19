package hud

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	TrialCardW     = 340
	TrialCardPad   = 8
	trialGoalLines = 2
	trialHintLines = 3
	trialPaceH     = 20
	trialStepW     = 14
	trialValueW    = 18
)

type trialCardLayout struct {
	Goal, hint []string `json:"-"`
	goalY      int
	progressY  int
	medalY     int
	hintY      int
	paceY      int
	buttonY    int
	height     int
}

func (h *HUD) TrialCardLayout() trialCardLayout {
	v := h.Trial.view
	w := TrialCardW - 2*TrialCardPad
	l := trialCardLayout{Goal: h.Font.WrapLines(trialText(v.Trial.Goal), w, 1, trialGoalLines)}
	y := TrialCardPad + ui.LineHeight + 3
	l.goalY = y
	y += len(l.Goal) * ui.LineHeight
	l.progressY = y
	y += ui.LineHeight
	l.medalY = y
	y += ui.LineHeight + 2
	if h.Trial.Hint {
		l.hint = h.Font.WrapLines(trialText(v.Trial.Hint), w, 1, trialHintLines)
		l.hintY = y
		y += len(l.hint)*ui.LineHeight + 2
	}
	if h.paceShown() {
		l.paceY = y
		y += trialPaceH
	}
	l.buttonY = y
	l.height = y + TrialBtnH + TrialCardPad - 2
	return l
}

func (h *HUD) paceShown() bool {
	v := h.Trial.view
	return v != nil && v.Trial.Pace && !(v.Status == view.TrialWon || v.Status == view.TrialLost)
}

func (h *HUD) TrialRect(f Frame) ui.Rect {
	r := Objective
	r.W, r.H = TrialCardW, h.TrialCardLayout().height
	return f.placed(PanelObjective, r)
}

func (h *HUD) trialCardButtons(f Frame) []Button {
	v := h.Trial.view
	r := h.TrialRect(f)
	l := h.TrialCardLayout()
	out := []Button{}
	if h.paceShown() {
		out = append(out, h.PaceButtons(r.X+TrialCardPad, r.Y+l.paceY)...)
	}
	var specs [][3]string
	switch v.Status {
	case view.TrialBriefing:
		specs = [][3]string{{TrialBegin, "START", "Start the trial: its clock starts now and does not pause"}, {"brief", "BRIEF", "Read the trial's brief again"}}
	case view.TrialRunning:
		specs = [][3]string{{"hint", hintLabel(h.Trial.Hint), "Show or hide how to win this trial"}, {TrialRetry, "RETRY", "Start this trial afresh"}}
	default:
		specs = [][3]string{{"result", "RESULT", "The medal and what it looks like in Kubernetes"}, {TrialRetry, "RETRY", "Play this trial again"}}
	}
	specs = append(specs, [3]string{TrialLeave, "LEAVE", "Back to your realm: it waits as you left it"})
	return append(out, h.buttonRow(r.X+TrialCardPad, r.Y+l.buttonY, specs)...)
}

func hintLabel(open bool) string {
	if open {
		return "HIDE HINT"
	}
	return "HINT"
}

func (h *HUD) PaceButtons(x, y int) []Button {
	p := h.Trial.Pace
	out := []Button{}
	x += h.Font.Width("PACE", 1) + 2*ui.Advance
	for _, field := range []struct {
		key, label, tip string
		value           int
	}{
		{"surge", paceLabel("surge"), ui.Bridge("The Vanguard: new huts the shift change may raise above the count before old ones go", "maxSurge: new Pods the rollout may create above the replica count"), p.Surge},
		{"gaps", paceLabel("gaps"), ui.Bridge("The Allowed Gaps: huts that may stand down at once, so the count may dip by this much", "maxUnavailable: Pods that may be unavailable at once during the rollout"), p.Gaps},
	} {
		x += h.Font.Width(field.label, 1) + ui.Advance
		out = append(out,
			Button{ID: trialPrefix + ":step:" + field.key + ":down", Label: "-", Tip: "Fewer. " + field.tip, Rect: ui.Rect{X: x, Y: y, W: trialStepW, H: trialStepW}},
			Button{ID: trialPrefix + ":step:" + field.key + ":up", Label: "+", Tip: "More. " + field.tip, Rect: ui.Rect{X: x + trialStepW + trialValueW, Y: y, W: trialStepW, H: trialStepW}})
		x += 2*trialStepW + trialValueW + 2*ui.Advance
	}
	set := Button{ID: trialPrefix + ":" + TrialPace, Label: "SET", Rect: ui.Rect{X: x, Y: y - 2, W: h.Font.Width("SET", 1) + 2*TrialBtnPad, H: TrialBtnH},
		Tip: "Write this pace into the porters' crew\n$ " + view.PaceCommand("porters", strconv.Itoa(p.Surge), strconv.Itoa(p.Gaps))}
	switch {
	case p.Surge+p.Gaps == 0:
		set.Disabled, set.Tip = true, "The Vanguard and the Gaps cannot both be 0: the shift could never change."
	case h.Trial.view.Status != view.TrialRunning:
		set.Disabled, set.Tip = true, "Press START first: the trial's crew listens once its clock runs."
	case !p.Stepped && p.Surge == h.Trial.view.Surge && p.Gaps == h.Trial.view.Gaps:
		set.Disabled, set.Tip = true, "Step the Vanguard or the Gaps first: SET writes them into the crew."
	}
	return append(out, set)
}

func (h *HUD) DrawTrialCard(dst *ebiten.Image, f Frame) {
	v := h.Trial.view
	r := h.TrialRect(f)
	l := h.TrialCardLayout()
	ui.Fill(dst, r.Offset(2, 3), ui.Tint(ui.Palette.Shadow, 180))
	ui.Fill(dst, r, ui.Tint(ui.Palette.Console, 235))
	ui.Stroke(dst, r, ui.Palette.Gold)
	x, w := r.X+TrialCardPad, r.W-2*TrialCardPad
	clock := view.TrialClock(v.Elapsed)
	cw := h.Font.Width(clock, 1)
	h.Font.DrawShadow(dst, ui.Clip(strings.ToUpper(trialText(v.Trial.Title)), ui.MaxChars(w-cw-ui.Advance, 1)), x, r.Y+TrialCardPad, 1, ui.Palette.Gold)
	h.Font.DrawShadow(dst, clock, r.X+r.W-TrialCardPad-cw, r.Y+TrialCardPad, 1, ui.Palette.GoldBright)
	for i, row := range l.Goal {
		h.Font.Draw(dst, row, x, r.Y+l.goalY+i*ui.LineHeight, 1, ui.Palette.Cream)
	}
	progress := ui.Palette.RedText
	if v.Verdict.Good {
		progress = ui.Palette.Green
	}
	h.Font.Draw(dst, ui.Clip(trialText(v.Verdict.Progress), ui.MaxChars(w, 1)), x, r.Y+l.progressY, 1, progress)
	medal, clr := MedalLine(v)
	seeLiteral.Draw(dst, seeLiteral.Clip(medal, ui.MaxChars(w, 1)), x, r.Y+l.medalY, 1, clr)
	for i, row := range l.hint {
		h.Font.Draw(dst, row, x, r.Y+l.hintY+i*ui.LineHeight, 1, ui.Palette.TextDim)
	}
	if h.paceShown() {
		h.drawPace(dst, x, r.Y+l.paceY)
	}
	h.drawTrialButtons(dst, h.trialCardButtons(f))
}

func MedalLine(v *TrialView) (string, color.RGBA) {
	t := v.Trial
	switch v.Status {
	case view.TrialBriefing:
		return "The clock waits for START: " + strings.ToLower(medalTimes(t.Times)), ui.Palette.TextDim
	case view.TrialLost:
		return "Lost after " + view.TrialClock(v.Elapsed) + ": RETRY starts afresh", ui.Palette.RedText
	case view.TrialWon:
		if v.Result.Medal == view.MedalNone {
			return "Completed in " + view.TrialClock(v.Result.Seconds) + ", past the bronze time", ui.Palette.Cream
		}
		return fmt.Sprintf("%s MEDAL in %s", v.Result.Medal.Name(), view.TrialClock(v.Result.Seconds)), medalColor(v.Result.Medal)
	}
	now := t.Times.Medal(v.Elapsed)
	if now == view.MedalNone {
		return "Past the bronze time: finish anyway, or RETRY", ui.Palette.TextDim
	}
	return fmt.Sprintf("On pace for %s: under %s", now.Name(), view.TrialClock(t.Times.Limit(now))), medalColor(now)
}

func (h *HUD) drawPace(dst *ebiten.Image, x, y int) {
	p := h.Trial.Pace
	h.Font.Draw(dst, "PACE", x, y+3, 1, ui.Palette.Gold)
	for _, b := range h.PaceButtons(x, y) {
		if !strings.HasSuffix(b.ID, ":down") {
			continue
		}
		label, value := paceLabel("surge"), p.Surge
		if strings.Contains(b.ID, ":gaps:") {
			label, value = paceLabel("gaps"), p.Gaps
		}
		h.Font.Draw(dst, label, b.Rect.X-h.Font.Width(label, 1)-ui.Advance, y+3, 1, ui.Palette.Text)
		h.Font.DrawCentered(dst, strconv.Itoa(value), ui.Rect{X: b.Rect.X + trialStepW, Y: y, W: trialValueW, H: trialStepW}, 1, ui.Palette.GoldBright)
	}
}

func paceLabel(field string) string {
	if field == "gaps" {
		return ui.Bridge("GAPS", "UNAVAILABLE")
	}
	return ui.Bridge("VANGUARD", "SURGE")
}
