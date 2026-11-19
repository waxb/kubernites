package hud

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	themeRows = 2
	themeGap  = 6
)

func StageThemeLine(f Frame) string {
	for _, a := range f.State.Road.Ages {
		if a.ID == f.State.Road.Current {
			return a.Theme
		}
	}
	return ""
}

func (h *HUD) BannerLines() (string, string) {
	return h.Juice.Banner, h.Juice.Cause
}

func (h *HUD) DrawStageTheme(dst *ebiten.Image, f Frame, r ui.Rect) int {
	theme := StageThemeLine(f)
	if theme == "" {
		return r.Y
	}
	y := r.Y
	for _, row := range h.Font.WrapLines(theme, r.W, 1, themeRows) {
		h.Font.Draw(dst, row, r.X, y, 1, ui.Palette.GoldBright)
		y += ui.LineHeight
	}
	return y + themeGap
}

func (h *HUD) VignetteAge() float64 {
	return h.events - h.Juice.vignetteAt
}
