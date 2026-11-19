package app

import "github.com/waxb/kubernites/pkg/client/hud"

const coverStep = 4

func topmostAt(all []hud.Button, x, y int) (hud.Button, bool) {
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].Rect.Contains(x, y) {
			return all[i], true
		}
	}
	return hud.Button{}, false
}

func visiblePoint(all []hud.Button, b hud.Button) (int, int, string) {
	r := b.Rect
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	first, ok := topmostAt(all, cx, cy)
	if ok && first.ID == b.ID {
		return cx, cy, ""
	}
	for y := r.Y + 1; y < r.Y+r.H; y += coverStep {
		for x := r.X + 1; x < r.X+r.W; x += coverStep {
			if top, found := topmostAt(all, x, y); found && top.ID == b.ID {
				return x, y, ""
			}
		}
	}
	if ok {
		return cx, cy, first.ID
	}
	return cx, cy, "something"
}
