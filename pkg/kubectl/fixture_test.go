package kubectl

import (
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/waxb/kubernites/pkg/sim/clock"
	"github.com/waxb/kubernites/pkg/sim/store"
)

type fakeCluster struct {
	st       *store.Store
	clk      *clock.Clock
	unlocks  map[string]bool
	logs     map[types.UID][]string
	prev     map[types.UID][]string
	usage    map[string]Usage
	sidecar  map[string][]string
	teaching bool
}

func (f *fakeCluster) Teaching() bool { return f.teaching }

func (f *fakeCluster) Store() *store.Store     { return f.st }
func (f *fakeCluster) Clock() *clock.Clock     { return f.clk }
func (f *fakeCluster) Unlocked(id string) bool { return f.unlocks[id] }
func (f *fakeCluster) MetricsAvailable() bool  { return f.unlocks["augury"] }
func (f *fakeCluster) PodUsage(uid string) (Usage, bool) {
	u, ok := f.usage[uid]
	return u, ok
}
func (f *fakeCluster) NodeUsage(name string) (Usage, bool) {
	u, ok := f.usage["node:"+name]
	return u, ok
}
func (f *fakeCluster) Logs(uid types.UID, previous bool) ([]string, bool) {
	if previous {
		l, ok := f.prev[uid]
		return l, ok
	}
	l, ok := f.logs[uid]
	return l, ok
}

func newCluster(t *testing.T) *fakeCluster {
	t.Helper()
	clk := clock.New(time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC))
	st := withAccounts(store.New(clk.MetaNow))
	f := &fakeCluster{st: st, clk: clk, unlocks: map[string]bool{}, logs: map[types.UID][]string{}, prev: map[types.UID][]string{}, usage: map[string]Usage{}, sidecar: map[string][]string{}}
	res := corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2"), corev1.ResourceMemory: resource.MustParse("4Gi"), corev1.ResourcePods: resource.MustParse("110")}
	_, _ = st.Create(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "lone-tower", Labels: map[string]string{"node-role.kubernetes.io/control-plane": ""}},
		Spec:       corev1.NodeSpec{Taints: []corev1.Taint{{Key: "node-role.kubernetes.io/control-plane", Effect: corev1.TaintEffectNoSchedule}}},
		Status: corev1.NodeStatus{Capacity: res, Allocatable: res, NodeInfo: corev1.NodeSystemInfo{KubeletVersion: "v1.37.0"},
			Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}}},
	})
	_, _ = st.Create(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "default"}, Status: corev1.NamespaceStatus{Phase: corev1.NamespaceActive}})
	clk.Advance(5)
	return f
}

func (f *fakeCluster) runningPod(t *testing.T, name string, restarts int32) *corev1.Pod {
	t.Helper()
	started := f.clk.MetaNow()
	p, err := f.st.Create(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", Labels: map[string]string{"app": "lantern"}},
		Spec: corev1.PodSpec{NodeName: "lone-tower", Containers: []corev1.Container{{Name: "lantern", Image: "porter:1.0",
			Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m")}}}}},
		Status: corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.244.0.2",
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
			ContainerStatuses: []corev1.ContainerStatus{{Name: "lantern", Ready: true, RestartCount: restarts, Image: "porter:1.0",
				State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: started}}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return p.(*corev1.Pod)
}

func run(f *fakeCluster, cmd string) Result {
	return Execute(f, cmd)
}

func (f *fakeCluster) wardenSet(t *testing.T, status appsv1.DaemonSetStatus) *appsv1.DaemonSet {
	t.Helper()
	labels := map[string]string{"app": "warden"}
	one, zero := intstr.FromInt32(1), intstr.FromInt32(0)
	created, err := f.st.Create(&appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "warden", Namespace: "default", Labels: labels},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{NodeSelector: map[string]string{"trade": "couriers"},
					Containers: []corev1.Container{{Name: "warden", Image: "warden:1.0"}}},
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{Type: appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{MaxUnavailable: &one, MaxSurge: &zero}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ds := created.(*appsv1.DaemonSet).DeepCopy()
	ds.Status = status
	updated, _ := f.st.UpdateStatus(ds)
	return updated.(*appsv1.DaemonSet)
}

func mustStatus(t *testing.T, f *fakeCluster, ds *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
	t.Helper()
	updated, err := f.st.UpdateStatus(ds)
	if err != nil {
		t.Fatal(err)
	}
	return updated.(*appsv1.DaemonSet), nil
}

func mustRun(t *testing.T, f *fakeCluster, cmd string) string {
	t.Helper()
	res := run(f, cmd)
	if !res.OK {
		t.Fatalf("%s: %s", cmd, res.Output)
	}
	return res.Output
}

func (f *fakeCluster) placed(t *testing.T, ns, name, node string, mutate func(*corev1.Pod)) {
	t.Helper()
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       corev1.PodSpec{NodeName: node, Containers: []corev1.Container{{Name: "c", Image: "porter:1.0"}}},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	if mutate != nil {
		mutate(p)
	}
	if _, err := f.st.Create(p); err != nil {
		t.Fatal(err)
	}
}

func mirrored(node string) func(*corev1.Pod) {
	return func(p *corev1.Pod) {
		yes := true
		p.Labels = map[string]string{"app": "lantern"}
		p.Annotations = map[string]string{
			"kubernetes.io/config.source": "file", "kubernetes.io/config.hash": "9c4e", corev1.MirrorPodAnnotationKey: "9c4e",
		}
		p.OwnerReferences = []metav1.OwnerReference{{APIVersion: "v1", Kind: "Node", Name: node, UID: "node-uid", Controller: &yes}}
		p.Spec.Containers[0].Ports = []corev1.ContainerPort{{Name: "http", ContainerPort: 8080, HostPort: 80, Protocol: corev1.ProtocolTCP}}
	}
}
