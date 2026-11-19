package hud

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const timelineGap = 12

type timelineRow struct {
	icon string
	text string
}

func timelineRows(c view.ChronicleView) []timelineRow {
	t := c.Timeline
	if t == nil {
		return nil
	}
	found := ui.Bridge("CAUSE NEVER FOUND", "root cause never found")
	if t.CauseAt > 0 {
		found = ui.Bridge("CAUSE FOUND "+Duration(t.CauseAt-t.StruckAt)+" LATER", "root cause found "+Duration(t.CauseAt-t.StruckAt)+" after detection")
	}
	return []timelineRow{
		{icon: art.TimelineIconName("struck"), text: ui.Bridge("STRUCK", "detected")},
		{icon: art.TimelineIconName("cause-found"), text: found},
		{icon: art.TimelineIconName("fixed"), text: ui.Bridge("MENDED "+Duration(t.FixedAt-t.StruckAt)+" AFTER IT STRUCK", "resolved "+Duration(t.FixedAt-t.StruckAt)+" after detection")},
		{icon: art.TimelineIconName("boxes-lost"), text: ui.Bridge(fmt.Sprintf("%s BOXES LOST", Compact(t.BoxesLost)), fmt.Sprintf("%s requests lost", Compact(t.BoxesLost)))},
	}
}

func timelineBeside(pic ui.Rect, rows int) ui.Rect {
	x := pic.X + pic.W + timelineGap
	return ui.Rect{X: x, Y: pic.Y, W: Overlay.X + Overlay.W - 20 - x, H: max(rows*clueRowH, pic.H)}
}

func (h *HUD) drawTimeline(dst *ebiten.Image, c view.ChronicleView, r ui.Rect) {
	y := r.Y
	width := r.W - clueTextX
	for _, row := range timelineRows(c) {
		h.icon(dst, row.icon, r.X, y)
		h.Font.Draw(dst, ui.Clip(row.text, ui.MaxChars(width, 1)), r.X+clueTextX, y+(clueRowH-ui.GlyphHeight)/2, 1, ui.Palette.Text)
		y += clueRowH
	}
}
