package ui

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var Palette = struct {
	Ink, Shadow, Parchment, ParchmentDark, Gold, GoldBright   color.RGBA
	Blue, BlueDark, Green, GreenDark, Red, RedDark, Muted     color.RGBA
	Night, Console, ConsoleEdge, Text, TextDim, Purple, Cream color.RGBA
	RedText, ChipBorder                                       color.RGBA
}{
	Ink:           color.RGBA{43, 37, 32, 255},
	Shadow:        color.RGBA{20, 25, 27, 220},
	Parchment:     color.RGBA{232, 211, 164, 255},
	ParchmentDark: color.RGBA{172, 137, 91, 255},
	Gold:          color.RGBA{213, 159, 52, 255},
	GoldBright:    color.RGBA{255, 218, 105, 255},
	Blue:          color.RGBA{74, 137, 156, 255},
	BlueDark:      color.RGBA{37, 68, 83, 255},
	Green:         color.RGBA{112, 176, 88, 255},
	GreenDark:     color.RGBA{45, 83, 50, 255},
	Red:           color.RGBA{206, 88, 74, 255},
	RedDark:       color.RGBA{104, 44, 43, 255},
	Muted:         color.RGBA{122, 119, 108, 255},
	Night:         color.RGBA{24, 30, 58, 255},
	Console:       color.RGBA{33, 44, 48, 245},
	ConsoleEdge:   color.RGBA{18, 24, 27, 255},
	Text:          color.RGBA{238, 226, 196, 255},
	TextDim:       color.RGBA{150, 160, 150, 255},
	Purple:        color.RGBA{150, 104, 170, 255},
	Cream:         color.RGBA{255, 244, 214, 255},
	RedText:       color.RGBA{240, 128, 112, 255},
	ChipBorder:    color.RGBA{232, 211, 164, 255},
}

type Rect struct{ X, Y, W, H int }

func (r Rect) Contains(x, y int) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.W && y < r.Y+r.H
}

func (r Rect) Inset(n int) Rect {
	return Rect{r.X + n, r.Y + n, max(r.W-2*n, 0), max(r.H-2*n, 0)}
}

func (r Rect) Offset(dx, dy int) Rect {
	return Rect{r.X + dx, r.Y + dy, r.W, r.H}
}

func Fill(dst *ebiten.Image, r Rect, clr color.Color) {
	vector.FillRect(dst, float32(r.X), float32(r.Y), float32(r.W), float32(r.H), clr, false)
}

func Stroke(dst *ebiten.Image, r Rect, clr color.Color) {
	vector.StrokeRect(dst, float32(r.X)+.5, float32(r.Y)+.5, float32(r.W-1), float32(r.H-1), 1, clr, false)
}

func Panel(dst *ebiten.Image, r Rect) {
	Fill(dst, r.Offset(2, 3), Palette.Shadow)
	Fill(dst, r, Palette.ParchmentDark)
	Fill(dst, r.Inset(2), Palette.Parchment)
	Stroke(dst, r, Palette.Ink)
	Stroke(dst, r.Inset(3), color.RGBA{250, 230, 184, 255})
}

func DarkPanel(dst *ebiten.Image, r Rect) {
	Fill(dst, r.Offset(2, 3), color.RGBA{11, 15, 18, 180})
	Fill(dst, r, Palette.ConsoleEdge)
	Fill(dst, r.Inset(1), Palette.BlueDark)
	Fill(dst, r.Inset(2), Palette.Console)
	Stroke(dst, r, Palette.Gold)
}

func Well(dst *ebiten.Image, r Rect) {
	Fill(dst, r, color.RGBA{16, 22, 24, 255})
	Stroke(dst, r, color.RGBA{70, 84, 80, 255})
}

func Bar(dst *ebiten.Image, r Rect, value float64, good color.Color) {
	value = min(max(value, 0), 1)
	Fill(dst, r, Palette.Shadow)
	inner := r.Inset(1)
	Fill(dst, inner, Palette.RedDark)
	inner.W = int(float64(inner.W) * value)
	if inner.W > 0 {
		Fill(dst, inner, good)
	}
	Stroke(dst, r, Palette.Ink)
}

type ButtonState struct {
	Hovered, Pressed, Disabled, Selected bool
}

func ButtonFace(dst *ebiten.Image, r Rect, state ButtonState) color.RGBA {
	face, border, text := Palette.BlueDark, Palette.Gold, Palette.Parchment
	switch {
	case state.Disabled:
		face, border, text = color.RGBA{58, 60, 56, 255}, Palette.Muted, Palette.Muted
	case state.Selected:
		face, border = Palette.GreenDark, Palette.Green
	case state.Hovered:
		face, border = Palette.Blue, Palette.GoldBright
	}
	if state.Pressed {
		r = r.Offset(1, 1)
	}
	Fill(dst, r.Offset(1, 2), Palette.Shadow)
	Fill(dst, r, face)
	Fill(dst, Rect{r.X + 1, r.Y + 1, r.W - 2, 1}, lighten(face, 30))
	Stroke(dst, r, border)
	return text
}

func Button(dst *ebiten.Image, font Font, r Rect, label string, state ButtonState) {
	text := ButtonFace(dst, r, state)
	font.DrawCentered(dst, label, r, 1, text)
}

func lighten(c color.RGBA, n int) color.RGBA {
	add := func(v uint8) uint8 { return uint8(min(int(v)+n, 255)) }
	return color.RGBA{add(c.R), add(c.G), add(c.B), c.A}
}

func Tint(c color.RGBA, alpha uint8) color.RGBA {
	scale := func(v uint8) uint8 { return uint8(int(v) * int(alpha) / 255) }
	return color.RGBA{scale(c.R), scale(c.G), scale(c.B), alpha}
}
