package hud

import (
	"image/color"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	DeltaLife = 1.6
	deltaRise = 14.0
	maxDeltas = 6
)

type delta struct {
	meter string
	Text  string `json:"-"`
	Good  bool   `json:"-"`
	start float64
	value float64
}

func DeltaText(v float64) string {
	switch {
	case math.Abs(v) < .005:
		return ""
	case v > 0:
		return "+" + Compact(v)
	}
	return "-" + Compact(-v)
}

func (h *HUD) Delta(meter string, v float64) {
	next := slices.Clone(h.LiveDeltas())
	if i := slices.IndexFunc(next, func(d delta) bool { return d.meter == meter }); i >= 0 {
		v += next[i].value
		next = slices.Delete(next, i, i+1)
	}
	if text := DeltaText(v); text != "" {
		next = append(next, delta{meter: meter, Text: text, Good: v > 0, start: h.Now, value: v})
	}
	h.deltas = next[max(len(next)-maxDeltas, 0):]
}

type deltaSpot struct {
	delta
	x, y  int
	rect  ui.Rect
	alpha uint8
}

func (h *HUD) deltaSpots(f Frame) []deltaSpot {
	out := []deltaSpot{}
	blocked := h.topCards(f)
	if box, _, ok := h.PurseTab(); ok {
		blocked = append(blocked, box)
	}
	if r, ok := h.gainRect(f); ok {
		blocked = append(blocked, r)
	}
	for _, st := range h.TopStats(f) {
		for _, d := range h.deltas {
			if st.Meter != d.meter {
				continue
			}
			rise, alpha := DeltaOffset(h.Now-d.start, f.Reduce)
			x, y := st.X+4, TopH+3+int(rise)
			r := h.textRect(d.Text, x, y).Inset(-1)
			if OverlapsAny(r, blocked) || OverlapsAny(r, spotRects(out)) {
				continue
			}
			out = append(out, deltaSpot{delta: d, x: x, y: y, rect: r, alpha: alpha})
		}
	}
	return out
}

func spotRects(spots []deltaSpot) []ui.Rect {
	out := make([]ui.Rect, 0, len(spots))
	for _, s := range spots {
		out = append(out, s.rect)
	}
	return out
}

func (d deltaSpot) color() color.RGBA {
	if !d.Good {
		return ui.Palette.RedText
	}
	return ui.Palette.Green
}

func (h *HUD) LiveDeltas() []delta {
	out := make([]delta, 0, len(h.deltas))
	for _, d := range h.deltas {
		if age := h.Now - d.start; age >= 0 && age < DeltaLife {
			out = append(out, d)
		}
	}
	return out
}

func DeltaOffset(age float64, reduce bool) (float64, uint8) {
	p := min(max(age/DeltaLife, 0), 1)
	alpha := uint8(255 * (1 - p*p))
	if reduce {
		return 0, alpha
	}
	return deltaRise * p, alpha
}

func (h *HUD) drawDeltas(dst *ebiten.Image, f Frame) {
	for _, d := range h.deltaSpots(f) {
		h.drawDelta(dst, d)
	}
}

func (h *HUD) drawDelta(dst *ebiten.Image, d deltaSpot) {
	h.Font.DrawShadowAs(dst, d.Text, d.x, d.y, 1, ui.Tint(d.color(), d.alpha), ui.Tint(ui.Palette.Ink, d.alpha))
}

type MeterDelta struct {
	Meter string
	Value float64
}

func MeterDeltas(before, after view.GameState) []MeterDelta {
	return []MeterDelta{{MeterGold, after.Gold - before.Gold}, {MeterWisdom, after.Wisdom - before.Wisdom}, {MeterRenown, after.Renown - before.Renown}}
}

func (h *HUD) textRect(text string, x, y int) ui.Rect {
	w, _ := h.Font.Measure(text, 1)
	return ui.Rect{X: x, Y: y, W: w + 1, H: ui.CapHeight + 1}
}

func (h *HUD) topCards(f Frame) []ui.Rect {
	out := []ui.Rect{}
	if h.CardsDrawn(f) && h.ObjectiveShown(f) {
		out = append(out, h.ObjectiveRectOf(f))
	}
	if h.LessonShown(f) {
		out = append(out, h.LessonRect(f))
	}
	return out
}
