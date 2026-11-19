package live

import (
	"fmt"
	"math"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	keyDistrict = "district"
	keyGate     = "gate"
	keySign     = "sign"
	strayReach  = 12
	strayFar    = 48
)

type wanted struct {
	key  string
	ns   string
	it   item
	gate string
}

func objectKey(class string, o view.ObjectRef) string {
	return class + "/" + o.Kind + "/" + o.Namespace + "/" + o.Name
}

func foreignHall(kind string) bool {
	if kind == KindForeign {
		return true
	}
	for _, k := range helperGroups {
		if k == kind {
			return true
		}
	}
	return false
}

func standingKey(st view.Structure) (string, bool) {
	switch {
	case strings.HasPrefix(st.Kind, "depot-") && st.Object.Kind == "Service", foreignHall(st.Kind):
		return objectKey(keyDistrict, st.Object), true
	case st.Kind == KindHarborGate:
		return objectKey(keyGate, st.Object), true
	case st.Kind == KindSignpost:
		return objectKey(keySign, st.Object), true
	}
	return "", false
}

func wantedOf(s Snapshot) []wanted {
	var out []wanted
	services := append([]corev1.Service(nil), s.Services...)
	sort.Slice(services, func(i, j int) bool {
		return services[i].Namespace+"/"+services[i].Name < services[j].Namespace+"/"+services[j].Name
	})
	fiefs := map[string]bool{}
	for _, ns := range s.Namespaces {
		fiefs[ns.Name] = true
	}
	ends := map[string]int{}
	for _, sl := range s.EndpointSlices {
		ends[sl.Namespace+"/"+sl.Labels["kubernetes.io/service-name"]] += len(sl.Endpoints)
	}
	for _, svc := range services {
		if fiefs[svc.Namespace] {
			it := depotItem(svc, ends[svc.Namespace+"/"+svc.Name])
			out = append(out, wanted{key: objectKey(keyDistrict, it.object), ns: svc.Namespace, it: it})
		}
	}
	for _, f := range foreignKinds(s) {
		if f.ns == "" || fiefs[f.ns] {
			it := f.item()
			out = append(out, wanted{key: objectKey(keyDistrict, it.object), ns: f.ns, it: it})
		}
	}
	for _, e := range harborEntries(s) {
		gate := objectKey(keyGate, e.object)
		out = append(out, wanted{key: gate, it: item{kind: KindHarborGate, w: 2, h: 2, object: e.object, label: e.label, state: view.StateReady, tip: e.tip}})
		for _, sign := range e.signs {
			out = append(out, wanted{key: objectKey(keySign, sign.object), it: sign, gate: gate})
		}
	}
	return out
}

func (w *World) Refurnish(s Snapshot) bool {
	want := wantedOf(s)
	desired := map[string]bool{}
	for _, d := range want {
		desired[d.key] = true
	}
	standing := map[string]int{}
	razed := map[string]bool{}
	for i, st := range w.Site.Structures {
		key, ok := standingKey(st)
		switch {
		case !ok:
		case desired[key]:
			standing[key] = i
		default:
			razed[st.ID] = true
		}
	}
	for _, d := range want {
		if i, ok := standing[d.key]; ok {
			st := &w.Site.Structures[i]
			st.Label = d.it.label
			if strings.HasPrefix(st.Kind, "depot-") {
				st.Kind = d.it.kind
			}
			w.Tips[st.ID] = d.it.tip
		}
	}
	w.raze(razed)
	have := map[string]view.Structure{}
	for _, st := range w.Site.Structures {
		if key, ok := standingKey(st); ok {
			have[key] = st
		}
	}
	roomy := true
	for _, d := range want {
		if _, ok := have[d.key]; ok || w.skipped[d.key] {
			continue
		}
		var st view.Structure
		placed := false
		switch {
		case d.gate != "":
			if g, ok := have[d.gate]; ok {
				st, placed = w.raiseSign(d.it, g)
			}
		case d.it.kind == KindHarborGate:
			st, placed = w.raiseGate(d.it)
		default:
			st, placed = w.raiseInFief(d.ns, d.it)
		}
		if placed {
			have[d.key] = st
		}
		roomy = roomy && placed
	}
	return roomy
}

func (w *World) noteSkipped(s Snapshot) {
	w.skipped = map[string]bool{}
	have := map[string]bool{}
	for _, st := range w.Site.Structures {
		if key, ok := standingKey(st); ok {
			have[key] = true
		}
	}
	for _, d := range wantedOf(s) {
		if !have[d.key] {
			w.skipped[d.key] = true
		}
	}
}

func (w *World) raze(ids map[string]bool) {
	if len(ids) == 0 {
		return
	}
	kept := make([]view.Structure, 0, len(w.Site.Structures))
	for _, st := range w.Site.Structures {
		if ids[st.ID] {
			delete(w.Tips, st.ID)
			continue
		}
		kept = append(kept, st)
	}
	w.Site.Structures = kept
	w.structureOf = map[string]int{}
	for i, st := range kept {
		w.structureOf[st.ID] = i
	}
}

