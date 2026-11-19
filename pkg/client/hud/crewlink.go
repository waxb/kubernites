package hud

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const crewLinkMax = 2

func crewLink(f Frame) (Button, string, bool) {
	if f.Selection.Kind != "hearth" && f.Selection.Kind != "walker" {
		return Button{}, "", false
	}
	c, ok := ChainOf(f)
	if !ok || c.Bare {
		return Button{}, "", false
	}
	box := PortraitBox()
	x := box.X + box.W + 8
	text := ui.Bridge("crew "+c.Name+": open the board", "Deployment "+c.Name+": open the board")
	w := CardBox.X + CardBox.W - x - 24
	h := len(crewLinkRows(text, w)) * ui.LineHeight
	rect := ui.Rect{X: x, Y: cardFloor() - h, W: w, H: h}
	return Button{ID: ChainButton, Rect: rect, Tip: openTip(c)}, text, true
}

func crewLinkRows(text string, w int) []string {
	return ui.Font{}.WrapLines(strings.Replace(text, ": ", ":  ", 1), w, 1, crewLinkMax)
}

func (h *HUD) drawCrewLink(dst *ebiten.Image, f Frame) {
	b, text, ok := crewLink(f)
	if !ok {
		return
	}
	hot := h.Hovering && h.Hover.Rect == b.Rect
	for i, row := range crewLinkRows(text, b.Rect.W) {
		y := b.Rect.Y + i*ui.LineHeight
		h.Font.Draw(dst, row, b.Rect.X, y, 1, ui.Palette.Blue)
		if hot {
			ui.Fill(dst, ui.Rect{X: b.Rect.X, Y: y + ui.CapHeight + 2, W: h.Font.Width(row, 1), H: 1}, ui.Palette.Blue)
		}
	}
}
