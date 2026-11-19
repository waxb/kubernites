package world

import (
	"math"
	"slices"
	"sort"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	StackHeight = 2
	spoilSlack  = .3
	SpoilLinger = 1.2
	resetSlack  = 1.0
	atScale     = 100
	heldAge     = .5
	heldSlack   = 1e-6
	matchSlack  = .01
)

type Lane struct {
	Kind    int
	Sealed  bool
	Crowned bool
}

type loadKey struct {
	lane Lane
	at   int64
}

const offYard = -1

type Stack struct {
	first, N, Spot int `json:"-"`
}

type Load struct {
	Lane      Lane
	At        float64
	key       loadKey
	seq       int
	seen      float64
	held      bool
	Haven     int
	Total     int
	Count     int
	Remaining float64
	TTL       float64
	Sizes     []uint8 `json:"-"`
	Stacks    []Stack `json:"-"`
}

type yardKey struct{ haven, kind int }

type spoilGhost struct {
	Boxes []QuayBox `json:"-"`
	start float64
}

type spentLoad struct {
	load  *Load
	until float64
}

const spentKeep = 90.0

type Manifest struct {
	spent   []spentLoad
	next    int
	Loads   map[loadKey]*Load           `json:"-"`
	Cooling map[yardKey]map[int]float64 `json:"-"`
	Ghosts  []spoilGhost                `json:"-"`
	last    float64
	primed  bool
}

type HavenPicker func(at float64, key uint32) int

func keyOf(lane Lane, at float64) loadKey {
	return loadKey{lane: lane, at: int64(math.Round(at * atScale))}
}

func (k loadKey) hash() uint32 {
	h := uint32(k.at) ^ uint32(k.lane.Kind)*2654435761
	if k.lane.Sealed {
		h ^= 0x9e3779b9
	}
	if k.lane.Crowned {
		h ^= 0x85ebca6b
	}
	return mix32(h)
}

func (m *Manifest) reset() {
	m.Loads, m.Cooling, m.Ghosts, m.spent, m.primed = map[loadKey]*Load{}, map[yardKey]map[int]float64{}, nil, nil, true
}

func (m *Manifest) Update(kinds []view.KindView, now, clock float64, pick HavenPicker) bool {
	fresh := !m.primed || now < m.last-resetSlack
	if fresh {
		m.reset()
		m.last = now
	}
	elapsed := now - m.last
	m.last = now
	m.Ghosts = liveGhosts(m.Ghosts, clock)
	m.spent = liveSpent(m.spent, clock)
	seen := make(map[loadKey]bool, len(m.Loads))
	changed := fresh
	for _, ls := range sightedLanes(kinds, now) {
		for i, key := range m.align(ls, elapsed, seen) {
			s := ls.sights[i]
			seen[key] = true
			changed = m.track(key, s.kv, s.c, s.at, pick) || changed
		}
	}
	changed = m.retire(seen, now, clock) || changed
	return m.place(clock) || changed
}

type sighting struct {
	kv view.KindView
	c  cohortView
	at float64
}

type laneSights struct {
	lane   Lane
	sights []sighting
}

func sightedLanes(kinds []view.KindView, now float64) []laneSights {
	out := []laneSights{}
	index := map[Lane]int{}
	for k, kv := range kinds {
		for _, c := range kv.Cohorts {
			lane := Lane{Kind: k, Sealed: c.Sealed, Crowned: c.Crowned}
			i, ok := index[lane]
			if !ok {
				i = len(out)
				index[lane] = i
				out = append(out, laneSights{lane: lane})
			}
			out[i].sights = append(out[i].sights, sighting{kv: kv, c: c, at: now - c.Age})
		}
	}
	return out
}

func heldCohort(c cohortView) bool {
	return c.TTL > 0 && math.Abs(c.Age-c.TTL*heldAge) <= heldSlack
}

func (m *Manifest) laneLoads(lane Lane) []*Load {
	out := []*Load{}
	for _, l := range m.Loads {
		if l.Lane == lane {
			out = append(out, l)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].seq < out[j].seq })
	return out
}

func (m *Manifest) align(ls laneSights, elapsed float64, taken map[loadKey]bool) []loadKey {
	old := m.laneLoads(ls.lane)
	claimed := map[*Load]bool{}
	keys := make([]loadKey, len(ls.sights))
	assigned := make([]bool, len(ls.sights))
	held := []int{}
	for j, s := range ls.sights {
		if heldCohort(s.c) {
			held = append(held, j)
			continue
		}
		if l := exactLoad(old, claimed, keyOf(ls.lane, s.at)); l != nil {
			claimed[l], keys[j], assigned[j] = true, l.key, true
		}
	}
	for j, l := range pairHeld(old, claimed, ls, held, elapsed) {
		claimed[l], keys[j], assigned[j] = true, l.key, true
	}
	for j := range ls.sights {
		if !assigned[j] {
			keys[j] = m.vacant(keyOf(ls.lane, ls.sights[j].at), taken, keys[:j])
		}
	}
	return keys
}

