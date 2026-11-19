package world

import (
	"math"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	nearRing0 = .5
	nearRing1 = 3.4
)

type nearKey struct {
	keep   string
	lo, hi Point
}

type nearSpot struct {
	at  Point
	gap float64
	ok  bool
}

func (h *Hamlet) depotAnchor(site view.Site, keep, depot view.Structure, fallback hutAnchor) hutAnchor {
	lo, hi := Footprint(depot)
	near, gap, ok := h.Nearest(site, keep, lo, hi)
	switch {
	case !ok:
		return fallback
	case gap <= depotRing1:
		return hutAnchor{lo: lo, hi: hi, ring0: depotRing0, ring1: depotRing1, spread: depotSpread}
	}
	return hutAnchor{lo: near, hi: near, ring0: nearRing0, ring1: nearRing1, spread: depotSpread}
}

func (h *Hamlet) Nearest(site view.Site, keep view.Structure, lo, hi Point) (Point, float64, bool) {
	key := nearKey{keep: keep.ID, lo: lo, hi: hi}
	if n, ok := h.Nears[key]; ok {
		return n.at, n.gap, n.ok
	}
	best := nearSpot{gap: math.Inf(1)}
	if area, known := h.areas[keep.ID]; known {
		for _, i := range h.list(site, area) {
			x, y := i%h.w, i/h.w
			if g := FootprintGap(lo, hi, x, y); g < best.gap {
				best = nearSpot{at: Point{x, y}, gap: g, ok: true}
			}
		}
	}
	h.Nears[key] = best
	return best.at, best.gap, best.ok
}
