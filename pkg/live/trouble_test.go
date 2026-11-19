package live

import (
	"encoding/json"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/view"
)

var calmBorn = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)

func ownedBy(kind, name, uid string) []metav1.OwnerReference {
	yes := true
	return []metav1.OwnerReference{{APIVersion: "apps/v1", Kind: kind, Name: name, UID: types.UID(uid), Controller: &yes}}
}

func runningPod(name, rs, hash string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "shop", UID: types.UID("uid-" + name), CreationTimestamp: metav1.NewTime(calmBorn),
			Labels: map[string]string{"app": "web", podTemplateHash: hash}, OwnerReferences: ownedBy("ReplicaSet", rs, "uid-"+rs)},
		Spec: corev1.PodSpec{NodeName: "worker-1", Containers: []corev1.Container{{Name: "app", Image: "registry.example/web:1.0",
			Resources: corev1.ResourceRequirements{Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("64Mi")}}}}},
		Status: corev1.PodStatus{Phase: corev1.PodRunning, StartTime: &metav1.Time{Time: calmBorn},
			Conditions:        []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}, {Type: corev1.PodScheduled, Status: corev1.ConditionTrue}},
			ContainerStatuses: []corev1.ContainerStatus{{Name: "app", Ready: true, Image: "registry.example/web:1.0", State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}},
	}
}

func calmSnapshot() Snapshot {
	two, five := int32(2), int32(5)
	class := "standard"
	s := Snapshot{TakenAt: calmBorn.Add(time.Hour), Version: "v1.37.0"}
	s.Nodes = []corev1.Node{{ObjectMeta: metav1.ObjectMeta{Name: "worker-1", UID: "uid-node-1", CreationTimestamp: metav1.NewTime(calmBorn)},
		Status: corev1.NodeStatus{Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue, Reason: "KubeletReady"},
			{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionFalse}},
			Allocatable: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4"), corev1.ResourceMemory: resource.MustParse("8Gi")}}}}
	s.Namespaces = []corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: "kube-system", UID: "uid-ns-ks"}}, {ObjectMeta: metav1.ObjectMeta{Name: "shop", UID: "uid-ns-shop"}}}
	s.Deployments = []appsv1.Deployment{{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "shop", UID: "uid-web"}, Spec: appsv1.DeploymentSpec{Replicas: &two},
		Status: appsv1.DeploymentStatus{Replicas: 2, UpdatedReplicas: 2, ReadyReplicas: 2, AvailableReplicas: 2}}}
	s.ReplicaSets = []appsv1.ReplicaSet{{ObjectMeta: metav1.ObjectMeta{Name: "web-abc", Namespace: "shop", UID: "uid-web-abc", Labels: map[string]string{podTemplateHash: "abc"},
		Annotations: map[string]string{revisionNote: "1"}, OwnerReferences: ownedBy("Deployment", "web", "uid-web")}, Status: appsv1.ReplicaSetStatus{Replicas: 2}}}
	s.Pods = []corev1.Pod{runningPod("web-abc-1", "web-abc", "abc"), runningPod("web-abc-2", "web-abc", "abc")}
	s.Jobs = []batchv1.Job{{ObjectMeta: metav1.ObjectMeta{Name: "nightly", Namespace: "shop", UID: "uid-nightly"}}}
	s.HPAs = []autoscalingv2.HorizontalPodAutoscaler{{ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "shop", UID: "uid-hpa"},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{MaxReplicas: five, ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{Kind: "Deployment", Name: "web"}}}}
	s.PVCs = []corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{Name: "data", Namespace: "shop", UID: "uid-pvc"}, Spec: corev1.PersistentVolumeClaimSpec{StorageClassName: &class},
		Status: corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound}}}
	return s
}

func warning(name, kind, ns, object, uid, reason, message string) corev1.Event {
	return corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, UID: types.UID("uid-ev-" + name)},
		InvolvedObject: corev1.ObjectReference{Kind: kind, Namespace: ns, Name: object, UID: types.UID(uid)}, Reason: reason, Message: message, Type: corev1.EventTypeWarning,
		Count: 3, FirstTimestamp: metav1.NewTime(calmBorn.Add(50 * time.Minute)), LastTimestamp: metav1.NewTime(calmBorn.Add(55 * time.Minute))}
}

