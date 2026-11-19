package live

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	LabelZone         = "topology.kubernetes.io/zone"
	LabelInstanceType = "node.kubernetes.io/instance-type"
	labelControlPlane = "node-role.kubernetes.io/control-plane"
	labelMaster       = "node-role.kubernetes.io/master"

	keepSide     = 3
	keepPlotArea = 225.0
	hutArea      = 2.0
	hutRoom      = 4.0
	keepCore     = 25.0
	plotSlack    = 1.2
	civicArea    = 500.0
	isleSlack    = 1.6
	coastTiles   = 4.0
	minIsle      = 14
	isleGrowth   = 1.25
	maxGrowths   = 16
	maxWiden     = 3
	widenUpTo    = 48
	isleGapTiles = 10
	worldMargin  = 6
	hutRingMin   = 2
	hutRingMax   = 8
	towerGap     = 9
	baseReach    = 5
	shareRoom    = 2.4
	bucketSide   = 16
)

var keepGaps = []int{13, 10, 7, 5, 4}

var poolLabels = []string{
	"cloud.google.com/gke-nodepool", "eks.amazonaws.com/nodegroup", "alpha.eksctl.io/nodegroup-name", "kubernetes.azure.com/agentpool",
	"karpenter.sh/nodepool", "node.kubernetes.io/pool", "doks.digitalocean.com/node-pool", "nodepool",
}

type Isle struct {
	ID    string
	Name  string
	Zone  string
	By    string
	Nodes []string
	Home  bool
}

type KeepPlot struct {
	Node     string
	UID      types.UID
	Isle     string
	X, Y     int
	Variant  string
	Instance string
	CPU      int
	MemGiB   int
	Control  bool
	Ready    corev1.ConditionStatus
	Reach    int
	Pods     int
	Shrines  int
}

type District struct {
	Namespace string
	UID       types.UID
	Isle      string
	Order     int
	Pods      int
	Center    view.Point
	Citadel   bool
}

type Crew struct {
	UID       types.UID
	Kind      string
	Story     string
	Namespace string
	Name      string
	Pods      []types.UID
}

type HutPlot struct {
	UID       types.UID
	Name      string
	Namespace string
	Node      string
	Crew      types.UID
	X, Y      int
	Placed    bool
	Shrine    bool
}

type World struct {
	Seed        uint64
	Site        view.Site
	Isles       []Isle
	Keeps       []KeepPlot
	Districts   []District
	Crews       []Crew
	Huts        []HutPlot
	Lots        []Lot
	Idle        []string
	Tips        map[string]Tip
	Chancery    string
	Gates       int
	Signposts   int
	wider       map[string]int
	taken       map[view.Point]types.UID
	buckets     map[view.Point][]int
	widest      int
	lands       map[string]*keepLand
	shrineOf    map[types.UID]view.Point
	structureOf map[string]int
	newFiefs    map[string]bool
	skipped     map[string]bool
}

func SeedOf(s Snapshot) uint64 {
	h := fnv.New64a()
	for _, ns := range s.Namespaces {
		if ns.Name == "kube-system" {
			h.Write([]byte(ns.UID))
			return h.Sum64()
		}
	}
	for _, n := range byUID(s.Nodes, func(n corev1.Node) types.UID { return n.UID }) {
		h.Write([]byte(n.UID))
	}
	return h.Sum64()
}

func byUID[T any](in []T, uid func(T) types.UID) []T {
	out := append([]T(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return uid(out[i]) < uid(out[j]) })
	return out
}

func olderFirst(a, b metav1.ObjectMeta) bool {
	if !a.CreationTimestamp.Equal(&b.CreationTimestamp) {
		return a.CreationTimestamp.Before(&b.CreationTimestamp)
	}
	return a.UID < b.UID
}

func controlPlane(n corev1.Node) bool {
	_, cp := n.Labels[labelControlPlane]
	_, master := n.Labels[labelMaster]
	return cp || master
}

func isleKey(n corev1.Node) (string, string) {
	if z := n.Labels[LabelZone]; z != "" {
		return "zone", z
	}
	for _, l := range poolLabels {
		if v := n.Labels[l]; v != "" {
			return "pool", v
		}
	}
	if v := n.Labels[LabelInstanceType]; v != "" {
		return "instance-type", v
	}
	return "", ""
}

func nodeReady(n corev1.Node) corev1.ConditionStatus {
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status
		}
	}
	return corev1.ConditionUnknown
}

