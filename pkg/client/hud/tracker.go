package hud

import (
	"image/color"
	"slices"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	FoldID      = "obj:fold"
	foldMarkW   = 12
	foldRowH    = 12
	foldTipOpen = "Fold the decree to one line [click]"
	foldTipShut = "Unfold the decree: its four halls [click]"
)

func (f Frame) decreePending() bool {
	return f.Guide.Staged && f.Revealed(view.BorgIntro) && !slices.Contains(f.Guide.Introduced, view.BorgIntro)
}

func (f Frame) waitsForDecree(o view.ObjectiveView) bool {
	return o.Quest != "" && f.decreePending()
}

func questProgress(q view.QuestView) string {
	done := 0
	for _, p := range q.Parts {
		if p.Done {
			done++
		}
	}
	return strconv.Itoa(done) + "/" + strconv.Itoa(len(q.Parts))
}

func (h *HUD) TrackerFolded() bool { return h.folded }

func (h *HUD) SetTrackerFolded(folded bool) { h.folded = folded }

func (h *HUD) ScrollOpen(f Frame) bool { return h.QuestCardShown(f) }

func (h *HUD) questLines(f Frame, o view.ObjectiveView, q view.QuestView) []Line {
	if h.folded {
		return h.foldedLines(f, o)
	}
	return h.trackerLines(f, o, q)
}

func (h *HUD) foldedLines(f Frame, o view.ObjectiveView) []Line {
	w := Objective.W - 16
	if o.Done {
		return append(wrapped(h, markDone, o.Text, w, objectiveTextLines, ui.Palette.GreenDark), wrapped(h, "", o.Lesson, w, objectiveHintLines, ui.Palette.Ink)...)
	}
	return h.trackerStep(f, o, w, "", true)
}

func (h *HUD) foldButton(f Frame) []Button {
	_, q, ok := f.questStep()
	if !ok {
		return nil
	}
	r := h.ObjectiveRectOf(f)
	title, _ := h.CardTitle(f)
	w, _ := h.Font.Measure(title, 1)
	tip := foldTipOpen
	if h.folded {
		tip = foldTipShut
	}
	return []Button{{ID: FoldID, Rect: ui.Rect{X: r.X, Y: r.Y + 2, W: titleX(true) + w + foldMarkW + 2, H: foldRowH}, Label: questProgress(q), Tip: tip}}
}

func (h *HUD) toggleFold() []Intent {
	h.folded = !h.folded
	return []Intent{{Kind: IntentPanels}, {Kind: IntentSound, Command: "click"}}
}

func (h *HUD) drawFoldTitle(dst *ebiten.Image, r ui.Rect, title string) {
	clr := ui.Palette.RedDark
	if h.Hovering && h.Hover.ID == FoldID {
		clr = ui.Palette.Red
	}
	x := r.X + titleX(true)
	h.Font.Draw(dst, title, x, r.Y+6, 1, clr)
	w, _ := h.Font.Measure(title, 1)
	drawFoldMark(dst, x+w+4, r.Y+6+1, h.folded, clr)
}

func drawFoldMark(dst *ebiten.Image, x, y int, folded bool, clr color.RGBA) {
	rows := []int{5, 3, 1}
	if folded {
		for i, w := range []int{1, 2, 3, 2, 1} {
			ui.Fill(dst, ui.Rect{X: x + 1, Y: y + i, W: w, H: 1}, clr)
		}
		return
	}
	for i, w := range rows {
		ui.Fill(dst, ui.Rect{X: x + i, Y: y + 1 + i, W: w, H: 1}, clr)
	}
}
