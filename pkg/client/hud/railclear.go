package hud

import "github.com/waxb/kubernites/pkg/client/ui"

const railClearGap = 4

func (f Frame) railBox() ui.Rect {
	shown, more := railCount(len(f.Notices))
	if shown == 0 || !f.Shown(SurfaceRail) || f.railQuiet() {
		return ui.Rect{}
	}
	if more {
		shown++
	}
	top, bottom := railSlot(0), railSlot(shown-1)
	home := ui.Rect{X: top.X, Y: top.Y - railGripH, W: top.W, H: bottom.Y + bottom.H - top.Y + railGripH}
	if p, ok := f.Panels[PanelNotices]; ok {
		return clampPanel(ui.Rect{X: p.X, Y: p.Y, W: home.W, H: home.H})
	}
	return home
}

func clearOf(r, rail ui.Rect) ui.Rect {
	if rail.W == 0 || !Overlaps(r, rail) {
		return r
	}
	best, bestCost, found := r, 0, false
	for _, c := range []ui.Rect{
		{X: rail.X - railClearGap - r.W, Y: r.Y, W: r.W, H: r.H},
		{X: rail.X + rail.W + railClearGap, Y: r.Y, W: r.W, H: r.H},
		{X: r.X, Y: rail.Y - railClearGap - r.H, W: r.W, H: r.H},
		{X: r.X, Y: rail.Y + rail.H + railClearGap, W: r.W, H: r.H},
	} {
		if clampPanel(c) != c {
			continue
		}
		cost := iabs(c.X-r.X) + iabs(c.Y-r.Y)
		if !found || cost < bestCost {
			best, bestCost, found = c, cost, true
		}
	}
	return best
}
