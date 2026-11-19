package world

import (
	"maps"
	"math"
	"sort"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	depotRing0  = 1.2
	depotRing1  = 3.8
	gateRing0   = 2.0
	gateRing1   = 4.6
	doorRing0   = 1.0
	DoorRing1   = 2.4
	depotSpread = .8
	gateSpread  = 1.0
	reachMargin = .5
	doorSpread  = 1.1
	crowdCost   = 45.0
	rowShare    = .35
	gardenCost  = 6.0
	gardenOdds  = 3
	gardenRoom  = 2
	frontAngle  = math.Pi / 4
	aimSteps    = 16
	aimSlack    = 1.5
	aimShare    = .8
	neverBorn   = -1e9
	coverCost   = 1e6
)

const (
	occFree uint8 = iota
	occHut
	occGarden
)

type hutSpot struct {
	At     Point
	Node   string
	Born   float64
	Garden Point
	Sown   bool
	Tier   int
	QoS    string
}

type goneHut struct {
	spot  hutSpot
	since float64
}

type hutAnchor struct {
	lo, hi       Point
	angle        float64
	ring0, ring1 float64
	spread       float64
}

type Hamlet struct {
	sig      uint64
	Builds   int `json:"-"`
	w, h     int
	primed   bool
	SpotList map[string]hutSpot `json:"-"`
	Homes    map[string]Vec     `json:"-"`
	gone     []goneHut
	occ      []uint8
	hard     []bool
	soft     []bool
	owner    []int16
	areas    map[string]int16
	reach    map[string]int
	lists    map[int16][]int
	aims     map[aimKey]float64
	Nears    map[nearKey]nearSpot `json:"-"`
	cover    func(x, y int) bool
	covSig   uint64
	names    map[string]bool
	Homeless map[string]string `json:"-"`
	bound    map[string]bool
	pins     map[string]Point
	pinAt    map[Point]bool
}

func (h *Hamlet) Pin(pins map[string]Point) {
	if !maps.Equal(pins, h.pins) {
		h.pins, h.pinAt = maps.Clone(pins), nil
	}
}

func (h *Hamlet) pinned(keep view.Structure, p model.PodView, soft []bool) (Point, bool) {
	at, ok := h.pins[p.UID]
	if !ok {
		return Point{}, false
	}
	i, in := h.index(at.X, at.Y)
	return at, in && i < len(soft) && soft[i] && h.owner[i] == h.areas[keep.ID] && h.occ[i] != occHut
}

func (h *Hamlet) reserved(p Point) bool {
	if h.pins == nil {
		return false
	}
	if h.pinAt == nil {
		h.pinAt = make(map[Point]bool, len(h.pins))
		for _, at := range h.pins {
			h.pinAt[at] = true
		}
	}
	return h.pinAt[p]
}

func NewHamlet() *Hamlet {
	return &Hamlet{SpotList: map[string]hutSpot{}, Homes: map[string]Vec{}, lists: map[int16][]int{}, aims: map[aimKey]float64{}, Nears: map[nearKey]nearSpot{}, Homeless: map[string]string{}}
}

func (h *Hamlet) Spots() map[string]Point {
	out := make(map[string]Point, len(h.SpotList))
	for name, s := range h.SpotList {
		out[name] = s.At
	}
	return out
}

func (h *Hamlet) Covered(sig uint64, cover func(x, y int) bool) {
	if sig != h.covSig {
		h.covSig, h.cover, h.soft = sig, cover, nil
		h.lists, h.aims, h.Nears = map[int16][]int{}, map[aimKey]float64{}, map[nearKey]nearSpot{}
		clear(h.Homeless)
	}
}

func (h *Hamlet) Place(site view.Site, pods []model.PodView, now float64) {
	h.sync(site)
	h.expire(now)
	fresh, matched := h.survey(site, pods, now)
	if matched < len(h.SpotList) {
		h.evict(pods, now)
	}
	if len(fresh) > 0 {
		h.settle(site, fresh, now)
	}
	h.primed = true
}

func (h *Hamlet) survey(site view.Site, pods []model.PodView, now float64) ([]model.PodView, int) {
	var fresh []model.PodView
	matched := 0
	for _, p := range pods {
		if p.System || p.Node == "" || p.State == model.HearthPending {
			continue
		}
		s, ok := h.SpotList[p.Name]
		switch {
		case ok && s.Node == p.Node:
			matched++
			continue
		case ok:
			h.drop(p.Name, now)
		case h.Homeless[p.Name] == p.Node:
			continue
		}
		if _, known := KeepForNode(site, p.Node); known {
			fresh = append(fresh, p)
		}
	}
	return fresh, matched
}

func (h *Hamlet) evict(pods []model.PodView, now float64) {
	if h.names == nil {
		h.names = map[string]bool{}
	}
	clear(h.names)
	for _, p := range pods {
		h.names[p.Name] = true
	}
	for name := range h.SpotList {
		if !h.names[name] {
			h.drop(name, now)
		}
	}
	for name := range h.Homeless {
		if !h.names[name] {
			delete(h.Homeless, name)
		}
	}
}

