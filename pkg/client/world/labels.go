package world

import (
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const labelGap = 2

func keepInView(box ui.Rect) ui.Rect {
	if box.W > ViewWidth {
		return box
	}
	box.X = min(max(box.X, 0), ViewWidth-box.W)
	return box
}

func SettleLabel(box ui.Rect, anchorX int, panels []ui.Rect) (ui.Rect, bool) {
	box = keepInView(box)
	if !overlapsAnyRect(box, panels) {
		return box, true
	}
	best, found := ui.Rect{}, false
	for _, p := range panels {
		if !Overlaps(box, p) {
			continue
		}
		for _, x := range []int{p.X - labelGap - box.W, p.X + p.W + labelGap} {
			c := ui.Rect{X: x, Y: box.Y, W: box.W, H: box.H}
			if !labelFits(c, anchorX, panels) {
				continue
			}
			if !found || Abs(x-box.X) < Abs(best.X-box.X) {
				best, found = c, true
			}
		}
	}
	return best, found
}

func labelFits(c ui.Rect, anchorX int, panels []ui.Rect) bool {
	return c.X <= anchorX && anchorX <= c.X+c.W && c.X >= 0 && c.X+c.W <= ViewWidth && !overlapsAnyRect(c, panels)
}

func (r *Renderer) labelObstacles(cam render.Camera, sc Scene) []ui.Rect {
	out := append([]ui.Rect(nil), sc.Panels...)
	if box, ok := r.signpostLabel(cam, sc); ok {
		out = append(out, box)
	}
	for _, g := range r.gates {
		sx, _, home := gateBox(cam, sc, g)
		if box, ok := SettleLabel(home, int(sx), sc.Panels); ok {
			out = append(out, box)
		}
	}
	return out
}

const stackTries = 12

func StackLabel(box ui.Rect, anchorX int, panels, placed []ui.Rect) (ui.Rect, bool) {
	for _, dir := range []int{-1, 1} {
		if got, ok := stackToward(box, anchorX, panels, placed, dir); ok {
			return got, true
		}
	}
	return ui.Rect{}, false
}

func stackToward(box ui.Rect, anchorX int, panels, placed []ui.Rect, dir int) (ui.Rect, bool) {
	cur := box
	for range stackTries {
		settled, ok := SettleLabel(cur, anchorX, panels)
		if !ok || settled.Y < 0 || settled.Y+settled.H > ViewHeight {
			return ui.Rect{}, false
		}
		cur = settled
		hit, found := firstOverlap(cur, placed)
		if !found {
			return cur, true
		}
		if dir < 0 {
			cur.Y = hit.Y - labelGap - cur.H
		} else {
			cur.Y = hit.Y + hit.H + labelGap
		}
	}
	return ui.Rect{}, false
}

func firstOverlap(box ui.Rect, rects []ui.Rect) (ui.Rect, bool) {
	for _, r := range rects {
		if Overlaps(box, r) {
			return r, true
		}
	}
	return ui.Rect{}, false
}

func midX(box ui.Rect) int { return box.X + box.W/2 }

func (r *Renderer) settleLabel(box ui.Rect, anchorX int, panels []ui.Rect) (ui.Rect, bool) {
	got, ok := StackLabel(box, anchorX, panels, r.placed)
	if ok {
		r.placed = append(r.placed, got)
		r.note(DrawnLabel, "label", "", got)
	}
	return got, ok
}
