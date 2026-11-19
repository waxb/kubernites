package view

import (
	"math"
)

const havenSeaNear = 18

const havenSeaFar = 30

const havenSeaStride = 4

const havenLateral = 24

const havenLateralStep = 3

const HavenLaneMin = 12

const havenLaneFull = 18

const havenLaneWeight = 4

const HavenSeaRoom = 3

func eachLaneTile(a, b Point, visit func(x, y int) bool) bool {
	nx, ny := Abs(b.X-a.X), Abs(b.Y-a.Y)
	sx, sy := sign(b.X-a.X), sign(b.Y-a.Y)
	x, y := a.X, a.Y
	if !visit(x, y) {
		return false
	}
	for ix, iy := 0, 0; ix < nx || iy < ny; {
		switch d := (1+2*ix)*ny - (1+2*iy)*nx; {
		case d == 0:
			if !visit(x+sx, y) || !visit(x, y+sy) {
				return false
			}
			x, y, ix, iy = x+sx, y+sy, ix+1, iy+1
		case d < 0:
			x, ix = x+sx, ix+1
		default:
			y, iy = y+sy, iy+1
		}
		if !visit(x, y) {
			return false
		}
	}
	return true
}

func sign(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

func (p *havenPlanner) lane(h Haven) (Point, bool) {
	head := h.PierHead()
	best, bestScore, found := Point{}, 0.0, false
	for dist := havenSeaFar; dist >= havenSeaNear; dist -= havenSeaStride {
		for step := 0; step <= havenLateral; step += havenLateralStep {
			for _, side := range [2]int{p.lean, -p.lean} {
				sea := p.clampSea(h.Quay.Step(h.Out, dist).Step(h.Along(), side*step))
				length := distance(sea, head)
				score := havenLaneWeight*math.Min(length, havenLaneFull) - float64(step)
				if length < HavenLaneMin || found && score <= bestScore || !p.laneClear(h, sea) {
					continue
				}
				best, bestScore, found = sea, score, true
			}
		}
	}
	return best, found
}

func (p *havenPlanner) clampSea(pt Point) Point {
	return Point{X: min(max(pt.X, 1), p.site.Width-2), Y: min(max(pt.Y, 1), p.site.Height-2)}
}

func (p *havenPlanner) laneClear(h Haven, sea Point) bool {
	if p.site.Tile(sea.X, sea.Y) != TileWater || !p.openAround(sea, HavenSeaRoom) {
		return false
	}
	if !eachLaneTile(sea, h.PierHead(), func(x, y int) bool { return p.openSea(x, y) && !p.claimed(x, y) }) {
		return false
	}
	for _, b := range h.Berths() {
		if !eachLaneTile(sea, b, func(x, y int) bool { return p.water(Point{X: x, Y: y}) && !p.claimed(x, y) }) {
			return false
		}
	}
	return true
}

func (p *havenPlanner) openSea(x, y int) bool {
	return p.water(Point{X: x, Y: y}) && p.openAround(Point{X: x, Y: y}, 1)
}

func (p *havenPlanner) openAround(c Point, room int) bool {
	for dy := -room; dy <= room; dy++ {
		for dx := -room; dx <= room; dx++ {
			if IsLand(p.site.Tile(c.X+dx, c.Y+dy)) {
				return false
			}
		}
	}
	return true
}

func havenBox(h Haven) Rect {
	water := spanRect(h.Quay.Step(h.Out, 1).Step(h.Along(), -1), h.Quay.Step(h.Out, h.Pier+1).Step(h.Along(), 1))
	yards := h.yardRects()
	box := h.QuayRect()
	for _, r := range [3]Rect{yards[0], yards[1], water} {
		box = Rect{X0: min(box.X0, r.X0), Y0: min(box.Y0, r.Y0), X1: max(box.X1, r.X1), Y1: max(box.Y1, r.Y1)}
	}
	return box
}

func (p *havenPlanner) claimed(x, y int) bool {
	for _, o := range p.placed {
		if havenBox(o).Contains(x, y) {
			return true
		}
	}
	return false
}

func (p *havenPlanner) clearOfPlaced(h Haven) bool {
	box := havenBox(h)
	outside := func(x, y int) bool { return !box.Contains(x, y) }
	for _, o := range p.placed {
		if havenBox(o).Gap(box) < havenSpacing || !eachLaneTile(o.Sea, o.PierHead(), outside) {
			return false
		}
		for _, b := range o.Berths() {
			if !eachLaneTile(o.Sea, b, outside) {
				return false
			}
		}
	}
	return true
}