func (h *Hamlet) roomy(area int16) bool {
	list, free := h.lists[area], 0
	for _, i := range list {
		if h.occ[i] == occFree {
			free++
		}
	}
	return free*gardenRoom > len(list)
}

func (h *Hamlet) drop(name string, now float64) {
	s := h.SpotList[name]
	delete(h.SpotList, name)
	delete(h.Homes, name)
	if h.primed {
		h.gone = append(h.gone, goneHut{spot: s, since: now})
		return
	}
	h.release(s)
}

func (h *Hamlet) release(s hutSpot) {
	clear(h.Homeless)
	h.mark(s.At, occFree)
	if s.Sown && h.occAt(s.Garden) == occGarden {
		h.mark(s.Garden, occFree)
	}
}

func (h *Hamlet) expire(now float64) {
	kept := h.gone[:0]
	for _, g := range h.gone {
		if now-g.since < FadeSeconds && now >= g.since {
			kept = append(kept, g)
			continue
		}
		h.release(g.spot)
	}
	h.gone = kept
}

func doorRank(p model.PodView) int {
	switch {
	case p.Static && strings.HasSuffix(p.Name, "-"+p.Node):
		return 0
	case p.Static:
		return 1
	}
	return 2
}

func settleOrder(pods []model.PodView) {
	sort.SliceStable(pods, func(i, j int) bool {
		if a, b := doorRank(pods[i]), doorRank(pods[j]); a != b {
			return a < b
		}
		return pods[i].Name < pods[j].Name
	})
}

func (h *Hamlet) settle(site view.Site, fresh []model.PodView, now float64) {
	settleOrder(fresh)
	born := now
	if !h.primed {
		born = neverBorn
	}
	for _, p := range fresh {
		keep, _ := KeepForNode(site, p.Node)
		at, ok := h.home(site, keep, p)
		if !ok {
			h.Homeless[p.Name] = p.Node
			continue
		}
		s := hutSpot{At: at, Node: p.Node, Born: born, Tier: p.Tier, QoS: p.QoS}
		h.mark(at, occHut)
		if h.roomy(h.areas[keep.ID]) {
			s.Garden, s.Sown = h.sow(at, model.Hash(p.Name), h.bound[p.Name])
		}
		h.SpotList[p.Name] = s
		h.Homes[p.Name] = at.Center()
	}
}

func (h *Hamlet) home(site view.Site, keep view.Structure, p model.PodView) (Point, bool) {
	if p.Static {
		if at, ok := DoorHut(site, keep); ok && h.hardAt(at) && h.occAt(at) == occFree {
			return at, true
		}
	}
	area := h.areas[keep.ID]
	if at, ok := h.pinned(keep, p, h.softGrid(site)); ok {
		return at, true
	}
	if pin, ok := h.pins[p.UID]; ok {
		return h.nearestPin(h.list(site, area), pin)
	}
	return h.pick(h.list(site, area), h.anchorFor(site, keep, p), model.Hash(p.Name))
}

func (h *Hamlet) nearestPin(list []int, pin Point) (Point, bool) {
	best, score, found := 0, math.Inf(1), false
	for _, i := range list {
		if h.occ[i] == occHut {
			continue
		}
		x, y := i%h.w, i/h.w
		s := float64((x-pin.X)*(x-pin.X) + (y-pin.Y)*(y-pin.Y))
		if h.occ[i] == occGarden {
			s += gardenCost
		}
		if h.covers(x, y) || h.reserved(Point{x, y}) {
			s += coverCost
		}
		if s < score {
			best, score, found = i, s, true
		}
	}
	return Point{best % max(h.w, 1), best / max(h.w, 1)}, found
}

func (h *Hamlet) pick(list []int, a hutAnchor, seed uint32) (Point, bool) {
	want := a.want(seed)
	best, score, found := 0, math.Inf(1), false
	for _, i := range list {
		if h.occ[i] == occHut {
			continue
		}
		x, y := i%h.w, i/h.w
		s := (float64(x)-want.X)*(float64(x)-want.X) + (float64(y)-want.Y)*(float64(y)-want.Y)
		s += h.crowding(x, y)
		if h.occ[i] == occGarden {
			s += gardenCost
		}
		if h.covers(x, y) {
			s += coverCost
		}
		if h.reserved(Point{x, y}) {
			s += coverCost
		}
		if s < score {
			best, score, found = i, s, true
		}
	}
	return Point{best % max(h.w, 1), best / max(h.w, 1)}, found
}

func (a hutAnchor) want(seed uint32) Vec {
	u := float64(seed&0xffff) / 0xffff
	v := float64(mix32(seed)&0xffff) / 0xffff
	r := a.ring0 + u*(a.ring1-a.ring0)
	th := a.angle + (v-.5)*2*a.spread
	cx, cy := float64(a.lo.X+a.hi.X)/2, float64(a.lo.Y+a.hi.Y)/2
	half := float64(max(a.hi.X-a.lo.X, a.hi.Y-a.lo.Y)+1) / 2
	return Vec{cx + math.Cos(th)*(half+r), cy + math.Sin(th)*(half+r)}
}

