package hud

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func SaveStatus(f Frame) (string, bool) {
	switch {
	case f.SaveError != "" && f.Saved:
		return fmt.Sprintf("NOT SAVED (last %s ago)", saveAgo(f.SavedAgo)), true
	case f.SaveError != "":
		return "NOT SAVED", true
	case f.Saved:
		return "Saved " + saveAgo(f.SavedAgo) + " ago", false
	}
	return "", false
}

func saveAgo(seconds float64) string {
	s := max(int(seconds), 0)
	switch {
	case s < 60:
		return fmt.Sprintf("%d s", s)
	case s < 3600:
		return fmt.Sprintf("%d min", s/60)
	}
	return fmt.Sprintf("%d h", s/3600)
}

func (h *HUD) drawSaveStatus(dst *ebiten.Image, f Frame) {
	text, failed := SaveStatus(f)
	if text == "" {
		return
	}
	clr := ui.Palette.TextDim
	if failed {
		clr = ui.Palette.RedText
	}
	box := h.menuBox()
	h.Font.DrawCentered(dst, text, ui.Rect{X: box.X, Y: box.Y + box.H - 16, W: box.W, H: 8}, 1, clr)
}
