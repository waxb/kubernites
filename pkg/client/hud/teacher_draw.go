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
	debriefW     = 720
	debriefPad   = 20
	debriefLabel = 130
)

func (h *HUD) drawTeacherCard(dst *ebiten.Image, f Frame) {
	v := *h.Yard.View
	r := h.TeacherRect(f)
	l := h.TeacherLayout()
	ui.Fill(dst, r.Offset(2, 3), ui.Tint(ui.Palette.Shadow, 180))
	ui.Fill(dst, r, ui.Tint(ui.Palette.Console, 235))
	ui.Stroke(dst, r, ui.Palette.Gold)
	portrait := ui.Rect{X: r.X + TeacherPad, Y: r.Y + TeacherPad, W: teacherPortrait, H: teacherPortrait}
	if !h.Lib.Portrait(dst, "advisor-borg", portrait) {
		ui.Fill(dst, portrait, ui.Palette.BlueDark)
	}
	x, colW := r.X+TeacherColX, r.W-TeacherColX-TeacherPad
	Literal.DrawShadow(dst, teacherName, x, r.Y+TeacherPad, 1, ui.Palette.Gold)
	if v.Status == view.ChapterRunning {
		h.drawTeacherPips(dst, v, r.X+r.W-TeacherPad, r.Y+TeacherPad)
	}
	headClr := ui.Palette.TextDim
	if v.Status == view.ChapterComplete {
		headClr = ui.Palette.GoldBright
	}
	headW := colW
	if l.Beat != "" {
		bw := Literal.Width(l.Beat, 1)
		headW -= bw + TeacherBeatGap
		Literal.Draw(dst, l.Beat, x+colW-bw, r.Y+TeacherPad+ui.LineHeight, 1, ui.Palette.GoldBright)
	}
	Literal.Draw(dst, Literal.Clip(l.Header, ui.MaxChars(headW, 1)), x, r.Y+TeacherPad+ui.LineHeight, 1, headClr)
	bodyClr := ui.Palette.Text
	if l.bodyGreen {
		bodyClr = ui.Palette.Green
	}
	for i, row := range l.Body {
		Literal.Draw(dst, row, x, r.Y+l.bodyY+i*ui.LineHeight, 1, bodyClr)
	}
	lx := r.X + TeacherPad
	if len(l.Happened) > 0 {
		Literal.Draw(dst, TeacherHappened, lx, r.Y+l.happenedY, 1, ui.Palette.Gold)
		for i, row := range l.Happened {
			Literal.Draw(dst, row, lx, r.Y+l.happenedY+(i+1)*ui.LineHeight, 1, ui.Palette.Cream)
		}
	}
	y := r.Y + l.rowsY
	y = drawLabelled(dst, "COURSE", l.Course, lx, y, ui.Palette.TextDim, ui.Palette.TextDim)
	y = drawLabelled(dst, "WHY", l.Why, lx, y, ui.Palette.TextDim, ui.Palette.TextDim)
	drawLabelled(dst, "TRY", l.Try, lx, y, ui.Palette.Green, ui.Palette.Green)
	for i, line := range l.Progress {
		clr := ui.Palette.Text
		if v.Progress.Met {
			clr = ui.Palette.Green
		}
		Literal.Draw(dst, line, lx, r.Y+l.ProgressY+i*ui.LineHeight, 1, clr)
	}
	if l.Ask != "" {
		Literal.Draw(dst, l.Ask, lx, r.Y+l.askY, 1, ui.Palette.GoldBright)
	}
	for _, b := range h.teacherCardButtons(f) {
		switch {
		case b.ID == TeacherPrefix+":"+TeacherChip:
			h.drawTeacherChip(dst, b.Rect, l.Chip, h.Hovering && h.Hover.ID == b.ID)
		case h.teacherPrimary(b.ID):
			h.drawPrimary(dst, b)
		default:
			h.drawButton(dst, b)
		}
	}
}