func waiting(p *corev1.Pod, reason, message string, last *corev1.ContainerStateTerminated) {
	cs := &p.Status.ContainerStatuses[0]
	cs.Ready, cs.RestartCount = false, 4
	cs.State = corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason, Message: message}}
	if last != nil {
		cs.LastTerminationState = corev1.ContainerState{Terminated: last}
	}
	p.Status.Conditions[0].Status = corev1.ConditionFalse
}

const (
	crashWords   = "back-off 40s restarting failed container=app pod=web-abc-1_shop(uid-web-abc-1)"
	pullWords    = "Back-off pulling image \"registry.example/web:0.0-never\""
	schedWords   = "0/1 nodes are available: 1 Insufficient memory. preemption: 0/1 nodes are available: 1 No preemption victims found for incoming pod."
	evictWords   = "Pod ephemeral local storage usage exceeds the total limit of containers 10Mi. "
	probeWords   = "Readiness probe failed: HTTP probe failed with statuscode: 503"
	nodeWords    = "Kubelet stopped posting node status."
	jobWords     = "Job has reached the specified backoff limit"
	claimWords   = "storageclass.storage.k8s.io \"nope\" not found"
	hornWords    = "the desired replica count is more than the maximum replica count"
	progressWord = "ReplicaSet \"web-def\" has timed out progressing."
)

type signalCase struct {
	name   string
	kind   string
	words  string
	break_ func(*Snapshot)
}

func ended(reason string, code int32) *corev1.ContainerStateTerminated {
	return &corev1.ContainerStateTerminated{Reason: reason, ExitCode: code, FinishedAt: metav1.NewTime(calmBorn.Add(58 * time.Minute))}
}

func unschedulable(p *corev1.Pod) {
	p.Spec.NodeName = ""
	p.Status = corev1.PodStatus{Phase: corev1.PodPending, Conditions: []corev1.PodCondition{{Type: corev1.PodScheduled, Status: corev1.ConditionFalse, Reason: reasonUnschedule, Message: schedWords}}}
}

