package live

import (
	"container/heap"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	KindShrine      = "shrine"
	KindForeign     = "foreign"
	KindHarborGate  = "harbor-gate"
	KindSignpost    = "signpost"
	KindSeaChancery = "sea-chancery"

	courtName    = ""
	civicClear   = 4
	harborClear  = 7
	shrineRing   = 3
	shrineRings  = 4
	gateSpacing  = 3
	harborArea   = 160.0
	squareSide   = 3
	squareItems  = 2
	itemRoom     = 1.5
	quarterRoom  = 5
	lotTries     = 4
	lotGrowth    = 1.4
	roadNoise    = 0.7
	lotNoise     = 1.1
	noiseCell    = 5.0
	seedSpacing  = 5
	anchorShare  = 0.55
	anchorMin    = 3.0
	frontWeight  = 3.0
	seedWeight   = 0.35
	strayCost    = 40.0
	foreignCost  = 400.0
	frontReach   = 6
	chanceryOff  = 4
	chanceryFar  = 9
	chanceryDry  = 2
	marketStride = 2
	laneItems    = 4
	laneReach    = 14
	laneMin      = 2
	laneStride   = 3
	laneBend     = 0.62
	backStreet   = 20.0
	laneTries    = 8
	streetMax    = 12
	streetWalk   = 3
	streetStub   = 1
)

var doorSteps = [][2]int{{2, 0}, {2, -1}, {0, 2}, {-1, 2}, {2, 1}, {1, 2}, {3, 0}, {0, 3}, {2, 2}}

var covers = [...][2]int{{1, 1}, {1, 0}, {0, 1}, {2, 1}, {1, 2}}

var sides = [4][2]int{{0, -1}, {1, 0}, {0, 1}, {-1, 0}}

type Tip struct {
	Story string
	Names string
}

type Lot struct {
	Namespace string
	Node      string
	Order     int
	Seed      view.Point
	Tiles     []view.Point
	Square    []view.Point
}

type item struct {
	isle    int
	kind    string
	w, h    int
	variant string
	object  view.ObjectRef
	label   string
	state   string
	tip     Tip
}

type districtPlan struct {
	ns    string
	uid   types.UID
	isle  int
	pods  int
	order int
	items []item
	keeps map[string]int
}

func (d *districtPlan) itemsOn(isle int) []item {
	var out []item
	for _, it := range d.items {
		if it.isle == isle || it.isle < 0 && d.isle == isle {
			out = append(out, it)
		}
	}
	return out
}

func lotArea(items []item) float64 {
	if len(items) == 0 {
		return 0
	}
	area := float64(squareSide * squareSide)
	for _, it := range items {
		area += float64((it.w + 2) * (it.h + 2))
	}
	return area * itemRoom
}

type landSlot struct {
	at    view.Point
	front int
}

type keepLand struct {
	node   string
	center view.Point
	slots  []landSlot
	lot    map[view.Point]string
	seeds  map[string]view.Point
	orders map[string][]int
}

func gib(q corev1.ResourceList) string {
	v, ok := q[corev1.ResourceStorage]
	if !ok {
		return "unsized"
	}
	return v.String()
}

func planDistricts(s Snapshot, isleOfNode map[string]int) []*districtPlan {
	names := make([]string, 0, len(s.Namespaces))
	for _, ns := range s.Namespaces {
		names = append(names, ns.Name)
	}
	heraldry := view.Heraldry(names)
	byNS := map[string]*districtPlan{}
	var out []*districtPlan
	for _, ns := range byUID(s.Namespaces, func(n corev1.Namespace) types.UID { return n.UID }) {
		d := &districtPlan{ns: ns.Name, uid: ns.UID, order: heraldry[ns.Name], keeps: map[string]int{}}
		byNS[ns.Name] = d
		out = append(out, d)
	}
	court := &districtPlan{ns: courtName, uid: "court", order: len(heraldry) + 1, keeps: map[string]int{}}
	for _, p := range s.Pods {
		if d, ok := byNS[p.Namespace]; ok {
			d.pods++
			if p.Spec.NodeName != "" && !daemonPod(p) {
				d.keeps[p.Spec.NodeName]++
			}
		}
	}
	for _, d := range out {
		counts := map[int]int{}
		for node, n := range d.keeps {
			counts[isleOfNode[node]] += n
		}
		best, most := 0, 0
		for i, n := range counts {
			if n > most || n == most && i < best {
				best, most = i, n
			}
		}
		d.isle = best
	}
	services := append([]corev1.Service(nil), s.Services...)
	sort.Slice(services, func(i, j int) bool {
		return services[i].Namespace+"/"+services[i].Name < services[j].Namespace+"/"+services[j].Name
	})
	for _, svc := range services {
		if d, ok := byNS[svc.Namespace]; ok {
			d.items = append(d.items, depotItem(svc, endpointsOf(s, svc)))
		}
	}
	isleOfZone := map[string]int{}
	for _, n := range s.Nodes {
		if z := n.Labels[LabelZone]; z != "" {
			isleOfZone[z] = isleOfNode[n.Name]
		}
	}
	claims := append([]corev1.PersistentVolumeClaim(nil), s.PVCs...)
	sort.Slice(claims, func(i, j int) bool {
		return claims[i].Namespace+"/"+claims[i].Name < claims[j].Namespace+"/"+claims[j].Name
	})
	for _, pvc := range claims {
		if d, ok := byNS[pvc.Namespace]; ok {
			holder := holderOf(s, pvc)
			it := vaultItem(pvc, holder.Name)
			it.isle = volumeIsle(s, pvc, holder, isleOfNode, isleOfZone)
			d.items = append(d.items, it)
		}
	}
	for _, f := range foreignKinds(s) {
		d, ok := byNS[f.ns]
		if f.ns == "" {
			d, ok = court, true
		}
		if ok {
			d.items = append(d.items, f.item())
		}
	}
	if len(court.items) > 0 {
		out = append(out, court)
	}
	return out
}

func daemonPod(p corev1.Pod) bool {
	ref := metav1.GetControllerOfNoCopy(&p.ObjectMeta)
	return ref != nil && ref.Kind == "DaemonSet"
}

func endpointsOf(s Snapshot, svc corev1.Service) int {
	n := 0
	for _, sl := range s.EndpointSlices {
		if sl.Namespace == svc.Namespace && sl.Labels["kubernetes.io/service-name"] == svc.Name {
			n += len(sl.Endpoints)
		}
	}
	return n
}

func depotItem(svc corev1.Service, ends int) item {
	box := "parcel"
	switch {
	case svc.Spec.ClusterIP == corev1.ClusterIPNone:
		box = "ledger"
	case svc.Spec.Type == corev1.ServiceTypeLoadBalancer || svc.Spec.Type == corev1.ServiceTypeNodePort:
		box = "dispatch"
	case svc.Spec.Type == corev1.ServiceTypeExternalName:
		box = "sealed"
	case hash64(string(svc.UID))%3 == 0:
		box = "crate"
	}
	addr := svc.Spec.ClusterIP
	if addr == corev1.ClusterIPNone {
		addr = "headless"
	}
	return item{isle: -1, kind: view.DepotStructure(box), w: 1, h: 2, variant: view.FaceSE,
		object: view.ObjectRef{Kind: "Service", Namespace: svc.Namespace, Name: svc.Name}, label: "Depot " + svc.Name, state: view.StateReady,
		tip: Tip{Story: fmt.Sprintf("Depot %s: %s on its roll", svc.Name, counted(ends, "hut", "huts")),
			Names: fmt.Sprintf("Service %s/%s (%s, %s): %s", svc.Namespace, svc.Name, svc.Spec.Type, addr, counted(ends, "endpoint", "endpoints"))}}
}

func counted(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}

func holderOf(s Snapshot, pvc corev1.PersistentVolumeClaim) corev1.Pod {
	for _, p := range s.Pods {
		if p.Namespace != pvc.Namespace {
			continue
		}
		for _, v := range p.Spec.Volumes {
			if v.PersistentVolumeClaim != nil && v.PersistentVolumeClaim.ClaimName == pvc.Name {
				return p
			}
		}
	}
	return corev1.Pod{}
}

var zoneKeys = map[string]bool{LabelZone: true, "topology.gke.io/zone": true, "failure-domain.beta.kubernetes.io/zone": true, "topology.ebs.csi.aws.com/zone": true, "topology.disk.csi.azure.com/zone": true}

