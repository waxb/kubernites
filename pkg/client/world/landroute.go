package world

import (
	"math"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	routeReach  = 3
	sightStep   = .3
	maxRouteMem = 8192
)

type landGrid struct {
	w, h int
	on   []bool
}

type routeKey struct{ a, b Vec }

type RouteBook struct {
	Sig    uint64 `json:"-"`
	grid   landGrid
	Routes map[routeKey][]Vec `json:"-"`
}

func LandGridOf(site view.Site) landGrid {
	g := landGrid{w: site.Width, h: site.Height, on: make([]bool, site.Width*site.Height)}
	blocked := footprintMask(site)
	for y := 0; y < site.Height; y++ {
		for x := 0; x < site.Width; x++ {
			i := y*site.Width + x
			g.on[i] = !blocked[i] && !Wet(site.Tile(x, y))
		}
	}
	return g
}

func (g landGrid) walk(p Point) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < g.w && p.Y < g.h && g.on[p.Y*g.w+p.X]
}

func (g landGrid) nearest(at Point) (Point, bool) {
	best, found, bestD := Point{}, false, math.MaxFloat64
	for dy := -routeReach; dy <= routeReach; dy++ {
		for dx := -routeReach; dx <= routeReach; dx++ {
			p := Point{at.X + dx, at.Y + dy}
			if d := math.Hypot(float64(dx), float64(dy)); g.walk(p) && d < bestD {
				best, found, bestD = p, true, d
			}
		}
	}
	return best, found
}

func (g landGrid) tiles(a, b Point) ([]Point, bool) {
	prev := map[Point]Point{a: a}
	queue := []Point{a}
	for len(queue) > 0 {
		at := queue[0]
		queue = queue[1:]
		if at == b {
			return unwind(prev, a, b), true
		}
		for _, d := range [4][2]int{{1, 0}, {0, 1}, {-1, 0}, {0, -1}} {
			p := Point{at.X + d[0], at.Y + d[1]}
			if _, seen := prev[p]; !seen && g.walk(p) {
				prev[p] = at
				queue = append(queue, p)
			}
		}
	}
	return nil, false
}

func (g landGrid) Clear(a, b Vec) bool {
	d := Dist(a, b)
	steps := int(math.Ceil(d / sightStep))
	for i := 1; i < steps; i++ {
		if !g.walk(TileOf(Lerp(a, b, float64(i)/float64(steps)))) {
			return false
		}
	}
	return true
}

func (g landGrid) smooth(pts []Vec) []Vec {
	out := []Vec{pts[0]}
	for i := 0; i < len(pts)-1; {
		j := len(pts) - 1
		for j > i+1 && !g.Clear(pts[i], pts[j]) {
			j--
		}
		out = append(out, pts[j])
		i = j
	}
	return out
}

func (g landGrid) Route(from, to Vec) []Vec {
	a, okA := g.nearest(TileOf(from))
	b, okB := g.nearest(TileOf(to))
	if !okA || !okB {
		return []Vec{from, to}
	}
	tiles, ok := g.tiles(a, b)
	if !ok {
		return []Vec{from, to}
	}
	pts := make([]Vec, 0, len(tiles)+2)
	pts = append(pts, from)
	for _, t := range tiles {
		pts = append(pts, t.Center())
	}
	return g.smooth(append(pts, to))
}

func (b *RouteBook) Route(site view.Site, sig uint64, from, to Vec) []Vec {
	if b.Sig != sig || b.Routes == nil || len(b.Routes) > maxRouteMem {
		b.Sig, b.grid, b.Routes = sig, LandGridOf(site), map[routeKey][]Vec{}
	}
	key := routeKey{a: from, b: to}
	if r, ok := b.Routes[key]; ok {
		return r
	}
	r := b.grid.Route(from, to)
	b.Routes[key] = r
	return r
}

type routeLeg struct {
	pts     []Vec
	length  float64
	pause   float64
	action  string
	carry   bool
	Ferry   *ferryHop `json:"-"`
	deliver bool
	passing bool
}