var signalCases = []signalCase{
	{"a container waiting in CrashLoopBackOff", view.TagCrashLoop, crashWords, func(s *Snapshot) {
		waiting(&s.Pods[0], reasonCrashLoop, crashWords, ended("Error", 1))
	}},
	{"a container whose last state is OOMKilled", view.TagOOMKilled, "OOMKilled: container \"app\" was terminated with exit code 137", func(s *Snapshot) {
		waiting(&s.Pods[0], reasonCrashLoop, crashWords, ended(reasonOOMKilled, 137))
	}},
	{"a container waiting in ImagePullBackOff", view.TagImagePull, pullWords, func(s *Snapshot) {
		waiting(&s.Pods[0], reasonPullBackOff, pullWords, nil)
	}},
	{"a container waiting in ErrImagePull", view.TagImagePull, "ErrImagePull: rpc error", func(s *Snapshot) {
		waiting(&s.Pods[0], reasonErrPull, "rpc error", nil)
	}},
	{"a Pod Pending with a FailedScheduling Event", view.TagUnhoused, schedWords, func(s *Snapshot) {
		p := runningPod("web-abc-3", "web-abc", "abc")
		unschedulable(&p)
		s.Pods = append(s.Pods, p)
		s.Events = append(s.Events, warning("sched", "Pod", "shop", p.Name, string(p.UID), reasonFailedSched, schedWords))
	}},
	{"a Pod failed with reason Evicted", view.TagEvicted, evictWords, func(s *Snapshot) {
		s.Pods[1].Status = corev1.PodStatus{Phase: corev1.PodFailed, Reason: reasonEvicted, Message: evictWords}
	}},
	{"a roll-out held by Pending new Pods", view.CalamityStalled, progressWord, func(s *Snapshot) {
		three := int32(3)
		d := &s.Deployments[0]
		d.Spec.Replicas = &three
		d.Status = appsv1.DeploymentStatus{Replicas: 3, UpdatedReplicas: 1, ReadyReplicas: 2,
			Conditions: []appsv1.DeploymentCondition{{Type: appsv1.DeploymentProgressing, Status: corev1.ConditionFalse, Reason: reasonDeadline, Message: progressWord}}}
		s.ReplicaSets = append(s.ReplicaSets, appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "web-def", Namespace: "shop", UID: "uid-web-def",
			Labels: map[string]string{podTemplateHash: "def"}, Annotations: map[string]string{revisionNote: "2"}, OwnerReferences: ownedBy("Deployment", "web", "uid-web")},
			Status: appsv1.ReplicaSetStatus{Replicas: 1}})
		p := runningPod("web-def-1", "web-def", "def")
		unschedulable(&p)
		s.Pods = append(s.Pods, p)
	}},
	{"failing probes", view.ReportHut, probeWords, func(s *Snapshot) {
		s.Pods[0].Status.ContainerStatuses[0].Ready = false
		s.Events = append(s.Events, warning("probe", "Pod", "shop", "web-abc-1", "uid-web-abc-1", reasonUnhealthy, probeWords))
	}},
	{"a NotReady node", view.ReportKeep, nodeWords, func(s *Snapshot) {
		s.Nodes[0].Status.Conditions[0] = corev1.NodeCondition{Type: corev1.NodeReady, Status: corev1.ConditionUnknown, Reason: "NodeStatusUnknown", Message: nodeWords}
	}},
	{"a node under MemoryPressure", view.ReportKeep, "MemoryPressure: KubeletHasInsufficientMemory: kubelet has insufficient memory available", func(s *Snapshot) {
		s.Nodes[0].Status.Conditions[1] = corev1.NodeCondition{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue, Reason: "KubeletHasInsufficientMemory", Message: "kubelet has insufficient memory available"}
	}},
	{"a cordoned node", view.ReportKeep, "SchedulingDisabled", func(s *Snapshot) {
		s.Nodes[0].Spec.Unschedulable = true
	}},
	{"a failed Job", view.ReportQuest, jobWords, func(s *Snapshot) {
		s.Jobs[0].Status = batchv1.JobStatus{Failed: 7, Conditions: []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Reason: "BackoffLimitExceeded", Message: jobWords}}}
	}},
	{"a Pending PVC", view.ReportDeed, claimWords, func(s *Snapshot) {
		s.PVCs[0].Status.Phase = corev1.ClaimPending
		s.Events = append(s.Events, warning("claim", "PersistentVolumeClaim", "shop", "data", "uid-pvc", "ProvisioningFailed", claimWords))
	}},
	{"an HPA held at its maximum", view.ReportHorn, hornWords, func(s *Snapshot) {
		s.HPAs[0].Status = autoscalingv2.HorizontalPodAutoscalerStatus{CurrentReplicas: 5, DesiredReplicas: 5,
			Conditions: []autoscalingv2.HorizontalPodAutoscalerCondition{{Type: autoscalingv2.ScalingLimited, Status: corev1.ConditionTrue, Reason: reasonTooMany, Message: hornWords}}}
	}},
}

var neverLive = []string{view.TagThrottled, view.CalamityFrozen, view.CalamityWarm, view.CalamityHornLow}

func TestACalmClusterRaisesNoSignal(t *testing.T) {
	s := calmSnapshot()
	if got := Signals(&s); len(got) != 0 {
		t.Fatalf("a calm cluster raised %+v", got)
	}
}

