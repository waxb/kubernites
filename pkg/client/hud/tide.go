package hud

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	tideRow    = FreightTip + "tide"
	idleBoxes  = 1.0
	servedSlip = 0.98
)

type tideSpan struct {
	until float64
	story string
	k8s   string
	clr   func() color.RGBA
}

var tideSpans = []tideSpan{
	{0.125, "Night: fewer ships until dawn", "Night: less traffic until dawn", func() color.RGBA { return ui.Palette.Blue }},
	{0.375, "Morning: more ships each hour", "Morning: traffic builds", func() color.RGBA { return ui.Palette.Gold }},
	{0.625, "Noon: the most ships", "Noon: peak traffic", func() color.RGBA { return ui.Palette.GoldBright }},
	{0.875, "Evening: the ships thin out", "Evening: traffic thins", func() color.RGBA { return ui.Palette.Gold }},
	{1.125, "Night: fewer ships until dawn", "Night: less traffic until dawn", func() color.RGBA { return ui.Palette.Blue }},
}

func tideOf(phase float64) tideSpan {
	if phase < tideSpans[0].until {
		phase++
	}
	for _, s := range tideSpans[1:] {
		if phase < s.until {
			return s
		}
	}
	return tideSpans[len(tideSpans)-1]
}

func TideLabel(phase float64) string {
	return ui.Bridge(tideOf(phase).story, tideOf(phase).k8s)
}

func TideLevel(phase float64) float64 { return view.DayFactor(phase) }

func TideShown(f Frame) bool { return BoardShown(f) }

func TideTip() string {
	return tipLines("The tide of ships: they swell at noon and ebb at night, between a third and the whole of the day's peak.",
		"When the tide is low fewer boxes arrive, so income falls and extra hands sit idle until the ships return.")
}

func (h *HUD) drawTideRow(dst *ebiten.Image, f Frame, r ui.Rect, x, y int) {
	phase := f.State.Pilgrims.DayPhase
	level := TideLevel(phase)
	h.drawBarRow(dst, r, x, y, TideLabel(phase), fmt.Sprintf("%d%%", int(level*100+.5)), level, tideOf(phase).clr())
}

func idleScale(st view.GameState) bool {
	n := st.Next
	if n == nil || n.Kind != view.NextMilestone {
		return false
	}
	for _, k := range st.Freight.Kinds {
		if k.CrewName != n.ID {
			continue
		}
		return k.Waiting < idleBoxes && k.SpoiledRate <= 0 && k.ServedRate >= k.ArrivalRate*servedSlip
	}
	return false
}

func WithoutIdleScale(st view.GameState) view.GameState {
	if idleScale(st) {
		st.Next = nil
	}
	return st
}