func (w *World) freshID(kind string) string {
	used := map[string]bool{}
	for _, st := range w.Site.Structures {
		used[st.ID] = true
	}
	for n := len(w.Site.Structures) + 1; ; n++ {
		if id := fmt.Sprintf("%s-%d", strings.TrimPrefix(kind, "depot-"), n); !used[id] {
			return id
		}
	}
}

func (w *World) disturbsHut(st view.Structure) bool {
	lift := view.StructureLift(st.Kind, st.Variant)
	for y := st.Y - lift - 2; y <= st.Y+st.H+1; y++ {
		for x := st.X - lift - 2; x <= st.X+st.W+1; x++ {
			if _, hut := w.taken[view.Point{X: x, Y: y}]; !hut {
				continue
			}
			if x >= st.X-1 && x <= st.X+st.W && y >= st.Y-1 && y <= st.Y+st.H || hiddenBy(st, x, y) {
				return true
			}
		}
	}
	return false
}

func (w *World) stand(st view.Structure, tip Tip) view.Structure {
	w.add(st, tip)
	for _, l := range w.lands {
		kept := l.slots[:0]
		for _, sl := range l.slots {
			x, y := sl.at.X, sl.at.Y
			if x >= st.X-1 && x <= st.X+st.W && y >= st.Y-1 && y <= st.Y+st.H || hiddenBy(st, x, y) {
				continue
			}
			kept = append(kept, sl)
		}
		if len(kept) != len(l.slots) {
			l.slots, l.orders = kept, map[string][]int{}
		}
	}
	return st
}

func (w *World) openHarbor() bool {
	next := w.Site.WithGateHaven()
	if len(next.Havens) == 0 {
		return false
	}
	for _, st := range w.Site.Structures {
		for y := st.Y; y < st.Y+max(st.H, 1); y++ {
			for x := st.X; x < st.X+max(st.W, 1); x++ {
				if next.HarborGround(x, y, 1) {
					return false
				}
			}
		}
	}
	for at := range w.taken {
		if next.HarborGround(at.X, at.Y, 1) {
			return false
		}
	}
	w.Site, w.Site.NoHarbor = next, false
	for _, l := range w.lands {
		kept := l.slots[:0]
		for _, sl := range l.slots {
			if !next.HarborGround(sl.at.X, sl.at.Y, 1) {
				kept = append(kept, sl)
			}
		}
		if len(kept) != len(l.slots) {
			l.slots, l.orders = kept, map[string][]int{}
		}
	}
	return true
}

func (w *World) raiseGate(it item) (view.Structure, bool) {
	if w.Site.NoHarbor && !w.openHarbor() {
		return view.Structure{}, false
	}
	i := w.Gates
	side := -3
	if i%2 == 1 {
		side = 2
	}
	gate := w.Site.Gate
	target := view.Point{X: gate.X + 2 + gateSpacing*(i/2)*2, Y: gate.Y + side}
	at, ok := w.spotNear(target, 2, 2, 1, 1, func(x, y int) bool {
		return !w.nearHaven(x, y) && !w.disturbsHut(view.Structure{Kind: KindHarborGate, X: x, Y: y, W: 2, H: 2})
	})
	if !ok {
		return view.Structure{}, false
	}
	w.Gates++
	return w.stand(view.Structure{ID: fmt.Sprintf("gate-%d", w.Gates), Kind: KindHarborGate, X: at.X, Y: at.Y, W: 2, H: 2, State: view.StateReady, Progress: 1, Label: it.label, Object: it.object}, it.tip), true
}

func (w *World) raiseSign(it item, gate view.Structure) (view.Structure, bool) {
	from := view.Point{X: gate.X, Y: gate.Y + 3}
	if gate.Y < w.Site.Gate.Y {
		from = view.Point{X: gate.X, Y: gate.Y - 2}
	}
	at, ok := w.spotNear(from, 1, 1, 1, 1, func(x, y int) bool {
		return !w.nearHaven(x, y) && !w.disturbsHut(view.Structure{Kind: KindSignpost, X: x, Y: y, W: 1, H: 1})
	})
	if !ok {
		return view.Structure{}, false
	}
	w.Signposts++
	return w.stand(view.Structure{ID: fmt.Sprintf("signpost-%d", w.Signposts), Kind: KindSignpost, X: at.X, Y: at.Y, W: 1, H: 1, State: view.StateReady, Progress: 1,
		Label: it.label, Object: it.object}, it.tip), true
}

type spot struct {
	x, y, w, h int
	score      float64
}