func volumeIsle(s Snapshot, pvc corev1.PersistentVolumeClaim, holder corev1.Pod, isleOfNode map[string]int, isleOfZone map[string]int) int {
	for _, pv := range s.PVs {
		if pv.Name != pvc.Spec.VolumeName || pv.Spec.NodeAffinity == nil || pv.Spec.NodeAffinity.Required == nil {
			continue
		}
		for _, term := range pv.Spec.NodeAffinity.Required.NodeSelectorTerms {
			for _, e := range term.MatchExpressions {
				if e.Operator != corev1.NodeSelectorOpIn || len(e.Values) == 0 {
					continue
				}
				if i, ok := isleOfZone[e.Values[0]]; ok && zoneKeys[e.Key] {
					return i
				}
				if i, ok := isleOfNode[e.Values[0]]; ok && e.Key == "kubernetes.io/hostname" {
					return i
				}
			}
		}
	}
	if i, ok := isleOfNode[holder.Spec.NodeName]; ok && holder.Spec.NodeName != "" {
		return i
	}
	return -1
}

func vaultItem(pvc corev1.PersistentVolumeClaim, holder string) item {
	class := ""
	if pvc.Spec.StorageClassName != nil {
		class = *pvc.Spec.StorageClassName
	}
	size := gib(pvc.Status.Capacity)
	if size == "unsized" {
		size = gib(pvc.Spec.Resources.Requests)
	}
	state, phase := "bound", string(pvc.Status.Phase)
	if pvc.Status.Phase != corev1.ClaimBound {
		state = "available"
	}
	held, heldBy := "no hut holds it", "no Pod mounts it"
	if holder != "" {
		held, heldBy = "held by "+holder, "mounted by "+holder
	}
	return item{isle: -1, kind: view.StructVault, w: 2, h: 2, variant: state,
		object: view.ObjectRef{Kind: "PersistentVolumeClaim", Namespace: pvc.Namespace, Name: pvc.Name}, label: "Vault " + pvc.Name, state: view.StateReady,
		tip: Tip{Story: fmt.Sprintf("Vault for the Deed %s: %s of the vault kind %s, %s", pvc.Name, size, orNone(class), held),
			Names: fmt.Sprintf("PVC %s/%s %s to PV %s, StorageClass %s, %s, %s", pvc.Namespace, pvc.Name, phase, orNone(pvc.Spec.VolumeName), orNone(class), size, heldBy)}}
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

type foreignKind struct {
	ns, group, kind, version string
	n                        int
}

var helperGroups = map[string]string{"cert-manager.io": "seal-smithy", "acme.cert-manager.io": "seal-smithy", "external-secrets.io": "key-gate"}

func (f foreignKind) item() item {
	kind := KindForeign
	if k, ok := helperGroups[f.group]; ok {
		kind = k
	}
	where, whereNames := "for the whole realm, in no fief", "cluster-scoped"
	if f.ns != "" {
		where, whereNames = "in the fief "+f.ns, "in namespace "+f.ns
	}
	return item{isle: -1, kind: kind, w: 2, h: 2, variant: f.group,
		object: view.ObjectRef{Kind: f.kind, Namespace: f.ns, Name: f.group}, label: "Foreign hall of " + f.kind, state: view.StateReady,
		tip: Tip{Story: fmt.Sprintf("Foreign hall of the %s from %s: %d %s", f.kind, f.group, f.n, where),
			Names: fmt.Sprintf("%d %s (%s/%s) %s", f.n, f.kind, f.group, f.version, whereNames)}}
}

func foreignKinds(s Snapshot) []foreignKind {
	version := map[string]string{}
	for _, c := range s.CRDs {
		version[c.Group+"/"+c.Kind] = c.Version
	}
	count := map[[3]string]int{}
	for _, r := range s.Custom {
		count[[3]string{r.Namespace, r.Group, r.Kind}]++
	}
	out := []foreignKind{}
	for k, n := range count {
		out = append(out, foreignKind{ns: k[0], group: k[1], kind: k[2], version: version[k[1]+"/"+k[2]], n: n})
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.ns != b.ns {
			return a.ns < b.ns
		}
		if a.group != b.group {
			return a.group < b.group
		}
		return a.kind < b.kind
	})
	return out
}

func (w *World) furnish(s Snapshot, groups []*isleGroup, plans []*districtPlan) ([]int, bool) {
	w.Tips = map[string]Tip{}
	w.structureOf = map[string]int{}
	if !anyControlPlane(s) && !w.raiseChancery(s) {
		return []int{0}, true
	}
	if len(harborEntries(s)) > 0 {
		w.Site = w.Site.WithGateHaven()
	}
	w.raiseShrines(s)
	w.raiseHarbor(s)
	var short []int
	for i := range groups {
		if !w.raiseTown(i, plans) {
			short = append(short, i)
		}
	}
	if len(short) > 0 {
		return short, true
	}
	w.measureLands()
	return w.roomless(groups)
}

func anyControlPlane(s Snapshot) bool {
	for _, n := range s.Nodes {
		if controlPlane(n) {
			return true
		}
	}
	return false
}

func (w *World) add(st view.Structure, tip Tip) {
	w.structureOf[st.ID] = len(w.Site.Structures)
	w.Site.Structures = append(w.Site.Structures, st)
	w.Tips[st.ID] = tip
}

func (w *World) index(x, y int) (int, bool) {
	if x < 0 || y < 0 || x >= w.Site.Width || y >= w.Site.Height {
		return 0, false
	}
	return y*w.Site.Width + x, true
}

func (w *World) land(x, y int, region uint8) bool {
	i, ok := w.index(x, y)
	return ok && w.Site.Region[i] == region && buildable(w.Site.Tiles[i])
}

func (w *World) occupied(x, y, wd, ht, margin int) bool {
	for _, st := range w.Site.Structures {
		if x < st.X+st.W+margin && st.X < x+wd+margin && y < st.Y+st.H+margin && st.Y < y+ht+margin {
			return true
		}
	}
	return false
}

func (w *World) free(x, y, wd, ht int, region uint8, margin int) bool {
	for dy := 0; dy < ht; dy++ {
		for dx := 0; dx < wd; dx++ {
			if !w.land(x+dx, y+dy, region) || w.civic(x+dx, y+dy) {
				return false
			}
		}
	}
	return !w.occupied(x, y, wd, ht, margin)
}

func (w *World) civic(x, y int) bool {
	s := w.Site
	if s.HarborGround(x, y, 1) {
		return true
	}
	if abs(x-s.Plaza.X) <= civicClear && abs(y-s.Plaza.Y) <= civicClear {
		return true
	}
	if abs(y-s.Gate.Y) <= 1 && x >= s.Gate.X-2 && x <= s.Plaza.X {
		return true
	}
	return abs(x-s.TowerSpot.X) <= 2 && abs(y-s.TowerSpot.Y) <= 2
}

func (w *World) spotNear(from view.Point, wd, ht int, region uint8, margin int, ok func(x, y int) bool) (view.Point, bool) {
	limit := w.Site.Width + w.Site.Height
	for r := 0; r < limit; r++ {
		var best view.Point
		found, score := false, 0
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if max(abs(dx), abs(dy)) != r {
					continue
				}
				x, y := from.X+dx, from.Y+dy
				if !w.free(x, y, wd, ht, region, margin) || ok != nil && !ok(x, y) {
					continue
				}
				if d := dx*dx + dy*dy; !found || d < score || d == score && (y < best.Y || y == best.Y && x < best.X) {
					best, found, score = view.Point{X: x, Y: y}, true, d
				}
			}
		}
		if found {
			return best, true
		}
	}
	return view.Point{}, false
}