func TestEachSignalShowsItsCampaignCardInTheClustersOwnWords(t *testing.T) {
	for _, c := range signalCases {
		t.Run(c.name, func(t *testing.T) {
			s := calmSnapshot()
			c.break_(&s)
			got := Signals(&s)
			if len(got) != 1 {
				t.Fatalf("%d signals: %+v", len(got), got)
			}
			v := got[0]
			if v.Kind != c.kind || !strings.Contains(v.Cause+" "+v.Why, c.words) {
				t.Errorf("kind %q cause %q why %q, want %q with %q", v.Kind, v.Cause, v.Why, c.kind, c.words)
			}
			if slices.Contains(neverLive, v.Kind) {
				t.Errorf("a live realm raised %s", v.Kind)
			}
			if len(v.Remedies) == 0 || !ReadOnlyCommand(v.Remedies[0].Command) || v.Remedies[0].Unwired {
				t.Errorf("the first remedy is not a read: %+v", v.Remedies)
			}
			for _, r := range v.Remedies[1:] {
				if !r.Unwired || ReadOnlyCommand(r.Command) || !strings.HasPrefix(r.Command, "kubectl ") {
					t.Errorf("a remedy that changes the cluster is not shown as an unwired command: %+v", r)
				}
			}
			calm := calmSnapshot()
			if again := Signals(&calm); len(again) != 0 {
				t.Errorf("undone, it still stands: %+v", again)
			}
		})
	}
}

func TestOneCalamityPerCrewListsItsHuts(t *testing.T) {
	s := calmSnapshot()
	waiting(&s.Pods[0], reasonCrashLoop, crashWords, ended("Error", 1))
	waiting(&s.Pods[1], reasonCrashLoop, "back-off 20s restarting failed container=app pod=web-abc-2_shop(uid-web-abc-2)", ended("Error", 1))
	got := Signals(&s)
	if len(got) != 1 || !strings.Contains(got[0].Why, "2 Pods of deployment shop/web: web-abc-1, web-abc-2") {
		t.Fatalf("signals %+v", got)
	}
	if got[0].ID != view.TagCrashLoop+":shop/Deployment/web" {
		t.Errorf("id %q", got[0].ID)
	}
}

func TestSignalsAreAFunctionOfTheMirrorAlone(t *testing.T) {
	s := calmSnapshot()
	for _, c := range signalCases {
		if c.kind != view.CalamityStalled {
			c.break_(&s)
		}
	}
	want := Signals(&s)
	if len(want) < 6 {
		t.Fatalf("only %d signals from every case", len(want))
	}
	rng := rand.New(rand.NewPCG(19, 36))
	for range 50 {
		shuffled := s.clone()
		rng.Shuffle(len(shuffled.Pods), func(i, j int) { shuffled.Pods[i], shuffled.Pods[j] = shuffled.Pods[j], shuffled.Pods[i] })
		rng.Shuffle(len(shuffled.Events), func(i, j int) { shuffled.Events[i], shuffled.Events[j] = shuffled.Events[j], shuffled.Events[i] })
		rng.Shuffle(len(shuffled.Nodes), func(i, j int) { shuffled.Nodes[i], shuffled.Nodes[j] = shuffled.Nodes[j], shuffled.Nodes[i] })
		if got := Signals(&shuffled); !equalViews(got, want) {
			t.Fatalf("the same mirror in another order gave other signals:\n%+v\n%+v", got, want)
		}
	}
}

func equalViews(a, b []view.CalamityView) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

func kindNamed(t testing.TB, name string) kindRead {
	t.Helper()
	for _, k := range kindReads {
		if k.name == name {
			return k
		}
	}
	t.Fatalf("no kind %s", name)
	return kindRead{}
}

func stream(t testing.TB, r *Realm, kind, typ string, obj any) {
	t.Helper()
	raw, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ingest(kindNamed(t, kind), typ, raw); err != nil {
		t.Fatal(err)
	}
}

