package world

import (
	"strconv"
	"strings"

	"github.com/waxb/kubernites/pkg/view"
)

type ferryHop struct {
	Lane     int `json:"-"`
	isle     string
	FromHome bool `json:"-"`
}

type trip struct {
	Walks [][]Vec    `json:"-"`
	Boats []ferryHop `json:"-"`
}

type ferryBook struct {
	sig   string
	lanes []FerryLane
	trips map[routeKey]trip
	decks map[Point]bool
}

func laneSig(site view.Site, land uint64) string {
	var b strings.Builder
	for _, is := range site.Islands {
		if is.Owned {
			b.WriteString(is.ID)
			b.WriteString(",")
		}
	}
	b.WriteString(strconv.FormatUint(land, 36))
	return b.String()
}

func (h *Harbor) SyncLanes(site view.Site) {
	land := h.landSig
	if land == 0 {
		land = StructureSig(site)
	}
	if sig := laneSig(site, land); sig != h.ferries.sig || h.ferries.trips == nil {
		lanes := PlanLanes(site)
		h.ferries = ferryBook{sig: sig, lanes: lanes, trips: map[routeKey]trip{}, decks: jettyDecks(lanes)}
		h.Lanes = lanes
	}
}

func isleOf(site view.Site, v Vec) (string, bool) {
	t := TileOf(v)
	is, ok := site.IslandAt(t.X, t.Y)
	return is.ID, ok
}

func (h *Harbor) laneOf(isle string, fromHome bool) (ferryHop, bool) {
	for i, l := range h.Lanes {
		if l.Isle == isle {
			return ferryHop{Lane: i, isle: isle, FromHome: fromHome}, true
		}
	}
	return ferryHop{}, false
}

func (h *Harbor) hopsBetween(site view.Site, from, to Vec) ([]ferryHop, bool) {
	a, okA := isleOf(site, from)
	b, okB := isleOf(site, to)
	if !okA || !okB || a == b {
		return nil, false
	}
	hops := []ferryHop{}
	for _, leg := range []struct {
		isle     string
		fromHome bool
	}{{a, false}, {b, true}} {
		if leg.isle == view.HomeIslandID {
			continue
		}
		hop, ok := h.laneOf(leg.isle, leg.fromHome)
		if !ok {
			return nil, false
		}
		hops = append(hops, hop)
	}
	return hops, len(hops) > 0
}

func (h *Harbor) Trip(site view.Site, from, to Vec) trip {
	if h.ferries.trips == nil {
		h.SyncLanes(site)
	}
	key := routeKey{a: from, b: to}
	if t, ok := h.ferries.trips[key]; ok {
		return t
	}
	t := h.planTrip(site, from, to)
	if len(h.ferries.trips) < maxRouteMem {
		h.ferries.trips[key] = t
	}
	return t
}

func (h *Harbor) planTrip(site view.Site, from, to Vec) trip {
	hops, ok := h.hopsBetween(site, from, to)
	if !ok {
		return walkTrip(h.route(site, from, to))
	}
	out := trip{Boats: hops}
	start, lead := from, []Vec(nil)
	for _, hop := range hops {
		dep, arr := h.Lanes[hop.Lane].ends(hop.FromHome)
		walk := append(append(lead, h.route(site, start, dep.Root.Center())...), dep.Planks...)
		out.Walks = append(out.Walks, walk)
		start, lead = arr.Root.Center(), reversed(arr.Planks)
	}
	out.Walks = append(out.Walks, append(lead, h.route(site, start, to)...))
	return out
}

func (l FerryLane) ends(fromHome bool) (Jetty, Jetty) {
	if fromHome {
		return l.Home, l.Away
	}
	return l.Away, l.Home
}

func walkTrip(path []Vec) trip { return trip{Walks: [][]Vec{path}} }

func (h *Harbor) crossesWater(site view.Site, stops ...Vec) bool {
	first, ok := isleOf(site, stops[0])
	if !ok {
		return false
	}
	for _, v := range stops[1:] {
		if isle, ok := isleOf(site, v); ok && isle != first {
			return true
		}
	}
	return false
}

func jettyDecks(lanes []FerryLane) map[Point]bool {
	out := map[Point]bool{}
	for _, l := range lanes {
		for _, j := range []Jetty{l.Home, l.Away} {
			for _, p := range j.Planks {
				out[TileOf(p)] = true
			}
		}
	}
	return out
}

type span struct{ from, to float64 }

func (h *Harbor) errandPath(site view.Site, from, to Vec) ([]Vec, []span) {
	t := h.Trip(site, from, to)
	path := append([]Vec(nil), t.Walks[0]...)
	water := []span{}
	for i, hop := range t.Boats {
		dep, arr := h.Lanes[hop.Lane].ends(hop.FromHome)
		start := PathLength(path)
		path = append(path, dep.Dock, arr.Dock)
		path = append(path, t.Walks[i+1]...)
		water = append(water, span{from: start, to: start + Dist(dep.Head, dep.Dock) + Dist(dep.Dock, arr.Dock) + Dist(arr.Dock, arr.Head)})
	}
	return path, water
}

func afloat(path []Vec, water []span, frac float64) bool {
	at := frac * PathLength(path)
	for _, w := range water {
		if at > w.from && at < w.to {
			return true
		}
	}
	return false
}
