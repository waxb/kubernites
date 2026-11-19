package world

import (
	"math"
	"strconv"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	SailSeconds = 6.0
	sailMost    = 9.0
	sailSpeed   = 3.0
	minUnload   = 2.5
	maxUnload   = 8.0
	unloadEach  = .3
	shipLinger  = 2.5
	berthClear  = 1.0
	heraldStay  = 7.0
	sameShip    = .3
	fadeInShare = .2
	fadeOutFrom = .55
)

var tierShips = []string{"cog", "caravel", "galleon", "carrack"}

type Ship struct {
	Serial   int
	Variant  int
	Cargo    int
	Class    string
	Berth    int
	Dock     float64
	Unload   float64
	Wait     float64
	Sail     float64
	Herald   bool
	Docked   bool
	Accepted int
	Counted  bool
}

func (s Ship) Arrive() float64 { return s.Dock - s.Sail }
func (s Ship) Leave() float64  { return s.Dock + s.Unload + s.Wait }
func (s Ship) Gone() float64   { return s.Leave() + s.Sail }

func SailTime(dist float64) float64 {
	return min(max(dist/sailSpeed, SailSeconds), sailMost)
}

type ShipPose struct {
	Pos       Vec
	Dir       string
	Alpha     float64
	Moving    bool
	Unloading bool
}

func ShipClassOf(tier int) string {
	return tierShips[min(max(tier, 1), len(tierShips))-1]
}

func UnloadTime(cargo int) float64 {
	return min(max(float64(cargo)*unloadEach, minUnload), maxUnload)
}

func ShipVariant(haven string, serial int) int {
	return int(model.Hash(haven+"/"+strconv.Itoa(serial)) % gfx.ShipVariants)
}

type fleet struct {
	haven string
	Ships []Ship `json:"-"`
}

func (f *fleet) prune(now float64) {
	out := f.Ships[:0]
	for _, s := range f.Ships {
		if s.Gone() > now {
			out = append(out, s)
		}
	}
	f.Ships = out
}

func (f *fleet) find(serial int) (int, bool) {
	for i, s := range f.Ships {
		if s.Serial == serial && !s.Herald {
			return i, true
		}
	}
	return 0, false
}

func (f *fleet) freeBerth(berths int, dock float64) int {
	busy := make([]float64, max(berths, 1))
	for _, s := range f.Ships {
		if s.Berth < len(busy) {
			busy[s.Berth] = max(busy[s.Berth], s.Leave()+berthClear)
		}
	}
	best := 0
	for b := range busy {
		if busy[b] <= dock {
			return b
		}
		if busy[b] < busy[best] {
			best = b
		}
	}
	return best
}

func (f *fleet) launch(s Ship, berths int, sail float64) Ship {
	s.Variant, s.Sail = ShipVariant(f.haven, s.Serial), max(sail, SailSeconds)
	s.Berth = f.freeBerth(berths, s.Dock)
	f.Ships = append(f.Ships, s)
	return s
}

type docks struct {
	seen   uint64
	tier   int
	primed bool
}

func (h *Harbor) schedule(fv view.FreightView, now float64, weights []float64) {
	d := fv.Docking
	if !h.docks.primed || d.Ships < h.docks.seen {
		h.docks = docks{seen: 0, tier: fv.Tier, primed: true}
		for _, f := range h.Fleets {
			f.Ships = nil
		}
	}
	for _, f := range h.Fleets {
		f.prune(now)
	}
	if d.Ships > h.docks.seen {
		h.landShip(fv, weights)
		h.docks.seen = d.Ships
	}
	if d.Ships > 0 && d.Next > now {
		h.expectShip(fv, int(d.Ships)+1, d.Next, weights)
	}
	if fv.Tier > h.docks.tier && h.docks.tier > 0 {
		h.herald(fv.Tier, now)
	}
	h.docks.tier = max(h.docks.tier, fv.Tier)
}