func keepsOf(s Snapshot) []corev1.Node {
	out := append([]corev1.Node(nil), s.Nodes...)
	known := map[string]bool{}
	for _, n := range s.Nodes {
		known[n.Name] = true
	}
	var unseen []string
	for _, p := range s.Pods {
		if n := p.Spec.NodeName; n != "" && !known[n] {
			known[n] = true
			unseen = append(unseen, n)
		}
	}
	sort.Strings(unseen)
	for _, n := range unseen {
		out = append(out, corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: n, UID: types.UID("node/" + n)},
			Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionUnknown}}}})
	}
	return out
}

type isleGroup struct {
	by, key string
	nodes   []corev1.Node
	pods    int
	control bool
	podsOn  map[string]int
	daemons map[string]int
}

func groupIsles(s Snapshot) []*isleGroup {
	nodes := keepsOf(s)
	sort.SliceStable(nodes, func(i, j int) bool { return olderFirst(nodes[i].ObjectMeta, nodes[j].ObjectMeta) })
	podsOn, daemons := map[string]int{}, map[string]int{}
	for _, p := range s.Pods {
		podsOn[p.Spec.NodeName]++
		if daemonPod(p) {
			daemons[p.Spec.NodeName]++
		}
	}
	byKey := map[string]*isleGroup{}
	var groups []*isleGroup
	for _, n := range nodes {
		by, key := isleKey(n)
		g, ok := byKey[by+"/"+key]
		if !ok {
			g = &isleGroup{by: by, key: key, podsOn: podsOn, daemons: daemons}
			byKey[by+"/"+key] = g
			groups = append(groups, g)
		}
		g.nodes = append(g.nodes, n)
		g.pods += podsOn[n.Name]
		g.control = g.control || controlPlane(n)
	}
	if len(groups) == 0 {
		groups = []*isleGroup{{podsOn: podsOn, daemons: daemons}}
	}
	sort.SliceStable(groups, func(i, j int) bool {
		a, b := groups[i], groups[j]
		if a.control != b.control {
			return a.control
		}
		if len(a.nodes) != len(b.nodes) {
			return len(a.nodes) > len(b.nodes)
		}
		return a.by+"/"+a.key < b.by+"/"+b.key
	})
	return groups
}

func keepReach(n corev1.Node, pods, wider int) int {
	base := view.TowerTerritory
	if v := keepVariant(n); v != "tower" {
		if kt, ok := view.KeepTypeOf(v); ok {
			base = kt.Territory
		}
	}
	need := int(math.Ceil(math.Sqrt((float64(pods)*hutRoom + keepCore) / math.Pi)))
	if wider > 0 {
		need = max(need, base) + wider
	}
	if need > base {
		return need
	}
	return 0
}

func plotArea(n corev1.Node, pods int) float64 {
	area := keepPlotArea + float64(pods)*hutArea
	if r := keepReach(n, pods, 0); r > 0 {
		side := float64(2*r+1) * plotSlack
		area = max(area, side*side)
	}
	return area
}

func isleRadius(g *isleGroup, home bool, extra float64) int {
	area := keepPlotArea + extra
	if len(g.nodes) > 0 {
		area = 0
	}
	for _, n := range g.nodes {
		area += plotArea(n, g.podsOn[n.Name])
	}
	if home {
		area += civicArea
	}
	r := math.Sqrt(area*isleSlack/math.Pi) + coastTiles
	return max(int(math.Ceil(r)), minIsle)
}

func isleSpan(r int) int {
	return int(math.Ceil(2*float64(r)*view.CoastCeiling)) + isleGapTiles
}

type placedIsle struct {
	cx, cy, r int
}

func layOut(radii []int) ([]placedIsle, int, int) {
	total := 0.0
	for _, r := range radii {
		d := float64(isleSpan(r))
		total += d * d
	}
	rowLimit := int(math.Max(math.Sqrt(total)*1.25, float64(isleSpan(radii[0]))))
	out := make([]placedIsle, len(radii))
	x, y, rowH, width := worldMargin, worldMargin, 0, 0
	for i, r := range radii {
		d := isleSpan(r)
		if x > worldMargin && x+d > rowLimit {
			x, y, rowH = worldMargin, y+rowH, 0
		}
		out[i] = placedIsle{cx: x + d/2, cy: y + d/2, r: r}
		x += d
		rowH = max(rowH, d)
		width = max(width, x)
	}
	return out, width + worldMargin, y + rowH + worldMargin
}

