package live

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/kubectl"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
	"github.com/waxb/kubernites/pkg/sim/clock"
	"github.com/waxb/kubernites/pkg/sim/store"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	ReadOnlyAnswer = "This realm is read-only: you are watching a real cluster."
	noAugur        = "error: Metrics API not available: no Augur in this realm (metrics-server not found)"
	noSecrets      = "error: Secrets are never read in a live realm: their values stay in the cluster."
	readTimeout    = 10 * time.Second
	liveNoon       = .5
	logTailLines   = 500
)

type podLogReader interface {
	podLogs(ctx context.Context, namespace, pod, container string, previous bool, tail int) (string, error)
}

type Realm struct {
	mu         sync.Mutex
	snap       Snapshot
	world      *World
	mirror     *store.Store
	clk        *clock.Clock
	logs       podLogReader
	usage      Usage
	watches    []func(store.Event)
	wire       *wire
	watching   bool
	stopped    bool
	cancel     context.CancelFunc
	health     map[string]kindHealth
	reads      readBook
	outbox     []store.Event
	dirty      bool
	drawn      uint64
	book       troubleBook
	replayed   bool
	sinceDrawn float64
	staleAt    time.Time
	staleWhy   string
	tree       *kubectltree.Tree
	gate       *write.Gate
}

func Open(s Snapshot, logs podLogReader, usage Usage) (*Realm, error) {
	return OpenContext(context.Background(), s, logs, usage)
}

func OpenContext(ctx context.Context, s Snapshot, logs podLogReader, usage Usage) (*Realm, error) {
	mirror, err := NewMirror(s)
	if err != nil {
		return nil, err
	}
	world, err := BuildContext(ctx, s)
	if err != nil {
		return nil, err
	}
	r := &Realm{sinceDrawn: PumpEvery, snap: s.clone(), world: world, mirror: mirror, clk: clock.New(s.TakenAt), logs: logs, usage: usage, health: map[string]kindHealth{}}
	mirror.Watch(func(e store.Event) { r.outbox = append(r.outbox, e) })
	r.book = newTroubleBook(signals(&r.snap))
	return r, nil
}

func Connect(ctx context.Context, paths []string, name string, now time.Time, progress func(Progress)) (*Realm, error) {
	conn, err := Dial(paths, name)
	if err != nil {
		return nil, err
	}
	r, err := connectOver(ctx, conn, now, progress)
	if err != nil {
		return nil, err
	}
	r.tree = openTree(paths, name, conn.rest, nil)
	return r, nil
}

func ConnectGated(ctx context.Context, paths []string, name string, g *write.Gate, now time.Time, progress func(Progress)) (*Realm, error) {
	conn, err := DialGated(paths, name, g)
	if err != nil {
		return nil, err
	}
	g.LearnLocalGroups(ctx)
	r, err := connectOver(ctx, conn, now, progress)
	if err != nil {
		return nil, err
	}
	r.gate = g
	g.StaleWhen(r.staleReason)
	r.tree = openTree(paths, name, conn.rest, g)
	return r, nil
}

func (r *Realm) Gate() *write.Gate {
	return r.gate
}

func (r *Realm) staleReason() string {
	f := r.Freshness()
	if f.StaleSince.IsZero() {
		return ""
	}
	return "stale since " + f.StaleSince.UTC().Format("15:04:05 UTC") + ": " + f.Why
}

func connectOver(ctx context.Context, conn *Conn, now time.Time, progress func(Progress)) (*Realm, error) {
	last := Progress{}
	tell := func(p Progress) {
		last = p
		if progress != nil {
			progress(p)
		}
	}
	who, _ := conn.WhoAmI(ctx)
	snap, err := Take(ctx, conn, now, tell)
	if err != nil {
		return nil, err
	}
	snap.Identity = who
	step := func(name string) {
		next := last
		next.Step = name
		tell(next)
	}
	step(StepMetrics)
	usage := conn.Usage(ctx)
	step(StepDrawing)
	r, err := OpenContext(ctx, snap, conn, usage)
	if err != nil {
		return nil, err
	}
	r.wire = wireOf(conn)
	return r, nil
}