func (w *World) fiefHome(ns string) (int, view.Point) {
	for _, d := range w.Districts {
		if d.Namespace != ns {
			continue
		}
		for i, is := range w.Isles {
			if is.ID == d.Isle {
				return i, d.Center
			}
		}
	}
	return 0, w.Site.Plaza
}

func (w *World) raiseInFief(ns string, it item) (view.Structure, bool) {
	isle, center := w.fiefHome(ns)
	region := uint8(isle + 1)
	owner := w.Site.KeepLand()
	lotOf := map[view.Point]string{}
	var mine []view.Point
	seed := center
	for _, l := range w.Lots {
		if l.Node != "" {
			continue
		}
		for _, p := range l.Tiles {
			lotOf[p] = l.Namespace
			if l.Namespace == ns {
				mine = append(mine, p)
			}
		}
		if l.Namespace == ns {
			seed = l.Seed
		}
	}
	ok := func(p view.Point, own bool) bool {
		i, in := w.index(p.X, p.Y)
		if !in || w.Site.Region[i] != region || owner[i] != 0 || !buildable(w.Site.Tiles[i]) || w.open(p.X, p.Y) || w.civic(p.X, p.Y) {
			return false
		}
		if own {
			return lotOf[p] == ns
		}
		_, other := lotOf[p]
		return !other
	}
	if sp, found := w.bestSpot(it, mine, seed, func(p view.Point) bool { return ok(p, true) }); found {
		return w.standItem(it, sp), true
	}
	var sp spot
	found := false
	for reach := strayReach; reach <= strayFar && !found; reach *= 2 {
		var near []view.Point
		for dy := -reach; dy <= reach; dy++ {
			for dx := -reach; dx <= reach; dx++ {
				if p := (view.Point{X: center.X + dx, Y: center.Y + dy}); ok(p, false) {
					near = append(near, p)
				}
			}
		}
		sp, found = w.bestSpot(it, near, center, func(p view.Point) bool { return ok(p, false) })
	}
	if !found {
		return view.Structure{}, false
	}
	st := w.standItem(it, sp)
	lot := Lot{Namespace: ns, Order: w.fiefOrder(ns), Seed: view.Point{X: sp.x, Y: sp.y}}
	for y := sp.y - 1; y <= sp.y+sp.h; y++ {
		for x := sp.x - 1; x <= sp.x+sp.w; x++ {
			if p := (view.Point{X: x, Y: y}); ok(p, false) || x >= sp.x && x < sp.x+sp.w && y >= sp.y && y < sp.y+sp.h {
				lot.Tiles = append(lot.Tiles, p)
			}
		}
	}
	sortPoints(lot.Tiles)
	w.Lots = append(w.Lots, lot)
	return st, true
}

func (w *World) fiefOrder(ns string) int {
	for _, d := range w.Districts {
		if d.Namespace == ns {
			return d.Order
		}
	}
	return 0
}

func (w *World) bestSpot(it item, tiles []view.Point, seed view.Point, ok func(view.Point) bool) (spot, bool) {
	shapes := [][2]int{{it.w, it.h}}
	if it.w != it.h {
		shapes = append(shapes, [2]int{it.h, it.w})
	}
	var best spot
	found := false
	for _, p := range tiles {
		for _, sh := range shapes {
			wd, ht := sh[0], sh[1]
			fits := true
			for dy := 0; dy < ht && fits; dy++ {
				for dx := 0; dx < wd && fits; dx++ {
					fits = ok(view.Point{X: p.X + dx, Y: p.Y + dy})
				}
			}
			if !fits || w.crowds(p.X, p.Y, wd, ht, nil) || w.disturbsHut(view.Structure{Kind: it.kind, Variant: it.variant, X: p.X, Y: p.Y, W: wd, H: ht}) {
				continue
			}
			front := 0.0
			switch {
			case wd*ht == 2 && !w.facesStreet(p.X, p.Y, wd):
				front = backStreet
			case wd*ht != 2 && !w.touchesStreet(p.X, p.Y, wd, ht):
				front = backStreet
			}
			cx, cy := float64(p.X)+float64(wd)/2, float64(p.Y)+float64(ht)/2
			sc := front + seedWeight*math.Hypot(cx-float64(seed.X), cy-float64(seed.Y))
			if !found || sc < best.score || sc == best.score && (p.Y < best.y || p.Y == best.y && p.X < best.x) {
				best, found = spot{p.X, p.Y, wd, ht, sc}, true
			}
		}
	}
	return best, found
}

func (w *World) standItem(it item, sp spot) view.Structure {
	variant := it.variant
	if strings.HasPrefix(it.kind, "depot-") {
		variant = view.FaceSE
		if sp.w == 2 {
			variant = view.FaceSW
		}
	}
	return w.stand(view.Structure{ID: w.freshID(it.kind), Kind: it.kind, Variant: variant, X: sp.x, Y: sp.y, W: sp.w, H: sp.h, State: it.state, Progress: 1, Label: it.label, Object: it.object}, it.tip)
}