func (w *World) raiseChancery(s Snapshot) bool {
	far, from := w.seaDistance()
	home := w.Site.Islands[0].Center
	best, found, score := view.Point{}, false, math.Inf(1)
	for y := 0; y+3 <= w.Site.Height; y++ {
		for x := 0; x+3 <= w.Site.Width; x++ {
			near, ok := w.openSea(x, y, far)
			if !ok || near < chanceryOff-chanceryDry || near > chanceryFar {
				continue
			}
			if i, _ := w.index(x+1, y+1); from[i] != 1 {
				continue
			}
			lean := float64((x + 1 - home.X) + (y + 1 - home.Y))
			if sc := lean + 4*math.Abs(float64(near-chanceryOff+chanceryDry)); sc < score {
				best, found, score = view.Point{X: x, Y: y}, true, sc
			}
		}
	}
	if !found {
		return false
	}
	server := strings.TrimRight(s.Context.Server, "/")
	w.Chancery = fmt.Sprintf("%s %s", orNone(server), orNone(s.Version))
	w.add(view.Structure{ID: "chancery", Kind: KindSeaChancery, X: best.X, Y: best.Y, W: 3, H: 3, State: view.StateReady, Progress: 1, Label: "The Citadel across the sea",
		Object: view.ObjectRef{Kind: "APIServer", Name: orNone(server)}},
		Tip{Story: fmt.Sprintf("The Citadel across the sea: its keepers are hidden, and only its Chancery answers, at %s (%s)", orNone(server), orNone(s.Version)),
			Names: fmt.Sprintf("kube-apiserver at %s (%s): a managed control plane, no control-plane node in this cluster", orNone(server), orNone(s.Version))})
	return true
}

func (w *World) openSea(x, y int, far []int) (int, bool) {
	near := math.MaxInt32
	for dy := -chanceryDry; dy < 3+chanceryDry; dy++ {
		for dx := -chanceryDry; dx < 3+chanceryDry; dx++ {
			i, in := w.index(x+dx, y+dy)
			if !in || w.Site.Tiles[i] != view.TileWater {
				return 0, false
			}
			near = min(near, far[i])
		}
	}
	return near, true
}

func (w *World) seaDistance() ([]int, []uint8) {
	n := len(w.Site.Tiles)
	far, from := make([]int, n), make([]uint8, n)
	queue := []int{}
	for i := range far {
		far[i] = -1
		if w.Site.Region[i] != 0 {
			far[i], from[i] = 0, w.Site.Region[i]
			queue = append(queue, i)
		}
	}
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		x, y := i%w.Site.Width, i/w.Site.Width
		for _, d := range sides {
			j, ok := w.index(x+d[0], y+d[1])
			if ok && far[j] < 0 {
				far[j], from[j] = far[i]+1, from[i]
				queue = append(queue, j)
			}
		}
	}
	return far, from
}

func (w *World) keepOf(node string) (KeepPlot, bool) {
	for _, k := range w.Keeps {
		if k.Node == node {
			return k, true
		}
	}
	return KeepPlot{}, false
}

func (w *World) raiseShrines(s Snapshot) {
	byNode := map[string][]corev1.Pod{}
	owner := map[types.UID]string{}
	for _, p := range s.Pods {
		if daemonPod(p) && p.Spec.NodeName != "" {
			byNode[p.Spec.NodeName] = append(byNode[p.Spec.NodeName], p)
			owner[p.UID] = metav1.GetControllerOfNoCopy(&p.ObjectMeta).Name
		}
	}
	w.shrineOf = map[types.UID]view.Point{}
	n := 0
	for _, k := range w.Keeps {
		pods := byNode[k.Node]
		sort.Slice(pods, func(i, j int) bool {
			a, b := owner[pods[i].UID], owner[pods[j].UID]
			if a != b {
				return a < b
			}
			return pods[i].UID < pods[j].UID
		})
		spots := w.shrineSpots(k, w.regionOfKeep(k), len(pods))
		for i, p := range pods {
			if i >= len(spots) {
				break
			}
			n++
			at, ds := spots[i], owner[p.UID]
			state := view.StateReady
			if !podReady(p) {
				state = view.StateDark
			}
			w.add(view.Structure{ID: fmt.Sprintf("shrine-%d", n), Kind: KindShrine, X: at.X, Y: at.Y, W: 1, H: 1, State: state, Progress: 1, Label: "Shrine of " + ds,
				Object: view.ObjectRef{Kind: "Pod", Namespace: p.Namespace, Name: p.Name}},
				Tip{Story: fmt.Sprintf("Shrine of %s at %s: one stands at every Keep it serves", ds, k.Node),
					Names: fmt.Sprintf("DaemonSet %s/%s: Pod %s on node %s", p.Namespace, ds, p.Name, k.Node)})
			w.shrineOf[p.UID] = at
		}
	}
}

func podReady(p corev1.Pod) bool {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

func (w *World) regionOfKeep(k KeepPlot) uint8 {
	i, _ := w.index(k.X+1, k.Y+1)
	return w.Site.Region[i]
}

func (w *World) shrineSpots(k KeepPlot, region uint8, want int) []view.Point {
	if want == 0 {
		return nil
	}
	cx, cy := k.X+1, k.Y+1
	var out []view.Point
	for ring := shrineRing; ring < shrineRing+2*shrineRings && len(out) < want; ring += 2 {
		type cand struct {
			at    view.Point
			flank float64
		}
		var cands []cand
		for dy := -ring; dy <= ring; dy++ {
			for dx := -ring; dx <= ring; dx++ {
				if max(abs(dx), abs(dy)) != ring || dx+dy > 0 && abs(dx-dy) < ring {
					continue
				}
				a := math.Atan2(float64(dy), float64(dx))
				flank := min(angleGap(a, math.Atan2(1, -1)), angleGap(a, math.Atan2(-1, 1)))
				cands = append(cands, cand{view.Point{X: cx + dx, Y: cy + dy}, flank})
			}
		}
		sort.Slice(cands, func(i, j int) bool {
			if cands[i].flank != cands[j].flank {
				return cands[i].flank < cands[j].flank
			}
			if cands[i].at.Y != cands[j].at.Y {
				return cands[i].at.Y < cands[j].at.Y
			}
			return cands[i].at.X < cands[j].at.X
		})
		for _, c := range cands {
			if len(out) < want && w.free(c.at.X, c.at.Y, 1, 1, region, 1) && !nearAny(out, c.at) {
				out = append(out, c.at)
			}
		}
	}
	return out
}

func nearAny(spots []view.Point, p view.Point) bool {
	for _, q := range spots {
		if abs(p.X-q.X) <= 1 && abs(p.Y-q.Y) <= 1 {
			return true
		}
	}
	return false
}

func angleGap(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 2*math.Pi)
	return min(d, 2*math.Pi-d)
}

type harborEntry struct {
	label  string
	object view.ObjectRef
	tip    Tip
	addrs  []string
	signs  []item
}

func lbAddresses(ing []corev1.LoadBalancerIngress) []string {
	var out []string
	for _, a := range ing {
		if a.IP != "" {
			out = append(out, a.IP)
		}
		if a.Hostname != "" {
			out = append(out, a.Hostname)
		}
	}
	return out
}

func harborEntries(s Snapshot) []*harborEntry {
	var out []*harborEntry
	services := append([]corev1.Service(nil), s.Services...)
	sort.Slice(services, func(i, j int) bool {
		return services[i].Namespace+"/"+services[i].Name < services[j].Namespace+"/"+services[j].Name
	})
	for _, svc := range services {
		if svc.Spec.Type != corev1.ServiceTypeLoadBalancer && svc.Spec.Type != corev1.ServiceTypeNodePort {
			continue
		}
		addrs := lbAddresses(svc.Status.LoadBalancer.Ingress)
		at := strings.Join(addrs, ", ")
		if svc.Spec.Type == corev1.ServiceTypeNodePort {
			ports := []string{}
			for _, p := range svc.Spec.Ports {
				ports = append(ports, fmt.Sprint(p.NodePort))
			}
			at = "every Keep's door " + strings.Join(ports, ", ")
		}
		out = append(out, &harborEntry{label: "Harbor gate " + svc.Name, addrs: addrs,
			object: view.ObjectRef{Kind: "Service", Namespace: svc.Namespace, Name: svc.Name},
			tip: Tip{Story: fmt.Sprintf("Harbor gate %s of the fief %s: the way in from the sea at %s", svc.Name, svc.Namespace, orNone(at)),
				Names: fmt.Sprintf("Service %s/%s, type %s, at %s", svc.Namespace, svc.Name, svc.Spec.Type, orNone(at))}})
	}
	gateways := append([]Gateway(nil), s.Gateways...)
	sort.Slice(gateways, func(i, j int) bool {
		return gateways[i].Namespace+"/"+gateways[i].Name < gateways[j].Namespace+"/"+gateways[j].Name
	})
	for _, g := range gateways {
		listeners := []string{}
		for _, l := range g.Spec.Listeners {
			listeners = append(listeners, fmt.Sprintf("%s %d", l.Protocol, l.Port))
		}
		out = append(out, &harborEntry{label: "Harbor gate " + g.Name, object: view.ObjectRef{Kind: "Gateway", Namespace: g.Namespace, Name: g.Name},
			tip: Tip{Story: fmt.Sprintf("Harbor gate %s of the fief %s, kept by %s: %s", g.Name, g.Namespace, orNone(g.Spec.GatewayClassName), orNone(strings.Join(listeners, ", "))),
				Names: fmt.Sprintf("Gateway %s/%s, GatewayClass %s, listeners %s", g.Namespace, g.Name, orNone(g.Spec.GatewayClassName), orNone(strings.Join(listeners, ", ")))}})
	}
	plain := map[string]*harborEntry{}
	for _, ing := range ingressesOf(s) {
		gate := gateFor(out, ing.addrs)
		if gate == nil {
			class := orNone(ing.class)
			if plain[class] == nil {
				plain[class] = &harborEntry{label: "Harbor gate " + class, object: view.ObjectRef{Kind: "IngressClass", Name: class},
					tip: Tip{Story: fmt.Sprintf("Plain harbor gate %s: its signposts have no harbor address yet", class),
						Names: fmt.Sprintf("IngressClass %s: Ingresses with no address that matches a LoadBalancer Service", class)}}
			}
			gate = plain[class]
		}
		gate.signs = append(gate.signs, ing.sign())
	}
	classes := make([]string, 0, len(plain))
	for c := range plain {
		classes = append(classes, c)
	}
	sort.Strings(classes)
	for _, c := range classes {
		out = append(out, plain[c])
	}
	return out
}