func (h *Harbor) landShip(fv view.FreightView, weights []float64) {
	d := fv.Docking
	serial, cargo := int(d.Ships), int(math.Round(d.LastCargo))
	for _, f := range h.Fleets {
		if i, ok := f.find(serial); ok {
			s := &f.Ships[i]
			s.Dock, s.Cargo, s.Unload, s.Docked = d.Last, cargo, UnloadTime(cargo), true
			return
		}
	}
	i := HavenFor(serial, weights)
	if i >= len(h.Fleets) {
		return
	}
	p := h.Plan(i)
	h.Fleets[i].launch(Ship{Serial: serial, Cargo: cargo, Class: ShipClassOf(fv.Tier), Dock: d.Last, Unload: UnloadTime(cargo),
		Wait: shipLinger, Docked: true}, len(p.Berths), p.SailTime())
}

func (h *Harbor) expectShip(fv view.FreightView, serial int, dock float64, weights []float64) {
	for _, f := range h.Fleets {
		if _, ok := f.find(serial); ok {
			return
		}
	}
	i := HavenFor(serial, weights)
	if i >= len(h.Fleets) || len(h.Plan(i).Berths) == 0 {
		return
	}
	p := h.Plan(i)
	h.Fleets[i].launch(Ship{Serial: serial, Class: ShipClassOf(fv.Tier), Dock: dock, Unload: minUnload, Wait: shipLinger},
		len(p.Berths), p.SailTime())
}

func (h *Harbor) herald(tier int, now float64) {
	if len(h.Fleets) == 0 || len(h.Plan(0).Berths) == 0 {
		return
	}
	p := h.Plan(0)
	sail := max(p.SailTime(), SailSeconds)
	h.Fleets[0].launch(Ship{Serial: -tier, Class: ShipClassOf(tier), Dock: now + sail, Wait: heraldStay, Herald: true}, len(p.Berths), sail)
}

func (h *Harbor) shipOfLanding(at float64) (int, Ship, bool) {
	for i, f := range h.Fleets {
		for _, s := range f.Ships {
			if !s.Herald && s.Docked && math.Abs(s.Dock-at) <= sameShip {
				return i, s, true
			}
		}
	}
	return 0, Ship{}, false
}

func HavenFor(serial int, weights []float64) int {
	total := 0.0
	for _, w := range weights {
		total += max(w, 0)
	}
	if total <= 0 {
		return 0
	}
	u := math.Mod(float64(serial)*weyl+.5, 1) * total
	for i, w := range weights {
		if u < max(w, 0) {
			return i
		}
		u -= max(w, 0)
	}
	return len(weights) - 1
}

const weyl = 0.6180339887498949

func ShipAt(s Ship, b Berth, sea Vec, now float64) (ShipPose, bool) {
	switch {
	case now < s.Arrive() || now >= s.Gone():
		return ShipPose{}, false
	case now < s.Dock:
		p := (now - s.Arrive()) / s.Sail
		pos, dir := PathAt([]Vec{sea, b.Lane, b.Moor}, 1-(1-p)*(1-p))
		return ShipPose{Pos: pos, Dir: ShipHeading(dir), Alpha: min(p/fadeInShare, 1), Moving: true}, true
	case now < s.Leave():
		in := Vec{b.Moor.X - b.Lane.X, b.Moor.Y - b.Lane.Y}
		return ShipPose{Pos: b.Moor, Dir: ShipHeading(in), Alpha: 1, Unloading: s.Cargo > 0 && now < s.Dock+s.Unload}, true
	}
	p := (now - s.Leave()) / s.Sail
	pos, dir := PathAt(AwayPath(b, sea), p*p)
	return ShipPose{Pos: pos, Dir: ShipHeading(dir), Alpha: 1 - max(0, (p-fadeOutFrom)/(1-fadeOutFrom)), Moving: true}, true
}

func AwayPath(b Berth, sea Vec) []Vec {
	return []Vec{b.Moor, b.Lane, sea}
}

func PathAt(points []Vec, frac float64) (Vec, Vec) {
	total := 0.0
	for i := 1; i < len(points); i++ {
		total += Dist(points[i-1], points[i])
	}
	left := min(max(frac, 0), 1) * total
	dir := Vec{1, 0}
	for i := 1; i < len(points); i++ {
		a, b := points[i-1], points[i]
		seg := Dist(a, b)
		if seg < 1e-9 {
			continue
		}
		dir = Vec{b.X - a.X, b.Y - a.Y}
		if left <= seg || i == len(points)-1 {
			return Lerp(a, b, left/seg), dir
		}
		left -= seg
	}
	return points[len(points)-1], dir
}
