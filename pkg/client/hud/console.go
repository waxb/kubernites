package hud

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

var (
	consoleWood  = color.RGBA{58, 44, 32, 255}
	consoleStone = color.RGBA{40, 50, 52, 255}
)

func drawStrip(dst *ebiten.Image) {
	ui.Fill(dst, ConsoleBox, consoleStone)
	ui.Fill(dst, ui.Rect{X: 0, Y: ConsoleY, W: Width, H: 4}, consoleWood)
	ui.Fill(dst, ui.Rect{X: 0, Y: ConsoleY + 4, W: Width, H: 1}, ui.Palette.Gold)
}

func drawPanel(dst *ebiten.Image, r ui.Rect) {
	ui.Fill(dst, r.Offset(2, 3), color.RGBA{8, 11, 12, 220})
	ui.Fill(dst, r, consoleWood)
	ui.Fill(dst, r.Inset(3), ui.Palette.Console)
	ui.Stroke(dst, r.Inset(2), ui.Palette.Gold)
}

func (h *HUD) drawConsole(dst *ebiten.Image, f Frame) {
	h.NoteAppearances(f)
	if h.StripShown(f) {
		ui.InRegion(ConsoleRegion+"strip", func() { h.withAppear(dst, keyStrip, f, drawStrip) })
	}
	if MinimapShown(f) {
		ui.InRegion(ConsoleRegion+"minimap", func() {
			h.withAppear(dst, KeyMinimap, f, func(d *ebiten.Image) { drawPanel(d, Minimap); h.drawMinimap(d, f) })
		})
	}
	if CardShown(f) {
		ui.InRegion(ConsoleRegion+"card", func() {
			h.withAppear(dst, KeyCard, f, func(d *ebiten.Image) { drawPanel(d, CardBox); h.DrawSelection(d, f) })
		})
	}
	if h.WellShown(f) {
		ui.InRegion(ConsoleRegion+"well", func() {
			h.withAppear(dst, keyWell, f, func(d *ebiten.Image) { drawPanel(d, InfoWell); h.drawWell(d, f) })
		})
	}
	if r := h.CommandFrame(f); r.W > 0 {
		ui.InRegion(ConsoleRegion+"commands", func() {
			drawPanel(dst, r)
			h.drawCommand(dst, f)
		})
	}
}