func crewKey(p model.PodView) string {
	switch {
	case p.Crew != "":
		return p.Crew
	case p.Owner != "":
		return p.Owner
	}
	return p.Depot
}

func (h *Hamlet) anchorFor(site view.Site, keep view.Structure, p model.PodView) hutAnchor {
	klo, khi := Footprint(keep)
	if p.Owner == "" {
		return hutAnchor{lo: klo, hi: khi, angle: frontAngle, ring0: doorRing0, ring1: DoorRing1, spread: doorSpread}
	}
	crew := mix32(model.Hash(crewKey(p)))
	a := hutAnchor{lo: klo, hi: khi, ring0: gateRing0, ring1: h.gateReach(keep), spread: gateSpread}
	if d, ok := DepotOf(site, p.Depot); ok && p.Depot != "" {
		a = h.depotAnchor(site, keep, d, a)
	}
	a.angle = h.aim(h.list(site, h.areas[keep.ID]), a, crew)
	return a
}

func (h *Hamlet) gateReach(keep view.Structure) float64 {
	half := float64(max(keep.W, keep.H)) / 2
	return min(max(float64(h.reach[keep.ID])-half-reachMargin, gateRing0), gateRing1)
}

type aimKey struct {
	lo, hi Point
	crew   uint32
}

func (h *Hamlet) aim(list []int, a hutAnchor, crew uint32) float64 {
	key := aimKey{lo: a.lo, hi: a.hi, crew: crew}
	if th, ok := h.aims[key]; ok {
		return th
	}
	var counts [aimSteps]int
	cx, cy := float64(a.lo.X+a.hi.X)/2, float64(a.lo.Y+a.hi.Y)/2
	for _, i := range list {
		x, y := i%h.w, i/h.w
		if g := FootprintGap(a.lo, a.hi, x, y); g < a.ring0 || g > a.ring1+aimSlack {
			continue
		}
		phi := math.Atan2(float64(y)-cy, float64(x)-cx)
		for k := range counts {
			if angleGap(phi, aimAngle(k)) <= a.spread {
				counts[k]++
			}
		}
	}
	th := aimAngle(pickAim(counts, crew))
	h.aims[key] = th
	return th
}

func aimAngle(k int) float64 {
	return frontAngle + 2*math.Pi*float64(k)/aimSteps
}

func angleGap(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 2*math.Pi)
	return min(d, 2*math.Pi-d)
}

func pickAim(counts [aimSteps]int, crew uint32) int {
	best := 0
	for _, c := range counts {
		best = max(best, c)
	}
	good := make([]int, 0, aimSteps)
	for k, c := range counts {
		if float64(c) >= aimShare*float64(best) {
			good = append(good, k)
		}
	}
	return good[int(crew%uint32(len(good)))]
}

func (h *Hamlet) crowding(x, y int) float64 {
	cost := 0.0
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if (dx != 0 || dy != 0) && h.occAt(Point{x + dx, y + dy}) == occHut {
				cost = max(cost, neighbourCost(dx, dy))
			}
		}
	}
	return cost
}

var coverSteps = [5]Point{{1, 1}, {1, 0}, {0, 1}, {2, 1}, {1, 2}}

func (h *Hamlet) covers(x, y int) bool {
	for _, d := range coverSteps {
		if h.occAt(Point{x + d.X, y + d.Y}) == occHut || h.occAt(Point{x - d.X, y - d.Y}) == occHut {
			return true
		}
	}
	return false
}

func neighbourCost(dx, dy int) float64 {
	if dx == -dy {
		return crowdCost * rowShare
	}
	return crowdCost
}

var gardenSides = [4]Point{{-1, 0}, {0, -1}, {1, 0}, {0, 1}}

var tendedOrder = [2][4]int{{2, 3, 0, 1}, {3, 2, 1, 0}}

func (h *Hamlet) sow(at Point, seed uint32, tended bool) (Point, bool) {
	if (seed%gardenOdds == 0 && !tended) || h.soft == nil {
		return Point{}, false
	}
	start := int(seed >> 4 % 4)
	if tended {
		start = 2 + int(seed>>4%2)
	}
	for k := range gardenSides {
		d := gardenSides[(start+k)%4]
		if tended {
			d = gardenSides[tendedOrder[start-2][k]]
		}
		g := Point{at.X + d.X, at.Y + d.Y}
		if i, ok := h.index(g.X, g.Y); ok && h.soft[i] && h.occ[i] == occFree && !h.crowdedBut(g, at) && !h.reserved(g) {
			h.occ[i] = occGarden
			return g, true
		}
	}
	return Point{}, false
}

func (h *Hamlet) crowdedBut(g, home Point) bool {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			p := Point{g.X + dx, g.Y + dy}
			if p != g && p != home && h.occAt(p) == occHut {
				return true
			}
		}
	}
	return false
}
