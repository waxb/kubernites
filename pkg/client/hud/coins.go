package hud

import (
	"fmt"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	goldX      = 80
	goldIcon   = 16
	coinFlight = .6
	coinLift   = 30.0
	coinPulse  = .35
	coinPlus   = .9
	coinGap    = .25
	purseTabH  = 10
	coinRadius = 3
	maxFlights = 24
)

type flight struct {
	from  [2]float64
	start float64
	value float64
	Label string `json:"-"`
}

type purse struct {
	Flights   []flight `json:"-"`
	Landed    float64  `json:"-"`
	landAt    float64
	batchAt   float64
	mode      string
	flown     int
	income    float64
	trueNames bool
	Gain      gainWatch `json:"-"`
}

type CoinTap struct {
	Carry float64
	Last  float64
	Fired bool
}

func (t CoinTap) Step(now, earned float64) (CoinTap, int) {
	return t.StepEvery(now, earned, coinGap)
}

func (t CoinTap) StepEvery(now, earned, gap float64) (CoinTap, int) {
	next := t
	next.Carry += max(earned, 0)
	if next.Carry < 1 || (t.Fired && now-t.Last < gap-1e-9) {
		return next, 0
	}
	v := int(math.Floor(next.Carry))
	next.Carry -= float64(v)
	next.Last, next.Fired = now, true
	return next, v
}

func GoldAnchor() (int, int) {
	return goldX + goldIcon/2, 1 + goldIcon/2
}

func CoinPath(from, to [2]float64, t float64) [2]float64 {
	t = min(max(t, 0), 1)
	u := t * t * (3 - 2*t)
	ctrl := [2]float64{(from[0] + to[0]) / 2, min(from[1], to[1]) - coinLift}
	a, b, c := (1-u)*(1-u), 2*(1-u)*u, u*u
	return [2]float64{a*from[0] + b*ctrl[0] + c*to[0], a*from[1] + b*ctrl[1] + c*to[1]}
}

func CounterPulse(age float64) float64 {
	if age < 0 || age >= coinPulse {
		return 0
	}
	return 1 - age/coinPulse
}

func (h *HUD) LaunchCoin(x, y, value float64) {
	h.launch(x, y, value, "", false)
}

func (h *HUD) LaunchHandCoin(x, y, value float64) {
	h.launch(x, y, value, GoldLabel(value), false)
}

func (h *HUD) LaunchGoldenCoin(x, y, value float64) {
	h.launch(x, y, value, GoldLabel(value), true)
}

func (h *HUD) launch(x, y, value float64, label string, golden bool) {
	if value <= 0 {
		return
	}
	if h.Reduce || !h.coinPolicy().Flies(value, golden) {
		h.land(value)
		return
	}
	p := &h.Coins
	p.flown++
	if len(p.Flights) >= maxFlights {
		p.Flights[len(p.Flights)-1].value += value
		return
	}
	p.Flights = append(p.Flights, flight{from: [2]float64{x, y}, start: h.events, value: value, Label: label})
}

func (h *HUD) land(value float64) {
	p := &h.Coins
	if p.Landed == 0 || h.events-p.batchAt > coinPlus || h.events < p.batchAt {
		p.Landed, p.batchAt = 0, h.events
	}
	p.Landed += value
	p.landAt = h.events
}

func (h *HUD) stepCoins(f Frame) {
	h.Coins.income, h.Coins.trueNames = f.State.GoldPerSec, f.TrueNames
	kept := make([]flight, 0, len(h.Coins.Flights))
	for _, fl := range h.Coins.Flights {
		if h.events-fl.start >= coinFlight || h.events < fl.start {
			h.land(fl.value)
			continue
		}
		kept = append(kept, fl)
	}
	h.Coins.Flights = kept
	if h.events-h.Coins.batchAt > coinPlus || h.events < h.Coins.batchAt {
		h.Coins.Landed = 0
	}
	h.stepGain(f)
}

func (h *HUD) CoinsInFlight() int { return len(h.Coins.Flights) }

func (h *HUD) GoldPulse() float64 {
	if h.Coins.Landed == 0 {
		return 0
	}
	return CounterPulse(h.events - h.Coins.landAt)
}

func (h *HUD) drawCoins(dst *ebiten.Image) {
	ax, ay := GoldAnchor()
	to := [2]float64{float64(ax), float64(ay)}
	for _, f := range h.Coins.Flights {
		p := CoinPath(f.from, to, (h.events-f.start)/coinFlight)
		name := fmt.Sprintf("fx/coin/%d", int(h.events*12)%6)
		if !h.Lib.Draw(dst, name, p[0], p[1]+4, 1, render.SpriteOptions{}) {
			vector.FillCircle(dst, float32(p[0]), float32(p[1]), coinRadius+1, ui.Palette.Ink, false)
			vector.FillCircle(dst, float32(p[0]), float32(p[1]), coinRadius, ui.Palette.GoldBright, false)
		}
		h.drawCoinLabel(dst, f.Label, p)
	}
}

func (h *HUD) drawCoinLabel(dst *ebiten.Image, label string, at [2]float64) {
	if label == "" {
		return
	}
	w, _ := h.Font.Measure(label, 1)
	box := ui.Rect{X: int(at[0]) + coinRadius + 3, Y: int(at[1]) - 5, W: w + 6, H: 11}
	ui.Fill(dst, box, ui.Tint(ui.Palette.ConsoleEdge, 210))
	h.Font.Draw(dst, label, box.X+3, box.Y+2, 1, ui.Palette.GoldBright)
}

func (h *HUD) drawGoldPulse(dst *ebiten.Image) {
	if pulse := h.GoldPulse(); pulse > 0 {
		grow := int(2 * (1 - pulse))
		r := ui.Rect{X: goldX - 1 - grow, Y: 0, W: goldIcon + 2 + 2*grow, H: TopH}
		ui.Stroke(dst, r, ui.Tint(ui.Palette.GoldBright, uint8(255*pulse)))
	}
}

func (h *HUD) PurseTab() (ui.Rect, string, bool) {
	if h.Coins.Landed == 0 {
		return ui.Rect{}, "", false
	}
	text := PurseLabel(h.Coins.Landed, h.Coins.trueNames)
	w, _ := h.Font.Measure(text, 1)
	ax, _ := GoldAnchor()
	return ui.Rect{X: ax - w/2 - 3, Y: TopH, W: w + 6, H: purseTabH}, text, true
}

func (h *HUD) DrawPurseTab(dst *ebiten.Image, f Frame) {
	h.drawGain(dst, f)
	h.drawPurseOnly(dst)
}

func (h *HUD) drawPurseOnly(dst *ebiten.Image) {
	box, text, ok := h.PurseTab()
	if !ok {
		return
	}
	age := h.events - h.Coins.batchAt
	alpha := uint8(255 * min(max(1-age/coinPlus, 0), 1))
	ui.Fill(dst, box, ui.Tint(ui.Palette.ConsoleEdge, alpha))
	h.Font.Draw(dst, text, box.X+3, box.Y+2, 1, ui.Tint(ui.Palette.GoldBright, alpha))
}