func drawLabelled(dst *ebiten.Image, label string, rows []string, x, y int, labelClr, textClr color.RGBA) int {
	for i, row := range rows {
		if i == 0 {
			Literal.Draw(dst, label, x, y, 1, labelClr)
		}
		Literal.Draw(dst, row, x+TeacherLabelW, y, 1, textClr)
		y += ui.LineHeight
	}
	return y
}

func (h *HUD) drawTeacherPips(dst *ebiten.Image, v TeacherView, right, y int) {
	count := fmt.Sprintf("%d / %d", v.Step+1, len(v.Chapter.Steps))
	w := Literal.Width(count, 1)
	Literal.Draw(dst, count, right-w, y, 1, ui.Palette.GoldBright)
	x := right - w - 6 - len(v.Chapter.Steps)*6
	for i := range v.Chapter.Steps {
		clr := ui.Palette.Muted
		if i <= v.Step {
			clr = ui.Palette.Gold
		}
		ui.Fill(dst, ui.Rect{X: x + i*6, Y: y + 3, W: 4, H: 3}, clr)
	}
}

func (h *HUD) teacherLit(f Frame) (Button, bool) {
	v := h.Yard.View
	if v == nil || h.OverlayID != OverlayNone || f.Placing != "" {
		return Button{}, false
	}
	goal := h.TeacherHUDGoal()
	if goal == "" || raising(goal, f) {
		return Button{}, false
	}
	if strings.HasPrefix(goal, TeacherPrefix+":") {
		return HitID(h.teacherCardButtons(f), goal)
	}
	if s, ok := h.stepOn(NavPath(goal, h.NavState(f)), f); ok && !s.Map {
		return Button{ID: s.ID, Rect: s.Rect}, true
	}
	return Button{}, false
}

func raising(goal string, f Frame) bool {
	kind, ok := strings.CutPrefix(goal, "build:")
	if !ok {
		return false
	}
	for _, s := range f.Site.Structures {
		if s.Kind == kind {
			return true
		}
	}
	return false
}

func (h *HUD) drawTeacherBeacons(dst *ebiten.Image, f Frame) {
	if b, ok := h.teacherLit(f); ok {
		h.drawBeacon(dst, b.Rect)
	}
}

func (h *HUD) Lit(f Frame) string {
	f = h.Stage(f)
	if h.Yard.View != nil {
		b, _ := h.teacherLit(f)
		return b.ID
	}
	return h.StepID(f)
}

func (h *HUD) DebriefRect() ui.Rect {
	if h.Yard.View == nil {
		return Overlay
	}
	height := h.debriefHeight()
	if h.Yard.PageID == 0 {
		height = h.DebriefLesson().height
	}
	return ui.Rect{X: Overlay.X, Y: Overlay.Y, W: debriefW, H: height}
}

func (h *HUD) debriefKube() view.TrialKube {
	d := h.Yard.View.Chapter.Debrief
	return view.TrialKube{Commands: d.Verify, Rule: d.Rule, Doc: d.Doc}
}

type debriefRow struct {
	Label  string   `json:"-"`
	Lines  []string `json:"-"`
	clr    color.RGBA
	Y      int `json:"-"`
	Struck int `json:"-"`
}

func (h *HUD) debriefRows() []debriefRow {
	v := h.Yard.View
	d := v.Chapter.Debrief
	w := debriefW - 2*debriefPad - debriefLabel
	rows := []debriefRow{}
	add := func(label, text string, clr color.RGBA) {
		if text != "" {
			rows = append(rows, debriefRow{Label: label, Lines: Literal.WrapLines(text, w, 1, 4), clr: clr})
		}
	}
	add("KUBEADM DECLARED", strings.Join(d.Kinds, " "), ui.Palette.Text)
	add("DO IT FOR REAL", d.Real, ui.Palette.Blue)
	add("SKIPPED HERE", d.Skips, ui.Palette.TextDim)
	add("NEXT", d.Pain, ui.Palette.Text)
	if t, ok := view.FindTrial(v.Chapter.Trial); ok {
		add("PRACTICE", t.Title.K8s+": MENU, then TRIALS, for a medal.", ui.Palette.Text)
	}
	return rows
}

