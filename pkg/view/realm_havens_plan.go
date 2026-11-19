package view

import (
	"cmp"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
)

const gateHavenName = "Gate Harbor"

const HavenPier = 6

const HavenMinPier = 4

const havenSpacing = 3

const havenCivicGap = 2

const havenGateReach = 2

const HavenGateWalk = 4

const havenCoastSlack = 6

const havenCoastSwing = 10

const havenWalkWeight = 10.0

const havenNearWeight = 0.01

const havenFillCost = 2.0

const havenClearCost = 0.5

const havenSalt = 0x2545f4914f6cdd1d

type havenCoast struct {
	out   Point
	names [3]string
}

var gateCoast = Point{X: -1}

var havenCoasts = [...]havenCoast{
	{out: Point{Y: -1}, names: [3]string{"North Haven", "Gullcliff Landing", "Frostwater Quay"}},
	{out: Point{Y: 1}, names: [3]string{"South Haven", "Sunward Wharf", "Amberbay"}},
	{out: Point{X: 1}, names: [3]string{"East Haven", "Dawnport", "Morrow Quay"}},
}

type havenDraws struct {
	order []int
	names [len(havenCoasts)]int
	swing [len(havenCoasts)]int
	lean  int
}

type havenPlanner struct {
	site   Site
	home   uint8
	gate   Rect
	plaza  Rect
	reach  []bool
	steps  []int
	lean   int
	placed []Haven
}

type havenCandidate struct {
	haven Haven
	score float64
}

type havenScore func(h Haven, cost float64) (float64, bool)

func drawHavens(seed uint64) havenDraws {
	rng := rand.New(rand.NewPCG(seed^havenSalt, seed*worldStream+havenSalt))
	d := havenDraws{order: rng.Perm(len(havenCoasts)), lean: 1 - 2*rng.IntN(2)}
	for i, c := range havenCoasts {
		d.names[i] = rng.IntN(len(c.names))
		d.swing[i] = rng.IntN(2*havenCoastSwing+1) - havenCoastSwing
	}
	return d
}

func (s Site) WithHavens(seed uint64) Site {
	if len(s.Havens) > 0 || len(s.Tiles) == 0 || len(s.Tiles) != s.Width*s.Height {
		return s
	}
	d := drawHavens(seed)
	p := newHavenPlanner(s, d.lean)
	havens := p.plan(d)
	if len(havens) == 0 {
		return s
	}
	return p.settled(havens)
}

func newHavenPlanner(s Site, lean int) *havenPlanner {
	gate := Rect{X0: s.Gate.X, Y0: s.Gate.Y - 1, X1: s.Gate.X + 1, Y1: s.Gate.Y}
	if g, ok := FirstStructure(s, StructGate); ok {
		gate = Footprint(g)
	}
	plaza := Rect{X0: s.Plaza.X - PlazaRadius, Y0: s.Plaza.Y - PlazaRadius, X1: s.Plaza.X + PlazaRadius, Y1: s.Plaza.Y + PlazaRadius}
	p := &havenPlanner{site: s, home: homeRegion(s), gate: gate, plaza: plaza, lean: lean}
	p.reach = homeReach(s)
	p.steps = p.gateSteps()
	return p
}

func (p *havenPlanner) plan(d havenDraws) []Haven {
	gate, ok := p.gateHaven()
	if !ok {
		return nil
	}
	gate.ID, gate.Name, gate.Open, gate.Share = GateHavenID, gateHavenName, true, 1
	p.placed = []Haven{gate}
	for _, c := range d.order {
		coast := havenCoasts[c]
		h, ok := p.best([]Point{coast.out}, p.nearPoint(p.coastTarget(coast.out, d.swing[c])))
		if !ok {
			continue
		}
		h.ID, h.Name = fmt.Sprintf("haven-%d", len(p.placed)), coast.names[d.names[c]]
		p.placed = append(p.placed, h)
	}
	return p.placed
}

func (p *havenPlanner) gateHaven() (Haven, bool) {
	if h, ok := p.best([]Point{gateCoast}, p.besideGate); ok {
		return h, true
	}
	if h, ok := p.best([]Point{gateCoast}, p.nearPoint(p.site.Gate)); ok {
		return h, true
	}
	outs := []Point{gateCoast}
	for _, c := range havenCoasts {
		outs = append(outs, c.out)
	}
	return p.best(outs, p.nearPoint(p.site.Gate))
}

func (p *havenPlanner) besideGate(h Haven, cost float64) (float64, bool) {
	walk := p.walk(h)
	if walk < 0 || walk > HavenGateWalk {
		return 0, false
	}
	return havenWalkWeight*float64(walk) + cost + havenNearWeight*distance(h.Quay, p.site.Gate), true
}

func (p *havenPlanner) nearPoint(target Point) havenScore {
	return func(h Haven, cost float64) (float64, bool) {
		return distance(h.Quay, target) + cost, true
	}
}

func distance(a, b Point) float64 {
	return math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y))
}

func (p *havenPlanner) coastTarget(out Point, swing int) Point {
	center := p.site.Home().Center
	extent := 0
	for i, ok := range p.reach {
		if ok {
			extent = max(extent, (i%p.site.Width-center.X)*out.X+(i/p.site.Width-center.Y)*out.Y)
		}
	}
	return center.Step(out, extent+havenCoastSlack).Step(Haven{Out: out}.Along(), swing)
}

func (p *havenPlanner) best(outs []Point, score havenScore) (Haven, bool) {
	var found []havenCandidate
	for _, out := range outs {
		for i := range p.site.Tiles {
			quay := Point{X: i % p.site.Width, Y: i / p.site.Width}
			if !p.shore(quay, out) {
				continue
			}
			h, cost, ok := p.fit(quay, out)
			if !ok {
				continue
			}
			if s, ok := score(h, cost); ok {
				found = append(found, havenCandidate{haven: h, score: s})
			}
		}
	}
	slices.SortStableFunc(found, func(a, b havenCandidate) int { return cmp.Compare(a.score, b.score) })
	for _, c := range found {
		if sea, ok := p.lane(c.haven); ok {
			h := c.haven
			h.Sea = sea
			return h, true
		}
	}
	return Haven{}, false
}

func (p *havenPlanner) shore(quay, out Point) bool {
	i, ok := p.site.Index(quay.X, quay.Y)
	return ok && (p.reach[i] || p.site.Tiles[i] == TileShallows) && p.water(quay.Step(out, 1))
}

func (p *havenPlanner) fit(quay, out Point) (Haven, float64, bool) {
	h := Haven{Quay: quay, Out: out}
	cost, ok := p.quayCost(h)
	if !ok {
		return Haven{}, 0, false
	}
	for pier := HavenPier; pier >= HavenMinPier; pier-- {
		h.Pier = pier
		if p.waterFits(h) && p.clearOfPlaced(h) {
			return h, cost, true
		}
	}
	return Haven{}, 0, false
}

func (p *havenPlanner) water(pt Point) bool {
	i, ok := p.site.Index(pt.X, pt.Y)
	return ok && !IsLand(p.site.Tiles[i])
}

func (p *havenPlanner) waterFits(h Haven) bool {
	for k := 1; k <= h.Pier+1; k++ {
		c := h.Quay.Step(h.Out, k)
		if !p.water(c) {
			return false
		}
		if k > 1 && (!p.water(c.Step(h.Along(), -1)) || !p.water(c.Step(h.Along(), 1))) {
			return false
		}
	}
	return true
}
