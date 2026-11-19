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
	ChipH              = 12
	chipPad            = 20
	chipMaxW           = 288
	ObjectiveTop       = 17
	objectiveTextLines = 3
	objectiveHintLines = 4
	objectiveWhyLines  = 2
	objectiveGoalLines = 4
	ProgressLines      = 12
	doneMark           = "DONE  "
	ForMark            = "FOR  "
	objectiveHint      = "obj:hint"
	hintW              = 36
	hintTip            = "You have %s before: the step now names only the goal. Click for the full how-to."
)

func (f Frame) current() (view.ObjectiveView, bool) {
	for _, o := range f.State.Objectives {
		if !o.Done {
			return o, true
		}
	}
	return view.ObjectiveView{}, false
}

func (f Frame) ShownObjective() (view.ObjectiveView, bool) {
	if f.Guide.Held != "" {
		if f.Guide.Last != nil {
			return *f.Guide.Last, true
		}
		return view.ObjectiveView{}, false
	}
	if o, ok := f.current(); ok && !f.waitsForDecree(o) {
		return o, true
	}
	for i := len(f.State.Objectives) - 1; i >= 0; i-- {
		if o := f.State.Objectives[i]; o.Done {
			return o, true
		}
	}
	return view.ObjectiveView{}, false
}

func (f Frame) chipAhead() bool {
	o, ok := f.ShownObjective()
	return ok && !o.Done && o.Command != "" && f.Guide.Held == ""
}

func (f Frame) SpeakCommand() string {
	if !f.Revealed("stone") || !f.Shown(SurfaceStone) || f.Guide.Held != "" {
		return ""
	}
	l, _ := f.ObjectiveLine()
	return l.Command
}

func (h *HUD) ObjectiveShown(f Frame) bool {
	if h.trialShown() || h.teacherShown() {
		return true
	}
	_, ok := f.ShownObjective()
	return ok && !h.QuestCardShown(f)
}

func (h *HUD) ObjectiveDrawn(f Frame) bool {
	if h.trialShown() || h.teacherShown() {
		return h.CardsDrawn(f) && !h.underWindow(h.ObjectiveRectOf(f))
	}
	_, ok := f.ShownObjective()
	return ok && h.CardsDrawn(f) && !h.QuestCardShown(f)
}

func (h *HUD) underWindow(r ui.Rect) bool {
	w, open := h.windowRect()
	return open && Overlaps(Shadowed(r), w)
}

func (h *HUD) rawObjectiveLines(f Frame) []Line {
	o, ok := f.ShownObjective()
	if !ok || h.QuestCardShown(f) {
		return nil
	}
	if _, q, ok := f.questStep(); ok {
		return h.questLines(f, o, q)
	}
	if o.Done {
		return h.doneLines(f, o)
	}
	w := Objective.W - 16
	out := []Line{}
	for _, row := range h.Font.WrapLines(f.QuoteObjective(o.Text), w, 1, objectiveTextLines) {
		out = append(out, Line{Text: row, Clr: ui.Palette.Text})
	}
	if o.PastDemo {
		out = append(out, wrapped(h, "", view.ExperimentalNote, w, objectiveWhyLines, ui.Palette.TextDim)...)
	}
	out = append(out, wrapped(h, ForMark, o.Purpose, w, objectiveWhyLines, ui.Palette.Cream)...)
	if f.Guide.Why != "" && o.Purpose == "" {
		for _, row := range h.Font.WrapLines(f.Guide.Why, w, 1, objectiveWhyLines) {
			out = append(out, Line{Text: row, Clr: ui.Palette.Green})
		}
	}
	if o.Progress != "" {
		out = append(out, h.statusLines(f.QuoteObjective(h.readyFrom(f, o.Progress)), w, ProgressLines, ui.Palette.GoldBright)...)
	}
	if o.Earn != "" && !samePace(o.Progress, o.Earn) {
		out = append(out, h.statusLines(upperFirst(o.Earn), w, objectiveGoalLines, ui.Palette.TextDim)...)
	}
	if o.Practiced && h.HintOpen == o.ID {
		return append(out, wrapped(h, "", f.QuoteObjective(o.Hint), w, objectiveHintLines, ui.Palette.TextDim)...)
	}
	if f.Guide.Why != "" || o.Practiced {
		return out
	}
	for _, row := range h.Font.WrapLines(f.QuoteObjective(o.Hint), w, 1, objectiveHintLines) {
		out = append(out, Line{Text: row, Clr: ui.Palette.TextDim})
	}
	return append(out, h.ageGoalLines(f, w)...)
}