func (h *HUD) debriefHeight() int {
	y := debriefPad + 3*ui.LineHeight + 8
	y += seeItHeight(h.debriefKube(), debriefW-2*debriefPad) + 6
	for _, r := range h.debriefRows() {
		y += len(r.Lines)*ui.LineHeight + 4
	}
	return y + TrialBtnH + debriefPad + 4
}

func (h *HUD) DebriefButtons(Frame) []Button {
	if h.Yard.View == nil {
		return []Button{closeButtonAs(h.DebriefRect(), DebriefPrefix+":close", "Close the debrief [Esc]")}
	}
	r := h.DebriefRect()
	x, y := r.X+debriefPad, r.Y+r.H-debriefPad-TrialBtnH
	out := []Button{}
	specs := [][3]string{{DebriefSee, seeItTitle, "The verification commands, the rule, and how to do it for real"}}
	if h.Yard.PageID == 0 {
		for _, q := range h.DebriefLesson().Questions {
			out = append(out, q.choices...)
		}
	} else {
		specs = [][3]string{{DebriefBack, "BACK", "Back to the chapter in one sentence and CHECK YOURSELF"}}
	}
	if h.Yard.View.HasNext {
		specs = append(specs, [3]string{TeacherNextChapter, "NEXT CHAPTER", "Start " + h.Yard.View.Next.Label()})
	}
	specs = append(specs, [3]string{TeacherMap, "MAP", "Back to the chapter map"})
	for _, s := range specs {
		b := Button{ID: DebriefPrefix + ":" + s[0], Label: s[1], Tip: s[2], Rect: ui.Rect{X: x, Y: y, W: h.Font.Width(s[1], 1) + 2*TrialBtnPad, H: TrialBtnH}}
		out = append(out, b)
		x += b.Rect.W + trialBtnGap
	}
	return append(out, closeButtonAs(r, DebriefPrefix+":close", "Close the debrief [Esc]"))
}

func (h *HUD) drawDebrief(dst *ebiten.Image, f Frame) {
	v := h.Yard.View
	if v == nil {
		return
	}
	r := h.DebriefRect()
	ui.DarkPanel(dst, r)
	x, y := r.X+debriefPad, r.Y+debriefPad
	Literal.DrawShadow(dst, fmt.Sprintf("CHAPTER %d COMPLETE", v.Chapter.Number), x, y, 2, ui.Palette.Gold)
	y += 2*ui.LineHeight + 2
	Literal.Draw(dst, Literal.Clip(v.Chapter.Title, ui.MaxChars(r.W-2*debriefPad, 1)), x, y, 1, ui.Palette.TextDim)
	y += ui.LineHeight + 6
	if h.Yard.PageID == 0 {
		h.drawDebriefLesson(dst, x, r.Y)
		h.drawDebriefButtons(dst, f)
		return
	}
	y = h.drawSeeIt(dst, h.debriefKube(), x, y, r.W-2*debriefPad, r.Y+r.H) + 6
	for _, row := range h.debriefRows() {
		Literal.Draw(dst, row.Label, x, y, 1, ui.Palette.Gold)
		for i, line := range row.Lines {
			Literal.Draw(dst, line, x+debriefLabel, y+i*ui.LineHeight, 1, row.clr)
		}
		y += len(row.Lines)*ui.LineHeight + 4
	}
	h.drawDebriefButtons(dst, f)
}

func (h *HUD) drawDebriefButtons(dst *ebiten.Image, f Frame) {
	for _, b := range h.DebriefButtons(f) {
		if strings.HasPrefix(b.ID, DebriefPrefix+":"+DebriefCheck+":") {
			continue
		}
		if IsClose(b) {
			h.drawButton(dst, Button{ID: b.ID, Rect: b.Face, Label: b.Label})
			continue
		}
		if b.ID == DebriefPrefix+":"+TeacherNextChapter {
			h.drawPrimary(dst, b)
			continue
		}
		h.drawButton(dst, b)
	}
}
