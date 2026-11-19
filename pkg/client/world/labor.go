package world

import (
	"math"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	laborStep     = 2.0
	laborIdle     = .03
	laborBudget   = 64
	maxLaborSlots = 6
)

var laborSlots = map[string]int{"tower": 2, "stone": 3, "great": 4, "citadel": 5}

type laborSite struct {
	id     string
	node   string
	seed   uint32
	door   Vec
	slots  int
	trees  []int
	rocks  []int
	lo, hi Point
}

type span2 struct{ since, until float64 }

type shift struct {
	active int
	now    [maxLaborSlots]span2
	prev   [maxLaborSlots]span2
	last   float64
	seen   bool
}

type labor struct {
	sig     uint64
	width   int
	sites   []laborSite
	shifts  map[string]*[2]shift
	hide    []bool
	hidden  []int
	scarBuf []scar
}

func (r *Renderer) laborOf() *labor {
	if r.labor == nil {
		r.labor = &labor{shifts: map[string]*[2]shift{}}
	}
	return r.labor
}

func CrewWanted(share float64, slots int) int {
	if share <= laborIdle || slots <= 0 {
		return 0
	}
	return min(slots, max(1, int(math.Round(share*float64(slots)))))
}

func (s shift) step(t float64, want int, backfill float64) shift {
	next := s
	want = min(max(want, 0), maxLaborSlots)
	switch {
	case !s.seen || t < s.last-laborStep:
		next = shift{active: want, last: t, seen: true}
		for i := range want {
			next.now[i] = span2{since: t - backfill, until: math.Inf(1)}
		}
	case want > s.active && t-s.last >= laborStep:
		next.prev[s.active], next.now[s.active] = s.now[s.active], span2{since: t, until: math.Inf(1)}
		next.active, next.last = s.active+1, t
	case want < s.active && t-s.last >= laborStep:
		next.active, next.last = s.active-1, t
		next.now[next.active].until = t
	}
	return next
}

func (s shift) works(slot int, start float64) bool {
	if slot < 0 || slot >= maxLaborSlots {
		return false
	}
	in := func(p span2) bool { return start >= p.since && start < p.until }
	return in(s.now[slot]) || in(s.prev[slot])
}

func laborShare(used, total float64) float64 {
	if total <= 0 || used <= 0 {
		return 0
	}
	return min(used/total, 1)
}

func NodeShares(m view.NodeMetrics) (float64, float64) {
	cpu := max(m.CPUMilli, float64(m.CPURequestedMilli))
	memory := max(m.MemoryBytes, float64(m.MemoryRequestedBytes))
	return laborShare(cpu, float64(m.CPUAllocatableMilli)), laborShare(memory, float64(m.MemoryAllocatableBytes))
}

func (l *labor) sync(site view.Site, hb *habitat, version uint64) {
	sig := StructureSig(site) ^ version*1099511628211 ^ uint64(site.Width)<<32
	if sig == l.sig && l.sites != nil {
		return
	}
	l.sig, l.width = sig, site.Width
	l.sites = planLabor(site, hb)
	l.hide, l.hidden = make([]bool, site.Width*site.Height), nil
	kept := map[string]*[2]shift{}
	for _, ls := range l.sites {
		if sh, ok := l.shifts[ls.id]; ok {
			kept[ls.id] = sh
		}
	}
	l.shifts = kept
}

func planLabor(site view.Site, hb *habitat) []laborSite {
	areas := site.Territories()
	owner := ownerOf(site, areas)
	out := make([]laborSite, len(areas))
	for i, a := range areas {
		out[i] = laborSite{id: a.Keep, lo: Point{a.Center.X, a.Center.Y}, hi: Point{a.Center.X, a.Center.Y}}
	}
	for i, o := range owner {
		if o == 0 {
			continue
		}
		ls := &out[o-1]
		x, y := i%site.Width, i/site.Width
		ls.lo, ls.hi = Point{min(ls.lo.X, x), min(ls.lo.Y, y)}, Point{max(ls.hi.X, x), max(ls.hi.Y, y)}
		switch {
		case site.Tile(x, y) == view.TileForest && fellable(site, hb, x, y):
			ls.trees = append(ls.trees, i)
		case site.Tile(x, y) == view.TileRock && quarriable(site, hb, x, y):
			ls.rocks = append(ls.rocks, i)
		}
	}
	return workingSites(site, out)
}

func fellable(site view.Site, hb *habitat, x, y int) bool {
	return quarriable(site, hb, x, y) && !hb.veiled(x, y) && hb.townAt(x, y) <= stumpTown
}

func quarriable(site view.Site, hb *habitat, x, y int) bool {
	return hb.BlockAt(x, y) == blockFree && RoadLevel(site, x, y) == RoadNone
}

func workingSites(site view.Site, planned []laborSite) []laborSite {
	out := make([]laborSite, 0, len(planned))
	for _, ls := range planned {
		k, ok := structureOf(site, ls.id)
		if !ok || k.Kind != view.StructKeep || k.Object.Name == "" || len(ls.trees)+len(ls.rocks) == 0 {
			continue
		}
		ls.node, ls.seed, ls.door = k.Object.Name, model.Hash(k.Object.Name), Door(k)
		ls.slots = keepSlots(k)
		out = append(out, ls)
	}
	return out
}

func keepSlots(k view.Structure) int {
	if n, ok := laborSlots[k.Variant]; ok {
		return n
	}
	return laborSlots["stone"]
}

func structureOf(site view.Site, id string) (view.Structure, bool) {
	for _, s := range site.Structures {
		if s.ID == id {
			return s, true
		}
	}
	return view.Structure{}, false
}

func (l *labor) ready(site view.Site, ls laborSite) bool {
	k, ok := structureOf(site, ls.id)
	return ok && k.State == view.StateReady
}

func (l *labor) shiftsFor(site view.Site, ls laborSite, t float64, m view.NodeMetrics) *[2]shift {
	cpu, memory := NodeShares(m)
	if !l.ready(site, ls) {
		cpu, memory = 0, 0
	}
	sh, ok := l.shifts[ls.id]
	if !ok {
		sh = &[2]shift{}
		l.shifts[ls.id] = sh
	}
	sh[0] = sh[0].step(t, CrewWanted(cpu, min(ls.slots, len(ls.trees))), cutterTrade.window())
	sh[1] = sh[1].step(t, CrewWanted(memory, min(ls.slots, len(ls.rocks))), masonTrade.window())
	return sh
}
