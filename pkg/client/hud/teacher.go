package hud

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	TeacherW           = 340
	TeacherPad         = 8
	teacherPortrait    = 48
	TeacherColX        = TeacherPad + teacherPortrait + 8
	TeacherLabelW      = 46
	TeacherRowMax      = 3
	teacherChipH       = 16
	teacherName        = "MAGISTER BORG"
	TeacherBeatGap     = 6
	teacherProgressMax = 3
	teacherTextMax     = 40
	TeacherHappened    = "WHAT JUST HAPPENED"
	teacherChipText    = 20
)

type teacherLayout struct {
	Header         string   `json:"-"`
	Beat           string   `json:"-"`
	Body           []string `json:"-"`
	bodyGreen      bool
	Why, Try       []string `json:"-"`
	Happened       []string `json:"-"`
	happenedY      int
	Course         []string `json:"-"`
	Progress       []string `json:"-"`
	Chip           []string `json:"-"`
	Ask            string   `json:"-"`
	askY           int
	choices        [][]Button
	choicesY       int
	bodyY, rowsY   int
	ProgressY      int `json:"-"`
	ChipY, ButtonY int `json:"-"`
	Height         int `json:"-"`
}

func (v TeacherView) Header() string {
	switch v.Status {
	case view.ChapterBriefing:
		return fmt.Sprintf("CHAPTER %d  BRIEFING", v.Chapter.Number)
	case view.ChapterComplete:
		return "CHAPTER COMPLETE"
	}
	return fmt.Sprintf("CHAPTER %d  STEP %d OF %d", v.Chapter.Number, v.Step+1, len(v.Chapter.Steps))
}

func (v TeacherView) bodyText() (string, bool) {
	switch {
	case v.Status == view.ChapterBriefing:
		return strings.Join(v.Chapter.Briefing, " "), false
	case v.Status == view.ChapterComplete:
		return v.Lesson.Praise + " " + v.Chapter.Debrief.Pain, false
	case v.Soaking:
		return v.Lesson.Praise, true
	case v.walking():
		return strings.TrimSpace(v.Verdict + " " + v.Hop.Line), false
	}
	s, _ := v.Current()
	if v.Armed && s.Warn != "" {
		return s.Warn, false
	}
	return strings.Join(s.Lines(), " "), false
}

func (v TeacherView) happened() string {
	switch {
	case v.Status == view.ChapterRunning && v.Soaking && v.LessonStep == v.Step:
		return v.Lesson.Fact
	}
	return ""
}

func (v TeacherView) predicting() bool {
	s := v.step()
	return v.Armed && s.Beat == view.BreakIt && s.Warn != ""
}

func (v TeacherView) running() bool {
	return v.Status == view.ChapterRunning && !v.Soaking
}

func (v TeacherView) step() view.TutorialStep {
	s, _ := v.Current()
	return s
}

func (h *HUD) TeacherLayout() teacherLayout {
	v := *h.Yard.View
	colW := TeacherW - TeacherColX - TeacherPad
	fullW := TeacherW - 2*TeacherPad
	text, green := v.bodyText()
	l := teacherLayout{Header: v.Header(), Body: Literal.WrapLines(text, colW, 1, teacherTextMax), bodyGreen: green}
	if v.Status == view.ChapterRunning {
		l.Beat = v.step().Beat
	}
	y := TeacherPad + 2*ui.LineHeight + 2
	l.bodyY = y
	y = max(y+len(l.Body)*ui.LineHeight, TeacherPad+teacherPortrait) + 4
	if fact := v.happened(); fact != "" {
		l.happenedY = y
		l.Happened = Literal.WrapLines(fact, fullW, 1, teacherTextMax)
		y += (1+len(l.Happened))*ui.LineHeight + 4
	}
	l.rowsY = y
	switch {
	case v.Status == view.ChapterBriefing:
		l.Course = Literal.WrapLines(v.Chapter.Course, fullW-TeacherLabelW, 1, 2)
		y += len(l.Course) * ui.LineHeight
	case v.running() && v.step().Spotlit() && !v.walking() && !v.predicting():
		s := v.step()
		l.Why = Literal.WrapLines(s.Why, fullW-TeacherLabelW, 1, TeacherRowMax)
		l.Try = Literal.WrapLines(s.Try, fullW-TeacherLabelW, 1, 2)
		y += (len(l.Why) + len(l.Try)) * ui.LineHeight
	}
	if v.Status == view.ChapterRunning {
		l.ProgressY = y + 2
		l.Progress = Literal.WrapLines(h.Yard.Line, TeacherW-2*TeacherPad, 1, teacherProgressMax)
		y += max(len(l.Progress), h.Yard.rows, 1)*ui.LineHeight + 4
	}
	if v.running() && v.Chip != "" {
		l.ChipY = y
		l.Chip = Literal.Wrap(v.Chip, TeacherW-2*TeacherPad-teacherChipText, 1)
		y += teacherChipRect(l.Chip, 0, 0).H + 4
	}
	if v.running() && v.Asking && v.Question.Picked < 0 {
		l.askY = y
		l.Ask = v.Question.Ask
		y += ui.LineHeight + 2
		l.choices = h.choiceRows(v.Question.Choices, TeacherW-2*TeacherPad)
		l.choicesY = y
		y += len(l.choices) * (TrialBtnH + 4)
	}
	l.ButtonY = y
	l.Height = y + TrialBtnH + TeacherPad
	return l
}

