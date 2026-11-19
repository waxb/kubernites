package world

import (
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	CalloutLife   = 4.0
	calloutFade   = .3
	calloutRise   = 8.0
	calloutLift   = 30
	calloutFace   = 16
	calloutPad    = 3
	calloutChars  = 44
	calloutLines  = 5
	calloutMargin = 2
)

type Callout struct {
	Portrait string
	Text     string
	At       Vec
	Age      float64
	Count    int
}

func CalloutAlpha(age float64, reduce bool) float64 {
	switch {
	case age < 0 || age >= CalloutLife:
		return 0
	case reduce:
		return 1
	}
	left := (CalloutLife - age) / (CalloutLife * calloutFade)
	return min(left, 1)
}

func CalloutText(c Callout) string {
	var font ui.Font
	text := c.Text
	if c.Count > 1 {
		text += " x" + Itoa(c.Count)
	}
	return strings.Join(font.WrapLines(text, calloutChars*ui.Advance-1, 1, calloutLines), "\n")
}

func calloutInner(text string) (int, int) {
	textH := strings.Count(text, "\n")*ui.LineHeight + ui.CapHeight
	return max(calloutFace, textH), textH
}

func CalloutRect(c Callout, sx, sy, zoom float64, reduce bool) ui.Rect {
	var font ui.Font
	text := CalloutText(c)
	w, _ := font.Measure(text, 1)
	inner, _ := calloutInner(text)
	rise := 0.0
	if !reduce {
		rise = calloutRise * min(c.Age/CalloutLife, 1)
	}
	boxW := calloutFace + 3*calloutPad + w
	x := min(max(int(sx)-boxW/2, calloutMargin), ViewWidth-boxW-calloutMargin)
	return ui.Rect{X: x, Y: int(sy-calloutLift*zoom-rise) - inner - calloutPad, W: boxW, H: inner + 2*calloutPad}
}

func clearOf(r ui.Rect, avoid []ui.Rect) (ui.Rect, bool) {
	for _, a := range avoid {
		if Overlaps(r, a) {
			moved := r
			moved.Y = a.Y + a.H + 4
			if moved.Y+moved.H > ViewHeight {
				return r, false
			}
			r = moved
		}
	}
	for _, a := range avoid {
		if Overlaps(r, a) {
			return r, false
		}
	}
	return r, true
}

func calloutShade(r ui.Rect) ui.Rect { return ui.Rect{X: r.X, Y: r.Y, W: r.W + 1, H: r.H + 2} }

type placedCallout struct {
	callout Callout
	box     ui.Rect
	alpha   float64
}

func placeCallouts(cam render.Camera, sc Scene) []placedCallout {
	avoid := append([]ui.Rect(nil), sc.Avoid...)
	out := []placedCallout{}
	for _, c := range sc.Callouts {
		alpha := CalloutAlpha(c.Age, sc.Reduce)
		sx, sy := ScreenPointOf(cam, sc.Site, c.At)
		if alpha <= 0 || !onScreen(sx, sy, cam.Scale()) {
			continue
		}
		box := CalloutRect(c, sx, sy, cam.Scale(), sc.Reduce)
		box.Y = max(box.Y, calloutMargin)
		shade, ok := clearOf(calloutShade(box), avoid)
		if !ok {
			continue
		}
		box.X, box.Y = shade.X, shade.Y
		avoid = append(avoid, shade)
		out = append(out, placedCallout{callout: c, box: box, alpha: alpha})
	}
	return out
}

func (r *Renderer) DrawCallouts(dst *ebiten.Image, cam render.Camera, sc Scene) {
	var font ui.Font
	avoid := slices.Clone(sc.Avoid)
	for _, b := range r.Drawn() {
		avoid = append(avoid, b.Rect)
	}
	sc.Avoid = avoid
	for _, p := range placeCallouts(cam, sc) {
		box, alpha, c := p.box, p.alpha, p.callout
		a := uint8(255 * alpha)
		ui.Fill(dst, box.Offset(1, 2), ui.Tint(ui.Palette.Shadow, a/2))
		ui.Fill(dst, box, ui.Tint(ui.Palette.ConsoleEdge, uint8(220*alpha)))
		ui.Stroke(dst, box, ui.Tint(ui.Palette.Blue, a))
		face := ui.Rect{X: box.X + calloutPad, Y: box.Y + calloutPad, W: calloutFace, H: calloutFace}
		if alpha > .5 {
			r.Lib.Portrait(dst, c.Portrait, face)
		}
		text := CalloutText(c)
		r.note(DrawnCallout, "callout:"+text, text, calloutShade(box))
		inner, textH := calloutInner(text)
		font.Draw(dst, text, face.X+face.W+calloutPad, box.Y+calloutPad+(inner-textH+1)/2, 1, ui.Tint(ui.Palette.Cream, a))
	}
}
