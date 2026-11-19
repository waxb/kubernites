package hud

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	DebriefSee       = "see"
	DebriefBack      = "back"
	DebriefCheck     = "check"
	DebriefSentence  = "IN ONE SENTENCE"
	DebriefMyth      = "MYTH / REALITY"
	debriefChecks    = "CHECK YOURSELF"
	debriefAskRows   = 2
	debriefWhyRows   = 2
	debriefGap       = 4
	DebriefChoicePad = 8
	pickedMark       = "[x] "
)

type debriefQuestion struct {
	Ask     []string `json:"-"`
	askY    int
	choices []Button
	Answer  []string `json:"-"`
	answerY int
	Right   bool `json:"-"`
}

type debriefLesson struct {
	Rows      []debriefRow      `json:"-"`
	CheckY    int               `json:"-"`
	Questions []debriefQuestion `json:"-"`
	height    int
}

func (h *HUD) Answer(q, c int) {
	if h.Yard.answers == nil {
		h.Yard.answers = map[int]int{}
	}
	h.Yard.answers[q] = c
}

func (h *HUD) DebriefLesson() debriefLesson {
	d := h.Yard.View.Chapter.Debrief
	x, top := Overlay.X+debriefPad, Overlay.Y
	full := debriefW - 2*debriefPad
	w := full - debriefLabel
	y := debriefPad + 3*ui.LineHeight + 8
	l := debriefLesson{}
	sentence := debriefRow{Label: DebriefSentence, Lines: Literal.WrapLines(d.Sentence, w, 1, 4), clr: ui.Palette.Text, Y: y}
	l.Rows = append(l.Rows, sentence)
	y += len(sentence.Lines)*ui.LineHeight + debriefGap + 2
	belief := Literal.WrapLines(d.Myth.Belief, w, 1, 2)
	myth := debriefRow{Label: DebriefMyth, Lines: append(belief, Literal.WrapLines(d.Myth.Truth, w, 1, 4)...), clr: ui.Palette.Text, Y: y, Struck: len(belief)}
	l.Rows = append(l.Rows, myth)
	y += len(myth.Lines)*ui.LineHeight + debriefGap + 2
	l.CheckY = y
	y += ui.LineHeight + 2
	cw := (full - 2*debriefGap) / 3
	for qi, q := range d.Check {
		dq := debriefQuestion{Ask: Literal.WrapLines(q.Ask, full, 1, debriefAskRows), askY: y}
		y += len(dq.Ask)*ui.LineHeight + 2
		picked, answered := h.Yard.answers[qi]
		for ci, ch := range q.Choices {
			label := ch.Text
			if answered && picked == ci {
				label = pickedMark + label
			}
			b := Button{ID: fmt.Sprintf("%s:%s:%d:%d", DebriefPrefix, DebriefCheck, qi+1, ci+1), Label: label, Tip: "Answer: the debrief says whether it is right, and why",
				Rect: ui.Rect{X: x + ci*(cw+debriefGap), Y: top + y, W: cw, H: TrialBtnH}}
			b.Selected = answered && picked == ci && ch.Right
			dq.choices = append(dq.choices, b)
		}
		y += TrialBtnH + 3
		dq.answerY = y
		if answered && picked >= 0 && picked < len(q.Choices) {
			dq.Answer = Literal.WrapLines(q.Choices[picked].Why, full, 1, debriefWhyRows)
			dq.Right = q.Choices[picked].Right
		}
		y += debriefWhyRows*ui.LineHeight + debriefGap
		l.Questions = append(l.Questions, dq)
	}
	l.height = y + TrialBtnH + debriefPad
	return l
}

func (h *HUD) pressCheck(rest string) {
	q, c, ok := strings.Cut(rest, ":")
	if !ok {
		return
	}
	qi, err1 := strconv.Atoi(q)
	ci, err2 := strconv.Atoi(c)
	v := h.Yard.View
	if err1 != nil || err2 != nil || v == nil || qi < 1 || qi > len(v.Chapter.Debrief.Check) || ci < 1 || ci > len(v.Chapter.Debrief.Check[qi-1].Choices) {
		return
	}
	h.Answer(qi-1, ci-1)
}

func (h *HUD) drawDebriefLesson(dst *ebiten.Image, x, top int) {
	l := h.DebriefLesson()
	for _, row := range l.Rows {
		Literal.Draw(dst, row.Label, x, top+row.Y, 1, ui.Palette.Gold)
		for i, line := range row.Lines {
			lx, ly := x+debriefLabel, top+row.Y+i*ui.LineHeight
			clr := row.clr
			if i < row.Struck {
				clr = ui.Palette.TextDim
			}
			Literal.Draw(dst, line, lx, ly, 1, clr)
			if i < row.Struck {
				ui.Fill(dst, ui.Rect{X: lx - 1, Y: ly + 4, W: Literal.Width(line, 1) + 2, H: 1}, ui.Palette.TextDim)
			}
		}
	}
	Literal.Draw(dst, debriefChecks, x, top+l.CheckY, 1, ui.Palette.Gold)
	for _, q := range l.Questions {
		for i, line := range q.Ask {
			Literal.Draw(dst, line, x, top+q.askY+i*ui.LineHeight, 1, ui.Palette.Text)
		}
		for _, b := range q.choices {
			h.drawButton(dst, b)
		}
		clr := ui.Palette.Text
		if q.Right {
			clr = ui.Palette.Green
		}
		for i, line := range q.Answer {
			Literal.Draw(dst, line, x, top+q.answerY+i*ui.LineHeight, 1, clr)
		}
	}
}