func gateFor(gates []*harborEntry, addrs []string) *harborEntry {
	for _, g := range gates {
		for _, a := range g.addrs {
			for _, b := range addrs {
				if a == b {
					return g
				}
			}
		}
	}
	return nil
}

type networkingIngress struct {
	ns, name, class string
	hosts, targets  []string
	addrs           []string
}

func ingressesOf(s Snapshot) []networkingIngress {
	var out []networkingIngress
	for _, ing := range s.Ingresses {
		n := networkingIngress{ns: ing.Namespace, name: ing.Name}
		if ing.Spec.IngressClassName != nil {
			n.class = *ing.Spec.IngressClassName
		}
		seen := map[string]bool{}
		note := func(svc string) {
			if svc != "" && !seen[svc] {
				seen[svc] = true
				n.targets = append(n.targets, svc)
			}
		}
		if b := ing.Spec.DefaultBackend; b != nil && b.Service != nil {
			note(b.Service.Name)
		}
		for _, r := range ing.Spec.Rules {
			if r.Host != "" {
				n.hosts = append(n.hosts, r.Host)
			}
			if r.HTTP == nil {
				continue
			}
			for _, p := range r.HTTP.Paths {
				if p.Backend.Service != nil {
					note(p.Backend.Service.Name)
				}
			}
		}
		for _, a := range ing.Status.LoadBalancer.Ingress {
			if a.IP != "" {
				n.addrs = append(n.addrs, a.IP)
			}
			if a.Hostname != "" {
				n.addrs = append(n.addrs, a.Hostname)
			}
		}
		out = append(out, n)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ns+"/"+out[i].name < out[j].ns+"/"+out[j].name })
	return out
}

func (n networkingIngress) sign() item {
	hosts := orNone(strings.Join(n.hosts, ", "))
	return item{isle: -1, kind: KindSignpost, w: 1, h: 1, object: view.ObjectRef{Kind: "Ingress", Namespace: n.ns, Name: n.name}, label: "Signpost " + n.name, state: view.StateReady,
		tip: Tip{Story: fmt.Sprintf("Signpost %s of the fief %s: %s, pointing to %s", n.name, n.ns, hosts, storyDepots(n.targets)),
			Names: fmt.Sprintf("Ingress %s/%s (class %s): %s -> %s", n.ns, n.name, orNone(n.class), hosts, namedServices(n.ns, n.targets))}}
}

func storyDepots(names []string) string {
	switch len(names) {
	case 0:
		return "no depot"
	case 1:
		return "the " + names[0] + " depot"
	}
	return "the " + strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1] + " depots"
}

func namedServices(ns string, names []string) string {
	if len(names) == 0 {
		return "no Service"
	}
	keys := make([]string, len(names))
	for i, name := range names {
		keys[i] = ns + "/" + name
	}
	if len(keys) == 1 {
		return "Service " + keys[0]
	}
	return "Services " + strings.Join(keys, ", ")
}

func (w *World) raiseHarbor(s Snapshot) {
	entries := harborEntries(s)
	w.Site.NoHarbor = len(entries) == 0
	gate := w.Site.Gate
	outside := func(x, y int) bool { return !w.nearHaven(x, y) }
	for i, e := range entries {
		side := -3
		if i%2 == 1 {
			side = 2
		}
		target := view.Point{X: gate.X + 2 + gateSpacing*(i/2)*2, Y: gate.Y + side}
		at, ok := w.spotNear(target, 2, 2, 1, 1, outside)
		if !ok {
			continue
		}
		w.Gates++
		w.add(view.Structure{ID: fmt.Sprintf("gate-%d", w.Gates), Kind: KindHarborGate, X: at.X, Y: at.Y, W: 2, H: 2, State: view.StateReady, Progress: 1, Label: e.label, Object: e.object}, e.tip)
		for _, sign := range e.signs {
			from := view.Point{X: at.X, Y: at.Y + 3}
			if side < 0 {
				from = view.Point{X: at.X, Y: at.Y - 2}
			}
			sp, ok := w.spotNear(from, 1, 1, 1, 1, outside)
			if !ok {
				continue
			}
			w.Signposts++
			w.add(view.Structure{ID: fmt.Sprintf("signpost-%d", w.Signposts), Kind: KindSignpost, X: sp.X, Y: sp.Y, W: 1, H: 1, State: view.StateReady, Progress: 1,
				Label: sign.label, Object: sign.object}, sign.tip)
		}
	}
}

func (w *World) nearHaven(x, y int) bool {
	g := w.Site.Gate
	return abs(x-g.X) <= 2 && abs(y-g.Y) <= harborClear || x < g.X
}

type smooth struct {
	seed uint64
}

func (n smooth) corner(x, y int) float64 {
	return float64(hash64(fmt.Sprintf("%d/%d/%d", n.seed, x, y))%1024) / 1023
}

func (n smooth) at(x, y int) float64 {
	gx, gy := float64(x)/noiseCell, float64(y)/noiseCell
	x0, y0 := int(math.Floor(gx)), int(math.Floor(gy))
	tx, ty := gx-float64(x0), gy-float64(y0)
	tx, ty = tx*tx*(3-2*tx), ty*ty*(3-2*ty)
	a, b := n.corner(x0, y0), n.corner(x0+1, y0)
	c, d := n.corner(x0, y0+1), n.corner(x0+1, y0+1)
	return a + (b-a)*tx + (c-a)*ty + (a-b-c+d)*tx*ty
}

type routeNode struct {
	i    int
	cost float64
}

type routeQueue []routeNode

func (q routeQueue) Len() int { return len(q) }
func (q routeQueue) Less(i, j int) bool {
	return q[i].cost < q[j].cost || q[i].cost == q[j].cost && q[i].i < q[j].i
}
func (q routeQueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }
func (q *routeQueue) Push(x any)   { *q = append(*q, x.(routeNode)) }
func (q *routeQueue) Pop() any {
	old := *q
	n := old[len(old)-1]
	*q = old[:len(old)-1]
	return n
}

func (w *World) stepCost(i int) float64 {
	switch w.Site.Tiles[i] {
	case view.TileRoad:
		return 0.4
	case view.TilePlaza:
		return 1
	case view.TileForest:
		return 4
	case view.TileRock:
		return 6
	case view.TileSand:
		return 3
	case view.TileWater, view.TileShallows:
		return -1
	}
	return 2
}