func WhoAmI(ctx context.Context, paths []string, name string) (Identity, error) {
	conn, err := Dial(paths, name)
	if err != nil {
		return Identity{Unknown: err.Error()}, err
	}
	return conn.WhoAmI(ctx)
}

func (r *Realm) World() *World {
	return r.world
}

func (r *Realm) Snapshot() Snapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snap.clone()
}

func (r *Realm) Advance(realSeconds float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clk.Advance(realSeconds)
}

const PumpEvery = .25

func (r *Realm) Pump(realSeconds float64) {
	r.mu.Lock()
	r.clk.Advance(realSeconds)
	r.sinceDrawn += realSeconds
	if r.dirty && r.sinceDrawn >= PumpEvery {
		r.dirty, r.sinceDrawn = false, 0
		r.redraw()
		r.book.reckon(signals(&r.snap), r.clk.Seconds(), &r.snap, r.clk.Now())
	} else {
		r.book.age(r.clk.Seconds())
	}
	out := r.outbox
	r.outbox = nil
	watches := r.watches
	r.mu.Unlock()
	for _, e := range out {
		for _, fn := range watches {
			fn(e)
		}
	}
}

func (r *Realm) Drawn() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.drawn
}

func (r *Realm) Watch(fn func(store.Event)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.watches = append(r.watches, fn)
}

func (r *Realm) WithSnapshot(fn func([]store.Object)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	fn(r.mirror.Snapshot().Objects)
}

func (r *Realm) Site() view.Site {
	site := r.world.Site
	site.Tiles = append([]view.TileKind(nil), site.Tiles...)
	site.Elevation = append([]uint8(nil), site.Elevation...)
	site.Structures = append([]view.Structure(nil), site.Structures...)
	site.Islands = append([]view.Island(nil), site.Islands...)
	site.Region = append([]uint8(nil), site.Region...)
	site.Plots = append([]view.Point(nil), site.Plots...)
	return site
}

func (r *Realm) Metrics() view.MetricsSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := view.MetricsSnapshot{Pods: map[string]view.PodMetrics{}, Nodes: map[string]view.NodeMetrics{}, Augury: r.usage.Available}
	for key, u := range r.usage.Pods {
		out.Pods[key] = view.PodMetrics{CPUMilli: u.CPUMilli, MemoryBytes: u.MemBytes}
	}
	for _, n := range store.View[*corev1.Node](r.mirror, store.KindNode, "") {
		nm := view.NodeMetrics{CPUAllocatableMilli: n.Status.Allocatable.Cpu().MilliValue(), MemoryAllocatableBytes: n.Status.Allocatable.Memory().Value()}
		if u, ok := r.usage.Nodes[n.Name]; ok {
			nm.CPUMilli, nm.MemoryBytes = u.CPUMilli, u.MemBytes
		}
		out.Nodes[n.Name] = nm
	}
	return out
}

func (r *Realm) Action(view.Action) (string, error) {
	return "", ErrReadOnly
}

func (r *Realm) Perform(view.Action) (view.Performed, error) {
	return view.Performed{Message: ReadOnlyAnswer}, ErrReadOnly
}

func (r *Realm) Apply(string) (string, bool) {
	return ReadOnlyAnswer + "\n", false
}

func (r *Realm) Marshal(time.Time) ([]byte, error) {
	return nil, errors.New("a live realm is never saved")
}

func (r *Realm) PlacementError(string, int, int) error {
	return ErrReadOnly
}

func (r *Realm) PlotSurvey(kind string, x, y int) view.Survey {
	return view.Survey{Kind: kind, X: x, Y: y, Reason: ReadOnlyAnswer}
}

func (r *Realm) SuggestPlacement(string) (int, int, bool) {
	return 0, 0, false
}

func (r *Realm) SuggestKeepFor(string, string) (int, int, bool) {
	return 0, 0, false
}

func (r *Realm) Bootstrap() {}