func openCalm(t *testing.T) *Realm {
	t.Helper()
	r, err := OpenRecorded(calmSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func liveCalamities(r *Realm) []view.CalamityView {
	return r.GameState().Calamities
}

func TestTroubleAtConnectIsQuietAndOnlyNewTroubleOpensACard(t *testing.T) {
	s := calmSnapshot()
	waiting(&s.Pods[0], reasonPullBackOff, pullWords, nil)
	r, err := OpenRecorded(s)
	if err != nil {
		t.Fatal(err)
	}
	got := liveCalamities(r)
	if len(got) != 1 || !got[0].Quiet || got[0].Kind != view.TagImagePull {
		t.Fatalf("at connect: %+v", got)
	}
	p := s.Pods[1]
	waiting(&p, reasonCrashLoop, crashWords, ended("Error", 1))
	p.OwnerReferences = nil
	stream(t, r, "pods", watchModified, p)
	r.Pump(PumpEvery)
	got = liveCalamities(r)
	if len(got) != 2 || !got[0].Quiet || got[1].Quiet || got[1].Kind != view.TagCrashLoop {
		t.Fatalf("after a crash while watching: %+v", got)
	}
}

func TestAMendFromTheWatchBecomesAChronicleOfWoe(t *testing.T) {
	r := openCalm(t)
	broken := calmSnapshot().Pods[0]
	waiting(&broken, reasonCrashLoop, crashWords, ended(reasonOOMKilled, 137))
	stream(t, r, "pods", watchModified, broken)
	r.Pump(PumpEvery)
	if got := liveCalamities(r); len(got) != 1 || got[0].Quiet || got[0].Mended != "" || got[0].Kind != view.TagOOMKilled {
		t.Fatalf("struck: %+v", got)
	}
	stream(t, r, "pods", watchModified, calmSnapshot().Pods[0])
	r.Pump(PumpEvery)
	if got := liveCalamities(r); len(got) != 1 || got[0].Mended != "" {
		t.Fatalf("mended before it settled: %+v", got)
	}
	r.Pump(MendSettle)
	got := liveCalamities(r)
	if len(got) != 1 || got[0].Mended != "2 of 2 Pods of deployment shop/web are Ready." {
		t.Fatalf("after the watch saw it mend: %+v", got)
	}
	st := r.GameState()
	if len(st.Chronicles) != 1 || !slices.Contains(st.Reveals, "chronicles") {
		t.Fatalf("chronicles %+v reveals %v", st.Chronicles, st.Reveals)
	}
	c := st.Chronicles[0]
	if !c.Plain || c.Question != "" || len(c.Options) != 0 || !c.Resolved || c.Struck != "09:58 UTC" || c.Healed == "" ||
		!strings.HasPrefix(c.Explanation, "OOMKilled: container \"app\"") || c.MendLine != "2 of 2 Pods of deployment shop/web are Ready." || !strings.Contains(c.Subject, "deployment shop/web") {
		t.Fatalf("the history entry %+v", c)
	}
	if _, err := r.Perform(view.Action{Action: "chronicle", ID: c.ID, Answer: "a"}); err == nil {
		t.Error("a live history entry took an answer")
	}
	r.Pump(MendedFor)
	if got := liveCalamities(r); len(got) != 0 {
		t.Errorf("a mended card stays: %+v", got)
	}
}

func TestAShortRecoveryInsideACrashLoopIsNotAMend(t *testing.T) {
	r := openCalm(t)
	broken := calmSnapshot().Pods[0]
	waiting(&broken, reasonCrashLoop, crashWords, ended("Error", 1))
	stream(t, r, "pods", watchModified, broken)
	r.Pump(PumpEvery)
	stream(t, r, "pods", watchModified, calmSnapshot().Pods[0])
	r.Pump(MendSettle / 3)
	stream(t, r, "pods", watchModified, broken)
	r.Pump(MendSettle)
	got := liveCalamities(r)
	if len(got) != 1 || got[0].Mended != "" || len(r.GameState().Chronicles) != 0 {
		t.Fatalf("a crash loop's running spell mended it: %+v", got)
	}
}

func TestTimeAloneNeverRaisesACalamity(t *testing.T) {
	s := calmSnapshot()
	for _, c := range signalCases[:5] {
		c.break_(&s)
	}
	r, err := OpenRecorded(s)
	if err != nil {
		t.Fatal(err)
	}
	before := liveCalamities(r)
	for range 200 {
		r.Pump(17)
	}
	if after := liveCalamities(r); !equalViews(before, after) {
		t.Fatalf("time changed the calamities:\n%+v\n%+v", before, after)
	}
}

func TestCalamitiesFollowTheMirrorAndNothingElse(t *testing.T) {
	cases := append([]signalCase{{name: "calm", break_: func(*Snapshot) {}}}, signalCases...)
	rng := rand.New(rand.NewPCG(36, 19))
	steps := make([]int, 120)
	for i := range steps {
		steps[i] = rng.IntN(len(cases))
	}
	play := func() ([]string, []view.CalamityView) {
		r := openCalm(t)
		dur := rand.New(rand.NewPCG(7, 7))
		raised := map[string]bool{}
		var seen []string
		for _, i := range steps {
			s := calmSnapshot()
			cases[i].break_(&s)
			replaceMirror(t, r, s)
			for _, v := range Signals(&s) {
				raised[v.ID] = true
			}
			r.Pump(float64(1 + dur.IntN(40)))
			for _, v := range liveCalamities(r) {
				if !raised[v.ID] {
					t.Fatalf("%s stands but the mirror never raised it", v.ID)
				}
				if v.Mended == "" && !slices.ContainsFunc(Signals(&s), func(w view.CalamityView) bool { return w.ID == v.ID }) && !r.book.pending(v.ID) {
					t.Fatalf("%s stands unmended with no signal and no mend pending", v.ID)
				}
				seen = append(seen, v.ID+"/"+v.Mended)
			}
		}
		return seen, liveCalamities(r)
	}
	a, endA := play()
	b, endB := play()
	if !slices.Equal(a, b) || !equalViews(endA, endB) {
		t.Fatal("the same mirror changes gave other calamities")
	}
}

func replaceMirror(t *testing.T, r *Realm, s Snapshot) {
	t.Helper()
	cur := r.Snapshot()
	for _, k := range []struct {
		name string
		objs func(*Snapshot) []any
		uid  func(any) types.UID
	}{
		{"nodes", func(s *Snapshot) []any { return anys(s.Nodes) }, nil},
		{"pods", func(s *Snapshot) []any { return anys(s.Pods) }, nil},
		{"deployments", func(s *Snapshot) []any { return anys(s.Deployments) }, nil},
		{"replicasets", func(s *Snapshot) []any { return anys(s.ReplicaSets) }, nil},
		{"jobs", func(s *Snapshot) []any { return anys(s.Jobs) }, nil},
		{"horizontalpodautoscalers", func(s *Snapshot) []any { return anys(s.HPAs) }, nil},
		{"persistentvolumeclaims", func(s *Snapshot) []any { return anys(s.PVCs) }, nil},
		{"events", func(s *Snapshot) []any { return anys(s.Events) }, nil},
	} {
		next := map[types.UID]bool{}
		for _, o := range k.objs(&s) {
			next[uidOf(o)] = true
			stream(t, r, k.name, watchModified, o)
		}
		for _, o := range k.objs(&cur) {
			if !next[uidOf(o)] {
				stream(t, r, k.name, watchDeleted, o)
			}
		}
	}
}

func anys[T any](list []T) []any {
	out := make([]any, len(list))
	for i := range list {
		out[i] = &list[i]
	}
	return out
}

func uidOf(o any) types.UID {
	return o.(metav1.Object).GetUID()
}

func TestAChronicleOfTroubleWithNoClusterTimeSaysWhenTheWatchSawIt(t *testing.T) {
	r := openCalm(t)
	r.Pump(3600)
	node := calmSnapshot().Nodes[0]
	node.Spec.Unschedulable = true
	stream(t, r, "nodes", watchModified, node)
	r.Pump(PumpEvery)
	stream(t, r, "nodes", watchModified, calmSnapshot().Nodes[0])
	r.Pump(PumpEvery)
	r.Pump(MendSettle)
	st := r.GameState()
	if len(st.Chronicles) != 1 || st.Chronicles[0].Struck != "11:00 UTC" {
		t.Fatalf("chronicles %+v", st.Chronicles)
	}
}