func (w *World) route(from view.Point, goal func(i int) bool, blocked []bool, wiggle smooth) []view.Point {
	start, ok := w.index(from.X, from.Y)
	if !ok {
		return nil
	}
	width := w.Site.Width
	dist := make([]float64, len(w.Site.Tiles))
	for i := range dist {
		dist[i] = math.Inf(1)
	}
	prev := make([]int, len(w.Site.Tiles))
	dist[start] = 0
	q := &routeQueue{{i: start}}
	for q.Len() > 0 {
		n := heap.Pop(q).(routeNode)
		if n.cost > dist[n.i] {
			continue
		}
		if goal(n.i) {
			var out []view.Point
			for i := n.i; i != start; i = prev[i] {
				out = append(out, view.Point{X: i % width, Y: i / width})
			}
			return append(out, from)
		}
		x, y := n.i%width, n.i/width
		for _, d := range sides {
			j, ok := w.index(x+d[0], y+d[1])
			if !ok || blocked[j] {
				continue
			}
			step := w.stepCost(j)
			if step < 0 {
				continue
			}
			if w.Site.Tiles[j] != view.TileRoad {
				step *= 1 + roadNoise*wiggle.at(x+d[0], y+d[1])
			}
			if c := n.cost + step; c < dist[j] {
				dist[j], prev[j] = c, n.i
				heap.Push(q, routeNode{i: j, cost: c})
			}
		}
	}
	return nil
}

func (w *World) structureMask(margin int) []bool {
	out := make([]bool, len(w.Site.Tiles))
	for _, st := range w.Site.Structures {
		for y := st.Y - margin; y < st.Y+st.H+margin; y++ {
			for x := st.X - margin; x < st.X+st.W+margin; x++ {
				if i, ok := w.index(x, y); ok {
					out[i] = true
				}
			}
		}
	}
	return out
}

func (w *World) pave(path []view.Point) {
	for _, p := range path {
		i, ok := w.index(p.X, p.Y)
		if ok && w.Site.Tiles[i] != view.TilePlaza && w.Site.Tiles[i] != view.TileWater && w.Site.Tiles[i] != view.TileShallows {
			w.Site.Tiles[i] = view.TileRoad
		}
	}
}

type townState struct {
	tiles     []view.TileKind
	elevation []uint8
	structs   int
	lots      int
}

func (w *World) save() townState {
	return townState{tiles: append([]view.TileKind(nil), w.Site.Tiles...), elevation: append([]uint8(nil), w.Site.Elevation...), structs: len(w.Site.Structures), lots: len(w.Lots)}
}

func (w *World) restore(t townState) {
	copy(w.Site.Tiles, t.tiles)
	copy(w.Site.Elevation, t.elevation)
	for _, st := range w.Site.Structures[t.structs:] {
		delete(w.Tips, st.ID)
		delete(w.structureOf, st.ID)
	}
	w.Site.Structures = w.Site.Structures[:t.structs]
	w.Lots = w.Lots[:t.lots]
}

func (w *World) raiseTown(isle int, plans []*districtPlan) bool {
	region := uint8(isle + 1)
	var keeps []KeepPlot
	for _, k := range w.Keeps {
		if k.Isle == w.Isles[isle].ID {
			keeps = append(keeps, k)
		}
	}
	var mine []*districtPlan
	for _, d := range plans {
		if len(d.itemsOn(isle)) > 0 || d.onIsle(w, isle) {
			mine = append(mine, d)
		}
	}
	if len(keeps) == 0 && len(mine) == 0 {
		return true
	}
	start := w.save()
	room := 1.0
	for try := 0; try < lotTries; try++ {
		if w.layTown(isle, region, keeps, mine, room, try < lotTries-1) {
			return true
		}
		w.restore(start)
		room *= lotGrowth
	}
	return false
}

func (d *districtPlan) onIsle(w *World, isle int) bool {
	for node := range d.keeps {
		if k, ok := w.keepOf(node); ok && k.Isle == w.Isles[isle].ID {
			return true
		}
	}
	return false
}

func (w *World) layTown(isle int, region uint8, keeps []KeepPlot, plans []*districtPlan, room float64, strict bool) bool {
	wiggle := smooth{seed: w.Seed ^ uint64(isle+1)}
	market := w.Site.Plaza
	if isle > 0 {
		market = w.townCentre(isle, region, keeps)
	}
	blocked := w.structureMask(0)
	for _, k := range keeps {
		gate := view.Point{X: k.X + 4, Y: k.Y + 4}
		w.pave(w.route(market, func(i int) bool {
			x, y := i%w.Site.Width, i/w.Site.Width
			return abs(x-gate.X) <= 1 && abs(y-gate.Y) <= 1
		}, blocked, wiggle))
	}
	owner := w.Site.KeepLand()
	seeds, ok := w.lotSeeds(isle, region, market, keeps, plans, owner, room)
	if !ok {
		return false
	}
	lots := w.growLots(region, seeds, owner, room, wiggle)
	for _, p := range lots {
		if p.Node != "" {
			w.laneQuarter(p, wiggle)
			continue
		}
		if !w.furnishLot(p, wiggle, strict) {
			return false
		}
	}
	for _, p := range lots {
		w.Lots = append(w.Lots, p.Lot)
	}
	return true
}

func (w *World) townCentre(isle int, region uint8, keeps []KeepPlot) view.Point {
	sx, sy := 0, 0
	for _, k := range keeps {
		sx, sy = sx+k.X+1, sy+k.Y+1
	}
	target := w.Site.Islands[isle].Center
	if len(keeps) > 0 {
		target = view.Point{X: sx / len(keeps), Y: sy / len(keeps)}
	}
	owner := w.Site.KeepLand()
	at, ok := w.spotNear(target, squareSide, squareSide, region, 1, func(x, y int) bool {
		for dy := -1; dy <= squareSide; dy++ {
			for dx := -1; dx <= squareSide; dx++ {
				if i, in := w.index(x+dx, y+dy); in && owner[i] != 0 {
					return false
				}
			}
		}
		return true
	})
	if !ok {
		return target
	}
	for dy := 0; dy < squareSide; dy++ {
		for dx := 0; dx < squareSide; dx++ {
			i, _ := w.index(at.X+dx, at.Y+dy)
			w.Site.Tiles[i] = view.TilePlaza
		}
	}
	return view.Point{X: at.X + 1, Y: at.Y + 1}
}

type growingLot struct {
	Lot
	plan   *districtPlan
	items  []item
	budget int
	keep   int16
}

func (w *World) streetFront(region uint8, owner []int16) []view.Point {
	taken := w.structureMask(1)
	var out []view.Point
	for i, t := range w.Site.Tiles {
		x, y := i%w.Site.Width, i/w.Site.Width
		if t == view.TileRoad || w.Site.Region[i] != region || owner[i] != 0 || taken[i] || !buildable(t) || w.civic(x, y) {
			continue
		}
		for _, d := range sides {
			if w.open(x+d[0], y+d[1]) {
				out = append(out, view.Point{X: x, Y: y})
				break
			}
		}
	}
	return out
}