func exactLoad(old []*Load, claimed map[*Load]bool, key loadKey) *Load {
	for _, l := range old {
		if !claimed[l] && keyOf(l.Lane, l.seen) == key {
			return l
		}
	}
	return nil
}

func pairHeld(old []*Load, claimed map[*Load]bool, ls laneSights, held []int, elapsed float64) map[int]*Load {
	out := map[int]*Load{}
	groups := heldGroups(ls, held)
	drifted := make([][]*Load, len(groups))
	for _, l := range old {
		if g, ok := driftGroup(ls, groups, l, elapsed); ok && !claimed[l] {
			drifted[g] = append(drifted[g], l)
		}
	}
	for g, members := range groups {
		loads := drifted[g]
		if len(members) < len(loads) {
			loads = loads[len(loads)-len(members):]
		}
		for i, l := range loads {
			out[members[i]] = l
		}
	}
	return out
}

func heldGroups(ls laneSights, held []int) [][]int {
	groups := [][]int{}
	for _, j := range held {
		at := ls.sights[j].at
		g := slices.IndexFunc(groups, func(members []int) bool { return math.Abs(ls.sights[members[0]].at-at) <= matchSlack })
		if g < 0 {
			groups = append(groups, []int{j})
			continue
		}
		groups[g] = append(groups[g], j)
	}
	return groups
}

func driftGroup(ls laneSights, groups [][]int, l *Load, elapsed float64) (int, bool) {
	best, gap := -1, math.Inf(1)
	for g, members := range groups {
		drift := ls.sights[members[0]].at - l.seen
		if drift < -matchSlack || drift > elapsed+matchSlack {
			continue
		}
		if d := math.Abs(drift - elapsed); d < gap {
			best, gap = g, d
		}
	}
	return best, best >= 0
}

func (m *Manifest) vacant(key loadKey, taken map[loadKey]bool, assigned []loadKey) loadKey {
	for {
		if _, used := m.Loads[key]; !used && !taken[key] && !slices.Contains(assigned, key) {
			return key
		}
		key.at++
	}
}

func (m *Manifest) track(key loadKey, kv view.KindView, c cohortView, at float64, pick HavenPicker) bool {
	count := int(math.Round(c.Count))
	l, ok := m.Loads[key]
	switch {
	case ok && count > l.Total:
		l = extend(l, kv, count)
		m.Loads[key] = l
		ok = false
	case !ok:
		l = seat(key, kv, at, pick(at, key.hash()), max(count, 0))
		m.next++
		l.seq = m.next
		m.Loads[key] = l
	}
	next := min(max(count, 0), l.Total)
	changed := !ok || l.Count != next || l.Remaining != c.Remaining
	l.Count, l.Remaining, l.TTL = next, c.Remaining, c.TTL
	l.seen, l.held = at, heldCohort(c)
	return changed
}

func seat(key loadKey, kv view.KindView, at float64, haven, total int) *Load {
	return &Load{Lane: key.lane, At: at, key: key, seen: at, Haven: haven, Total: total, Sizes: LoadSizes(key.hash(), kv.SizeMix, total), Stacks: NewStacks(0, total)}
}

func NewStacks(from, to int) []Stack {
	out := make([]Stack, 0, (to-from+StackHeight-1)/StackHeight)
	for first := from; first < to; first += StackHeight {
		out = append(out, Stack{first: first, N: min(StackHeight, to-first), Spot: offYard})
	}
	return out
}

func extend(l *Load, kv view.KindView, total int) *Load {
	next := *l
	next.Total = total
	next.Sizes = append(append([]uint8(nil), l.Sizes...), LoadSizes(l.key.hash()+uint32(len(l.Sizes)), kv.SizeMix, total-len(l.Sizes))...)
	next.Stacks = append(append([]Stack(nil), l.Stacks...), NewStacks(l.Total, total)...)
	return &next
}

func LoadSizes(seed uint32, mix [4]float64, n int) []uint8 {
	out := make([]uint8, n)
	for i := range out {
		out[i] = uint8(SizeFromMix(mix, float64(mix32(seed+uint32(i)*0x27d4eb2d)%10000)/10000))
	}
	return out
}

func SizeFromMix(mix [4]float64, u float64) int {
	total := 0.0
	for _, w := range mix {
		total += max(w, 0)
	}
	if total <= 0 {
		return 0
	}
	u *= total
	for s, w := range mix {
		if u < max(w, 0) {
			return s
		}
		u -= max(w, 0)
	}
	return len(mix) - 1
}

