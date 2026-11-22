package live

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/view"
)

func snapshotAt(t testing.TB, path string) Snapshot {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s, err := ReadSnapshot(f)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestRefurnishingAFreshlyDrawnRealmChangesNothing(t *testing.T) {
	for _, path := range []string{"testdata/kind-snapshot.json", "testdata/managed-snapshot.json", "testdata/gke-staging-snapshot.json"} {
		s := snapshotAt(t, path)
		w, err := Build(s)
		if err != nil {
			t.Fatal(err)
		}
		structures := append([]view.Structure(nil), w.Site.Structures...)
		lots := len(w.Lots)
		tips := map[string]Tip{}
		for id, tip := range w.Tips {
			tips[id] = tip
		}
		if !w.Refurnish(s) {
			t.Errorf("%s: a freshly drawn realm asks to be drawn again", path)
		}
		if !reflect.DeepEqual(structures, w.Site.Structures) || lots != len(w.Lots) || !reflect.DeepEqual(tips, w.Tips) {
			t.Errorf("%s: refurnishing a freshly drawn realm changed it", path)
		}
	}
}

func service(ns, name string, typ corev1.ServiceType, lb string) corev1.Service {
	svc := corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, UID: types.UID("svc-" + ns + "-" + name), ResourceVersion: "2000"},
		Spec: corev1.ServiceSpec{Type: typ, ClusterIP: "10.96.7.7", Ports: []corev1.ServicePort{{Port: 80, NodePort: 30080}}}}
	if lb != "" {
		svc.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{{IP: lb}}
	}
	return svc
}

func ingress(ns, name, class, backend, lb string) networkingv1.Ingress {
	ing := networkingv1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, UID: types.UID("ing-" + ns + "-" + name), ResourceVersion: "2000"},
		Spec: networkingv1.IngressSpec{IngressClassName: &class, Rules: []networkingv1.IngressRule{{Host: name + ".example", IngressRuleValue: networkingv1.IngressRuleValue{
			HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{Path: "/", Backend: networkingv1.IngressBackend{Service: &networkingv1.IngressServiceBackend{Name: backend}}}}}}}}}}
	if lb != "" {
		ing.Status.LoadBalancer.Ingress = []networkingv1.IngressLoadBalancerIngress{{IP: lb}}
	}
	return ing
}

func gateway(ns, name string) Gateway {
	return Gateway{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, UID: types.UID("gw-" + ns + "-" + name), ResourceVersion: "2000"},
		Spec: GatewaySpec{GatewayClassName: "none-installed", Listeners: []GatewayListener{{Name: "http", Port: 80, Protocol: "HTTP"}}}}
}

type standing struct {
	structures map[string]view.Structure
	huts       map[types.UID]HutPlot
}

func standingOf(w *World) standing {
	out := standing{structures: map[string]view.Structure{}, huts: hutsAt(w)}
	for _, st := range w.Site.Structures {
		out.structures[st.ID] = st
	}
	return out
}

func (was standing) unmoved(t *testing.T, w *World, when string) {
	t.Helper()
	now := standingOf(w)
	for id, st := range was.structures {
		got, ok := now.structures[id]
		if !ok {
			continue
		}
		if got.X != st.X || got.Y != st.Y || got.W != st.W || got.H != st.H || got.Object != st.Object {
			t.Errorf("%s: %s %s moved from %d,%d to %d,%d", when, st.ID, st.Label, st.X, st.Y, got.X, got.Y)
		}
	}
	for uid, h := range was.huts {
		if got, ok := now.huts[uid]; ok && (got.X != h.X || got.Y != h.Y || got.Placed != h.Placed) {
			t.Errorf("%s: hut %s moved from %d,%d to %d,%d", when, h.Name, h.X, h.Y, got.X, got.Y)
		}
	}
}

func overlapping(w *World) []string {
	var out []string
	sts := w.Site.Structures
	for i := range sts {
		for j := i + 1; j < len(sts); j++ {
			a, b := sts[i], sts[j]
			if a.X < b.X+max(b.W, 1) && b.X < a.X+max(a.W, 1) && a.Y < b.Y+max(b.H, 1) && b.Y < a.Y+max(a.H, 1) {
				out = append(out, a.ID+" and "+b.ID)
			}
		}
		for _, h := range w.Huts {
			if h.Placed && !h.Shrine && h.X >= sts[i].X && h.X < sts[i].X+max(sts[i].W, 1) && h.Y >= sts[i].Y && h.Y < sts[i].Y+max(sts[i].H, 1) {
				out = append(out, sts[i].ID+" on hut "+h.Name)
			}
		}
	}
	return out
}

