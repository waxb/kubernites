package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	surveyPad    = 6
	surveyIconW  = 18
	surveyIconH  = 16
	SurveyMaxW   = 250
	SurveyMinW   = 130
	SurveyGap    = 26
	surveyWrap   = 8
	surveyHeadH  = 13
	refusedPulse = 1.2
	IconWood     = "icon/wood"
	IconStone    = "icon/stone"
)

type SurveyPanel struct {
	Survey    view.Survey
	Name      string
	Keep      bool
	Land      string
	At        [2]int
	Reach     int
	RefusedAt float64
	Note      string
	Payoff    string
}

type SurveyRow struct {
	Icon string
	Text string
	Clr  color.RGBA
}

func ClassLine(class, instance string) string {
	if instance == "" {
		return class
	}
	return fmt.Sprintf("%s (%s)", class, instance)
}

func SurveyRows(p SurveyPanel) []SurveyRow {
	sv := p.Survey
	if !p.Keep {
		if !sv.OK {
			return []SurveyRow{{Text: sv.Reason, Clr: ui.Palette.RedText}}
		}
		row := SurveyRow{Text: "Free to build here", Clr: ui.Palette.Green}
		if p.Land != "" {
			row.Text = "On " + p.Land + "'s land"
		}
		if p.Note != "" {
			return []SurveyRow{row, {Text: p.Note, Clr: ui.Palette.Text}}
		}
		return []SurveyRow{row}
	}
	rows := []SurveyRow{
		{Icon: IconWood, Text: fmt.Sprintf("%s from %s", foodOfCores(float64(sv.CPU)), plural(sv.Wood, "tree", "trees")), Clr: ui.Palette.Text},
		{Icon: IconStone, Text: fmt.Sprintf("%s from %s", housingOfGiB(float64(sv.MemoryGiB)), plural(sv.Stone, "stone", "stones")), Clr: ui.Palette.Text},
	}
	if sv.Class != "" {
		rows = append(rows, SurveyRow{Text: ClassLine(sv.Class, sv.Instance), Clr: model.ClassColor(sv.Class)})
	}
	if !sv.OK && sv.Reason != "" {
		rows = append(rows, SurveyRow{Text: sv.Reason, Clr: ui.Palette.RedText})
	}
	if p.Payoff != "" {
		rows = append(rows, SurveyRow{Text: p.Payoff, Clr: ui.Palette.GoldBright})
	}
	return rows
}

type surveyBlock struct {
	Row   SurveyRow `json:"-"`
	Lines []string  `json:"-"`
	h     int
}

func (h *HUD) SurveyBlocks(p SurveyPanel) ([]surveyBlock, int, int) {
	textW := SurveyMaxW - 2*surveyPad
	out := []surveyBlock{}
	head, _ := h.Font.Measure(strings.ToUpper(p.Name), 1)
	w, height := head, surveyHeadH
	for _, row := range SurveyRows(p) {
		indent := 0
		if row.Icon != "" {
			indent = surveyIconW
		}
		b := surveyBlock{Row: row, Lines: h.Font.WrapLines(row.Text, textW-indent, 1, surveyWrap)}
		b.h = surveyTextTop(b) + len(b.Lines)*ui.LineHeight
		if row.Icon != "" {
			b.h = max(b.h, surveyIconH)
		}
		for _, l := range b.Lines {
			lw, _ := h.Font.Measure(l, 1)
			w = max(w, lw+indent)
		}
		out = append(out, b)
		height += b.h
	}
	return out, min(max(w+2*surveyPad, SurveyMinW), SurveyMaxW), height + 2*surveyPad
}

func surveyTextTop(b surveyBlock) int {
	if b.Row.Icon == "" {
		return 0
	}
	return (surveyIconH - ui.CapHeight) / 2
}

func SurveySpots(at [2]int, reach, w, h int) []ui.Rect {
	x, y := at[0], at[1]
	side, rise := max(SurveyGap, reach), max(SurveyGap, reach/2)
	right := Width - w - railRight - railSize - cardMargin
	out := []ui.Rect{
		{X: x + side, Y: y - h/2, W: w, H: h}, {X: x - side - w, Y: y - h/2, W: w, H: h},
		{X: x - w/2, Y: y - rise - SurveyGap - h, W: w, H: h}, {X: x - w/2, Y: y + rise, W: w, H: h},
		{X: right, Y: TopH, W: w, H: h}, {X: right, Y: ConsoleY, W: w, H: h}, {X: 0, Y: ConsoleY, W: w, H: h}, {X: 0, Y: TopH, W: w, H: h},
	}
	for i, r := range out {
		out[i].X = min(max(r.X, cardMargin), Width-w-cardMargin)
		out[i].Y = min(max(r.Y, TopH+cardMargin), ConsoleY-h-cardMargin)
	}
	return out
}

func (h *HUD) surveyBlocked(f Frame) []ui.Rect {
	out := append(inflated(h.GuideTargets(f), targetClear), FreightRects(f)...)
	if h.ObjectiveShown(f) {
		out = append(out, h.ObjectiveRectOf(f))
	}
	if h.LessonShown(f) {
		out = append(out, h.LessonRect(f))
	}
	if r, ok := h.CalamityRect(f); ok {
		out = append(out, r)
	}
	if rail := h.RailRect(f); rail.W > 0 {
		out = append(out, rail.Inset(-2))
	}
	return out
}

func (h *HUD) SurveyRect(f Frame) (ui.Rect, bool) {
	if f.Survey == nil || h.OverlayID != OverlayNone {
		return ui.Rect{}, false
	}
	_, w, height := h.SurveyBlocks(*f.Survey)
	board := []ui.Rect{}
	for _, r := range FreightRects(f) {
		board = append(board, Shadowed(r))
	}
	return ClearAvoiding(SurveySpots(f.Survey.At, f.Survey.Reach, w, height), board, h.surveyBlocked(f)), true
}

func SurveyBorder(p SurveyPanel, now float64, reduce bool) color.RGBA {
	age := now - p.RefusedAt
	if p.RefusedAt > 0 && age >= 0 && age < refusedPulse {
		if reduce || int(age*6)%2 == 0 {
			return ui.Palette.Red
		}
	}
	if !p.Survey.OK {
		return ui.Palette.RedText
	}
	return ui.Palette.Gold
}

func (h *HUD) drawSurvey(dst *ebiten.Image, f Frame) {
	r, ok := h.SurveyRect(f)
	if !ok {
		return
	}
	p := *f.Survey
	blocks, _, _ := h.SurveyBlocks(p)
	ui.Fill(dst, r.Offset(2, 3), ui.Palette.Shadow)
	ui.Fill(dst, r, ui.Tint(ui.Palette.Console, 240))
	ui.Stroke(dst, r, SurveyBorder(p, f.Clock, f.Reduce))
	h.Font.DrawShadow(dst, strings.ToUpper(p.Name), r.X+surveyPad, r.Y+surveyPad, 1, ui.Palette.Gold)
	y := r.Y + surveyPad + surveyHeadH
	for _, b := range blocks {
		x := r.X + surveyPad
		ty := y + surveyTextTop(b)
		if b.Row.Icon != "" {
			h.icon(dst, b.Row.Icon, x, y)
			x += surveyIconW
		}
		for i, l := range b.Lines {
			h.Font.Draw(dst, l, x, ty+i*ui.LineHeight, 1, b.Row.Clr)
		}
		y += b.h
	}
}
