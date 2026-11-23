package live_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/view"
)

const loadNode = "worker-a"

func quantities(pairs ...string) corev1.ResourceList {
	out := corev1.ResourceList{}
	for i := 0; i+1 < len(pairs); i += 2 {
		out[corev1.ResourceName(pairs[i])] = resource.MustParse(pairs[i+1])
	}
	return out
}

func loadedNode(name string, conditions ...corev1.NodeCondition) corev1.Node {
	alloc := quantities("cpu", "4", "memory", "8Gi", "pods", "110", "ephemeral-storage", "20Gi")
	return corev1.Node{TypeMeta: metav1.TypeMeta{Kind: "Node", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID("node-" + name), CreationTimestamp: metav1.NewTime(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))},
		Status: corev1.NodeStatus{Capacity: alloc, Allocatable: alloc,
			Conditions: append([]corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}}, conditions...)}}
}

func loadPod(name, node string, phase corev1.PodPhase, spec corev1.PodSpec) corev1.Pod {
	spec.NodeName = node
	return corev1.Pod{TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "shop", UID: types.UID("pod-" + name), ResourceVersion: "1",
			CreationTimestamp: metav1.NewTime(time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC))},
		Spec: spec, Status: corev1.PodStatus{Phase: phase}}
}

func asking(name string, requests corev1.ResourceList) corev1.Container {
	return corev1.Container{Name: name, Image: "nginx:1.27", Resources: corev1.ResourceRequirements{Requests: requests}}
}

func loadSnapshot(pods ...corev1.Pod) live.Snapshot {
	return live.Snapshot{Version: "v1.37.0", TakenAt: time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC),
		Nodes:      []corev1.Node{loadedNode(loadNode), loadedNode("worker-b", corev1.NodeCondition{Type: corev1.NodeDiskPressure, Status: corev1.ConditionTrue})},
		Namespaces: []corev1.Namespace{{ObjectMeta: metav1.ObjectMeta{Name: "shop", UID: "ns-shop"}}},
		Pods:       pods}
}

func schedulerPods() []corev1.Pod {
	always := corev1.ContainerRestartPolicyAlways
	sidecar := asking("proxy", quantities("cpu", "50m", "memory", "32Mi"))
	sidecar.RestartPolicy = &always
	resized := loadPod("resized", loadNode, corev1.PodRunning, corev1.PodSpec{Containers: []corev1.Container{asking("app", quantities("cpu", "100m"))}})
	resized.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "app", AllocatedResources: quantities("cpu", "300m")}}
	return []corev1.Pod{
		loadPod("init-heavy", loadNode, corev1.PodRunning, corev1.PodSpec{
			InitContainers: []corev1.Container{asking("migrate", quantities("cpu", "500m", "memory", "128Mi"))},
			Containers:     []corev1.Container{asking("app", quantities("cpu", "100m", "memory", "64Mi", "ephemeral-storage", "1Gi"))}}),
		loadPod("with-sidecar", loadNode, corev1.PodRunning, corev1.PodSpec{
			InitContainers: []corev1.Container{sidecar, asking("warm", quantities("cpu", "300m"))},
			Containers:     []corev1.Container{asking("app", quantities("cpu", "200m", "memory", "100Mi"))}}),
		loadPod("overhead", loadNode, corev1.PodPending, corev1.PodSpec{Overhead: quantities("cpu", "10m", "memory", "16Mi"),
			Containers: []corev1.Container{asking("app", quantities("cpu", "100m"))}}),
		loadPod("pod-level", loadNode, corev1.PodRunning, corev1.PodSpec{Resources: &corev1.ResourceRequirements{Requests: quantities("cpu", "1")},
			Containers: []corev1.Container{asking("app", quantities("cpu", "100m", "memory", "10Mi"))}}),
		resized,
		loadPod("done", loadNode, corev1.PodSucceeded, corev1.PodSpec{Containers: []corev1.Container{asking("app", quantities("cpu", "2"))}}),
		loadPod("broken", loadNode, corev1.PodFailed, corev1.PodSpec{Containers: []corev1.Container{asking("app", quantities("cpu", "2"))}}),
		loadPod("unplaced", "", corev1.PodPending, corev1.PodSpec{Containers: []corev1.Container{asking("app", quantities("cpu", "2"))}}),
		loadPod("elsewhere", "worker-b", corev1.PodRunning, corev1.PodSpec{Containers: []corev1.Container{asking("app", quantities("cpu", "250m", "memory", "1Gi", "ephemeral-storage", "2Gi"))}}),
	}
}

