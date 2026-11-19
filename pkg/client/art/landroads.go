package art

import "math"

type segment struct {
	ax, ay, bx, by float64
	la, lb         uint8
}

var roadHalfWidth = [RoadLevels + 1]float64{0, 0.22, 0.33, 0.4}

func roadSegments(l *Land) ([]segment, [][]int32) {
	links := make([][]int32, l.W*l.H)
	segs := []segment{}
	add := func(x0, y0, x1, y1 int, la, lb uint8) {
		id := int32(len(segs))
		segs = append(segs, segment{float64(x0), float64(y0), float64(x1), float64(y1), la, lb})
		for _, p := range [2][2]int{{x0, y0}, {x1, y1}} {
			i := p[1]*l.W + p[0]
			links[i] = append(links[i], id)
		}
	}
	for y := 0; y < l.H; y++ {
		for x := 0; x < l.W; x++ {
			lv := l.roadAt(x, y)
			if lv == 0 {
				continue
			}
			if !linkRoad(l, x, y, lv, add) {
				add(x, y, x, y, lv, lv)
			}
		}
	}
	return segs, links
}

func linkRoad(l *Land, x, y int, lv uint8, add func(x0, y0, x1, y1 int, la, lb uint8)) bool {
	linked := false
	for _, d := range [4][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
		nx, ny := x+d[0], y+d[1]
		other := l.roadAt(nx, ny)
		switch {
		case other > 0:
			linked = true
			if d[0]+d[1] > 0 {
				add(x, y, nx, ny, lv, other)
			}
		case l.groundAt(nx, ny) == GroundPlaza:
			linked = true
			add(x, y, nx, ny, lv, lv)
		}
	}
	for _, d := range [2][2]int{{1, 1}, {1, -1}} {
		nx, ny := x+d[0], y+d[1]
		other := l.roadAt(nx, ny)
		if other > 0 && l.roadAt(x+d[0], y) == 0 && l.roadAt(x, y+d[1]) == 0 {
			add(x, y, nx, ny, lv, other)
			linked = true
		}
	}
	for _, d := range [2][2]int{{-1, 1}, {-1, -1}} {
		nx, ny := x+d[0], y+d[1]
		if l.roadAt(nx, ny) > 0 && l.roadAt(x+d[0], y) == 0 && l.roadAt(x, y+d[1]) == 0 {
			linked = true
		}
	}
	return linked
}

type roadHit struct {
	level  uint8
	margin float64
	side   float64
}

func (l *Land) roadNear(gx, gy float64) roadHit {
	best := roadHit{margin: -1}
	if len(l.segs) == 0 {
		return best
	}
	tx, ty := int(math.Floor(gx+0.5)), int(math.Floor(gy+0.5))
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			i, ok := l.index(tx+dx, ty+dy)
			if !ok {
				continue
			}
			for _, id := range l.links[i] {
				if h := l.segs[id].hit(gx, gy); h.margin > best.margin {
					best = h
				}
			}
		}
	}
	return best
}

func (s segment) hit(px, py float64) roadHit {
	dx, dy := s.bx-s.ax, s.by-s.ay
	t := 0.0
	if n := dx*dx + dy*dy; n > 0 {
		t = min(max(((px-s.ax)*dx+(py-s.ay)*dy)/n, 0), 1)
	}
	cx, cy := s.ax+dx*t-px, s.ay+dy*t-py
	d := math.Hypot(cx, cy)
	hw := roadHalfWidth[s.la] + (roadHalfWidth[s.lb]-roadHalfWidth[s.la])*t
	level := s.la
	if t > 0.5 {
		level = s.lb
	}
	side := 0.0
	if n := math.Hypot(dx, dy); n > 0 {
		side = (dx*(py-s.ay) - dy*(px-s.ax)) / n
	}
	return roadHit{level: level, margin: hw - d, side: side}
}