var readVerbs = map[string]bool{
	"get": true, "describe": true, "logs": true, "events": true, "top": true,
	"explain": true, "version": true, "cluster-info": true, "api-resources": true, "help": true,
}

const (
	stoneClientVersion = "v1.37.0"
	notReadYet         = "error: a live realm does not read api-resources yet; the Stone reads get, describe, logs, events and top."
	clusterService     = "kubernetes.io/cluster-service"
	clusterServiceName = "kubernetes.io/name"
)

var readRollouts = map[string]bool{"status": true, "history": true}

func commandWords(command string) []string {
	fields := strings.Fields(command)
	if len(fields) > 0 && fields[0] == "kubectl" {
		fields = fields[1:]
	}
	return fields
}

func verbOf(fields []string) (string, []string) {
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		switch {
		case f == "-n" || f == "--namespace" || f == "--context" || f == "--kubeconfig":
			i++
		case strings.HasPrefix(f, "-"):
		default:
			return f, fields[i+1:]
		}
	}
	return "", nil
}

func ReadOnlyCommand(command string) bool {
	fields := commandWords(command)
	verb, rest := verbOf(fields)
	if verb == "rollout" {
		sub, _ := verbOf(rest)
		return readRollouts[sub]
	}
	return readVerbs[verb]
}

func touchesSecrets(command string) bool {
	verb, rest := verbOf(commandWords(command))
	if verb != "get" && verb != "describe" {
		return false
	}
	for _, f := range rest {
		for _, part := range strings.Split(strings.ToLower(f), ",") {
			part, _, _ = strings.Cut(part, "/")
			if part == "secret" || part == "secrets" || part == "secret.v1" {
				return true
			}
		}
	}
	return false
}