func TestANodesRequestsAreCountedAsTheSchedulerCountsThem(t *testing.T) {
	r, err := live.Open(loadSnapshot(schedulerPods()...), nil, live.Usage{})
	if err != nil {
		t.Fatal(err)
	}
	m := r.Metrics()
	got := m.Nodes[loadNode]
	mi := int64(1 << 20)
	want := view.NodeMetrics{CPUAllocatableMilli: 4000, MemoryAllocatableBytes: 8 << 30,
		CPURequestedMilli: 500 + 350 + 110 + 1000 + 300, MemoryRequestedBytes: 128*mi + 132*mi + 16*mi + 10*mi,
		Pods: 5, PodsAllocatable: 110, EphemeralRequestedBytes: 1 << 30, EphemeralAllocatableBytes: 20 << 30}
	if got != want {
		t.Errorf("node %s reads\n %+v\nwant\n %+v", loadNode, got, want)
	}
	other := m.Nodes["worker-b"]
	if other.CPURequestedMilli != 250 || other.Pods != 1 || !other.DiskPressure || other.EphemeralRequestedBytes != 2<<30 {
		t.Errorf("worker-b reads %+v, want 250m requested, 1 Pod, 2Gi ephemeral requested and DiskPressure", other)
	}
	if m.Augury || got.Measured || got.CPUMilli != 0 {
		t.Errorf("no metrics-server answered, yet the realm reads usage: %+v", m)
	}
}

func TestUsageIsKnownOnlyForTheNodesMetricsServerAnswersFor(t *testing.T) {
	usage := live.Usage{Available: true, Nodes: map[string]live.UsageOf{loadNode: {CPUMilli: 1234, MemBytes: 2 << 30}},
		Pods: map[string]live.UsageOf{"shop/init-heavy": {CPUMilli: 40, MemBytes: 50 << 20}}}
	r, err := live.Open(loadSnapshot(schedulerPods()...), nil, usage)
	if err != nil {
		t.Fatal(err)
	}
	m := r.Metrics()
	if a := m.Nodes[loadNode]; !a.Measured || a.CPUMilli != 1234 || a.MemoryBytes != 2<<30 {
		t.Errorf("%s reads %+v, want its metrics-server usage", loadNode, a)
	}
	if b := m.Nodes["worker-b"]; b.Measured || b.CPUMilli != 0 {
		t.Errorf("metrics-server did not answer for worker-b, yet it reads %+v", b)
	}
	if p := m.Pods["shop/init-heavy"]; p.CPUMilli != 40 {
		t.Errorf("the Pod's usage reads %+v", p)
	}
}

func TestTheNumbersFollowTheWatchAndAreNotCountedOnEveryRead(t *testing.T) {
	pods := schedulerPods()
	r, err := live.Open(loadSnapshot(pods...), nil, live.Usage{})
	if err != nil {
		t.Fatal(err)
	}
	before := r.Metrics().Nodes[loadNode]
	more := loadPod("late", loadNode, corev1.PodRunning, corev1.PodSpec{Containers: []corev1.Container{asking("app", quantities("cpu", "400m", "memory", "1Gi"))}})
	more.ResourceVersion = "9"
	if err := r.Replay("pods", strings.NewReader(string(live.WatchEvent("ADDED", &more)))); err != nil {
		t.Fatal(err)
	}
	r.Pump(live.PumpEvery)
	after := r.Metrics().Nodes[loadNode]
	if after.CPURequestedMilli != before.CPURequestedMilli+400 || after.MemoryRequestedBytes != before.MemoryRequestedBytes+1<<30 || after.Pods != before.Pods+1 {
		t.Errorf("after a new Pod the node reads %+v, before %+v", after, before)
	}
	big := make([]corev1.Pod, 0, 3000)
	for i := range 3000 {
		big = append(big, loadPod(fmt.Sprintf("many-%d", i), loadNode, corev1.PodRunning,
			corev1.PodSpec{Containers: []corev1.Container{asking("app", quantities("cpu", "1m"))}}))
	}
	r, err = live.Open(loadSnapshot(big...), nil, live.Usage{})
	if err != nil {
		t.Fatal(err)
	}
	if allocs := testing.AllocsPerRun(20, func() { r.Metrics() }); allocs > 1 {
		t.Errorf("one read of the metrics allocates %.0f times over 3000 Pods: it counts them again on every read", allocs)
	}
}

func TestTheControlPlaneHoldsTheRequestsOfItsOwnPodsOnly(t *testing.T) {
	r, err := live.OpenRecorded(kindFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	m := r.Metrics().Nodes
	if cp := m["kubernites-control-plane"]; cp.ControlPlaneCPUMilli != 650 || cp.ControlPlaneMemoryBytes != 100<<20 {
		t.Errorf("the control-plane node holds %dm and %d bytes for the control plane, want 650m and 100Mi (etcd, kube-apiserver, kube-controller-manager, kube-scheduler)",
			cp.ControlPlaneCPUMilli, cp.ControlPlaneMemoryBytes)
	}
	for _, name := range []string{"kubernites-worker", "kubernites-worker2"} {
		if w := m[name]; w.ControlPlaneCPUMilli != 0 || w.ControlPlaneMemoryBytes != 0 || w.CPURequestedMilli == 0 {
			t.Errorf("%s reads %+v: kube-proxy and the CNI are not the control plane, and its Pods request CPU", name, w)
		}
	}
}