func isleNames(groups []*isleGroup) []string {
	used := map[string]bool{}
	out := make([]string, len(groups))
	for i, g := range groups {
		h := fnv.New32a()
		h.Write([]byte(g.by + "/" + g.key))
		start := int(h.Sum32() % uint32(len(storyIsles)))
		name := ""
		for round := 1; name == ""; round++ {
			for k := 0; k < len(storyIsles); k++ {
				cand := storyIsles[(start+k)%len(storyIsles)]
				if round > 1 {
					cand = fmt.Sprintf("%s %s", cand, roman(round))
				}
				if !used[cand] {
					name = cand
					break
				}
			}
		}
		used[name] = true
		out[i] = name
	}
	return out
}

func roman(n int) string {
	vals := []int{10, 9, 5, 4, 1}
	syms := []string{"X", "IX", "V", "IV", "I"}
	var b strings.Builder
	for i, v := range vals {
		for n >= v {
			b.WriteString(syms[i])
			n -= v
		}
	}
	return b.String()
}

func Build(s Snapshot) (*World, error) {
	return BuildContext(context.Background(), s)
}

func BuildContext(ctx context.Context, s Snapshot) (*World, error) {
	groups := groupIsles(s)
	if len(groups) > 250 {
		return nil, errors.New("the cluster has more node groups than the realm can draw")
	}
	isleOfNode := map[string]int{}
	for i, g := range groups {
		for _, n := range g.nodes {
			isleOfNode[n.Name] = i
		}
	}
	plans := planDistricts(s, isleOfNode)
	extra := make([]float64, len(groups))
	for _, d := range plans {
		for i := range extra {
			extra[i] += lotArea(d.itemsOn(i))
		}
	}
	if len(harborEntries(s)) > 0 {
		extra[0] += harborArea
	}
	radii := make([]int, len(groups))
	for i, g := range groups {
		radii[i] = isleRadius(g, i == 0, extra[i])
	}
	if radii[0] < 27 {
		radii[0] = 27
	}
	names := isleNames(groups)
	seed := SeedOf(s)
	var world *World
	wider := map[string]int{}
	for tries, growths := 1, 0; ; tries++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("drawing the realm took too long (%d nodes, %d pods, %d layouts tried): %w", len(s.Nodes), len(s.Pods), tries, err)
		}
		w, short, err := lay(seed, groups, names, radii, wider)
		if err != nil {
			return nil, fmt.Errorf("the cluster is larger than the realm can draw (%d nodes, %d pods): %w", len(s.Nodes), len(s.Pods), err)
		}
		grow := true
		if len(short) == 0 {
			short, grow = w.furnish(s, groups, plans)
		}
		if len(short) == 0 {
			world = w
			break
		}
		if !grow {
			continue
		}
		if growths == maxGrowths {
			return nil, fmt.Errorf("the cluster's %d nodes found no room on their isles", len(s.Nodes))
		}
		growths++
		for _, i := range short {
			radii[i] = int(math.Ceil(float64(radii[i]) * isleGrowth))
		}
	}
	world.crews(s)
	world.Settle(s.Pods)
	world.districts(s, plans)
	world.idle(s)
	world.noteSkipped(s)
	return world, nil
}

func lay(seed uint64, groups []*isleGroup, names []string, radii []int, wider map[string]int) (*World, []int, error) {
	placed, wide, high := layOut(radii)
	plan := view.WorldPlan{Width: max(wide, 64), Height: max(high, 64)}
	for i, g := range groups {
		plan.Isles = append(plan.Isles, view.IslePlan{Name: names[i], Zone: g.key, FX: float64(placed[i].cx) / float64(plan.Width), FY: float64(placed[i].cy) / float64(plan.Height), Radius: placed[i].r})
	}
	world := &World{Seed: seed, wider: wider, taken: map[view.Point]types.UID{}, buckets: map[view.Point][]int{}}
	site, err := view.GenerateWorld(seed, plan)
	if err != nil {
		return nil, nil, err
	}
	world.Site = site
	for i, g := range groups {
		is := Isle{ID: site.Islands[i].ID, Name: site.Islands[i].Name, Zone: g.key, By: g.by, Home: i == 0}
		for _, n := range g.nodes {
			is.Nodes = append(is.Nodes, n.Name)
		}
		world.Isles = append(world.Isles, is)
	}
	return world, world.seatKeeps(groups), nil
}

