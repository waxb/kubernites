package hud

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func (h *HUD) floatingRects(f Frame) []ui.Rect {
	out := h.panelRects(f)
	if h.Hovering && h.Hover.Tip != "" {
		out = append(out, h.TooltipRect(h.tipSize(h.Hover.Tip), f))
	}
	return out
}

func (h *HUD) panelRects(f Frame) []ui.Rect {
	out := []ui.Rect{}
	if h.OverlayID != OverlayNone {
		out = append(out, h.OverlayRect())
	}
	if h.CardsDrawn(f) && h.ObjectiveShown(f) {
		out = append(out, h.ObjectiveRectOf(f))
	}
	if h.LessonShown(f) {
		out = append(out, h.LessonRect(f))
	}
	if h.IntroShown(f) {
		out = append(out, h.IntroCard(f))
	}
	if r, ok := h.TrayRect(f); ok {
		out = append(out, r)
	}
	if r, ok := h.CalamityRect(f); ok {
		out = append(out, r)
	}
	if rail := h.RailRect(f); rail.W > 0 {
		out = append(out, rail)
	}
	if h.LegendShown(f) {
		out = append(out, LegendRect(h.lens))
	}
	if SiegeActive(f) {
		out = append(out, SiegeStripRect(f))
	}
	out = append(out, FreightRects(f)...)
	return append(out, h.ToastRects(f)...)
}

func OverlapsAny(r ui.Rect, rects []ui.Rect) bool {
	for _, c := range rects {
		if Overlaps(c, r) {
			return true
		}
	}
	return false
}

func (h *HUD) Covers(r ui.Rect, f Frame) bool {
	return h.IntroShown(f) || h.Hides(r, f)
}

func (h *HUD) Hides(r ui.Rect, f Frame) bool {
	return Overlaps(r, TopBar) || OverlapsAny(r, h.floatingRects(f)) || OverlapsAny(r, h.consoleRects(f))
}

func (h *HUD) drawTargetsOnTop(dst *ebiten.Image, f Frame) {
	if h.IntroShown(f) {
		if target, ok := h.IntroHole(f); ok {
			if h.holeStroked(f) {
				h.strokeHole(dst, target)
			}
			h.drawClearPointer(dst, f, h.IntroCard(f), target)
		}
		return
	}
	for _, r := range h.guideBeacons(f) {
		h.drawBeacon(dst, r)
	}
}

func (h *HUD) guideBeacons(f Frame) []ui.Rect {
	out := []ui.Rect{}
	if s, ok := h.Step(f); ok && !s.Map && !(researchStep(f) && h.Voice(f) == VoiceObjective) && !h.guideFaded(f) {
		out = append(out, s.Rect)
	}
	if b, ok := HitID(h.FreightButtons(f), nextRow); ok && h.Voice(f) == VoiceNext {
		out = append(out, b.Rect)
	}
	if r, ok := h.researchPage(f); ok {
		out = append(out, r)
	}
	return out
}

func (h *HUD) holeStroked(f Frame) bool { return !h.IntroHidden(f) }

func (h *HUD) researchPage(f Frame) (ui.Rect, bool) {
	o, ok := f.current()
	if !ok || o.Page == "" || h.OverlayID != OverlayGrimoire {
		return ui.Rect{}, false
	}
	b, ok := HitID(h.GrimoireButtons(f), "grim:"+o.Page)
	return b.Rect, ok
}

func (h *HUD) CardRects(f Frame) []ui.Rect {
	out := []ui.Rect{}
	if h.CardsDrawn(f) && h.ObjectiveShown(f) {
		out = append(out, h.ObjectiveRectOf(f))
	}
	if h.LessonShown(f) {
		out = append(out, h.LessonRect(f))
	}
	if h.IntroShown(f) {
		out = append(out, h.IntroCard(f))
	}
	if h.Hovering && h.Hover.Tip != "" {
		out = append(out, h.TooltipRect(h.tipSize(h.Hover.Tip), f))
	}
	return out
}
