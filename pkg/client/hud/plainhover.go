package hud

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

const (
	plainTipLines = 3
	plainTipGap   = 8
)

type plainSpot struct {
	rect  ui.Rect
	Words []lexicon.Plain `json:"-"`
}

type plainWatch struct {
	mu   sync.Mutex
	spot *plainSpot
}

func (w *plainWatch) Reset() {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.spot = nil
}

func (w *plainWatch) set(s plainSpot) {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.spot = &s
}

func (w *plainWatch) Get() (plainSpot, bool) {
	if w == nil {
		return plainSpot{}, false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.spot == nil {
		return plainSpot{}, false
	}
	return *w.spot, true
}

func (h *HUD) SeeTextAt(text string, locate func() (ui.Rect, bool)) {
	if !ui.StoryNaming() || h.hoverCovered() {
		return
	}
	r, ok := locate()
	if !ok || !r.Contains(h.MouseX, h.MouseY) {
		return
	}
	words := lexicon.PlainMatches(ui.Plain(text))
	if len(words) == 0 {
		return
	}
	if len(words) > plainTipLines {
		words = words[:plainTipLines]
	}
	h.Plain.set(plainSpot{rect: r, Words: words})
}

func (h *HUD) hoverCovered() bool {
	return h.Pin.On() || h.Hovering && h.Hover.Tip != ""
}

func (h *HUD) plainTipLines(f Frame) ([]string, ui.Rect, bool) {
	spot, ok := h.Plain.Get()
	if !ok || h.hoverCovered() {
		return nil, ui.Rect{}, false
	}
	lines := []string{}
	for _, w := range spot.Words {
		lines = append(lines, h.Font.Wrap(w.Line(), TipWidth, 1)...)
	}
	width := 0
	for _, l := range lines {
		lw, _ := h.Font.Measure(l, 1)
		width = max(width, lw)
	}
	b := spot.rect
	r := ui.Rect{X: b.X, Y: b.Y - len(lines)*ui.LineHeight - 6 - plainTipGap, W: width + 10, H: len(lines)*ui.LineHeight + 6}
	spots := []ui.Rect{r, {X: r.X, Y: b.Y + b.H + plainTipGap, W: r.W, H: r.H}}
	blocked := h.TipBlocked(f)
	for _, c := range blocked {
		if c.Contains(b.X+b.W/2, b.Y+b.H/2) {
			spots = append(spots, ui.Rect{X: r.X, Y: c.Y - r.H - plainTipGap, W: r.W, H: r.H}, ui.Rect{X: r.X, Y: c.Y + c.H + plainTipGap, W: r.W, H: r.H})
		}
	}
	for i, c := range spots {
		spots[i].X, spots[i].Y = min(max(c.X, 2), Width-c.W-2), min(max(c.Y, TopH), Height-c.H-2)
	}
	return lines, FirstClear(slidPast(spots, blocked), blocked), true
}

func (h *HUD) PlainTipRect(f Frame) (ui.Rect, bool) {
	_, r, ok := h.plainTipLines(f)
	return r, ok
}

func (h *HUD) DrawPlainTip(dst *ebiten.Image, f Frame) {
	lines, r, ok := h.plainTipLines(f)
	if !ok {
		return
	}
	ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, 235))
	ui.Stroke(dst, r, ui.Palette.Gold)
	for i, l := range lines {
		h.Font.Draw(dst, l, r.X+5, r.Y+4+i*ui.LineHeight, 1, ui.Palette.Cream)
	}
}