func (w *World) seatKeeps(groups []*isleGroup) []int {
	towered := false
	var short []int
	for i, g := range groups {
		region := uint8(i + 1)
		cands := w.keepCandidates(region, w.Site.Islands[i].Center)
		roomy := true
		for _, n := range g.nodes {
			pods := g.podsOn[n.Name]
			k := KeepPlot{Node: n.Name, UID: n.UID, Isle: w.Isles[i].ID, Instance: n.Labels[LabelInstanceType], Control: controlPlane(n), Ready: nodeReady(n), Variant: keepVariant(n),
				CPU: int(n.Status.Allocatable.Cpu().Value()), MemGiB: memoryGiB(*n.Status.Allocatable.Memory()), Reach: keepReach(n, pods, w.wider[n.Name]), Pods: pods, Shrines: g.daemons[n.Name]}
			if k.Control && !towered && i == 0 {
				towered = true
				k.X, k.Y = w.Site.TowerSpot.X-1, w.Site.TowerSpot.Y-1
			} else if x, y, ok := w.keepSpot(cands, k.Reach); ok {
				k.X, k.Y = x, y
			} else {
				roomy = false
				break
			}
			w.seat(k)
		}
		if !roomy {
			short = append(short, i)
		}
	}
	if len(short) > 0 {
		return short
	}
	for i, k := range w.Keeps {
		w.Site.Structures = append(w.Site.Structures, keepStructure(i, k))
	}
	return nil
}

func (w *World) seat(k KeepPlot) {
	i := len(w.Keeps)
	w.Keeps = append(w.Keeps, k)
	cell := bucketOf(k.X+1, k.Y+1)
	w.buckets[cell] = append(w.buckets[cell], i)
	w.widest = max(w.widest, k.Reach)
}

func bucketOf(x, y int) view.Point {
	return view.Point{X: x / bucketSide, Y: y / bucketSide}
}

func keepVariant(n corev1.Node) string {
	if controlPlane(n) {
		return "tower"
	}
	cpu := n.Status.Allocatable[corev1.ResourceCPU]
	switch c := cpu.MilliValue(); {
	case c >= 16000:
		return "citadel"
	case c >= 8000:
		return "great"
	}
	return "stone"
}

func keepStructure(i int, k KeepPlot) view.Structure {
	state := view.StateReady
	if k.Ready != corev1.ConditionTrue {
		state = view.StateDark
	}
	return view.Structure{
		ID: fmt.Sprintf("keep-%d", i+1), Kind: view.StructKeep, Variant: k.Variant, X: k.X, Y: k.Y, W: keepSide, H: keepSide,
		State: state, Progress: 1, Label: "Keep " + k.Node, Territory: k.Reach,
		Object: view.ObjectRef{Kind: "Node", Name: k.Node}, Instance: k.Instance, CPU: k.CPU, MemoryGiB: k.MemGiB,
	}
}

func (w *World) footprintFree(x, y int, region uint8) bool {
	for dy := 0; dy < keepSide; dy++ {
		for dx := 0; dx < keepSide; dx++ {
			i := (y+dy)*w.Site.Width + x + dx
			if x+dx < 0 || y+dy < 0 || x+dx >= w.Site.Width || y+dy >= w.Site.Height || w.Site.Region[i] != region || !buildable(w.Site.Tiles[i]) {
				return false
			}
		}
	}
	return true
}

func buildable(t view.TileKind) bool {
	switch t {
	case view.TileWater, view.TileShallows, view.TileSand, view.TileRoad, view.TilePlaza:
		return false
	}
	return true
}

func pairGap(gap, a, b int) int {
	sum := max(a, baseReach) + max(b, baseReach)
	if sum <= 2*baseReach {
		return gap
	}
	return int(math.Ceil(float64(gap) * float64(sum) / float64(2*baseReach)))
}