func (m *Manifest) retire(seen map[loadKey]bool, now, clock float64) bool {
	changed := false
	gone := []*Load{}
	for key, l := range m.Loads {
		if !seen[key] {
			gone = append(gone, l)
		}
	}
	sortLoads(gone)
	for _, l := range gone {
		if Spoiled(*l, now) {
			m.Ghosts = append(m.Ghosts, spoilGhost{Boxes: spoiledBoxes(*l), start: clock})
		}
		for _, st := range l.Stacks {
			if st.Spot != offYard {
				m.cool(l.yard(), st.Spot, clock)
			}
		}
		m.spent = append(m.spent, spentLoad{load: l, until: clock + spentKeep})
		delete(m.Loads, l.key)
		changed = true
	}
	return changed
}

func Spoiled(l Load, now float64) bool {
	return l.Count >= 1 && l.seen+l.TTL <= now+spoilSlack
}

func spoiledBoxes(l Load) []QuayBox {
	out := []QuayBox{}
	for _, b := range l.Boxes() {
		if b.Column != offYard {
			b.Look.Spoiled = true
			out = append(out, b)
		}
	}
	return out
}

func (l *Load) yard() yardKey { return yardKey{haven: l.Haven, kind: l.Lane.Kind} }

func (l *Load) live(st Stack) bool { return st.first+st.N > l.Total-l.Count }

func (m *Manifest) cool(yk yardKey, spot int, clock float64) {
	if m.Cooling[yk] == nil {
		m.Cooling[yk] = map[int]float64{}
	}
	m.Cooling[yk][spot] = clock
}

func (m *Manifest) place(clock float64) bool {
	changed := false
	held := map[yardKey]map[int]bool{}
	loads := m.Sorted()
	for _, l := range loads {
		for i, st := range l.Stacks {
			switch {
			case st.Spot == offYard:
			case l.live(st):
				if held[l.yard()] == nil {
					held[l.yard()] = map[int]bool{}
				}
				held[l.yard()][st.Spot] = true
			default:
				m.cool(l.yard(), st.Spot, clock)
				l.Stacks[i].Spot, changed = offYard, true
			}
		}
	}
	for _, l := range loads {
		for i, st := range l.Stacks {
			if st.Spot != offYard || !l.live(st) {
				continue
			}
			spot, ok := m.freeSpot(l.yard(), held, clock)
			if !ok {
				break
			}
			if held[l.yard()] == nil {
				held[l.yard()] = map[int]bool{}
			}
			held[l.yard()][spot] = true
			l.Stacks[i].Spot, changed = spot, true
		}
	}
	return changed
}

func (m *Manifest) freeSpot(yk yardKey, held map[yardKey]map[int]bool, clock float64) (int, bool) {
	for spot := range YardColumns {
		if held[yk][spot] {
			continue
		}
		if at, cooling := m.Cooling[yk][spot]; cooling && clock-at < SpoilLinger && clock >= at {
			continue
		}
		delete(m.Cooling[yk], spot)
		return spot, true
	}
	return 0, false
}

func liveGhosts(ghosts []spoilGhost, clock float64) []spoilGhost {
	out := ghosts[:0]
	for _, g := range ghosts {
		if age := clock - g.start; age >= 0 && age < SpoilLinger {
			out = append(out, g)
		}
	}
	return out
}

func liveSpent(spent []spentLoad, clock float64) []spentLoad {
	out := spent[:0]
	for _, s := range spent {
		if clock < s.until {
			out = append(out, s)
		}
	}
	return out
}

func (m *Manifest) landed() []*Load {
	out := make([]*Load, 0, len(m.Loads)+len(m.spent))
	for _, l := range m.Loads {
		out = append(out, l)
	}
	for _, s := range m.spent {
		out = append(out, s.load)
	}
	sortLoads(out)
	return out
}

func sortLoads(out []*Load) {
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.At != b.At {
			return a.At < b.At
		}
		if a.Lane.Kind != b.Lane.Kind {
			return a.Lane.Kind < b.Lane.Kind
		}
		if laneOrder(a.Lane) != laneOrder(b.Lane) {
			return laneOrder(a.Lane) < laneOrder(b.Lane)
		}
		return a.seq < b.seq
	})
}

func (m *Manifest) Sorted() []*Load {
	out := make([]*Load, 0, len(m.Loads))
	for _, l := range m.Loads {
		out = append(out, l)
	}
	sortLoads(out)
	return out
}

func laneOrder(l Lane) int {
	n := 0
	if l.Sealed {
		n++
	}
	if l.Crowned {
		n += 2
	}
	return n
}

func NextToServe(loads []*Load, kind, haven int) (*Load, bool) {
	var best *Load
	for _, l := range loads {
		if l.Lane.Kind != kind || l.Count <= 0 || (haven >= 0 && l.Haven != haven) {
			continue
		}
		if best == nil || before(*l, *best) {
			best = l
		}
	}
	return best, best != nil
}

func before(a, b Load) bool {
	if a.Lane.Crowned != b.Lane.Crowned {
		return a.Lane.Crowned
	}
	return a.At < b.At
}