func structuresFor(w *World, kind, ns, name string) []view.Structure {
	var out []view.Structure
	for _, st := range w.Site.Structures {
		hall := st.Object.Name == "" && strings.HasSuffix(st.Object.Resource, "."+name)
		if st.Object.Kind == kind && st.Object.Namespace == ns && (st.Object.Name == name || hall) {
			out = append(out, st)
		}
	}
	return out
}

func TestAddingAndRemovingServicesIngressesAndGatewaysOnAnEventStreamMovesNothingStanding(t *testing.T) {
	s := kindSnapshot(t)
	r, err := OpenRecorded(s)
	if err != nil {
		t.Fatal(err)
	}
	w := r.World()
	before := standingOf(w)
	born := s.TakenAt.Add(time.Hour)
	services := []corev1.Service{
		service("shop", "basket", corev1.ServiceTypeClusterIP, ""),
		service("shop", "till", corev1.ServiceTypeClusterIP, ""),
		service("batch", "queue", corev1.ServiceTypeClusterIP, ""),
		service("watch", "beacon", corev1.ServiceTypeLoadBalancer, "172.18.0.200"),
		service("default", "front", corev1.ServiceTypeNodePort, ""),
	}
	ingresses := []networkingv1.Ingress{
		ingress("watch", "beacon", "lantern", "beacon", "172.18.0.200"),
		ingress("shop", "basket", "nginx", "basket", ""),
		ingress("batch", "queue", "none-yet", "queue", ""),
	}
	gateways := []Gateway{gateway("shop", "gate"), gateway("watch", "tower")}
	stream(t, r, "namespaces", watchAdded, corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "newfief", UID: "uid-newfief", CreationTimestamp: metav1.NewTime(born)}})
	services = append(services, service("newfief", "first", corev1.ServiceTypeClusterIP, ""))
	for i, svc := range services {
		stream(t, r, "services", watchAdded, svc)
		stream(t, r, "pods", watchAdded, syntheticPod(500+i, w.Keeps[1+i%2].Node, born))
		if i%2 == 0 {
			r.Pump(PumpEvery)
		}
	}
	for _, ing := range ingresses {
		stream(t, r, "ingresses", watchAdded, ing)
	}
	for _, g := range gateways {
		stream(t, r, "gateways", watchAdded, g)
	}
	r.Pump(PumpEvery)
	if r.World() != w {
		t.Fatal("new Services, Ingresses and Gateways redrew the realm")
	}
	before.unmoved(t, w, "after they came")
	for _, svc := range services {
		depots := 0
		for _, st := range structuresFor(w, "Service", svc.Namespace, svc.Name) {
			if st.Kind != KindHarborGate {
				depots++
			}
		}
		if depots != 1 {
			t.Errorf("the Service %s/%s stands as %d depots", svc.Namespace, svc.Name, depots)
		}
	}
	if gates := structuresFor(w, "Service", "watch", "beacon"); len(gates) != 2 {
		t.Errorf("the LoadBalancer Service stands as %+v, not a depot and a harbor gate", gates)
	}
	for _, ing := range ingresses {
		if got := structuresFor(w, "Ingress", ing.Namespace, ing.Name); len(got) != 1 || got[0].Kind != KindSignpost {
			t.Errorf("the Ingress %s/%s stands as %+v", ing.Namespace, ing.Name, got)
		}
	}
	if got := structuresFor(w, "IngressClass", "", "none-yet"); len(got) != 1 || got[0].Kind != KindHarborGate {
		t.Errorf("the new IngressClass's plain gate stands as %+v", got)
	}
	for _, g := range gateways {
		got := structuresFor(w, "Gateway", g.Namespace, g.Name)
		if len(got) != 1 || got[0].Kind != KindHarborGate {
			t.Errorf("the Gateway %s/%s stands as %+v", g.Namespace, g.Name, got)
		} else if tip := w.Tips[got[0].ID]; tip.Names != "Gateway "+g.Namespace+"/"+g.Name+", GatewayClass none-installed, listeners HTTP 80" {
			t.Errorf("the Gateway's tip %+v", tip)
		}
	}
	if found := false; true {
		for _, l := range w.Lots {
			found = found || l.Namespace == "newfief"
		}
		if !found {
			t.Error("the new fief's depot stands on no land of its fief")
		}
	}
	if bad := overlapping(w); len(bad) > 0 {
		t.Errorf("structures overlap: %v", bad)
	}
	middle := standingOf(w)
	for _, g := range gateways {
		stream(t, r, "gateways", watchDeleted, g)
	}
	for _, ing := range ingresses {
		stream(t, r, "ingresses", watchDeleted, ing)
	}
	for _, svc := range services {
		stream(t, r, "services", watchDeleted, svc)
	}
	r.Pump(PumpEvery)
	if r.World() != w {
		t.Fatal("deleting Services, Ingresses and Gateways redrew the realm")
	}
	middle.unmoved(t, w, "after they went")
	for _, svc := range services {
		if got := structuresFor(w, "Service", svc.Namespace, svc.Name); len(got) != 0 {
			t.Errorf("the deleted Service %s/%s still stands: %+v", svc.Namespace, svc.Name, got)
		}
	}
	for _, ing := range ingresses {
		if got := structuresFor(w, "Ingress", ing.Namespace, ing.Name); len(got) != 0 {
			t.Errorf("the deleted Ingress %s/%s still stands", ing.Namespace, ing.Name)
		}
	}
	for _, g := range gateways {
		if got := structuresFor(w, "Gateway", g.Namespace, g.Name); len(got) != 0 {
			t.Errorf("the deleted Gateway %s/%s still stands", g.Namespace, g.Name)
		}
	}
	if got := structuresFor(w, "IngressClass", "", "none-yet"); len(got) != 0 {
		t.Error("a plain gate with no signpost left still stands")
	}
	for id := range before.structures {
		if _, ok := standingOf(w).structures[id]; !ok {
			t.Errorf("the standing structure %s went", id)
		}
	}
	if len(w.Site.Structures) != len(before.structures) {
		t.Errorf("%d structures after everything went, %d before", len(w.Site.Structures), len(before.structures))
	}
}

