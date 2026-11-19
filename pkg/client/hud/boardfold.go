package hud

import (
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	boardMore     = FreightTip + "more"
	boardFoldBody = 3
	boardFoldTop  = 2
	moreLabel     = "MORE"
	lessLabel     = "LESS"
)

var (
	hiddenBoardGoals = []string{tideRow, nextRow, boostRow, DemandRowID, FlowbarMoney, flowbarLosses, SurfaceLosses}
	hiddenBoardWords = []string{"board's NEXT", "PORTS row"}
)

func (h *HUD) withBoard(f Frame) Frame {
	f = h.withSteadyBoard(f)
	f.BoardForced = boardNeeded(f)
	f.BoardCompact = !h.BoardOpen && !f.BoardForced
	return f
}

func boardNeeded(f Frame) bool {
	if !BoardShown(f) {
		return false
	}
	if NextSpeaks(f) || f.Teaching(SurfaceLosses) {
		return true
	}
	goals := []string{f.Guide.Target, f.Guide.Button, ObjectiveGoal(f)}
	if f.Guide.Intro != nil {
		goals = append(goals, f.Guide.Intro.Target)
	}
	for _, g := range goals {
		if slices.Contains(hiddenBoardGoals, g) || strings.HasPrefix(g, FreightTip+"kind:") || strings.HasPrefix(g, freightPortID) || opensAPort(g) {
			return true
		}
	}
	return namesHiddenRow(f)
}

func opensAPort(goal string) bool {
	verb, _, ok := PortTarget(goal)
	return ok && verb == PortVerbOpen
}

func namesHiddenRow(f Frame) bool {
	texts := []string{}
	if o, ok := f.current(); ok {
		texts = append(texts, o.Progress, o.Hint, o.Text, o.Purpose)
	}
	if in := f.Guide.Intro; in != nil {
		texts = append(texts, in.Try, in.Purpose, in.Title)
	}
	for _, t := range texts {
		for _, w := range hiddenBoardWords {
			if strings.Contains(t, w) {
				return true
			}
		}
	}
	return false
}

func boardTopRows(f Frame) int {
	if f.BoardCompact {
		return boardFoldTop
	}
	return boardRows
}

func boardBodyRows(f Frame) int {
	if f.BoardCompact {
		return boardFoldBody
	}
	return boardRows + boardExtra(f) + flowbarRows(f)
}

func boardFoldShown(f Frame) bool { return BoardShown(f) && !f.BoardForced }

func boardFoldY(f Frame, r ui.Rect) int {
	y := r.Y + BoardPad + boardBodyRows(f)*ui.LineHeight
	if !f.BoardCompact {
		if hasChips(f.State.Freight) {
			y += chipRowH
		}
		if PortsRowShown(f) {
			y += portsRowH
		}
	}
	return y
}

func boardFoldButton(f Frame, r ui.Rect, open bool) Button {
	label, tip := moreLabel, "Show every line of the harbor board: income, upkeep, the tide, what comes next, the ports."
	if open {
		label, tip = lessLabel, "Fold the harbor board down to its tier, its flow and its bottleneck."
	}
	return Button{ID: boardMore, Rect: ui.Rect{X: r.X + r.W - BoardPad - 36, Y: boardFoldY(f, r), W: 36, H: ui.LineHeight}, Label: label, Tip: tip}
}

func (h *HUD) boardFoldButtons(f Frame, r ui.Rect) []Button {
	if !boardFoldShown(f) {
		return nil
	}
	return []Button{boardFoldButton(f, r, !f.BoardCompact)}
}

func (h *HUD) drawBoardFold(dst *ebiten.Image, f Frame, r ui.Rect) {
	for _, b := range h.boardFoldButtons(f, r) {
		if note := LostNote(EconomyOf(f.State)); note != "" {
			drawClipped(dst, note, r.X+BoardPad, b.Rect.Y+3, b.Rect.X-r.X-BoardPad-FlowbarMinGap, ui.Palette.RedText)
		}
		h.drawButton(dst, b)
	}
}

func (h *HUD) toggleBoard() []Intent {
	h.BoardOpen = !h.BoardOpen
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) OpenBoard() { h.BoardOpen = true }
