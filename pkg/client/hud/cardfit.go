package hud

import (
	"slices"
)

func (h *HUD) LineRows(lines []Line, w int) int {
	rows := 0
	for _, l := range lines {
		if l.Bar {
			rows++
			continue
		}
		rows += len(h.Font.WrapLines(l.Text, w, 1, len(l.Text)+1))
	}
	return rows
}

func lastOptional(lines []Line) int {
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i].Optional {
			return i
		}
	}
	return -1
}

func (h *HUD) FitLines(lines []Line, w, room int) []Line {
	out := slices.Clone(lines)
	for h.LineRows(out, w) > room {
		i := lastOptional(out)
		if i < 0 {
			return out
		}
		out = slices.Delete(out, i, i+1)
	}
	return out
}
