package view

import (
	"math"
)

type HandFlow struct {
	Seen   float64 `json:"-"`
	Rate   float64 `json:"-"`
	Primed bool    `json:"-"`
}

func (h HandFlow) Observe(total, dt, tau float64) HandFlow {
	if !h.Primed || dt <= 0 || tau <= 0 {
		return HandFlow{Seen: total, Rate: h.Rate, Primed: true}
	}
	sample := max(0, total-h.Seen) / dt
	return HandFlow{Seen: total, Rate: h.Rate + (1-math.Exp(-dt/tau))*(sample-h.Rate), Primed: true}
}

func (g PathGrid) endTile(e RouteEnd) int {
	if e.Exact {
		return e.Tile
	}
	d, _ := g.Door(e.S)
	return d
}

func (g PathGrid) AroundYards(havens []Haven) PathGrid {
	if len(havens) == 0 {
		return g
	}
	blocked := append([]bool(nil), g.Blocked...)
	for _, h := range havens {
		eachYardTile(h, func(t Point) bool {
			if i, ok := g.site.Index(t.X, t.Y); ok {
				blocked[i] = true
			}
			return true
		})
	}
	return PathGrid{site: g.site, Blocked: blocked}
}