var backLabels = map[string]string{backStep: "BACK", SelClose: "X on this card"}

const paceWords = "the realm earns"

func (h *HUD) goalInView(f Frame) bool {
	return len(f.Beacons) > 0 && inView(f.Beacons[0])
}

func samePace(progress, earn string) bool {
	return strings.Contains(progress, paceWords) && strings.Contains(earn, paceWords) && !strings.Contains(earn, "Faster")
}

func (h *HUD) readyFrom(f Frame, progress string) string {
	rest, ready := strings.CutPrefix(progress, readyMark+" ")
	if !ready || f.Guide.Held != "" {
		return progress
	}
	goal := h.readyGoal(f)
	if MapTarget(goal) && f.GoalAt != nil && !f.GoalOfNext && !h.goalInView(f) {
		return readyMark + " press SHOW ME, then " + strings.TrimPrefix(rest, "press ")
	}
	path := NavPath(goal, h.NavState(f))
	if inner, ok := h.pastFirstPress(f, rest, path); ok {
		return readyMark + " press " + inner
	}
	if len(path) < 2 {
		return progress
	}
	lead, ok := backLabels[path[0]]
	if path[0] == SelClose && f.cardSelection() {
		if title := strings.TrimSpace(h.CardFor(f).Title); title != "" {
			lead = closeLead(title)
		}
	}
	if !ok && strings.HasSuffix(path[0], ":close") {
		lead, ok = "X on this panel", true
	}
	if !ok {
		return progress
	}
	return readyMark + " press " + lead + ", then " + strings.TrimPrefix(rest, "press ")
}

func (h *HUD) pastFirstPress(f Frame, rest string, path []string) (string, bool) {
	first, after, _ := strings.Cut(strings.TrimPrefix(rest, "press "), ", then ")
	if first != "BUILD" || h.PageID != PageBuild || len(path) == 0 {
		return "", false
	}
	if after != "" {
		return after, true
	}
	for _, b := range h.CommandButtons(f) {
		if b.ID == path[0] && b.Label != "" {
			return b.Label, true
		}
	}
	return "", false
}

func noLoneWord(font ui.Font, rows []string, width int) []string {
	n := len(rows)
	if n < 2 || strings.Contains(rows[n-1], " ") || strings.HasSuffix(rows[n-1], "...") || strings.Count(rows[n-2], " ") < 2 {
		return rows
	}
	cut := strings.LastIndex(rows[n-2], " ")
	moved := rows[n-2][cut+1:] + " " + rows[n-1]
	if font.Width(moved, 1) > width {
		return rows
	}
	out := append([]string(nil), rows...)
	out[n-2], out[n-1] = rows[n-2][:cut], moved
	return out
}

func (h *HUD) doneLines(f Frame, o view.ObjectiveView) []Line {
	w := Objective.W - 16
	out := []Line{}
	for i, row := range noLoneWord(h.Font, h.Font.WrapLines(f.QuoteObjective(o.Text), w-len(doneMark)*ui.Advance, 1, objectiveTextLines), w-len(doneMark)*ui.Advance) {
		if i == 0 {
			row = doneMark + row
		}
		out = append(out, Line{Text: row, Clr: ui.Palette.TextDim})
	}
	whyLines := 1
	if o.Major {
		whyLines = objectiveWhyLines
	}
	for _, row := range h.Font.WrapLines(o.Why, w, 1, whyLines) {
		out = append(out, Line{Text: row, Clr: ui.Palette.Green})
	}
	return append(out, wrapped(h, "", o.Lesson, w, objectiveHintLines, ui.Palette.Cream)...)
}

