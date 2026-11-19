package hud

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	IntroTray  = introPrefix + ":tray"
	trayW      = 70
	TrayRows   = 5
	trayGap    = 4
	trayHeader = "WAITING, IN ORDER"
)

func TrayLabel(n int) string { return fmt.Sprintf("+%d MORE", n) }

func trayTip(queue []string) string {
	return "Waiting after this, one at a time:\n" + strings.Join(queue, "\n")
}

func (h *HUD) trayButton(f Frame, card ui.Rect, face int) (Button, bool) {
	q := f.Guide.Queue
	if len(q) == 0 {
		return Button{}, false
	}
	r := ui.Rect{X: card.X + card.W/2 - trayW/2, Y: face, W: trayW, H: 16}
	return Button{ID: IntroTray, Face: r, Label: TrayLabel(len(q)), Tip: trayTip(q), Selected: h.Director.TrayOpen}, true
}

func (h *HUD) TrayLines(f Frame) []string {
	q := f.Guide.Queue
	if !h.Director.TrayOpen || len(q) == 0 || !h.IntroShown(f) {
		return nil
	}
	out := []string{trayHeader}
	for i, title := range q[:min(len(q), TrayRows)] {
		out = append(out, fmt.Sprintf("%d. %s", i+1, title))
	}
	if extra := len(q) - TrayRows; extra > 0 {
		out = append(out, fmt.Sprintf("and %d more", extra))
	}
	return out
}

func (h *HUD) TrayRect(f Frame) (ui.Rect, bool) {
	lines := h.TrayLines(f)
	if len(lines) == 0 {
		return ui.Rect{}, false
	}
	card := h.IntroCard(f)
	height := 8 + len(lines)*ui.LineHeight
	spots := clampAll([]ui.Rect{
		{X: card.X, Y: card.Y + card.H + trayGap, W: card.W, H: height},
		{X: card.X, Y: card.Y - trayGap - height, W: card.W, H: height},
	})
	blocked := inflated(h.GuideTargets(f), targetClear)
	return FirstClear(spots, append(blocked, card)), true
}

func (h *HUD) DrawTray(dst *ebiten.Image, f Frame) {
	r, ok := h.TrayRect(f)
	if !ok {
		return
	}
	ui.Fill(dst, r.Offset(2, 3), ui.Palette.Shadow)
	ui.Fill(dst, r, ui.Tint(ui.Palette.Console, 235))
	ui.Stroke(dst, r, ui.Palette.Gold)
	for i, l := range h.TrayLines(f) {
		clr := ui.Palette.Cream
		if i == 0 {
			clr = ui.Palette.Gold
		}
		h.Font.Draw(dst, ui.Clip(l, ui.MaxChars(r.W-2*introPad, 1)), r.X+introPad, r.Y+5+i*ui.LineHeight, 1, clr)
	}
}

func (h *HUD) toggleTray() []Intent {
	h.Director.TrayOpen = !h.Director.TrayOpen
	return []Intent{{Kind: IntentSound, Command: "click"}}
}
