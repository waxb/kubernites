package art

import (
	"math"
	"sort"
)

func (s iso) paintKeep(k KeepLook) {
	s.apron(keepFoot, 0.05)
	switch k.Size {
	case "tower":
		s.loneTower(k)
	case "great":
		s.greatKeep(k)
	case "citadel":
		s.citadelKeep(k)
	default:
		s.stoneKeep(k)
	}
	s.keepMotifs(k, keepYards(k, s.lk))
}

func (s iso) keepHallChimney(k KeepLook) {
	c := keepChimney(k, s.lk)
	s.chimney(c.gx, c.gy, c.z0, c.z1)
}

func (s iso) keepApex(t towerSpec, top float64) float64 {
	if s.lk.age == ageEmpire {
		return top + t.roofH*0.8 + 6
	}
	return top + t.roofH
}

func (s iso) keepFlag(k KeepLook, t towerSpec, top float64, pole float64) {
	s.pen(matTrim, func() { s.flag(t.cx, t.cy, s.keepApex(t, top)-1, pole, keepCloth(k)) })
}

func (s iso) keepTower(k KeepLook, t towerSpec, flag bool) float64 {
	top := s.tower(t)
	s.towerOpenings(t, s.windowPlan(t, t.z0, top).upTo(keepFloors(t.h)))
	if flag {
		s.keepFlag(k, t, top, 7+float64(s.lk.pick(int(t.cx*31+t.cy*17), 4)))
	}
	return top
}

func (s iso) loneTower(k KeepLook) {
	s.keepTower(k, towerSpec{cx: 0.45, cy: 0.45, r: 0.24, h: 34 + s.lk.jit(2, 4), roofH: 13}, s.lk.chance(3, 0.5))
	if annexLeft(s.lk) {
		s.hall(hallSpec{x0: 0.2, y0: 1.2, x1: 1.0, y1: 2.2, h: annexH, rise: 9, alongY: true})
		s.window(sideL(2.2), 0.45, 8)
	} else {
		s.hall(hallSpec{x0: 1.2, y0: 0.2, x1: 2.2, y1: 1.0, h: annexH, rise: 9})
		s.window(sideR(2.2), 0.45, 8)
	}
	s.keepHallChimney(k)
	t := towerSpec{cx: 1.62, cy: 1.55, r: 0.6, h: 58 + s.lk.jit(4, 4), roofH: 24}
	s.keepTower(k, t, true)
	s.towerOpenings(t, s.doorOnly(t, 100, 13))
}

func keepFloors(h float64) int {
	if h > 44 {
		return 2
	}
	return 1
}

var stoneTowerSpots = [3][2][2]float64{
	{{0.4, 0.4}, {0.4, 1.75}},
	{{0.4, 0.4}, {1.75, 0.4}},
	{{0.4, 1.75}, {1.75, 0.4}},
}

var stoneWalls = [3][4]float64{
	{0.3, 0.55, 0.5, 1.65},
	{0.55, 0.3, 1.65, 0.5},
	{0.3, 1.5, 0.5, 1.95},
}

func (s iso) stoneKeep(k KeepLook) {
	layout := s.lk.pick(2, 3)
	w := stoneWalls[layout]
	s.keepCurtain(w[0], w[1], w[2], w[3], 14)
	if layout == 2 {
		s.keepCurtain(1.5, 0.3, 1.95, 0.5, 14)
	}
	for i, at := range stoneTowerSpots[layout] {
		t := towerSpec{cx: at[0], cy: at[1], r: 0.3, h: 40 + s.lk.jit(10+i, 6), roofH: 15}
		s.keepTower(k, t, i == 0 || s.lk.chance(12, 0.6))
	}
	alongY := s.lk.pick(3, 2) == 0
	top := s.hall(hallSpec{x0: 0.72, y0: 0.72, x1: 2.1, y1: 2.08, h: stoneHallH, rise: 14, alongY: alongY})
	s.keepHallChimney(k)
	s.doorway(sideL(2.08), 1.25, 0.32, 12)
	s.window(sideL(2.08), 0.9, top*0.55)
	s.window(sideR(2.1), 1.1, top*0.55)
	s.window(sideR(2.1), 1.6, top*0.55)
}

func (s iso) keepCurtain(x0, y0, x1, y1, h float64) {
	k := s.kit()
	top := h * k.scale
	if s.lk.age == ageDark {
		s.keepPalisade(x0, y0, x1, y1, top)
		return
	}
	s.block(x0, y0, x1, y1, 0, top, k.wall, flatMat(stoneR[3]), k.wallTag)
	if s.lk.age >= ageFeudal {
		s.pen(matStone, func() { s.crenels(x0, y0, x1, y1, top, stoneR) })
	}
}