func (h *HUD) CardTitle(f Frame) (string, bool) {
	if _, q, ok := f.questStep(); ok {
		return strings.ToUpper(q.Title) + " " + questProgress(q), true
	}
	o, ok := f.ShownObjective()
	switch {
	case ok && o.Done:
		return lastStepTitle, false
	case ok && o.PastDemo:
		return experimentalStepTitle, false
	}
	return NextStepTitle, false
}

func (h *HUD) ObjectiveRect(f Frame) ui.Rect { return h.ObjectiveRectOf(f) }

func (h *HUD) ObjectiveRectOf(f Frame) ui.Rect {
	if h.teacherShown() {
		return h.TeacherRect(f)
	}
	if h.trialShown() {
		return h.TrialRect(f)
	}
	r := Objective
	r.H = ObjectiveTop + len(h.ObjectiveLines(f))*ui.LineHeight + 4 + h.objectiveExtras(f)
	return f.placed(PanelObjective, r)
}

func (h *HUD) objectiveExtras(f Frame) int {
	extra := 0
	if h.chipRoom(f) {
		extra += ChipH + 4
	}
	if h.BridgeRowShown(f) {
		extra += BridgeRowH
	}
	if h.nextStepShown(f) {
		extra += NextStepRowH
	}
	return extra
}

func titleX(decree bool) int {
	if decree {
		return 8 + 2*decreeSealR + 6
	}
	return 8
}

func ChipLabel(cmd string) string {
	return "kubectl " + strings.TrimPrefix(strings.TrimSpace(cmd), "kubectl ")
}

func ChipRect(card ui.Rect, cmd string) ui.Rect {
	w := min(len(ChipLabel(cmd))*ui.Advance+chipPad, chipMaxW)
	return ui.Rect{X: card.X + card.W - 6 - w, Y: card.Y + card.H - ChipH - 3, W: w, H: ChipH}
}

func showRect(card ui.Rect) ui.Rect {
	return ui.Rect{X: card.X + card.W - 58, Y: card.Y + 3, W: 54, H: 12}
}

func (h *HUD) ObjectiveButtons(f Frame) []Button {
	if !h.ObjectiveDrawn(f) {
		return nil
	}
	if h.teacherShown() {
		return h.teacherCardButtons(f)
	}
	if h.trialShown() {
		return h.trialCardButtons(f)
	}
	out := []Button{}
	if f.GoalAt != nil && !f.GoalOfNext {
		out = append(out, Button{ID: "obj:goal", Rect: showRect(h.ObjectiveRectOf(f)), Label: "SHOW ME"})
	}
	out = append(out, h.foldButton(f)...)
	out = append(out, h.objectiveHintButton(f)...)
	out = append(out, h.bridgeRowButtons(f)...)
	out = append(out, h.nextStepButton(f, h.ObjectiveRectOf(f))...)
	return append(append(out, h.objectiveAlmanacButton(f)...), h.objectiveLineButtons(f)...)
}

func (h *HUD) objectiveHintButton(f Frame) []Button {
	o, ok := f.current()
	if !ok || !o.Practiced || o.Hint == "" {
		return nil
	}
	show := showRect(h.ObjectiveRectOf(f))
	return []Button{{ID: objectiveHint, Rect: ui.Rect{X: show.X - hintW - 4, Y: show.Y, W: hintW, H: show.H}, Label: "HOW?", Tip: HintTip(o)}}
}

func closeLead(title string) string {
	return "X on the " + strings.TrimPrefix(title, "THE ") + " card"
}

func HintTip(o view.ObjectiveView) string {
	if o.DoneAct == "" {
		return "The step names only the goal. Click for the full how-to."
	}
	return fmt.Sprintf(hintTip, o.DoneAct)
}

func (h *HUD) guideFaded(f Frame) bool {
	o, ok := f.current()
	return ok && o.Practiced && h.Voice(f) == VoiceObjective
}

