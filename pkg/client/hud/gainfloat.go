package hud

import (
	"math"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	gainDelay  = 3.0
	GainWindow = 20.0
	gainShare  = .6
	gainFloor  = .05
	gainShift  = .05
	GainLife   = 2.4
	gainRise   = 14.0
	gainFade   = .6
	gainTabH   = 11
	gainPad    = 3
	statIconW  = 18
)

var gainActions = map[string]bool{
	view.ActScaleCrew: true, "raiseCrew": true, "openDepot": true, "buyKeep": true, view.ActAutoscaleCrew: true,
}

type gainFloat struct {
	text  string
	Good  bool `json:"-"`
	start float64
	x     int
	life  float64
}

func (g gainFloat) span() float64 {
	if g.life > 0 {
		return g.life
	}
	return GainLife
}

type gainWatch struct {
	Armed  bool `json:"-"`
	base   float64
	Expect float64 `json:"-"`
	at     float64
	Float  gainFloat `json:"-"`
	shown  bool
}

func GrowthAction(a view.Action) bool {
	return gainActions[a.Action] || a.Action == "build" && IsKeepKind(a.ID)
}

func expectedGain(a view.Action, st view.GameState) float64 {
	if a.Action != view.ActScaleCrew {
		return 0
	}
	for _, c := range st.Crews {
		if c.Name == a.Name {
			return c.PerReplica.Income * float64(a.Replicas-c.Replicas)
		}
	}
	return 0
}

func (h *HUD) WatchGain(a view.Action, before view.GameState) {
	if !GrowthAction(a) {
		return
	}
	h.Coins.Gain.Armed = true
	h.Coins.Gain.base = IncomeRate(before)
	h.Coins.Gain.Expect = expectedGain(a, before)
	h.Coins.Gain.at = before.RealmTime
}

func (w gainWatch) threshold() float64 {
	if w.Expect != 0 {
		return max(gainFloor, gainShare*math.Abs(w.Expect))
	}
	return max(gainFloor, gainShift*math.Abs(w.base))
}

func (w gainWatch) moved(income, now float64) (float64, bool) {
	delta := income - w.base
	if now-w.at < gainDelay || math.Abs(delta) < w.threshold() {
		return delta, false
	}
	return delta, w.Expect == 0 && delta > 0 || delta*w.Expect > 0
}

func GainLabel(v float64, trueNames bool) string {
	sign := "+"
	if v < 0 {
		sign, v = "-", -v
	}
	amount := Compact(v)
	if v < 1 {
		amount = strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
	}
	if trueNames {
		return sign + "$" + amount + "/S"
	}
	return sign + amount + "/S"
}

func (h *HUD) stepGain(f Frame) {
	w := &h.Coins.Gain
	if w.shown && (h.events-w.Float.start >= w.Float.span() || h.events < w.Float.start) {
		w.shown = false
	}
	now := f.State.RealmTime
	if !w.Armed {
		return
	}
	if now < w.at || now-w.at > GainWindow || h.CoinFlights() == CoinFlightsOff {
		w.Armed = false
		return
	}
	delta, fire := w.moved(IncomeRate(f.State), now)
	if !fire {
		return
	}
	w.Armed, w.shown = false, true
	w.Float = gainFloat{text: GainLabel(delta, f.TrueNames), Good: delta > 0, start: h.events, x: h.incomeX(f)}
}

func (h *HUD) incomeX(f Frame) int {
	stats := h.TopStats(f)
	if len(stats) == 0 {
		return goldX
	}
	return stats[0].X + statIconW + h.Font.Width(stats[0].slot, 1) + ui.Advance
}

func (h *HUD) GainFloat() (string, bool) {
	return h.Coins.Gain.Float.text, h.Coins.Gain.shown
}

func GainOffset(age float64, reduce bool) (float64, uint8) {
	p := min(max(age/GainLife, 0), 1)
	alpha := uint8(255)
	if p > 1-gainFade {
		alpha = uint8(255 * (1 - (p-(1-gainFade))/gainFade))
	}
	if reduce {
		return 0, alpha
	}
	return gainRise * (1 - p), alpha
}

func (h *HUD) gainRect(f Frame) (ui.Rect, bool) {
	g := h.Coins.Gain
	if !g.shown {
		return ui.Rect{}, false
	}
	lift, _ := GainOffset((h.events-g.Float.start)*GainLife/g.Float.span(), h.Reduce)
	w := h.Font.Width(g.Float.text, 1)
	r := ui.Rect{X: g.Float.x - gainPad, Y: TopH + 1 + int(lift), W: w + 2*gainPad, H: gainTabH}
	for _, c := range inflated(h.topCards(f), TipClear) {
		if Overlaps(r, c) {
			r.Y = min(r.Y, c.Y-r.H)
		}
	}
	return r, r.Y >= TopH
}

func (h *HUD) drawGain(dst *ebiten.Image, f Frame) {
	g := h.Coins.Gain
	box, ok := h.gainRect(f)
	if !ok {
		return
	}
	_, alpha := GainOffset((h.events-g.Float.start)*GainLife/g.Float.span(), h.Reduce)
	clr := ui.Palette.Green
	if !g.Float.Good {
		clr = ui.Palette.RedText
	}
	ui.Fill(dst, box, ui.Tint(ui.Palette.ConsoleEdge, alpha))
	h.Font.Draw(dst, g.Float.text, box.X+gainPad, box.Y+2, 1, ui.Tint(clr, alpha))
}