func (w *World) lotSeeds(isle int, region uint8, market view.Point, keeps []KeepPlot, plans []*districtPlan, owner []int16, room float64) ([]*growingLot, bool) {
	keepIndex := map[string]int16{}
	for i, st := range w.Site.Structures {
		if st.Kind == view.StructKeep {
			keepIndex[st.Object.Name] = int16(i + 1)
		}
	}
	taken := w.structureMask(1)
	frontage := w.streetFront(region, owner)
	ordered := append([]*districtPlan(nil), plans...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := len(ordered[i].itemsOn(isle)), len(ordered[j].itemsOn(isle))
		if a != b {
			return a > b
		}
		return ordered[i].ns < ordered[j].ns
	})
	var out []*growingLot
	var placed []lotDisc
	rank := 0
	for _, d := range ordered {
		items := d.itemsOn(isle)
		if len(items) == 0 {
			continue
		}
		target := w.civicTarget(d, market, keeps)
		want := 4.0 + float64(rank*marketStride)
		rank++
		need := lotArea(items) * room
		disc := lotDisc{r: math.Sqrt(need / math.Pi)}
		roomy := func(p view.Point) bool {
			disc.at = p
			return disc.clearOf(placed) && w.freeAround(p, region, owner, taken, placed, int(math.Ceil(need)))
		}
		best, found, score := view.Point{}, false, math.Inf(1)
		for _, p := range frontage {
			if !roomy(p) {
				continue
			}
			dm := math.Hypot(float64(p.X-market.X), float64(p.Y-market.Y))
			dt := math.Hypot(float64(p.X-target.X), float64(p.Y-target.Y))
			sc := math.Abs(dm-want) + dt
			if d.ns == view.CitadelFief {
				sc = dt
			}
			if sc < score {
				best, found, score = p, true, sc
			}
		}
		if !found {
			best, found = w.nearestTile(target, func(i int) bool {
				return w.districtLand(i, region, owner, taken) && roomy(view.Point{X: i % w.Site.Width, Y: i / w.Site.Width})
			})
		}
		if !found {
			return nil, false
		}
		placed = append(placed, lotDisc{at: best, r: disc.r})
		out = append(out, &growingLot{Lot: Lot{Namespace: d.ns, Order: d.order, Seed: best}, plan: d, items: items, budget: int(math.Ceil(lotArea(items)))})
	}
	for _, k := range keeps {
		land := keepIndex[k.Node]
		center := view.Point{X: k.X + 1, Y: k.Y + 1}
		reach := float64(max(k.Reach, baseReach))
		for _, d := range plans {
			if d.keeps[k.Node] == 0 {
				continue
			}
			toward := market
			for _, g := range out {
				if g.Namespace == d.ns && g.Node == "" {
					toward = g.Seed
				}
			}
			a := math.Atan2(float64(toward.Y-center.Y), float64(toward.X-center.X))
			if toward == center {
				h := fnv.New32a()
				h.Write([]byte(d.ns))
				a = float64(h.Sum32()%360) * math.Pi / 180
			}
			r := max(reach*anchorShare, anchorMin)
			aim := view.Point{X: center.X + int(math.Round(r*math.Cos(a))), Y: center.Y + int(math.Round(r*math.Sin(a)))}
			seed, ok := w.nearestTile(aim, func(i int) bool {
				return owner[i] == land && !taken[i] && w.Site.Tiles[i] != view.TileRoad && buildable(w.Site.Tiles[i])
			})
			if !ok {
				continue
			}
			out = append(out, &growingLot{Lot: Lot{Namespace: d.ns, Node: k.Node, Order: d.order, Seed: seed}, plan: d, budget: d.keeps[k.Node] * quarterRoom, keep: land})
		}
	}
	return out, true
}

type lotDisc struct {
	at view.Point
	r  float64
}

func (d lotDisc) holds(p view.Point) bool {
	return math.Hypot(float64(p.X-d.at.X), float64(p.Y-d.at.Y)) < d.r
}

func (d lotDisc) clearOf(placed []lotDisc) bool {
	for _, q := range placed {
		if math.Hypot(float64(d.at.X-q.at.X), float64(d.at.Y-q.at.Y)) < max(seedSpacing, d.r+q.r) {
			return false
		}
	}
	return true
}

func (w *World) districtLand(i int, region uint8, owner []int16, taken []bool) bool {
	t := w.Site.Tiles[i]
	return w.Site.Region[i] == region && owner[i] == 0 && !taken[i] && t != view.TileRoad && t != view.TilePlaza && buildable(t) && !w.civic(i%w.Site.Width, i/w.Site.Width)
}

func (w *World) freeAround(from view.Point, region uint8, owner []int16, taken []bool, placed []lotDisc, need int) bool {
	start, ok := w.index(from.X, from.Y)
	if !ok {
		return false
	}
	free := func(i int) bool {
		p := view.Point{X: i % w.Site.Width, Y: i / w.Site.Width}
		if !w.districtLand(i, region, owner, taken) {
			return false
		}
		for _, q := range placed {
			if q.holds(p) {
				return false
			}
		}
		return true
	}
	if !free(start) {
		return false
	}
	seen := map[int]bool{start: true}
	queue := []int{start}
	for len(queue) > 0 && len(seen) < need {
		i := queue[0]
		queue = queue[1:]
		x, y := i%w.Site.Width, i/w.Site.Width
		for _, d := range sides {
			if j, in := w.index(x+d[0], y+d[1]); in && !seen[j] && free(j) {
				seen[j] = true
				queue = append(queue, j)
			}
		}
	}
	return len(seen) >= need
}

func (w *World) civicTarget(d *districtPlan, market view.Point, keeps []KeepPlot) view.Point {
	if d.ns == courtName {
		return market
	}
	if d.ns == view.CitadelFief {
		best, most := KeepPlot{}, -1
		for _, k := range keeps {
			if k.Pods > most {
				best, most = k, k.Pods
			}
		}
		if most >= 0 {
			return view.Point{X: best.X + 1, Y: best.Y + 1}
		}
	}
	sx, sy, n := 0, 0, 0
	nodes := make([]string, 0, len(d.keeps))
	for node := range d.keeps {
		nodes = append(nodes, node)
	}
	sort.Strings(nodes)
	for _, node := range nodes {
		if k, ok := w.keepOf(node); ok {
			c := d.keeps[node]
			sx, sy, n = sx+(k.X+1)*c, sy+(k.Y+1)*c, n+c
		}
	}
	if n == 0 {
		return market
	}
	return view.Point{X: (sx/n + market.X) / 2, Y: (sy/n + market.Y) / 2}
}

func (w *World) nearestTile(from view.Point, ok func(i int) bool) (view.Point, bool) {
	limit := w.Site.Width + w.Site.Height
	for r := 0; r < limit; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if max(abs(dx), abs(dy)) != r {
					continue
				}
				if i, in := w.index(from.X+dx, from.Y+dy); in && ok(i) {
					return view.Point{X: from.X + dx, Y: from.Y + dy}, true
				}
			}
		}
	}
	return view.Point{}, false
}

func (w *World) growLots(region uint8, seeds []*growingLot, owner []int16, room float64, wiggle smooth) []*growingLot {
	if len(seeds) == 0 {
		return nil
	}
	claimed := make([]int, len(w.Site.Tiles))
	taken := w.structureMask(1)
	for i := range claimed {
		claimed[i] = -1
	}
	allowed := func(g *growingLot, i int) bool {
		x, y := i%w.Site.Width, i/w.Site.Width
		t := w.Site.Tiles[i]
		if w.Site.Region[i] != region || taken[i] || t == view.TileRoad || t == view.TilePlaza || !buildable(t) || w.civic(x, y) {
			return false
		}
		if g.keep != 0 {
			return owner[i] == g.keep
		}
		return owner[i] == 0
	}
	noise := make([]smooth, len(seeds))
	left := make([]int, len(seeds))
	q := &routeQueue{}
	for k, g := range seeds {
		noise[k] = smooth{seed: wiggle.seed ^ hash64(g.Namespace+"/"+g.Node)}
		left[k] = int(math.Ceil(float64(g.budget) * room))
		if i, ok := w.index(g.Seed.X, g.Seed.Y); ok {
			heap.Push(q, routeNode{i: i*len(seeds) + k})
		}
	}
	for q.Len() > 0 {
		n := heap.Pop(q).(routeNode)
		i, k := n.i/len(seeds), n.i%len(seeds)
		g := seeds[k]
		if left[k] <= 0 || claimed[i] >= 0 || !allowed(g, i) {
			continue
		}
		claimed[i] = k
		left[k]--
		x, y := i%w.Site.Width, i/w.Site.Width
		g.Tiles = append(g.Tiles, view.Point{X: x, Y: y})
		for _, d := range sides {
			j, ok := w.index(x+d[0], y+d[1])
			if !ok || claimed[j] >= 0 {
				continue
			}
			if step := w.stepCost(j); step >= 0 {
				heap.Push(q, routeNode{i: j*len(seeds) + k, cost: n.cost + step*(1+lotNoise*noise[k].at(x+d[0], y+d[1]))})
			}
		}
	}
	for _, g := range seeds {
		sortPoints(g.Tiles)
	}
	return seeds
}

func sortPoints(ps []view.Point) {
	sort.Slice(ps, func(a, b int) bool {
		if ps[a].Y != ps[b].Y {
			return ps[a].Y < ps[b].Y
		}
		return ps[a].X < ps[b].X
	})
}