func (w *World) clearOf(cx, cy, gap, reach int) bool {
	far := pairGap(gap, reach, w.widest)
	lo, hi := bucketOf(max(cx-far, 0), max(cy-far, 0)), bucketOf(cx+far, cy+far)
	for by := lo.Y; by <= hi.Y; by++ {
		for bx := lo.X; bx <= hi.X; bx++ {
			for _, i := range w.buckets[view.Point{X: bx, Y: by}] {
				k := w.Keeps[i]
				need := pairGap(gap, reach, k.Reach)
				if abs(k.X+1-cx) < need && abs(k.Y+1-cy) < need {
					return false
				}
			}
		}
	}
	for _, p := range []view.Point{w.Site.TowerSpot, w.Site.Plaza, w.Site.Gate} {
		if abs(p.X-cx) < towerGap && abs(p.Y-cy) < towerGap {
			return false
		}
	}
	return true
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

type keepCand struct{ x, y, d int }

func (w *World) keepCandidates(region uint8, anchor view.Point) []keepCand {
	var cands []keepCand
	for i, r := range w.Site.Region {
		if r != region {
			continue
		}
		x, y := i%w.Site.Width-1, i/w.Site.Width-1
		if w.footprintFree(x, y, region) {
			cands = append(cands, keepCand{x, y, (x+1-anchor.X)*(x+1-anchor.X) + (y+1-anchor.Y)*(y+1-anchor.Y)})
		}
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].d != cands[j].d {
			return cands[i].d < cands[j].d
		}
		if cands[i].y != cands[j].y {
			return cands[i].y < cands[j].y
		}
		return cands[i].x < cands[j].x
	})
	return cands
}

func (w *World) keepSpot(cands []keepCand, reach int) (int, int, bool) {
	for _, gap := range keepGaps {
		for _, c := range cands {
			if w.clearOf(c.x+1, c.y+1, gap, reach) {
				return c.x, c.y, true
			}
		}
	}
	return 0, 0, false
}

func hash64(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}

func (w *World) Settle(pods []corev1.Pod) {
	live := map[types.UID]corev1.Pod{}
	for _, p := range pods {
		live[p.UID] = p
	}
	kept := w.Huts[:0]
	for _, h := range w.Huts {
		p, ok := live[h.UID]
		if ok && p.Spec.NodeName == h.Node {
			kept = append(kept, h)
			continue
		}
		if h.Placed && !h.Shrine {
			delete(w.taken, view.Point{X: h.X, Y: h.Y})
		}
	}
	w.Huts = kept
	have := map[types.UID]bool{}
	for _, h := range w.Huts {
		have[h.UID] = true
	}
	fresh := []corev1.Pod{}
	for _, p := range pods {
		if !have[p.UID] {
			fresh = append(fresh, p)
		}
	}
	sort.SliceStable(fresh, func(i, j int) bool { return olderFirst(fresh[i].ObjectMeta, fresh[j].ObjectMeta) })
	crewOf := w.crewIndex()
	for _, p := range fresh {
		h := HutPlot{UID: p.UID, Name: p.Name, Namespace: p.Namespace, Node: p.Spec.NodeName, Crew: crewOf[p.UID]}
		if at, ok := w.shrineOf[p.UID]; ok && p.Spec.NodeName != "" {
			h.X, h.Y, h.Placed, h.Shrine = at.X, at.Y, true, true
		} else if at, ok := w.plot(p); ok {
			h.X, h.Y, h.Placed = at.X, at.Y, true
			w.taken[at] = p.UID
		}
		w.Huts = append(w.Huts, h)
	}
	sort.SliceStable(w.Huts, func(i, j int) bool { return w.Huts[i].UID < w.Huts[j].UID })
	w.Site.Plots = w.Site.Plots[:0]
	for _, h := range w.Huts {
		if h.Placed && !h.Shrine {
			w.Site.Plots = append(w.Site.Plots, view.Point{X: h.X, Y: h.Y})
		}
	}
}

func (w *World) plot(p corev1.Pod) (view.Point, bool) {
	l := w.lands[p.Spec.NodeName]
	if l == nil {
		return view.Point{}, false
	}
	order := l.order(p.Namespace)
	for _, roomy := range []bool{true, false} {
		for _, i := range order {
			at := l.slots[i].at
			if _, busy := w.taken[at]; !busy && (!roomy || !w.coversHut(at)) {
				return at, true
			}
		}
	}
	return view.Point{}, false
}

func (w *World) crewIndex() map[types.UID]types.UID {
	out := map[types.UID]types.UID{}
	for _, c := range w.Crews {
		for _, p := range c.Pods {
			out[p] = c.UID
		}
	}
	return out
}

func crewStory(kind string) string {
	switch kind {
	case "":
		return "lone hut"
	case "Node":
		return "hut at the door"
	}
	if name, ok := lexicon.StoryKind(kind); ok {
		return name
	}
	return "foreign " + strings.ToLower(kind)
}

