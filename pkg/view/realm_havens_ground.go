package view

var compass = [4]Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}}

func eachQuayTile(h Haven, visit func(t Point, depth int) bool) bool {
	for depth := 0; depth < QuayDepth; depth++ {
		for k := -QuayHalf; k <= QuayHalf; k++ {
			if !visit(h.Quay.Step(h.Along(), k).Step(h.Out, -depth), depth) {
				return false
			}
		}
	}
	return true
}

func eachYardTile(h Haven, visit func(t Point) bool) bool {
	for _, y := range h.yardRects() {
		for ty := y.Y0; ty <= y.Y1; ty++ {
			for tx := y.X0; tx <= y.X1; tx++ {
				if !visit(Point{X: tx, Y: ty}) {
					return false
				}
			}
		}
	}
	return true
}

func (p *havenPlanner) quayCost(h Haven) (float64, bool) {
	q := h.QuayRect()
	if p.plaza.Gap(q) <= havenCivicGap || p.gate.Gap(q) < 1 || !p.yardsClear(h) {
		return 0, false
	}
	for _, s := range p.site.Structures {
		if s.Kind != StructGate && Footprint(s).Gap(q) < havenCivicGap {
			return 0, false
		}
	}
	cost := 0.0
	ok := eachQuayTile(h, func(t Point, depth int) bool {
		c, fits := p.quayGround(t, depth)
		cost += c
		return fits
	}) && eachYardTile(h, func(t Point) bool {
		c, fits := p.quayGround(t, QuayDepth)
		cost += c
		return fits
	})
	return cost, ok
}

func (p *havenPlanner) yardsClear(h Haven) bool {
	for _, y := range h.yardRects() {
		if p.plaza.Gap(y) <= havenCivicGap || p.gate.Gap(y) < havenCivicGap {
			return false
		}
		for _, s := range p.site.Structures {
			if Footprint(s).Gap(y) < havenCivicGap {
				return false
			}
		}
	}
	return true
}

func (p *havenPlanner) quayGround(t Point, depth int) (float64, bool) {
	i, ok := p.site.Index(t.X, t.Y)
	if !ok || p.nearTower(t) {
		return 0, false
	}
	switch tile := p.site.Tiles[i]; {
	case tile == TileShallows:
		return havenFillCost, depth < QuayDepth-1 && p.fillable(t)
	case !p.reach[i] || p.site.ElevationAt(t.X, t.Y) != 0:
		return 0, false
	case tile == TileForest:
		return havenClearCost, true
	default:
		return 0, tile == TileSand || tile == TileGrass || tile == TileMeadow
	}
}

func (p *havenPlanner) nearTower(t Point) bool {
	dx, dy, clear := t.X-p.site.TowerSpot.X, t.Y-p.site.TowerSpot.Y, TowerClearing+havenCivicGap
	return dx*dx+dy*dy <= clear*clear
}

func (p *havenPlanner) fillable(t Point) bool {
	for _, d := range compass {
		n := t.Step(d, 1)
		if j, ok := p.site.Index(n.X, n.Y); ok && IsLand(p.site.Tiles[j]) && !homeLand(p.site, j) {
			return false
		}
	}
	return true
}

func (p *havenPlanner) settled(havens []Haven) Site {
	next := p.site
	next.Havens = havens
	next.Tiles = append([]TileKind(nil), p.site.Tiles...)
	next.Region = append([]uint8(nil), p.site.Region...)
	shape := func(t Point) bool {
		i, _ := next.Index(t.X, t.Y)
		switch next.Tiles[i] {
		case TileForest:
			next.Tiles[i] = TileGrass
		case TileShallows:
			next.Tiles[i] = TileSand
			if i < len(next.Region) {
				next.Region[i] = p.home
			}
		}
		return true
	}
	for _, h := range havens {
		eachQuayTile(h, func(t Point, _ int) bool { return shape(t) })
		eachYardTile(h, shape)
	}
	return next
}

func (p *havenPlanner) gateSteps() []int {
	steps := make([]int, len(p.site.Tiles))
	blocked := NewPathGrid(p.site).Blocked
	queue := []int{}
	for i, t := range p.site.Tiles {
		steps[i] = -1
		x, y := i%p.site.Width, i/p.site.Width
		if t == TileRoad && !blocked[i] && p.gate.Gap(Rect{X0: x, Y0: y, X1: x, Y1: y}) <= havenGateReach {
			steps[i] = 0
			queue = append(queue, i)
		}
	}
	for ; len(queue) > 0; queue = queue[1:] {
		i := queue[0]
		if steps[i] >= HavenGateWalk-1 {
			continue
		}
		for _, d := range compass {
			if j, ok := p.site.Index(i%p.site.Width+d.X, i/p.site.Width+d.Y); ok && steps[j] < 0 && p.reach[j] && !blocked[j] {
				steps[j] = steps[i] + 1
				queue = append(queue, j)
			}
		}
	}
	return steps
}

func (p *havenPlanner) walk(h Haven) int {
	q, best := h.QuayRect(), -1
	eachQuayTile(h, func(t Point, _ int) bool {
		for _, d := range compass {
			n := t.Step(d, 1)
			i, ok := p.site.Index(n.X, n.Y)
			if ok && !q.Contains(n.X, n.Y) && p.steps[i] >= 0 && (best < 0 || p.steps[i]+1 < best) {
				best = p.steps[i] + 1
			}
		}
		return true
	})
	return best
}

func homeRegion(s Site) uint8 {
	for i, is := range s.Islands {
		if is.ID == HomeIslandID {
			return uint8(i + 1)
		}
	}
	return 1
}

func homeReach(s Site) []bool {
	reach := make([]bool, len(s.Tiles))
	start, ok := s.Index(s.Plaza.X, s.Plaza.Y)
	if !ok || !homeLand(s, start) {
		start, ok = s.Index(s.Gate.X, s.Gate.Y)
	}
	if !ok || !homeLand(s, start) {
		return reach
	}
	reach[start] = true
	for queue := []int{start}; len(queue) > 0; queue = queue[1:] {
		x, y := queue[0]%s.Width, queue[0]/s.Width
		for _, d := range compass {
			if j, ok := s.Index(x+d.X, y+d.Y); ok && !reach[j] && homeLand(s, j) {
				reach[j] = true
				queue = append(queue, j)
			}
		}
	}
	return reach
}

func homeLand(s Site, i int) bool {
	if !IsLand(s.Tiles[i]) {
		return false
	}
	isle, ok := s.IslandAt(i%s.Width, i/s.Width)
	return !ok && len(s.Islands) == 0 || ok && isle.ID == HomeIslandID
}
