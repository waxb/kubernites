package live_test

import (
	"math/rand/v2"
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/view"
)

func build(t *testing.T, s live.Snapshot) *live.World {
	t.Helper()
	w, err := live.Build(s)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func structuresOf(w *live.World, kind string) []view.Structure {
	var out []view.Structure
	for _, st := range w.Site.Structures {
		if st.Kind == kind {
			out = append(out, st)
		}
	}
	return out
}

func TestTheManagedRealmHasTwoHarborGatesAndSevenSignposts(t *testing.T) {
	w := build(t, livetest.Managed())
	gates, signs := structuresOf(w, live.KindHarborGate), structuresOf(w, live.KindSignpost)
	if len(gates) != 2 || len(signs) != 7 || w.Site.NoHarbor {
		t.Fatalf("%d gates, %d signposts, no harbor %v", len(gates), len(signs), w.Site.NoHarbor)
	}
	kinds := map[string]bool{}
	for _, g := range gates {
		kinds[g.Object.Kind+"/"+g.Object.Name] = true
	}
	if !kinds["Service/gate-controller"] || !kinds["Service/admin-portal"] {
		t.Errorf("gates %v: one per LoadBalancer and NodePort Service", kinds)
	}
	for _, sgn := range signs {
		if sgn.Object.Kind != "Ingress" {
			t.Errorf("signpost %s stands for %s", sgn.ID, sgn.Object.Kind)
		}
		if tip := w.Tips[sgn.ID]; tip.Story == "" || tip.Names == "" {
			t.Errorf("signpost %s has no tip", sgn.ID)
		}
	}
	if len(w.Site.Havens) != 1 {
		t.Fatalf("%d havens, want the one harbor the gates stand at", len(w.Site.Havens))
	}
	quay := w.Site.Havens[0].Quay
	for _, g := range gates {
		if d := max(abs(g.X-quay.X), abs(g.Y-quay.Y)); d > harborReach {
			t.Errorf("harbor gate %s stands %d tiles from the quay", g.ID, d)
		}
	}
	for _, st := range w.Site.Structures {
		for y := st.Y; y < st.Y+st.H; y++ {
			for x := st.X; x < st.X+st.W; x++ {
				if w.Site.HarborGround(x, y, 0) {
					t.Errorf("%s stands on the quay at %d,%d", st.ID, x, y)
				}
			}
		}
	}
	for _, h := range w.Huts {
		if h.Placed && !h.Shrine && w.Site.HarborGround(h.X, h.Y, 0) {
			t.Errorf("hut %s stands on the quay", h.Name)
		}
	}
}

const harborReach = 10

func TestAnIngressWithNoAddressStandsAtAPlainGate(t *testing.T) {
	s := livetest.Managed()
	s.Ingresses = append([]networkingv1.Ingress(nil), s.Ingresses...)
	s.Ingresses[0].Status = networkingv1.IngressStatus{}
	w := build(t, s)
	gates := structuresOf(w, live.KindHarborGate)
	plain := 0
	for _, g := range gates {
		if g.Object.Kind == "IngressClass" && g.Object.Name == "nginx" {
			plain++
		}
	}
	if len(gates) != 3 || plain != 1 || len(structuresOf(w, live.KindSignpost)) != 7 {
		t.Errorf("%d gates (%d plain), %d signposts", len(gates), plain, len(structuresOf(w, live.KindSignpost)))
	}
}

func withoutEntries(s live.Snapshot) live.Snapshot {
	s.Ingresses, s.Gateways = nil, nil
	var services []corev1.Service
	for _, svc := range s.Services {
		if svc.Spec.Type == corev1.ServiceTypeClusterIP {
			services = append(services, svc)
		}
	}
	s.Services = services
	return s
}

func TestAClusterWithNoWayInHasNoHarbor(t *testing.T) {
	s := withoutEntries(livetest.Managed())
	for _, w := range []*live.World{build(t, s), build(t, withoutEntries(livetest.Snapshot(t, "testdata/kind-snapshot.json")))} {
		if !w.Site.NoHarbor || len(w.Site.Havens) != 0 || len(structuresOf(w, live.KindHarborGate)) != 0 || len(structuresOf(w, live.KindSignpost)) != 0 {
			t.Errorf("no way in, yet the harbor stands: no harbor %v, %d gates", w.Site.NoHarbor, len(structuresOf(w, live.KindHarborGate)))
		}
	}
}

func covered(a, b view.Point) bool {
	for _, d := range [][2]int{{1, 1}, {1, 0}, {0, 1}, {2, 1}, {1, 2}} {
		if a.X-b.X == d[0] && a.Y-b.Y == d[1] || b.X-a.X == d[0] && b.Y-a.Y == d[1] {
			return true
		}
	}
	return false
}

func TestEveryPodHasItsPlaceOnItsKeepAndNoHutCoversAnother(t *testing.T) {
	w := build(t, livetest.Managed())
	if len(w.Huts) != 58 {
		t.Errorf("%d huts", len(w.Huts))
	}
	shrines := placedWithoutCovering(t, w)
	if shrines != 21 {
		t.Errorf("%d shrines", shrines)
	}
	placedWithoutCovering(t, build(t, livetest.Snapshot(t, "testdata/kind-snapshot.json")))
}

func placedWithoutCovering(t *testing.T, w *live.World) int {
	t.Helper()
	owner := w.Site.KeepLand()
	keepAt := map[string]int16{}
	for i, st := range w.Site.Structures {
		if st.Kind == view.StructKeep {
			keepAt[st.Object.Name] = int16(i + 1)
		}
	}
	var huts []live.HutPlot
	shrines := 0
	for _, h := range w.Huts {
		if !h.Placed {
			if h.Node != "" {
				t.Errorf("pod %s/%s has no place", h.Namespace, h.Name)
			}
			continue
		}
		if h.Shrine {
			shrines++
			continue
		}
		huts = append(huts, h)
		if owner[h.Y*w.Site.Width+h.X] != keepAt[h.Node] {
			t.Errorf("hut %s stands off its Keep %s's land at %d,%d", h.Name, h.Node, h.X, h.Y)
		}
		if tile := w.Site.Tile(h.X, h.Y); tile == view.TileRoad || tile == view.TilePlaza || tile == view.TileWater {
			t.Errorf("hut %s stands on %v", h.Name, tile)
		}
	}
	for i, a := range huts {
		for _, b := range huts[i+1:] {
			if covered(view.Point{X: a.X, Y: a.Y}, view.Point{X: b.X, Y: b.Y}) {
				t.Errorf("hut %s at %d,%d covers %s at %d,%d", a.Name, a.X, a.Y, b.Name, b.X, b.Y)
			}
		}
	}
	return shrines
}

func TestShrinesStandBesideTheirKeepsAndIdleShrinesAreListed(t *testing.T) {
	w := build(t, livetest.Managed())
	keeps := map[string]view.Structure{}
	for _, st := range structuresOf(w, view.StructKeep) {
		keeps[st.Object.Name] = st
	}
	node := map[string]string{}
	for _, h := range w.Huts {
		node[h.Name] = h.Node
	}
	shrines := structuresOf(w, live.KindShrine)
	if len(shrines) != 21 {
		t.Fatalf("%d shrines for 21 DaemonSet Pods", len(shrines))
	}
	for _, sh := range shrines {
		k := keeps[node[sh.Object.Name]]
		if d := max(abs(sh.X-k.X-1), abs(sh.Y-k.Y-1)); d > 9 {
			t.Errorf("shrine %s stands %d tiles from its Keep %s", sh.ID, d, k.Object.Name)
		}
	}
	if len(w.Idle) != 32 {
		t.Errorf("%d idle shrines listed, want 32", len(w.Idle))
	}
}

func TestTheCitadelAcrossTheSeaStandsOnlyWithoutAControlPlaneNode(t *testing.T) {
	w := build(t, livetest.Managed())
	sea := structuresOf(w, live.KindSeaChancery)
	if len(sea) != 1 || w.Chancery != livetest.ManagedServer+" "+livetest.ManagedVersion {
		t.Fatalf("%d Citadels across the sea, label %q", len(sea), w.Chancery)
	}
	c := sea[0]
	for y := c.Y; y < c.Y+c.H; y++ {
		for x := c.X; x < c.X+c.W; x++ {
			if w.Site.Tile(x, y) != view.TileWater {
				t.Errorf("the Citadel across the sea stands on %v at %d,%d", w.Site.Tile(x, y), x, y)
			}
		}
	}
	kind := build(t, livetest.Snapshot(t, "testdata/kind-snapshot.json"))
	if len(structuresOf(kind, live.KindSeaChancery)) != 0 || kind.Chancery != "" || kind.Keeps[0].Variant != "tower" {
		t.Errorf("a cluster with a control-plane node drew the Citadel across the sea")
	}
}

func TestFiefBuildingsFrontTheirStreets(t *testing.T) {
	w := build(t, livetest.Managed())
	open := func(x, y int) bool {
		t := w.Site.Tile(x, y)
		return t == view.TileRoad || t == view.TilePlaza
	}
	front, total := 0, 0
	for _, st := range w.Site.Structures {
		switch {
		case view.IsStall(st):
			total++
			if st.W == 1 && (open(st.X+1, st.Y) || open(st.X+1, st.Y+1)) || st.W == 2 && (open(st.X, st.Y+1) || open(st.X+1, st.Y+1)) {
				front++
			}
		case st.Kind == view.StructVault || st.Kind == live.KindForeign || st.Kind == "seal-smithy" || st.Kind == "key-gate":
			total++
			for dy := -1; dy <= st.H; dy++ {
				for dx := -1; dx <= st.W; dx++ {
					if (dx == -1 || dx == st.W || dy == -1 || dy == st.H) && open(st.X+dx, st.Y+dy) {
						front++
						dx, dy = st.W+1, st.H+1
					}
				}
			}
		}
	}
	if total == 0 || front*10 < total*9 {
		t.Errorf("%d of %d fief buildings front a street", front, total)
	}
}

func TestAVaultStandsOnItsVolumesIsle(t *testing.T) {
	s := livetest.Snapshot(t, "testdata/kind-snapshot.json")
	w := build(t, s)
	isleOf := map[string]string{}
	for _, is := range w.Isles {
		for _, n := range is.Nodes {
			isleOf[n] = is.ID
		}
	}
	want := map[string]string{}
	for _, pvc := range s.PVCs {
		for _, pv := range s.PVs {
			if pv.Name != pvc.Spec.VolumeName || pv.Spec.NodeAffinity == nil || pv.Spec.NodeAffinity.Required == nil {
				continue
			}
			for _, term := range pv.Spec.NodeAffinity.Required.NodeSelectorTerms {
				for _, e := range term.MatchExpressions {
					if e.Key == "kubernetes.io/hostname" && len(e.Values) == 1 {
						want[pvc.Name] = isleOf[e.Values[0]]
					}
				}
			}
		}
	}
	if len(want) != 2 {
		t.Fatalf("volumes pinned to nodes: %v", want)
	}
	vaults := structuresOf(w, view.StructVault)
	if len(vaults) != 2 {
		t.Fatalf("%d vaults", len(vaults))
	}
	for _, v := range vaults {
		is, ok := w.Site.IslandAt(v.X, v.Y)
		if !ok || is.ID != want[v.Object.Name] {
			t.Errorf("vault %s stands on %s, its volume is on %s", v.Object.Name, is.ID, want[v.Object.Name])
		}
	}
}

func TestTheManagedRealmIsTheSameWhateverTheOrder(t *testing.T) {
	s := livetest.Managed()
	first := build(t, s)
	for seed := uint64(1); seed <= 3; seed++ {
		rng := rand.New(rand.NewPCG(seed, seed))
		o := s
		o.Pods = append([]corev1.Pod(nil), s.Pods...)
		o.Services = append([]corev1.Service(nil), s.Services...)
		o.PVCs = append([]corev1.PersistentVolumeClaim(nil), s.PVCs...)
		o.Custom = append([]live.CustomResource(nil), s.Custom...)
		rng.Shuffle(len(o.Pods), func(i, j int) { o.Pods[i], o.Pods[j] = o.Pods[j], o.Pods[i] })
		rng.Shuffle(len(o.Services), func(i, j int) { o.Services[i], o.Services[j] = o.Services[j], o.Services[i] })
		rng.Shuffle(len(o.PVCs), func(i, j int) { o.PVCs[i], o.PVCs[j] = o.PVCs[j], o.PVCs[i] })
		rng.Shuffle(len(o.Custom), func(i, j int) { o.Custom[i], o.Custom[j] = o.Custom[j], o.Custom[i] })
		w := build(t, o)
		if len(w.Site.Structures) != len(first.Site.Structures) {
			t.Fatalf("shuffle %d: %d structures, first %d", seed, len(w.Site.Structures), len(first.Site.Structures))
		}
		for i := range w.Site.Structures {
			if w.Site.Structures[i] != first.Site.Structures[i] {
				t.Fatalf("shuffle %d: structure %d is %+v, first %+v", seed, i, w.Site.Structures[i], first.Site.Structures[i])
			}
		}
		for i := range w.Site.Tiles {
			if w.Site.Tiles[i] != first.Site.Tiles[i] {
				t.Fatalf("shuffle %d: tile %d differs", seed, i)
			}
		}
	}
}
