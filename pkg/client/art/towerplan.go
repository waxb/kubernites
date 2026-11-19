package art

import "math"

const (
	columnArc  = 150.0
	floorSlack = 7.0
	mostFloors = 3
	lintelRows = 2.0
	doorGap    = 3.0
	planKey    = 310
)

var columnTurn = [BuildingVariants]float64{0, 6, -6}

type towerPlan struct {
	floors, angles []float64
	door           int
	doorH, doorTop float64
	sparse         int
}

type pane struct {
	floor, col int
	deg, z     float64
}

func columnCount(r float64) int {
	switch {
	case r < 0.2:
		return 1
	case r < 0.4:
		return 2
	case r < 0.55:
		return 3
	}
	return 4
}

func columnAngles(shape, n int, turn float64) []float64 {
	switch shape {
	case shapeOct:
		return octColumns(n)
	case shapeSquare:
		return squareColumns(n)
	}
	return ringColumns(n, turn)
}

func octColumns(n int) []float64 {
	switch {
	case n <= 1:
		return []float64{45}
	case n == 2:
		return []float64{0, 90}
	}
	return []float64{0, 45, 90}
}

func squareColumns(n int) []float64 {
	switch {
	case n <= 1:
		return []float64{90}
	case n < 4:
		return []float64{0, 90}
	}
	return []float64{-22, 22, 68, 112}
}

func ringColumns(n int, turn float64) []float64 {
	n = max(n, 1)
	step := columnArc / float64(n)
	out := make([]float64, n)
	for i := range out {
		out[i] = 45 + turn + (float64(i)-float64(n-1)/2)*step
	}
	return out
}

func floorLevels(lo, hi, pitch float64) []float64 {
	out := []float64{}
	for z := hi; z >= lo && len(out) < mostFloors; z -= pitch {
		out = append(out, math.Round(z))
	}
	return out
}

func nearestColumn(angles []float64, want float64) int {
	best := 0
	for i, a := range angles {
		if math.Abs(a-want) < math.Abs(angles[best]-want) {
			best = i
		}
	}
	return best
}

func (s iso) columns(t towerSpec, n int) []float64 {
	return columnAngles(s.shapeOf(t), n, columnTurn[s.lk.variant%BuildingVariants])
}

func (s iso) capRows() float64 {
	if s.lk.age >= ageCastle {
		return 3
	}
	return 1
}

func (s iso) windowPlan(t towerSpec, z0, top float64) towerPlan {
	wh := float64(windowHeight[s.lk.age])
	return towerPlan{
		floors: floorLevels(z0, top-s.capRows()-wh-lintelRows, wh+floorSlack),
		angles: s.columns(t, columnCount(t.r)),
		door:   -1,
		sparse: s.lk.pick(planKey+int(t.cx*31+t.cy*17), 2),
	}
}

func (s iso) ringPlan(t towerSpec, z float64, n int) towerPlan {
	return towerPlan{floors: []float64{math.Round(z)}, angles: s.columns(t, n), door: -1, sparse: -1}
}

func (s iso) doorClear(h float64) float64 {
	return math.Ceil(h*s.kit().scale) + lintelRows + doorGap
}

func (s iso) withDoor(p towerPlan, want, h float64) towerPlan {
	p.door, p.doorH, p.doorTop = nearestColumn(p.angles, want), h, s.doorClear(h)
	return p
}

func (s iso) doorOnly(t towerSpec, want, h float64) towerPlan {
	return s.withDoor(towerPlan{angles: s.columns(t, columnCount(t.r)), door: -1, sparse: -1}, want, h)
}

func (s iso) doorPlan(t towerSpec, top, want, h float64) towerPlan {
	return s.withDoor(s.windowPlan(t, s.doorClear(h), top), want, h)
}

func (p towerPlan) upTo(n int) towerPlan {
	p.floors = p.floors[:min(n, len(p.floors))]
	return p
}

func (p towerPlan) open(floor, col int) bool {
	if col == p.door && p.floors[floor] < p.doorTop {
		return false
	}
	return floor == 0 || p.sparse < 0 || len(p.angles) < 2 || thinKeeps(col, len(p.angles), p.sparse)
}

func thinKeeps(col, n, parity int) bool {
	if n < 3 {
		return col%2 == parity
	}
	return min(col, n-1-col)%2 == parity
}

func (p towerPlan) panes() []pane {
	out := []pane{}
	for i, z := range p.floors {
		for j, deg := range p.angles {
			if p.open(i, j) {
				out = append(out, pane{floor: i, col: j, deg: deg, z: z})
			}
		}
	}
	return out
}

func (s iso) towerOpenings(t towerSpec, p towerPlan) {
	for _, q := range p.panes() {
		s.towerWindow(t, q.deg, q.z)
	}
	if p.door >= 0 && p.door < len(p.angles) {
		s.towerDoor(t, p.angles[p.door], p.doorH)
	}
}