type RouteLoop struct {
	Legs  []routeLeg `json:"-"`
	total float64
}

func NewRouteLoop(out, carry, back []Vec) RouteLoop {
	return NewTripLoop(walkTrip(out), walkTrip(carry), walkTrip(back))
}

func NewTripLoop(out, carry, back trip) RouteLoop {
	l := RouteLoop{}
	l = l.withTrip(out, routeLeg{pause: WorkPause, action: "work"})
	l = l.withTrip(carry, routeLeg{pause: BankPause, action: "idle", carry: true})
	l = l.withTrip(back, routeLeg{pause: HomePause, action: "idle"})
	l.total = max(l.total, .001)
	return l
}

func NewGateLoop(out, toGate, onward, back trip) RouteLoop {
	l := RouteLoop{}
	l = l.withTrip(out, routeLeg{pause: WorkPause, action: "work"})
	l = l.withTrip(toGate, routeLeg{pause: gatePause, action: "idle", carry: true, passing: true})
	l = l.withTrip(onward, routeLeg{pause: BankPause, action: "idle", carry: true})
	l = l.withTrip(back, routeLeg{pause: HomePause, action: "idle"})
	l.total = max(l.total, .001)
	return l
}

func (l RouteLoop) withTrip(t trip, end routeLeg) RouteLoop {
	out := RouteLoop{Legs: append([]routeLeg(nil), l.Legs...), total: l.total}
	for i, w := range t.Walks {
		leg := routeLeg{pts: w, length: PathLength(w), action: "idle", carry: end.carry}
		if i == len(t.Walks)-1 {
			leg.pause, leg.action, leg.deliver = end.pause, end.action, end.carry && !end.passing
		}
		out.Legs = append(out.Legs, leg)
		out.total += leg.length/WalkSpeed + leg.pause
		if i < len(t.Boats) && len(w) > 0 {
			hop := t.Boats[i]
			out.Legs = append(out.Legs, routeLeg{pts: []Vec{w[len(w)-1]}, action: "idle", carry: end.carry, Ferry: &hop})
		}
	}
	return out
}

func (l RouteLoop) DeliveryAge(clock float64) float64 {
	at := 0.0
	for _, leg := range l.Legs {
		travel := leg.length / WalkSpeed
		if leg.deliver {
			return wrap(wrap(clock, l.total)-(at+travel), l.total)
		}
		at += travel + leg.pause
	}
	return l.total
}

func (l RouteLoop) Crossing(from, to float64) (float64, routeLeg, bool) {
	best, found, at := to, routeLeg{}, false
	offset := 0.0
	for _, leg := range l.Legs {
		offset += leg.length/WalkSpeed + leg.pause
		if leg.Ferry == nil {
			continue
		}
		k := offset + math.Floor((from-offset)/l.total)*l.total
		if k <= from {
			k += l.total
		}
		if k <= best {
			best, found, at = k, leg, true
		}
	}
	return best, found, at
}

func (l RouteLoop) Length() float64 { return l.total }

func (l RouteLoop) At(clock float64) Pose {
	phase := wrap(clock, l.total)
	for i, leg := range l.Legs {
		travel := leg.length / WalkSpeed
		if phase < travel {
			pos, dir := PathAt(leg.pts, phase/max(travel, .001))
			return Pose{Pos: pos, Action: carryAction(leg.carry), Dir: Facing(Vec{}, dir), Frame: Frame(clock, 6), Carry: leg.carry}
		}
		if phase < travel+leg.pause || i == len(l.Legs)-1 {
			end := leg.pts[len(leg.pts)-1]
			_, dir := PathAt(leg.pts, 1)
			return Pose{Pos: end, Action: leg.action, Dir: Facing(Vec{}, dir), Frame: Frame(clock, 4)}
		}
		phase -= travel + leg.pause
	}
	return Pose{}
}
