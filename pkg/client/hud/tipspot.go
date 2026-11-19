package hud

import "github.com/waxb/kubernites/pkg/client/ui"

const (
	tipSlideStep    = 48
	tipSlides       = 6
	TipBlockerLimit = 30000
)

func (h *HUD) pinBlockers(buttons []Button, hovered Button, f Frame) []ui.Rect {
	out := h.otherTargets(f)
	_, holder, held := h.tipHolder(f)
	for _, r := range h.TipBlocked(f) {
		if !held || !holderItself(holder, r) {
			out = append(out, r)
		}
	}
	for _, b := range buttons {
		if TipButtonID(b.ID) || b.ID == hovered.ID && b.Rect == hovered.Rect || b.Rect.W*b.Rect.H > TipBlockerLimit {
			continue
		}
		out = append(out, b.Rect)
	}
	return out
}

func (h *HUD) pinSpots(r ui.Rect) []ui.Rect {
	out := h.tipSpots(r, false)
	for _, y := range []int{ConsoleY - r.H - tipGap, TopH + tipGap} {
		for i := 0; i <= tipSlides; i++ {
			for _, dx := range []int{i * tipSlideStep, -i * tipSlideStep} {
				out = append(out, ui.Rect{X: min(max(r.X+dx, 2), Width-r.W-2), Y: y, W: r.W, H: r.H})
			}
		}
	}
	return out
}

func (h *HUD) liveButton(b Button, buttons []Button) bool {
	for _, c := range buttons {
		if c.ID == b.ID && c.Rect == b.Rect && !TipButtonID(c.ID) {
			return true
		}
	}
	return false
}
