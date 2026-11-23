package hud

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	dialW        = 78
	dialH        = 56
	dialRadius   = 19
	dialSegments = 10
	dialDots     = 3
	dialDot      = 3
	dialLowZone  = 4
	dialMidZone  = 7
)

type Dial struct {
	Label string
	Icon  string
	Value float64
	Known bool
	Tip   string
	Text  string
}

func RatingDials(f Frame) []Dial {
	if f.Live {
		return LiveDials(f)
	}
	r, known := f.Caesar.Ratings, f.Caesar.Rated
	return withRatingEffects(f, []Dial{
		{Label: "RELIABILITY", Icon: "icon/rating-reliability", Value: r.Reliability, Known: known, Text: Nines(f.State.SLO.Current), Tip: "Reliability: the share of boxes delivered fresh, the Nines (the dial's arc grades it)"},
		{Label: "EFFICIENCY", Icon: "icon/rating-efficiency", Value: r.Efficiency, Known: known, Tip: "Efficiency: food eaten against food pledged, and no idle Keeps"},
		{Label: "RESILIENCE", Icon: "icon/rating-resilience", Value: r.Resilience, Known: known, Tip: "Resilience: crews spread over Keeps and isles, storms weathered"},
		{Label: "FAVOR", Icon: "icon/rating-favor", Value: r.Favor, Known: known, Tip: "Favor: the Crown's regard, won by answering Decrees"},
	})
}

func DialAngle(v float64) float64 {
	return math.Pi * (1 - min(max(v, 0), 100)/100)
}

func DialLit(v float64, segment int) bool {
	return v >= float64(segment+1)*100/dialSegments-100/dialSegments/2
}

func DialZone(segment int) int {
	switch {
	case segment < dialLowZone:
		return ToneBad
	case segment < dialMidZone:
		return ToneWarn
	}
	return ToneGood
}

func DialText(d Dial) string {
	if !d.Known {
		return NoValue
	}
	if d.Text != "" {
		return d.Text
	}
	return fmt.Sprint(int(math.Round(min(max(d.Value, 0), 100))))
}

func DialsBox() ui.Rect {
	return ui.Rect{X: Overlay.X + Overlay.W - 12 - closeReach - 4*dialW, Y: Overlay.Y + 24, W: 4 * dialW, H: dialH}
}

func DialRect(i int) ui.Rect {
	b := DialsBox()
	return ui.Rect{X: b.X + i*dialW, Y: b.Y, W: dialW - 4, H: dialH}
}

func dimmed(c color.RGBA) color.RGBA {
	return color.RGBA{c.R / 3, c.G / 3, c.B / 3, 255}
}

func (h *HUD) drawDials(dst *ebiten.Image, f Frame) {
	for i, d := range RatingDials(f) {
		h.drawDial(dst, DialRect(i), d)
	}
}

func (h *HUD) drawDial(dst *ebiten.Image, r ui.Rect, d Dial) {
	ui.Fill(dst, r, consoleInk)
	ui.Stroke(dst, r, ui.Palette.BlueDark)
	cx, cy := float64(r.X+r.W/2), float64(r.Y+27)
	for s := range dialSegments {
		clr := toneColor(DialZone(s))
		if !d.Known || !DialLit(d.Value, s) {
			clr = dimmed(clr)
		}
		for k := range dialDots {
			a := math.Pi * (1 - (float64(s)+(float64(k)+.5)/dialDots)/dialSegments)
			x, y := cx+dialRadius*math.Cos(a), cy-dialRadius*math.Sin(a)
			ui.Fill(dst, ui.Rect{X: int(x) - 1, Y: int(y) - 1, W: dialDot, H: dialDot}, clr)
		}
	}
	if d.Known {
		a := DialAngle(d.Value)
		tip := float32(dialRadius - 5)
		vector.StrokeLine(dst, float32(cx), float32(cy), float32(cx)+tip*float32(math.Cos(a)), float32(cy)-tip*float32(math.Sin(a)), 1, ui.Palette.Cream, false)
	}
	ui.Fill(dst, ui.Rect{X: int(cx) - 1, Y: int(cy) - 1, W: 3, H: 3}, ui.Palette.Gold)
	h.Font.DrawCentered(dst, DialText(d), ui.Rect{X: r.X, Y: int(cy) + 2, W: r.W, H: 9}, 1, ui.Palette.Cream)
	h.Font.DrawCentered(dst, d.Label, ui.Rect{X: r.X, Y: int(cy) + 13, W: r.W, H: 9}, 1, ui.Palette.Gold)
}

func dialButtons(f Frame) []Button {
	out := []Button{}
	for i, d := range RatingDials(f) {
		out = append(out, Button{ID: "info:dial:" + d.Label, Rect: DialRect(i), Tip: d.Tip})
	}
	return out
}