func (w *World) crews(s Snapshot) {
	type ownerKey struct {
		ns, kind, name string
	}
	uids := map[ownerKey]types.UID{}
	parent := map[ownerKey]ownerKey{}
	note := func(ns, kind string, m metav1.ObjectMeta) {
		k := ownerKey{ns, kind, m.Name}
		uids[k] = m.UID
		if ref := metav1.GetControllerOfNoCopy(&m); ref != nil {
			parent[k] = ownerKey{ns, ref.Kind, ref.Name}
		}
	}
	for _, o := range s.Deployments {
		note(o.Namespace, "Deployment", o.ObjectMeta)
	}
	for _, o := range s.ReplicaSets {
		note(o.Namespace, "ReplicaSet", o.ObjectMeta)
	}
	for _, o := range s.StatefulSets {
		note(o.Namespace, "StatefulSet", o.ObjectMeta)
	}
	for _, o := range s.DaemonSets {
		note(o.Namespace, "DaemonSet", o.ObjectMeta)
	}
	for _, o := range s.Jobs {
		note(o.Namespace, "Job", o.ObjectMeta)
	}
	for _, o := range s.CronJobs {
		note(o.Namespace, "CronJob", o.ObjectMeta)
	}
	byCrew := map[ownerKey]*Crew{}
	for _, p := range byUID(s.Pods, func(p corev1.Pod) types.UID { return p.UID }) {
		top := ownerKey{p.Namespace, "", p.Name}
		if ref := metav1.GetControllerOfNoCopy(&p.ObjectMeta); ref != nil {
			top = ownerKey{p.Namespace, ref.Kind, ref.Name}
			for seen := 0; seen < 4; seen++ {
				up, ok := parent[top]
				if !ok {
					break
				}
				top = up
			}
		}
		c, ok := byCrew[top]
		if !ok {
			uid := uids[top]
			if top.kind == "" {
				uid = p.UID
			}
			if uid == "" {
				uid = types.UID(top.kind + "/" + top.ns + "/" + top.name)
			}
			c = &Crew{UID: uid, Kind: top.kind, Story: crewStory(top.kind), Namespace: top.ns, Name: top.name}
			byCrew[top] = c
		}
		c.Pods = append(c.Pods, p.UID)
	}
	for _, c := range byCrew {
		w.Crews = append(w.Crews, *c)
	}
	sort.Slice(w.Crews, func(i, j int) bool { return w.Crews[i].UID < w.Crews[j].UID })
}

func (w *World) districts(s Snapshot, plans []*districtPlan) {
	for _, d := range plans {
		center, civic := view.Point{}, false
		for _, p := range w.Lots {
			if p.Namespace == d.ns && p.Node == "" {
				center, civic = p.Seed, true
			}
		}
		if !civic {
			center = w.hutCenter(d.ns, d.isle)
		}
		w.Districts = append(w.Districts, District{Namespace: d.ns, UID: d.uid, Isle: w.Isles[d.isle].ID, Order: d.order, Pods: d.pods,
			Citadel: d.ns == view.CitadelFief, Center: center})
	}
	sort.SliceStable(w.Districts, func(i, j int) bool { return w.Districts[i].UID < w.Districts[j].UID })
}

func (w *World) hutCenter(ns string, isle int) view.Point {
	sx, sy, n := 0, 0, 0
	for _, h := range w.Huts {
		if h.Namespace == ns && h.Placed {
			sx, sy, n = sx+h.X, sy+h.Y, n+1
		}
	}
	if n == 0 {
		return w.Site.Islands[isle].Center
	}
	return view.Point{X: sx / n, Y: sy / n}
}

func (w *World) idle(s Snapshot) {
	running := map[string]bool{}
	for _, p := range s.Pods {
		if ref := metav1.GetControllerOfNoCopy(&p.ObjectMeta); ref != nil && ref.Kind == "DaemonSet" {
			running[p.Namespace+"/"+ref.Name] = true
		}
	}
	for _, d := range s.DaemonSets {
		if key := d.Namespace + "/" + d.Name; !running[key] {
			w.Idle = append(w.Idle, key)
		}
	}
	sort.Strings(w.Idle)
}

func memoryGiB(q resource.Quantity) int {
	return int(q.Value() >> 30)
}