func (s iso) keepPalisade(x0, y0, x1, y1, top float64) {
	alongX := x1-x0 >= y1-y0
	span := max(x1-x0, y1-y0)
	for u := 0.0; u <= span; u += 0.13 {
		gx, gy := x0+u, (y0+y1)/2
		if !alongX {
			gx, gy = (x0+x1)/2, y0+u
		}
		s.keepStake(gx, gy, top+s.lk.jit(90+int(u*50), 2))
	}
}

func (s iso) keepStake(gx, gy, h float64) {
	s.block(gx-0.06, gy-0.06, gx+0.06, gy+0.06, 0, h, logMat(s.lk.seed+int(gx*31+gy*17)), flatMat(woodR[3]), matWall)
	tip := s.p(gx+0.06, gy+0.06, h)
	s.pen(matWall, func() { s.c.dot(int(tip.x), int(tip.y)-1, woodR[2]) })
}

type keepArc struct {
	cx, cy, r, from, to, h float64
}

func (a keepArc) at(deg float64) (float64, float64) {
	rad := deg * math.Pi / 180
	return a.cx + a.r*math.Cos(rad), a.cy + a.r*math.Sin(rad)
}

func (s iso) keepArcWall(a keepArc) {
	k := s.kit()
	top := a.h * k.scale
	steps := int(math.Ceil((a.to - a.from) / 9))
	type seg struct{ x, y float64 }
	segs := make([]seg, 0, steps+1)
	for i := 0; i <= steps; i++ {
		x, y := a.at(a.from + (a.to-a.from)*float64(i)/float64(steps))
		segs = append(segs, seg{x, y})
	}
	sort.SliceStable(segs, func(i, j int) bool { return segs[i].x+segs[i].y < segs[j].x+segs[j].y })
	for _, g := range segs {
		if s.lk.age == ageDark {
			s.keepStake(g.x, g.y, top+s.lk.jit(int(g.x*40+g.y*70), 2))
			continue
		}
		s.block(g.x-0.1, g.y-0.1, g.x+0.1, g.y+0.1, 0, top, k.wall, flatMat(stoneR[3]), k.wallTag)
		if s.lk.age >= ageFeudal && int(g.x*13+g.y*7)%2 == 0 {
			s.block(g.x-0.06, g.y-0.06, g.x+0.06, g.y+0.06, top, top+3, k.wall, flatMat(stoneR[3]), matStone)
		}
	}
}

func (s iso) greatKeep(k KeepLook) {
	wall := keepArc{cx: 1.5, cy: 1.5, r: 1.3, from: 135, to: 315, h: 20}
	s.keepArcWall(wall)
	dx, dy := wall.at(225)
	donjon := towerSpec{cx: dx + 0.18, cy: dy + 0.18, r: 0.42, h: 64 + s.lk.jit(6, 4), roofH: 20}
	s.keepTower(k, donjon, true)
	flank := [2]float64{0.4, 2.02}
	if s.lk.pick(5, 2) == 1 {
		flank = [2]float64{2.02, 0.4}
	}
	s.keepTower(k, towerSpec{cx: flank[0], cy: flank[1], r: 0.26, h: 40 + s.lk.jit(7, 4), roofH: 14}, s.lk.chance(8, 0.6))
	hall := s.hall(hallSpec{x0: 1.02, y0: 1.02, x1: 2.12, y1: 2.1, h: greatHallH, rise: 16, alongY: s.lk.pick(9, 2) == 0})
	s.keepHallChimney(k)
	s.doorway(sideL(2.1), 1.3, 0.34, 13)
	for _, u := range []float64{1.12, 1.78} {
		s.window(sideL(2.1), u, hall*0.55)
		s.window(sideR(2.12), u, hall*0.55)
	}
}

func (s iso) citadelKeep(k KeepLook) {
	wall := keepArc{cx: 1.5, cy: 1.5, r: 1.22, from: 125, to: 325, h: 30}
	s.keepArcWall(wall)
	for i, deg := range []float64{225, 135, 315} {
		x, y := wall.at(deg)
		h := 60.0
		if i == 0 {
			h = 70
		}
		s.keepTower(k, towerSpec{cx: x, cy: y, r: 0.3, h: h + s.lk.jit(14+i, 4), roofH: 19}, true)
	}
	top := s.hall(hallSpec{x0: 0.9, y0: 0.9, x1: 2.12, y1: 2.12, h: citadelHallH, rise: 18, hip: true})
	s.keepHallChimney(k)
	spire := towerSpec{cx: 1.5, cy: 1.5, r: 0.27, z0: top + 4, h: 20, roofH: 28}
	stop := s.tower(spire)
	s.towerOpenings(spire, s.ringPlan(spire, stop-10, 1))
	s.keepFlag(k, spire, stop, 9)
	s.doorway(sideL(2.12), 1.3, 0.4, 17)
	for _, u := range []float64{1.0, 1.75} {
		s.window(sideL(2.12), u, top*0.45)
		s.window(sideL(2.12), u, top*0.75)
		s.window(sideR(2.12), u+0.05, top*0.6)
	}
}