func (r *Realm) Kubectl(command string) view.CommandResult {
	command = lexicon.CommandFromStory(command)
	trimmed := strings.TrimSpace(command)
	if strings.HasPrefix(trimmed, "etcdctl") || !ReadOnlyCommand(trimmed) {
		return view.CommandResult{Output: ReadOnlyAnswer + "\n"}
	}
	if touchesSecrets(trimmed) {
		return view.CommandResult{Output: noSecrets + "\n"}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	verb, _ := verbOf(commandWords(trimmed))
	if verb == "top" && !r.usage.Available {
		return view.CommandResult{Output: noAugur + "\n"}
	}
	if out, ok := r.answer(verb); ok {
		return view.CommandResult{Output: out + "\n", OK: verb != "api-resources"}
	}
	res := kubectl.Execute(stone{r}, trimmed)
	return view.CommandResult{Output: res.Output, OK: res.OK}
}

func (r *Realm) answer(verb string) (string, bool) {
	switch verb {
	case "version":
		return "Client Version: " + stoneClientVersion + "\nServer Version: " + r.snap.Version, true
	case "cluster-info":
		return r.clusterInfo(), true
	case "api-resources":
		return notReadYet, true
	}
	return "", false
}

func (r *Realm) clusterInfo() string {
	server := strings.TrimRight(r.snap.Context.Server, "/")
	lines := []string{"Kubernetes control plane is running at " + server}
	for _, svc := range store.View[*corev1.Service](r.mirror, store.KindService, "kube-system") {
		if svc.Labels[clusterService] != "true" {
			continue
		}
		name := svc.Labels[clusterServiceName]
		if name == "" {
			name = svc.Name
		}
		target := svc.Name
		if len(svc.Spec.Ports) > 0 && svc.Spec.Ports[0].Name != "" {
			target += ":" + svc.Spec.Ports[0].Name
		}
		lines = append(lines, fmt.Sprintf("%s is running at %s/api/v1/namespaces/kube-system/services/%s/proxy", name, server, target))
	}
	return strings.Join(lines, "\n") + "\n\nTo further debug and diagnose cluster problems, use 'kubectl cluster-info dump'."
}

func (r *Realm) Inspect(command string) view.CommandResult {
	return r.Kubectl(command)
}

func (r *Realm) GameState() view.GameState {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := view.GameState{FoundedAt: r.snap.TakenAt, RealmTime: r.clk.Seconds(), Speed: 1}
	st.Reveals = slices.Clone(liveReveals)
	st.Calamities = r.book.views()
	if len(r.book.chronicles) > 0 {
		st.Chronicles = slices.Clone(r.book.chronicles)
		st.Reveals = append(st.Reveals, "chronicles")
	}
	st.CalmRealm = true
	st.Pilgrims.DayPhase = liveNoon
	return st
}

var liveReveals = []string{"stone", "tidings", "heraldry"}

type stone struct {
	r *Realm
}

func (s stone) Store() *store.Store    { return s.r.mirror }
func (s stone) Clock() *clock.Clock    { return s.r.clk }
func (s stone) Unlocked(string) bool   { return true }
func (s stone) MetricsAvailable() bool { return s.r.usage.Available }
func (s stone) Logs(uid types.UID, previous bool) ([]string, bool) {
	return s.ContainerLogs(uid, "", previous)
}

func (s stone) PodUsage(uid string) (kubectl.Usage, bool) {
	p := s.pod(types.UID(uid))
	if p == nil {
		return kubectl.Usage{}, false
	}
	u, ok := s.r.usage.Pods[p.Namespace+"/"+p.Name]
	return kubectl.Usage{CPUMilli: u.CPUMilli, MemBytes: u.MemBytes}, ok
}

func (s stone) pod(uid types.UID) *corev1.Pod {
	for _, p := range store.View[*corev1.Pod](s.r.mirror, store.KindPod, "") {
		if p.UID == uid {
			return p
		}
	}
	return nil
}

func (s stone) NodeUsage(name string) (kubectl.Usage, bool) {
	u, ok := s.r.usage.Nodes[name]
	return kubectl.Usage{CPUMilli: u.CPUMilli, MemBytes: u.MemBytes}, ok
}

func (s stone) ContainerLogs(uid types.UID, container string, previous bool) ([]string, bool) {
	pod := s.pod(uid)
	if pod == nil || s.r.logs == nil {
		return []string{"(the Chronicle of this hut was not read: this realm was recorded, not opened)"}, s.r.logs == nil && pod != nil
	}
	if container == "" && len(pod.Spec.Containers) > 0 {
		container = pod.Spec.Containers[0].Name
	}
	ctx, cancel := context.WithTimeout(context.Background(), readTimeout)
	defer cancel()
	text, err := s.r.logs.podLogs(ctx, pod.Namespace, pod.Name, container, previous, logTailLines)
	if err != nil {
		if previous {
			return nil, false
		}
		return []string{fmt.Sprintf("(the Chronicle could not be read: %v)", err)}, true
	}
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil, true
	}
	return strings.Split(text, "\n"), true
}

func (r *Realm) CronJobOf(namespace string, owners []metav1.OwnerReference) (*batchv1.CronJob, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ref := range owners {
		if ref.Kind != "Job" {
			continue
		}
		for i := range r.snap.Jobs {
			j := &r.snap.Jobs[i]
			if j.Namespace != namespace || j.UID != ref.UID {
				continue
			}
			for _, up := range j.OwnerReferences {
				if up.Kind != "CronJob" {
					continue
				}
				for k := range r.snap.CronJobs {
					if cj := &r.snap.CronJobs[k]; cj.Namespace == namespace && cj.UID == up.UID {
						return cj.DeepCopy(), true
					}
				}
			}
		}
	}
	return nil, false
}

func (r *Realm) Budgets(namespace string) []policyv1.PodDisruptionBudget {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []policyv1.PodDisruptionBudget
	for i := range r.snap.PDBs {
		if r.snap.PDBs[i].Namespace == namespace {
			out = append(out, *r.snap.PDBs[i].DeepCopy())
		}
	}
	return out
}

func (r *Realm) Identity() Identity {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snap.Identity
}

func (r *Realm) Job(namespace, name string) (*batchv1.Job, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.snap.Jobs {
		if j := &r.snap.Jobs[i]; j.Namespace == namespace && j.Name == name {
			return j.DeepCopy(), true
		}
	}
	return nil, false
}
