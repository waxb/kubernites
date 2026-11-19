package hud

import (
	"image/color"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	stoneIndent = "  "
	stoneLead   = 2
)

type stoneRow struct {
	Text string `json:"-"`
	clr  color.RGBA
	line int
}

func (h *HUD) StoneWidth() int {
	return ui.MaxChars(h.stoneOutput().W-8, 1)
}

func (h *HUD) StoneRows() []stoneRow {
	width := h.StoneWidth()
	out := make([]stoneRow, 0, len(h.Stone.Output))
	for i, l := range h.Stone.Output {
		for _, part := range wrapStoneText(ConsoleText(l), width) {
			out = append(out, stoneRow{Text: part, clr: l.color(), line: i})
		}
	}
	return out
}

func wrapStoneText(text string, width int) []string {
	runes := []rune(text)
	if len(runes) <= width || width <= len(stoneIndent)+1 {
		return []string{text}
	}
	out := []string{}
	for len(runes) > width {
		cut := width
		if at := strings.LastIndexAny(string(runes[:width]), " |"); at > width/2 {
			cut = len([]rune(string(runes[:width])[:at])) + 1
		}
		out = append(out, strings.TrimRight(string(runes[:cut]), " "))
		runes = append([]rune(stoneIndent), []rune(strings.TrimLeft(string(runes[cut:]), " "))...)
	}
	return append(out, string(runes))
}

func (h *HUD) scrollStone(step int) {
	h.Stone.Scroll = h.Stone.Scroll.By(-step, len(h.StoneRows()), h.StoneVisible())
}

func (h *HUD) PageStone(dir int) {
	h.scrollStone(dir * max(h.StoneVisible()-1, 1))
}

func (h *HUD) pointStone(written int) {
	v := h.Teacher()
	if v == nil {
		return
	}
	step, ok := v.Current()
	if !ok || step.Points == "" {
		return
	}
	from := len(h.Stone.Output) - (h.Stone.Written - written)
	rows := h.StoneRows()
	for i, r := range rows {
		if r.line >= from && strings.Contains(r.Text, step.Points) {
			visible := h.StoneVisible()
			h.Stone.Scroll = ui.Scroll{Offset: len(rows) - max(i-stoneLead, 0) - visible}.Clamp(len(rows), visible)
			return
		}
	}
}