func (h *HUD) toggleHint(f Frame) []Intent {
	o, _ := f.current()
	if h.HintOpen == o.ID {
		h.HintOpen = ""
	} else {
		h.HintOpen = o.ID
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) pressObjective(rest string, f Frame) []Intent {
	if rest == "goal" && f.GoalAt != nil && !f.GoalOfNext {
		return h.focus(*f.GoalAt)
	}
	if rest == "almanac" {
		p, _ := objectivePage(f)
		return h.OpenBook(p.ID, f)
	}
	if rest == "fold" {
		return h.toggleFold()
	}
	if rest == "hint" {
		return h.toggleHint(f)
	}
	if rest == "next" {
		return h.pressNextStep(f)
	}
	l, _ := f.ObjectiveLine()
	if rest == "doit" {
		o, _ := f.ShownObjective()
		return h.doObjectiveStep(l, f.QuoteObjective(o.Text))
	}
	return h.UseLine(l, FromObjectiveCard)
}

func (h *HUD) drawObjective(dst *ebiten.Image, f Frame) {
	if h.teacherShown() {
		ui.InRegion(TeacherRegion, func() { h.drawTeacherCard(dst, f) })
		return
	}
	if h.trialShown() {
		h.DrawTrialCard(dst, f)
		return
	}
	lines := h.ObjectiveLines(f)
	if len(lines) == 0 {
		return
	}
	r := h.ObjectiveRectOf(f)
	title, decree := h.CardTitle(f)
	if decree {
		drawScroll(dst, r)
		drawSeal(dst, r.X+8+decreeSealR, r.Y+4+decreeSealR)
		h.drawFoldTitle(dst, r, title)
	} else {
		ui.Fill(dst, r.Offset(2, 3), color.RGBA{8, 11, 12, 180})
		ui.Fill(dst, r, ui.Tint(ui.Palette.Console, 235))
		ui.Stroke(dst, r, ui.Palette.Gold)
		h.Font.DrawShadow(dst, title, r.X+titleX(decree), r.Y+6, 1, ui.Palette.Gold)
	}
	y := r.Y + ObjectiveTop
	for _, l := range lines {
		h.Font.Draw(dst, l.Text, r.X+8, y, 1, l.Clr)
		y += ui.LineHeight
	}
	h.drawObjectiveLine(dst, f, r)
	h.drawBridgeRow(dst, f, decree)
	h.drawNextStep(dst, f, r)
	if f.GoalAt != nil && !f.GoalOfNext {
		b := Button{ID: "obj:goal", Rect: showRect(r), Label: "SHOW ME"}
		h.drawButton(dst, b)
	}
	for _, b := range append(h.objectiveAlmanacButton(f), h.objectiveHintButton(f)...) {
		h.drawButton(dst, b)
	}
}

func (h *HUD) drawChip(dst *ebiten.Image, r ui.Rect, label string, hot bool) {
	fill := consoleInk
	if hot {
		fill = ui.Palette.BlueDark
	}
	ui.Fill(dst, r, fill)
	ui.Stroke(dst, r, ui.Palette.ChipBorder)
	h.Font.Draw(dst, ">", r.X+4, r.Y+3, 1, ui.Palette.Gold)
	h.Font.Draw(dst, ui.Clip(label, ui.MaxChars(r.W-20, 1)), r.X+14, r.Y+3, 1, ui.Palette.Text)
}

func (h *HUD) shortStatus(f Frame, rows int) ([]Line, bool) {
	o, ok := f.ShownObjective()
	if !ok || o.Done || o.Progress == "" || h.QuestCardShown(f) {
		return nil, false
	}
	if _, _, quest := f.questStep(); quest {
		return nil, false
	}
	w := Objective.W - 16
	text := f.QuoteObjective(h.readyFrom(f, o.Progress))
	for {
		cut := strings.LastIndex(text, ": ")
		if cut <= strings.Index(text, ": ") {
			return nil, false
		}
		text = text[:cut]
		out := h.statusLines(text, w, ProgressLines, ui.Palette.GoldBright)
		if o.Earn != "" && !samePace(o.Progress, o.Earn) {
			out = append(out, h.statusLines(upperFirst(o.Earn), w, objectiveGoalLines, ui.Palette.TextDim)...)
		}
		if len(out) <= rows {
			return out, true
		}
	}
}
