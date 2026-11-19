package live

import (
	"context"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/view"
)

const redrawLimit = 20 * time.Second

func (w *World) Restaff(s Snapshot) {
	w.Crews = nil
	w.crews(s)
	crewOf := w.crewIndex()
	for i := range w.Huts {
		w.Huts[i].Crew = crewOf[w.Huts[i].UID]
	}
	w.Settle(s.Pods)
	w.Idle = nil
	w.idle(s)
	w.rekeep(s.Nodes)
	w.refief(s)
}

func (w *World) rekeep(nodes []corev1.Node) {
	ready := map[string]corev1.ConditionStatus{}
	for _, n := range nodes {
		ready[n.Name] = nodeReady(n)
	}
	for i := range w.Keeps {
		if st, ok := ready[w.Keeps[i].Node]; ok {
			w.Keeps[i].Ready = st
		}
	}
	for i, st := range w.Site.Structures {
		if st.Kind != view.StructKeep || st.Object.Kind != "Node" {
			continue
		}
		if r, ok := ready[st.Object.Name]; ok {
			w.Site.Structures[i].State = view.StateReady
			if r != corev1.ConditionTrue {
				w.Site.Structures[i].State = view.StateDark
			}
		}
	}
}

func (w *World) refief(s Snapshot) {
	present := map[string]bool{}
	for _, ns := range s.Namespaces {
		present[ns.Name] = true
	}
	w.Districts = slices.DeleteFunc(w.Districts, func(d District) bool { return d.Namespace != courtName && d.Namespace != "" && !present[d.Namespace] })
	known := map[string]bool{}
	for _, d := range w.Districts {
		known[d.Namespace] = true
	}
	pods := map[string]int{}
	for _, p := range s.Pods {
		pods[p.Namespace]++
	}
	for _, ns := range byUIDNamespaces(s.Namespaces) {
		if known[ns.Name] || len(w.Isles) == 0 {
			continue
		}
		w.Districts = append(w.Districts, District{Namespace: ns.Name, UID: ns.UID, Isle: w.Isles[0].ID, Order: len(w.Districts) + 1,
			Citadel: ns.Name == view.CitadelFief, Center: w.hutCenter(ns.Name, 0)})
		if w.newFiefs == nil {
			w.newFiefs = map[string]bool{}
		}
		w.newFiefs[ns.Name] = true
	}
	for i := range w.Districts {
		if n, ok := pods[w.Districts[i].Namespace]; ok || present[w.Districts[i].Namespace] {
			w.Districts[i].Pods = n
		}
	}
	slices.SortStableFunc(w.Districts, func(a, b District) int {
		switch {
		case a.UID < b.UID:
			return -1
		case a.UID > b.UID:
			return 1
		}
		return 0
	})
}

func byUIDNamespaces(list []corev1.Namespace) []corev1.Namespace {
	return byUID(list, func(n corev1.Namespace) types.UID { return n.UID })
}

func (w *World) Roomless() int {
	n := 0
	for _, h := range w.Huts {
		if h.Node != "" && !h.Placed && !h.Shrine {
			n++
		}
	}
	return n
}

func (w *World) newFiefRoomless() int {
	n := 0
	for _, h := range w.Huts {
		if h.Node != "" && !h.Placed && !h.Shrine && w.newFiefs[h.Namespace] {
			n++
		}
	}
	return n
}

func (w *World) keepsDiffer(s Snapshot) bool {
	now := keepsOf(s)
	if len(now) != len(w.Keeps) {
		return true
	}
	had := map[string]bool{}
	for _, k := range w.Keeps {
		had[k.Node] = true
	}
	for _, n := range now {
		if !had[n.Name] {
			return true
		}
	}
	return false
}

func (r *Realm) redraw() {
	if !r.world.keepsDiffer(r.snap) {
		before := r.world.newFiefRoomless()
		r.world.Restaff(r.snap)
		roomy := r.world.Refurnish(r.snap)
		r.drawn++
		if roomy && r.world.newFiefRoomless() <= before {
			return
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), redrawLimit)
	defer cancel()
	if w, err := BuildContext(ctx, r.snap.clone()); err == nil {
		r.world = w
		r.drawn++
	}
}
