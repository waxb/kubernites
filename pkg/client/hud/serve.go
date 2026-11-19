package hud

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	CrewTakesToast   = "The porters carry the parcels now. Your hands still earn: click the pile any time."
	HandsNeededToast = "No crew carries the parcels! Carry them by hand."
	freshBonus       = .25
	crownedPays      = 2
	sealedPays       = 1.5
	tipCursorGap     = 14
)

func parcelKind(fv view.FreightView) (view.KindView, bool) {
	return kindOf(fv, "parcel")
}

func kindOf(fv view.FreightView, kind string) (view.KindView, bool) {
	for _, k := range fv.Kinds {
		if k.Kind == kind {
			return k, true
		}
	}
	return view.KindView{}, false
}

func HandsNeeded(fv view.FreightView) bool {
	k, ok := parcelKind(fv)
	return !ok || !k.DepotOpen || k.Crew == 0
}

func nextHandBox(k view.KindView) (view.CohortView, bool) {
	var best view.CohortView
	found := false
	for _, c := range k.Cohorts {
		if c.Count < .5 {
			continue
		}
		if !found || c.Crowned && !best.Crowned || c.Crowned == best.Crowned && c.Remaining < best.Remaining {
			best, found = c, true
		}
	}
	return best, found
}

func HandValue(fv view.FreightView) float64 {
	return HandValueOf(fv, "parcel")
}

func HandValueOf(fv view.FreightView, kind string) float64 {
	k, ok := kindOf(fv, kind)
	if !ok {
		return 0
	}
	c, ok := nextHandBox(k)
	if !ok {
		return k.Value * (1 + freshBonus)
	}
	fresh := 1.0
	if c.TTL > 0 {
		fresh = min(max(c.Remaining/c.TTL, 0), 1)
	}
	v := k.Value * (1 + freshBonus*fresh)
	if c.Crowned {
		v *= crownedPays
	}
	if c.Sealed {
		v *= sealedPays
	}
	return v
}

func GoldLabel(v float64) string {
	if v >= 10 {
		v = math.Round(v)
	}
	return "+" + Compact(v) + " gold"
}

func (h *HUD) WorldTipRect(f Frame) (ui.Rect, bool) {
	r, ok, _ := h.worldTip(f)
	return r, ok
}

func (h *HUD) worldTip(f Frame) (ui.Rect, bool, bool) {
	if f.Tip == "" {
		return ui.Rect{}, false, false
	}
	lines := h.tipSize(f.Tip)
	w := 0
	font := h.tipFont(f.Tip)
	for _, l := range lines {
		lw, _ := font.Measure(l, 1)
		w = max(w, lw)
	}
	blocked := h.TipBlocked(f)
	spots := []ui.Rect{}
	for _, dx := range []int{tipCursorGap, -tipCursorGap - w - 10} {
		for _, dy := range []int{tipCursorGap, -tipCursorGap - len(lines)*ui.LineHeight - 6} {
			r := ui.Rect{X: f.TipAt[0] + dx, Y: f.TipAt[1] + dy, W: w + 10, H: len(lines)*ui.LineHeight + 6}
			r.X = min(max(r.X, cardMargin), Width-r.W-cardMargin)
			r.Y = min(max(r.Y, TopH+cardMargin), ConsoleY-r.H-cardMargin)
			spots = append(spots, r)
		}
	}
	candidates := slidPast(spots, blocked)
	for i, r := range candidates {
		candidates[i].Y = min(max(r.Y, TopH+cardMargin), ConsoleY-r.H-cardMargin)
	}
	r := FirstClear(candidates, blocked)
	return r, true, !OverlapsAny(r, blocked)
}

func (h *HUD) drawWorldTip(dst *ebiten.Image, f Frame) {
	r, ok := h.WorldTipRect(f)
	if !ok || h.Hovering {
		return
	}
	ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, 235))
	ui.Stroke(dst, r, ui.Palette.Gold)
	lines, styles := h.layoutTip(f.Tip)
	h.drawTipLines(dst, r, lines, styles)
}