func (w *World) furnishLot(g *growingLot, wiggle smooth, strict bool) bool {
	in := map[view.Point]bool{}
	for _, p := range g.Tiles {
		in[p] = true
	}
	from := g.Seed
	if len(g.items) >= squareItems {
		if sq, ok := w.squareIn(g, in); ok {
			g.Square = sq
			for _, p := range sq {
				i, _ := w.index(p.X, p.Y)
				w.Site.Tiles[i] = view.TilePlaza
			}
			from = sq[len(sq)/2]
			w.laneOut(g, in, sq, wiggle)
		}
	}
	w.joinRoads(from, in, wiggle)
	w.streets(in)
	mine := map[string]bool{}
	for _, it := range g.items {
		id, fronts, ok := w.placeItem(g, in, it, mine, false)
		for tries := 0; ok && !fronts && tries < laneTries && w.extendLane(g, in); tries++ {
			id, fronts, ok = w.placeItem(g, in, it, mine, false)
		}
		if !ok || strict && !fronts {
			return false
		}
		id, _, _ = w.placeItem(g, in, it, mine, true)
		mine[id] = true
	}
	return true
}

func (w *World) extendLane(g *growingLot, in map[view.Point]bool) bool {
	best, dir, run, score := view.Point{}, view.Point{}, 0, math.Inf(1)
	for _, p := range g.Tiles {
		if w.open(p.X, p.Y) || w.occupied(p.X, p.Y, 1, 1, 0) {
			continue
		}
		for _, d := range sides {
			if !w.open(p.X-d[0], p.Y-d[1]) {
				continue
			}
			n := 0
			for q := p; in[q] && !w.open(q.X, q.Y) && !w.occupied(q.X, q.Y, 1, 1, 0) && n < laneReach; q = (view.Point{X: q.X + d[0], Y: q.Y + d[1]}) {
				n++
			}
			sc := math.Hypot(float64(p.X-g.Seed.X), float64(p.Y-g.Seed.Y)) - float64(n)
			if n >= laneMin && sc < score {
				best, dir, run, score = p, view.Point{X: d[0], Y: d[1]}, n, sc
			}
		}
	}
	if run == 0 {
		return false
	}
	var path []view.Point
	for q, k := best, 0; k < run; k++ {
		path = append(path, q)
		q = view.Point{X: q.X + dir.X, Y: q.Y + dir.Y}
	}
	w.pave(path)
	return true
}

func (w *World) laneOut(g *growingLot, in map[view.Point]bool, sq []view.Point, wiggle smooth) {
	lo, hi := sq[0], sq[len(sq)-1]
	mid := view.Point{X: (lo.X + hi.X) / 2, Y: (lo.Y + hi.Y) / 2}
	starts := []struct{ at, dir view.Point }{
		{view.Point{X: mid.X, Y: lo.Y - 1}, view.Point{X: 0, Y: -1}},
		{view.Point{X: hi.X + 1, Y: mid.Y}, view.Point{X: 1, Y: 0}},
		{view.Point{X: mid.X, Y: hi.Y + 1}, view.Point{X: 0, Y: 1}},
		{view.Point{X: lo.X - 1, Y: mid.Y}, view.Point{X: -1, Y: 0}},
	}
	lanes := min(len(starts), 1+len(g.items)/laneItems)
	first := int(hash64(g.Namespace) % uint64(len(starts)))
	for k := 0; k < lanes; k++ {
		st := starts[(first+k)%len(starts)]
		w.lane(st.at, st.dir, in, wiggle)
	}
}

func (w *World) lane(at, dir view.Point, in map[view.Point]bool, wiggle smooth) {
	side := view.Point{X: dir.Y, Y: dir.X}
	var path []view.Point
	for step := 0; step < laneReach && in[at] && !w.open(at.X, at.Y); step++ {
		path = append(path, at)
		next := view.Point{X: at.X + dir.X, Y: at.Y + dir.Y}
		if n := wiggle.at(at.X*7, at.Y*7); n > laneBend && step%laneStride == laneStride-1 {
			shift := view.Point{X: at.X + side.X, Y: at.Y + side.Y}
			if n > (1+laneBend)/2 {
				shift = view.Point{X: at.X - side.X, Y: at.Y - side.Y}
			}
			if in[shift] && !w.open(shift.X, shift.Y) {
				path = append(path, shift)
				next = view.Point{X: shift.X + dir.X, Y: shift.Y + dir.Y}
			}
		}
		at = next
	}
	if len(path) >= laneMin {
		w.pave(path)
	}
}

func (w *World) joinRoads(from view.Point, in map[view.Point]bool, wiggle smooth) {
	start, ok := w.index(from.X, from.Y)
	if !ok {
		return
	}
	reach := map[int]bool{start: true}
	queue := []int{start}
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		x, y := i%w.Site.Width, i/w.Site.Width
		for _, d := range sides {
			j, ok := w.index(x+d[0], y+d[1])
			if !ok || reach[j] || !w.open(x+d[0], y+d[1]) {
				continue
			}
			reach[j] = true
			queue = append(queue, j)
		}
	}
	blocked := w.structureMask(0)
	w.pave(w.route(from, func(i int) bool { return !reach[i] && w.Site.Tiles[i] == view.TileRoad }, blocked, wiggle))
}

func (w *World) laneQuarter(g *growingLot, wiggle smooth) {
	if len(g.Tiles) < laneMin*laneMin {
		return
	}
	in := map[view.Point]bool{}
	for _, p := range g.Tiles {
		in[p] = true
	}
	w.joinRoads(g.Seed, in, wiggle)
	w.streets(in)
}

func (w *World) streets(in map[view.Point]bool) {
	tiles := make([]view.Point, 0, len(in))
	for p := range in {
		tiles = append(tiles, p)
	}
	sortPoints(tiles)
	for n := 0; n < streetMax; n++ {
		far, prev := map[view.Point]int{}, map[view.Point]view.Point{}
		var queue []view.Point
		for _, p := range tiles {
			for _, d := range sides {
				if q := (view.Point{X: p.X + d[0], Y: p.Y + d[1]}); w.open(q.X, q.Y) && !w.open(p.X, p.Y) {
					if _, seen := far[p]; !seen {
						far[p], prev[p] = 1, q
						queue = append(queue, p)
					}
				}
			}
		}
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			for _, d := range sides {
				q := view.Point{X: p.X + d[0], Y: p.Y + d[1]}
				if _, seen := far[q]; seen || !in[q] || w.open(q.X, q.Y) || w.occupied(q.X, q.Y, 1, 1, 0) {
					continue
				}
				far[q], prev[q] = far[p]+1, p
				queue = append(queue, q)
			}
		}
		end, most := view.Point{}, 0
		for _, p := range tiles {
			if d, ok := far[p]; ok && d > most {
				end, most = p, d
			}
		}
		if most <= streetWalk {
			return
		}
		var path []view.Point
		for p := end; !w.open(p.X, p.Y); p = prev[p] {
			path = append(path, p)
		}
		w.pave(path[streetStub:])
	}
}

func (w *World) squareIn(g *growingLot, in map[view.Point]bool) ([]view.Point, bool) {
	best, found, score := view.Point{}, false, math.Inf(1)
	for _, p := range g.Tiles {
		fits := true
		for dy := 0; dy < squareSide && fits; dy++ {
			for dx := 0; dx < squareSide && fits; dx++ {
				fits = in[view.Point{X: p.X + dx, Y: p.Y + dy}]
			}
		}
		if !fits {
			continue
		}
		if sc := math.Hypot(float64(p.X+1-g.Seed.X), float64(p.Y+1-g.Seed.Y)); sc < score {
			best, found, score = p, true, sc
		}
	}
	if !found {
		return nil, false
	}
	var out []view.Point
	for dy := 0; dy < squareSide; dy++ {
		for dx := 0; dx < squareSide; dx++ {
			out = append(out, view.Point{X: best.X + dx, Y: best.Y + dy})
		}
	}
	return out, true
}

func (w *World) open(x, y int) bool {
	t := w.Site.Tile(x, y)
	return t == view.TileRoad || t == view.TilePlaza
}

func (w *World) crowds(x, y, wd, ht int, mine map[string]bool) bool {
	for _, st := range w.Site.Structures {
		margin := 1
		if mine[st.ID] {
			margin = 0
		}
		if x < st.X+st.W+margin && st.X < x+wd+margin && y < st.Y+st.H+margin && st.Y < y+ht+margin {
			return true
		}
	}
	return false
}