func TestCustomResourcesComingAndGoingRaiseAndRazeTheirForeignHall(t *testing.T) {
	s := kindSnapshot(t)
	r, err := OpenRecorded(s)
	if err != nil {
		t.Fatal(err)
	}
	w := r.World()
	before := standingOf(w)
	c := CRD{Name: "dragons.lair.example", UID: "uid-dragons", Group: "lair.example", Kind: "Dragon", Plural: "dragons", Scope: scopeNamespaced, Version: "v1"}
	stream(t, r, "customresourcedefinitions", watchAdded, crdJSON(c))
	dragonA := metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{Name: "dragon-a", Namespace: "shop", UID: "uid-dragon-a"}}
	streamAny(t, r, c.List(), watchAdded, dragonA)
	r.Pump(PumpEvery)
	halls := structuresFor(w, "Dragon", "shop", "lair.example")
	if r.World() != w || len(halls) != 1 || !foreignHall(halls[0].Kind) {
		t.Fatalf("the new foreign kind stands as %+v (redrawn %v)", halls, r.World() != w)
	}
	if tip := w.Tips[halls[0].ID]; tip.Names != "1 Dragon (lair.example/v1) in namespace shop" {
		t.Errorf("the hall's tip %+v", tip)
	}
	dragonB := metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{Name: "dragon-b", Namespace: "shop", UID: "uid-dragon-b"}}
	streamAny(t, r, c.List(), watchAdded, dragonB)
	r.Pump(PumpEvery)
	if tip := w.Tips[halls[0].ID]; tip.Names != "2 Dragon (lair.example/v1) in namespace shop" {
		t.Errorf("the hall's tip after a second instance %+v", tip)
	}
	streamAny(t, r, c.List(), watchDeleted, dragonA)
	streamAny(t, r, c.List(), watchDeleted, dragonB)
	r.Pump(PumpEvery)
	if got := structuresFor(w, "Dragon", "shop", "lair.example"); len(got) != 0 {
		t.Errorf("a foreign hall with no instance left still stands: %+v", got)
	}
	before.unmoved(t, w, "after the dragons")
	if r.World() != w {
		t.Error("custom resources coming and going redrew the realm")
	}
}

func streamAny(t testing.TB, r *Realm, list, typ string, obj any) {
	t.Helper()
	k, ok := r.kindOf(list)
	if !ok {
		t.Fatalf("no kind %s", list)
	}
	raw, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ingest(k, typ, raw); err != nil {
		t.Fatal(err)
	}
}

func crdJSON(c CRD) map[string]any {
	return map[string]any{"metadata": map[string]any{"name": c.Name, "uid": c.UID},
		"spec": map[string]any{"group": c.Group, "scope": c.Scope, "names": map[string]any{"kind": c.Kind, "plural": c.Plural},
			"versions": []map[string]any{{"name": c.Version, "served": true, "storage": true}}}}
}

