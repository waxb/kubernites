package ui

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestDrawingSmoke(t *testing.T) {
	dst := ebiten.NewImage(64, 64)
	var f Font
	f.Draw(dst, "Ab1\n?", 1, 1, 1, Palette.Text)
	f.DrawShadow(dst, "x", 1, 1, 2, Palette.Text)
	f.DrawCentered(dst, "ok", Rect{0, 0, 64, 20}, 1, Palette.Text)
	Panel(dst, Rect{0, 0, 30, 30})
	DarkPanel(dst, Rect{0, 0, 30, 30})
	Well(dst, Rect{0, 0, 30, 30})
	Bar(dst, Rect{0, 0, 30, 6}, 2, Palette.Green)
	for _, s := range []ButtonState{{}, {Hovered: true}, {Pressed: true}, {Disabled: true}, {Selected: true}} {
		Button(dst, f, Rect{0, 0, 40, 14}, "b", s)
	}
}
