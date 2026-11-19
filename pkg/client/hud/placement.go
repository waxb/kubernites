package hud

import "github.com/waxb/kubernites/pkg/client/ui"

const (
	cardMargin  = 6
	targetClear = 4
	beaconPad   = 2
)

func clampCard(r ui.Rect) ui.Rect {
	next := r
	next.X = min(max(r.X, cardMargin), Width-r.W-cardMargin)
	next.Y = min(max(r.Y, TopH+cardMargin), ConsoleY-r.H-cardMargin)
	return next
}

func overlapArea(a, b ui.Rect) int {
	w := min(a.X+a.W, b.X+b.W) - max(a.X, b.X)
	h := min(a.Y+a.H, b.Y+b.H) - max(a.Y, b.Y)
	if w <= 0 || h <= 0 {
		return 0
	}
	return w * h
}

func blockedArea(r ui.Rect, blocked []ui.Rect) int {
	total := 0
	for _, b := range blocked {
		total += overlapArea(r, b)
	}
	return total
}

func FirstClear(candidates, blocked []ui.Rect) ui.Rect {
	best, bestArea := ui.Rect{}, -1
	for _, c := range candidates {
		area := blockedArea(c, blocked)
		if area == 0 {
			return c
		}
		if bestArea < 0 || area < bestArea {
			best, bestArea = c, area
		}
	}
	return best
}

func ClearAvoiding(candidates, hard, soft []ui.Rect) ui.Rect {
	safe := make([]ui.Rect, 0, len(candidates))
	for _, c := range candidates {
		if blockedArea(c, hard) == 0 {
			safe = append(safe, c)
		}
	}
	if len(safe) > 0 {
		return FirstClear(safe, soft)
	}
	return FirstClear(candidates, append(append([]ui.Rect{}, hard...), soft...))
}

func around(target ui.Rect, w, h, gap int) []ui.Rect {
	cx, cy := target.X+target.W/2-w/2, target.Y+target.H/2-h/2
	return []ui.Rect{
		{X: cx, Y: target.Y - gap - h, W: w, H: h},
		{X: cx, Y: target.Y + target.H + gap, W: w, H: h},
		{X: target.X - gap - w, Y: cy, W: w, H: h},
		{X: target.X + target.W + gap, Y: cy, W: w, H: h},
		{X: target.X - gap - w, Y: target.Y, W: w, H: h},
		{X: target.X + target.W + gap, Y: target.Y, W: w, H: h},
	}
}

func clampAll(rects []ui.Rect) []ui.Rect {
	out := make([]ui.Rect, 0, len(rects))
	for _, r := range rects {
		out = append(out, clampCard(r))
	}
	return out
}

func inflated(rects []ui.Rect, n int) []ui.Rect {
	out := make([]ui.Rect, 0, len(rects))
	for _, r := range rects {
		out = append(out, r.Inset(-n))
	}
	return out
}

func (h *HUD) GuideTargets(f Frame) []ui.Rect {
	if h.IntroShown(f) {
		if r, ok := h.IntroHole(f); ok {
			return []ui.Rect{r.Inset(-HolePad - 1)}
		}
		return nil
	}
	out := append([]ui.Rect(nil), f.Beacons...)
	if s, ok := h.Step(f); ok && !s.Map {
		out = append(out, s.Rect.Inset(-beaconPad))
	}
	return out
}

func (h *HUD) cardBlocked(f Frame) []ui.Rect {
	blocked := inflated(h.GuideTargets(f), targetClear)
	if h.OverlayID != OverlayNone {
		blocked = append(blocked, h.OverlayRect())
	}
	return blocked
}

func lessonSpots(below ui.Rect, h int, anchor ui.Rect, anchored bool) []ui.Rect {
	w := Objective.W
	right := Width - w - railRight - railSize - cardMargin
	out := []ui.Rect{}
	if anchored {
		out = append(out, ui.Rect{X: anchor.X, Y: anchor.Y + anchor.H + lessonGap, W: w, H: h},
			ui.Rect{X: anchor.X, Y: anchor.Y - lessonGap - h, W: w, H: h})
	}
	return append(out, ui.Rect{X: below.X, Y: below.Y, W: w, H: h}, ui.Rect{X: right, Y: below.Y, W: w, H: h},
		ui.Rect{X: below.X, Y: ConsoleY - h - cardMargin, W: w, H: h}, ui.Rect{X: right, Y: ConsoleY - h - cardMargin, W: w, H: h})
}