func TestAFiefWithNoRoomLeftForANewDepotFallsBackToTheRedraw(t *testing.T) {
	s := kindSnapshot(t)
	r, err := OpenRecorded(s)
	if err != nil {
		t.Fatal(err)
	}
	w := r.World()
	var added []corev1.Service
	for i := 0; i < 400 && r.World() == w; i++ {
		svc := service("batch", fmt.Sprintf("crowd-%03d", i), corev1.ServiceTypeClusterIP, "")
		added = append(added, svc)
		stream(t, r, "services", watchAdded, svc)
		r.Pump(PumpEvery)
	}
	if r.World() == w {
		t.Fatal("400 new depots in one fief never filled it")
	}
	now := r.World()
	for _, svc := range added {
		if got := structuresFor(now, "Service", svc.Namespace, svc.Name); len(got) != 1 {
			t.Errorf("after the redraw the Service %s stands as %d structures", svc.Name, len(got))
		}
	}
	if bad := overlapping(now); len(bad) > 0 {
		t.Errorf("structures overlap after the redraw: %v", bad)
	}
}

func harborless(t testing.TB) Snapshot {
	s := kindSnapshot(t)
	services := s.Services[:0]
	for _, svc := range s.Services {
		if svc.Spec.Type != corev1.ServiceTypeLoadBalancer && svc.Spec.Type != corev1.ServiceTypeNodePort {
			services = append(services, svc)
		}
	}
	s.Services, s.Ingresses, s.Gateways = services, nil, nil
	return s
}

func TestTheFirstGateOfARealmWithNoHarborOpensTheHarborWhereItStandsOrRedraws(t *testing.T) {
	s := harborless(t)
	r, err := OpenRecorded(s)
	if err != nil {
		t.Fatal(err)
	}
	w := r.World()
	if !w.Site.NoHarbor {
		t.Fatal("the fixture has a harbor")
	}
	before := standingOf(w)
	stream(t, r, "gateways", watchAdded, gateway("shop", "first"))
	stream(t, r, "ingresses", watchAdded, ingress("shop", "web", "nginx", "web", ""))
	r.Pump(PumpEvery)
	if r.World() != w {
		t.Fatal("the first harbor gate redrew a realm whose harbor fits where it stands")
	}
	if w.Site.NoHarbor || len(w.Site.Havens) == 0 {
		t.Fatalf("the harbor did not open: %+v", w.Site.Havens)
	}
	before.unmoved(t, w, "after the harbor opened")
	if got := structuresFor(w, "Gateway", "shop", "first"); len(got) != 1 {
		t.Errorf("the first Gateway stands as %+v", got)
	}
	if got := structuresFor(w, "Ingress", "shop", "web"); len(got) != 1 {
		t.Errorf("the Ingress stands as %+v", got)
	}
	for _, h := range w.Huts {
		if h.Placed && !h.Shrine && w.Site.HarborGround(h.X, h.Y, 1) {
			t.Errorf("hut %s stands on harbor ground", h.Name)
		}
	}
	if bad := overlapping(w); len(bad) > 0 {
		t.Errorf("structures overlap: %v", bad)
	}

	crowded, err := OpenRecorded(harborless(t))
	if err != nil {
		t.Fatal(err)
	}
	cw := crowded.World()
	next := cw.Site.WithGateHaven()
	var on view.Point
	for y := 0; y < next.Height && on == (view.Point{}); y++ {
		for x := 0; x < next.Width; x++ {
			if next.HarborGround(x, y, 0) {
				on = view.Point{X: x, Y: y}
				break
			}
		}
	}
	cw.taken[on] = "uid-in-the-way"
	stream(t, crowded, "gateways", watchAdded, gateway("shop", "first"))
	crowded.Pump(PumpEvery)
	if crowded.World() == cw {
		t.Error("a harbor whose ground holds a hut opened where it stands instead of redrawing")
	}
}

func TestWhatAFullDrawCouldNotPlaceWaitsForTheNextFullDraw(t *testing.T) {
	s := kindSnapshot(t)
	r, err := OpenRecorded(s)
	if err != nil {
		t.Fatal(err)
	}
	w := r.World()
	g := gateway("shop", "unplaced")
	w.skipped[objectKey(keyGate, view.ObjectRef{Kind: "Gateway", Namespace: g.Namespace, Name: g.Name})] = true
	stream(t, r, "gateways", watchAdded, g)
	r.Pump(PumpEvery)
	if r.World() != w {
		t.Fatal("an object the full draw could not place redrew the realm")
	}
	if got := structuresFor(w, "Gateway", "shop", "unplaced"); len(got) != 0 {
		t.Errorf("an object the full draw could not place was tried again: %+v", got)
	}
}