func (h *HUD) choiceRows(choices []string, width int) [][]Button {
	rows := [][]Button{}
	row, x := []Button{}, 0
	for i, c := range choices {
		b := h.teacherButton(fmt.Sprintf("%s:%d", TeacherWho, i+1), c, "Say who writes the next part", 0, 0)
		if x > 0 && x+b.Rect.W > width {
			rows, row, x = append(rows, row), []Button{}, 0
		}
		b.Rect.X = x
		row = append(row, b)
		x += b.Rect.W + 4
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	return rows
}

func (h *HUD) TeacherRect(f Frame) ui.Rect {
	r := Objective
	r.W, r.H = TeacherW, h.TeacherLayout().Height
	return f.placed(PanelObjective, r)
}

func (h *HUD) teacherButton(id, label, tip string, x, y int) Button {
	return Button{ID: TeacherPrefix + ":" + id, Label: label, Tip: tip, Rect: ui.Rect{X: x, Y: y, W: h.Font.Width(label, 1) + 2*TrialBtnPad, H: TrialBtnH}}
}

func (h *HUD) teacherCardButtons(f Frame) []Button {
	v := *h.Yard.View
	r := h.TeacherRect(f)
	l := h.TeacherLayout()
	out := []Button{}
	if l.ChipY > 0 {
		out = append(out, Button{ID: TeacherPrefix + ":" + TeacherChip, Label: v.Chip, Tip: "Write this command on the command line, ready to run: Enter runs it",
			Rect: teacherChipRect(l.Chip, r.X+TeacherPad, r.Y+l.ChipY)})
	}
	specs := [][3]string{}
	switch v.Status {
	case view.ChapterBriefing:
		specs = append(specs, [3]string{TeacherBegin, "BEGIN", "Begin the chapter: Borg gives the first order"}, [3]string{TeacherAlmanac, "ALMANAC", "Read this chapter's first page in the Almanac"})
	case view.ChapterComplete:
		specs = append(specs, [3]string{TeacherDebrief, "DEBRIEF", "The verification command, the rule, and how to do it for real"})
		if v.HasNext {
			specs = append(specs, [3]string{TeacherNextChapter, "NEXT CHAPTER", "Start " + v.Next.Label()})
		}
		specs = append(specs, [3]string{TeacherMap, "MAP", "Back to the chapter map"})
	default:
		s := v.step()
		if v.running() && s.Spotlit() && len(s.Targets) > 0 {
			specs = append(specs, [3]string{TeacherShow, "SHOW ME", "Glide the camera to what Borg names, one by one"})
		}
		if s.Page != "" && v.running() {
			specs = append(specs, [3]string{TeacherAlmanac, "ALMANAC", "Read this step's page in the Almanac"})
		}
		if v.Closing {
			specs = append(specs, [3]string{TeacherDebrief, "DEBRIEF", "The chapter's debrief: the verification command, the rule, and how to do it for real"})
		} else if v.Soaking || s.Talk || v.Armed || v.LastHop {
			specs = append(specs, [3]string{TeacherNext, "NEXT", "Go on to Borg's next order"})
		}
	}
	for row, buttons := range l.choices {
		for _, b := range buttons {
			b.Rect.X += r.X + TeacherPad
			b.Rect.Y = r.Y + l.choicesY + row*(TrialBtnH+4)
			out = append(out, b)
		}
	}
	x, y := r.X+TeacherPad, r.Y+l.ButtonY
	for _, s := range specs {
		b := h.teacherButton(s[0], s[1], s[2], x, y)
		out = append(out, b)
		x += b.Rect.W + 4
	}
	leave := h.teacherButton(TeacherLeave, "LEAVE", "Leave the chapter: it is never saved, and START begins it afresh", 0, y)
	leave.Rect.X = r.X + r.W - TeacherPad - leave.Rect.W
	return append(out, leave)
}

func teacherChipRect(rows []string, x, y int) ui.Rect {
	w := TeacherW - 2*TeacherPad
	if len(rows) == 1 {
		w = min(Literal.Width(rows[0], 1)+teacherChipText, w)
	}
	return ui.Rect{X: x, Y: y, W: w, H: teacherChipH + max(len(rows)-1, 0)*ui.LineHeight}
}

func (h *HUD) drawTeacherChip(dst *ebiten.Image, r ui.Rect, rows []string, hot bool) {
	fill := consoleInk
	if hot {
		fill = ui.Palette.BlueDark
	}
	ui.Fill(dst, r, fill)
	ui.Stroke(dst, r, ui.Palette.ChipBorder)
	h.Font.Draw(dst, ">", r.X+4, r.Y+3, 1, ui.Palette.Gold)
	for i, row := range rows {
		Literal.Draw(dst, row, r.X+14, r.Y+3+i*ui.LineHeight, 1, ui.Palette.Text)
	}
}

func (h *HUD) teacherPrimary(id string) bool {
	v := h.Yard.View
	switch id {
	case TeacherPrefix + ":" + TeacherBegin, TeacherPrefix + ":" + TeacherDebrief:
		return true
	case TeacherPrefix + ":" + TeacherNext:
		return v != nil && !v.Soaking
	}
	return false
}

func (h *HUD) TeacherHUDGoal() string {
	v := h.Yard.View
	if v == nil || !v.running() {
		return ""
	}
	s := v.step()
	if !s.Spotlit() {
		return ""
	}
	goal := ""
	for _, t := range s.Targets {
		if MapTarget(t) || strings.HasPrefix(t, "plot:") || strings.HasPrefix(t, "stone:") {
			continue
		}
		goal = t
	}
	return goal
}

func TeacherWorldTargets(v *TeacherView) []string {
	if v != nil && v.walking() {
		return []string{v.Hop.Target}
	}
	if v == nil || !v.running() || !v.step().Spotlit() {
		return nil
	}
	out := []string{}
	for _, t := range v.step().Targets {
		if MapTarget(t) || strings.HasPrefix(t, "stone:") {
			out = append(out, t)
		}
	}
	return out
}