func (w *World) placeItem(g *growingLot, in map[view.Point]bool, it item, mine map[string]bool, commit bool) (string, bool, bool) {
	shapes := [][2]int{{it.w, it.h}}
	if it.w != it.h {
		shapes = append(shapes, [2]int{it.h, it.w})
	}
	type cand struct {
		x, y, w, h int
		score      float64
		fronts     bool
	}
	var best cand
	found := false
	for _, p := range g.Tiles {
		for _, sh := range shapes {
			wd, ht := sh[0], sh[1]
			fits := true
			for dy := 0; dy < ht && fits; dy++ {
				for dx := 0; dx < wd && fits; dx++ {
					q := view.Point{X: p.X + dx, Y: p.Y + dy}
					fits = in[q] && !w.open(q.X, q.Y)
				}
			}
			if !fits || w.crowds(p.X, p.Y, wd, ht, mine) {
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
			sc := front + seedWeight*math.Hypot(cx-float64(g.Seed.X), cy-float64(g.Seed.Y))
			if !found || sc < best.score {
				best, found = cand{p.X, p.Y, wd, ht, sc, front == 0}, true
			}
		}
	}
	if !found {
		return "", false, false
	}
	if !commit {
		return "", best.fronts, true
	}
	variant := it.variant
	if strings.HasPrefix(it.kind, "depot-") {
		variant = view.FaceSE
		if best.w == 2 {
			variant = view.FaceSW
		}
	}
	id := fmt.Sprintf("%s-%d", strings.TrimPrefix(it.kind, "depot-"), len(w.Site.Structures)+1)
	w.add(view.Structure{ID: id, Kind: it.kind, Variant: variant, X: best.x, Y: best.y, W: best.w, H: best.h, State: it.state, Progress: 1, Label: it.label, Object: it.object}, it.tip)
	return id, best.fronts, true
}

func (w *World) facesStreet(x, y, wd int) bool {
	if wd == 1 {
		return w.open(x+1, y) || w.open(x+1, y+1)
	}
	return w.open(x, y+1) || w.open(x+1, y+1)
}

func (w *World) touchesStreet(x, y, wd, ht int) bool {
	for dy := -1; dy <= ht; dy++ {
		for dx := -1; dx <= wd; dx++ {
			if (dx == -1 || dx == wd || dy == -1 || dy == ht) && w.open(x+dx, y+dy) {
				return true
			}
		}
	}
	return false
}

func (w *World) measureLands() {
	owner := w.Site.KeepLand()
	nogo := w.nogo()
	keepIdx := map[int16]string{}
	for i, st := range w.Site.Structures {
		if st.Kind == view.StructKeep {
			keepIdx[int16(i+1)] = st.Object.Name
		}
	}
	w.lands = map[string]*keepLand{}
	for _, k := range w.Keeps {
		w.lands[k.Node] = &keepLand{node: k.Node, center: view.Point{X: k.X + 1, Y: k.Y + 1}, lot: map[view.Point]string{}, seeds: map[string]view.Point{}, orders: map[string][]int{}}
	}
	for _, p := range w.Lots {
		l, ok := w.lands[p.Node]
		if !ok {
			continue
		}
		l.seeds[p.Namespace] = p.Seed
		for _, t := range p.Tiles {
			l.lot[t] = p.Namespace
		}
	}
	front := w.streetDistance()
	for i, o := range owner {
		node, ok := keepIdx[o]
		if !ok || nogo[i] {
			continue
		}
		l := w.lands[node]
		l.slots = append(l.slots, landSlot{at: view.Point{X: i % w.Site.Width, Y: i / w.Site.Width}, front: front[i]})
	}
}

func (w *World) streetDistance() []int {
	far := make([]int, len(w.Site.Tiles))
	queue := []int{}
	for i, t := range w.Site.Tiles {
		far[i] = -1
		if t == view.TileRoad || t == view.TilePlaza {
			far[i] = 0
			queue = append(queue, i)
		}
	}
	for len(queue) > 0 {
		i := queue[0]
		queue = queue[1:]
		x, y := i%w.Site.Width, i/w.Site.Width
		for _, d := range sides {
			if j, ok := w.index(x+d[0], y+d[1]); ok && far[j] < 0 {
				far[j] = far[i] + 1
				queue = append(queue, j)
			}
		}
	}
	for i := range far {
		if far[i] < 0 {
			far[i] = frontReach
		}
	}
	return far
}

func (w *World) nogo() []bool {
	out := make([]bool, len(w.Site.Tiles))
	mark := func(x, y int) {
		if i, ok := w.index(x, y); ok {
			out[i] = true
		}
	}
	for i, t := range w.Site.Tiles {
		if !hutGround(t) || w.Site.HarborGround(i%w.Site.Width, i/w.Site.Width, 1) {
			out[i] = true
		}
	}
	for _, st := range w.Site.Structures {
		for y := st.Y - 1; y <= st.Y+st.H; y++ {
			for x := st.X - 1; x <= st.X+st.W; x++ {
				mark(x, y)
			}
		}
		if st.Kind == view.StructKeep {
			hx, hy := st.X+st.W-1, st.Y+st.H-1
			for _, d := range doorSteps {
				mark(hx+d[0], hy+d[1])
			}
		}
	}
	for i := range out {
		if out[i] {
			continue
		}
		x, y := i%w.Site.Width, i/w.Site.Width
		for _, st := range w.Site.Structures {
			if hiddenBy(st, x, y) {
				out[i] = true
				break
			}
		}
	}
	return out
}

func hutGround(t view.TileKind) bool {
	return t == view.TileGrass || t == view.TileMeadow || t == view.TileSand
}

func hiddenBy(s view.Structure, x, y int) bool {
	hiX, hiY := s.X+max(s.W, 1)-1, s.Y+max(s.H, 1)-1
	clo, chi := s.X-hiY-1, hiX-s.Y+1
	if x-y-1 >= chi || clo >= x-y+1 {
		return false
	}
	diff := hiX + hiY - (x + y)
	return diff > -2 && diff < view.StructureLift(s.Kind, s.Variant)
}

func (w *World) roomless(groups []*isleGroup) ([]int, bool) {
	short := map[int]bool{}
	grow := false
	for gi, g := range groups {
		for _, n := range g.nodes {
			k, _ := w.keepOf(n.Name)
			l, ok := w.lands[n.Name]
			switch huts := k.Pods - k.Shrines; {
			case !ok:
			case len(l.slots) < huts:
				short[gi], grow = true, true
			case huts <= widenUpTo && l.room() < huts && w.wider[n.Name] < maxWiden:
				short[gi] = true
				w.wider[n.Name]++
			}
		}
	}
	out := []int{}
	for gi := range short {
		out = append(out, gi)
	}
	sort.Ints(out)
	return out, grow
}

func (l *keepLand) order(ns string) []int {
	if o, ok := l.orders[ns]; ok {
		return o
	}
	seed, has := l.seeds[ns]
	if !has {
		seed = l.center
	}
	o := make([]int, len(l.slots))
	score := make([]float64, len(l.slots))
	for i, s := range l.slots {
		o[i] = i
		own := 0.0
		switch l.lot[s.at] {
		case ns:
		case "":
			own = strayCost
		default:
			own = foreignCost
		}
		score[i] = own + frontWeight*float64(min(s.front, frontReach)) + math.Hypot(float64(s.at.X-seed.X), float64(s.at.Y-seed.Y))
	}
	sort.SliceStable(o, func(a, b int) bool {
		if score[o[a]] != score[o[b]] {
			return score[o[a]] < score[o[b]]
		}
		pa, pb := l.slots[o[a]].at, l.slots[o[b]].at
		if pa.Y != pb.Y {
			return pa.Y < pb.Y
		}
		return pa.X < pb.X
	})
	l.orders[ns] = o
	return o
}

func (l *keepLand) room() int {
	placed := map[view.Point]bool{}
	for _, s := range l.slots {
		if !coveredBy(placed, s.at) {
			placed[s.at] = true
		}
	}
	return len(placed)
}

func coveredBy(placed map[view.Point]bool, at view.Point) bool {
	for _, d := range covers {
		if placed[view.Point{X: at.X + d[0], Y: at.Y + d[1]}] || placed[view.Point{X: at.X - d[0], Y: at.Y - d[1]}] {
			return true
		}
	}
	return false
}

func (w *World) coversHut(at view.Point) bool {
	for _, d := range covers {
		if _, ok := w.taken[view.Point{X: at.X + d[0], Y: at.Y + d[1]}]; ok {
			return true
		}
		if _, ok := w.taken[view.Point{X: at.X - d[0], Y: at.Y - d[1]}]; ok {
			return true
		}
	}
	return false
}
