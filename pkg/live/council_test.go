package live_test

import (
	"reflect"
	"slices"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/view"
)

func councilMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: name, Namespace: "shop", UID: types.UID("uid-" + name), CreationTimestamp: metav1.NewTime(time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC))}
}

func councilPods() []corev1.Pod {
	unschedulable := loadPod("web-7c9-x2", "", corev1.PodPending, corev1.PodSpec{Containers: []corev1.Container{asking("app", quantities("cpu", "8"))}})
	unschedulable.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodScheduled, Status: corev1.ConditionFalse, Reason: corev1.PodReasonUnschedulable,
		Message: "0/3 nodes are available: 3 Insufficient cpu."}}
	ready := loadPod("api-5d-a1", loadNode, corev1.PodRunning, corev1.PodSpec{Containers: []corev1.Container{asking("app", nil)}})
	ready.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}
	unready := loadPod("api-5d-b2", loadNode, corev1.PodRunning, corev1.PodSpec{Containers: []corev1.Container{asking("app", nil)}})
	unready.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionFalse}}
	return []corev1.Pod{unschedulable,
		loadPod("db-0", "", corev1.PodPending, corev1.PodSpec{Containers: []corev1.Container{asking("db", nil)}}),
		loadPod("fresh", "", corev1.PodPending, corev1.PodSpec{Containers: []corev1.Container{asking("app", nil)}}),
		ready, unready,
		loadPod("job-done", loadNode, corev1.PodSucceeded, corev1.PodSpec{Containers: []corev1.Container{asking("app", nil)}}),
	}
}

func councilSnapshot() live.Snapshot {
	s := loadSnapshot(councilPods()...)
	down := loadedNode("worker-c")
	down.Status.Conditions = []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionUnknown}}
	s.Nodes = append(s.Nodes, down)
	at := metav1.NewTime(time.Date(2026, 11, 20, 11, 0, 0, 0, time.UTC))
	later := metav1.NewTime(time.Date(2026, 11, 20, 11, 5, 0, 0, time.UTC))
	event := func(name, message string, when metav1.Time) corev1.Event {
		return corev1.Event{ObjectMeta: councilMeta(name), Reason: "FailedScheduling", Message: message, Type: corev1.EventTypeWarning, LastTimestamp: when,
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "shop", Name: "db-0", UID: "pod-db-0"}}
	}
	s.Events = []corev1.Event{event("db-0.1", "0/3 nodes are available: pod has unbound immediate PersistentVolumeClaims.", at),
		event("db-0.2", "0/3 nodes are available: 1 node(s) had untolerated taint {dedicated: db}.", later)}
	three, two, zero := int32(3), int32(2), int32(0)
	s.Deployments = []appsv1.Deployment{
		{ObjectMeta: councilMeta("web"), Spec: appsv1.DeploymentSpec{Replicas: &three}, Status: appsv1.DeploymentStatus{ReadyReplicas: 2}},
		{ObjectMeta: councilMeta("api"), Spec: appsv1.DeploymentSpec{Replicas: &two}, Status: appsv1.DeploymentStatus{ReadyReplicas: 2}},
		{ObjectMeta: councilMeta("idle"), Spec: appsv1.DeploymentSpec{Replicas: &zero}},
	}
	s.StatefulSets = []appsv1.StatefulSet{{ObjectMeta: councilMeta("db"), Spec: appsv1.StatefulSetSpec{Replicas: &three}, Status: appsv1.StatefulSetStatus{ReadyReplicas: 1}}}
	s.DaemonSets = []appsv1.DaemonSet{{ObjectMeta: councilMeta("agent"), Status: appsv1.DaemonSetStatus{DesiredNumberScheduled: 3, NumberReady: 3}}}
	service := func(name string, selector map[string]string, kind corev1.ServiceType) corev1.Service {
		return corev1.Service{ObjectMeta: councilMeta(name), Spec: corev1.ServiceSpec{Selector: selector, Type: kind}}
	}
	s.Services = []corev1.Service{
		service("web", map[string]string{"app": "web"}, corev1.ServiceTypeClusterIP),
		service("api", map[string]string{"app": "api"}, corev1.ServiceTypeClusterIP),
		service("lonely", map[string]string{"app": "nobody"}, corev1.ServiceTypeClusterIP),
		service("manual", nil, corev1.ServiceTypeClusterIP),
		service("elsewhere", nil, corev1.ServiceTypeExternalName),
	}
	slice := func(name, svc string, ready ...*bool) discoveryv1.EndpointSlice {
		out := discoveryv1.EndpointSlice{ObjectMeta: councilMeta(name), AddressType: discoveryv1.AddressTypeIPv4}
		out.Labels = map[string]string{discoveryv1.LabelServiceName: svc}
		for i, r := range ready {
			out.Endpoints = append(out.Endpoints, discoveryv1.Endpoint{Addresses: []string{"10.0.0." + string(rune('1'+i))}, Conditions: discoveryv1.EndpointConditions{Ready: r}})
		}
		return out
	}
	s.EndpointSlices = []discoveryv1.EndpointSlice{slice("web-abc", "web", flag(false)), slice("api-abc", "api", nil, flag(false))}
	return s
}

func TestTheDashboardsFactsAreCountedFromTheSnapshot(t *testing.T) {
	usage := live.Usage{Available: true, Pods: map[string]live.UsageOf{"shop/api-5d-a1": {CPUMilli: 40, MemBytes: 30 << 20}, "shop/api-5d-b2": {CPUMilli: 250, MemBytes: 10 << 20},
		"kube-system/coredns-1": {CPUMilli: 40, MemBytes: 20 << 20}}}
	r, err := live.Open(councilSnapshot(), nil, usage)
	if err != nil {
		t.Fatal(err)
	}
	c := r.Council()
	counts := [][3]int{{c.NodesReady, c.Nodes, 2}, {c.PodsReady, c.PodsActive, 1}, {c.WorkloadsFull, c.Workloads, 3}, {c.ServicesServed, c.Services, 1}}
	wants := [][2]int{{2, 3}, {1, 5}, {3, 5}, {1, 3}}
	for i, got := range counts {
		if got[0] != wants[i][0] || got[1] != wants[i][1] {
			t.Errorf("gauge %d counts %d of %d, want %d of %d", i, got[0], got[1], wants[i][0], wants[i][1])
		}
	}
	pending := []view.LivePending{
		{Namespace: "shop", Name: "db-0", Reason: "0/3 nodes are available: 1 node(s) had untolerated taint {dedicated: db}."},
		{Namespace: "shop", Name: "fresh"},
		{Namespace: "shop", Name: "web-7c9-x2", Reason: "0/3 nodes are available: 3 Insufficient cpu."},
	}
	if !reflect.DeepEqual(c.Pending, pending) {
		t.Errorf("Pending Pods %+v, want %+v", c.Pending, pending)
	}
	short := []view.LiveWorkload{{Kind: "StatefulSet", Namespace: "shop", Name: "db", Ready: 1, Desired: 3}, {Kind: "Deployment", Namespace: "shop", Name: "web", Ready: 2, Desired: 3}}
	if !reflect.DeepEqual(c.Short, short) {
		t.Errorf("workloads below desired %+v, want %+v, the most replicas missing first", c.Short, short)
	}
	unserved := []view.LiveService{{Namespace: "shop", Name: "lonely"}, {Namespace: "shop", Name: "web", Endpoints: 1}}
	if !reflect.DeepEqual(c.Unserved, unserved) {
		t.Errorf("Services without a ready endpoint %+v, want %+v", c.Unserved, unserved)
	}
	busiest := []view.LivePodUsage{{Namespace: "shop", Name: "api-5d-b2", CPUMilli: 250, MemoryBytes: 10 << 20},
		{Namespace: "kube-system", Name: "coredns-1", CPUMilli: 40, MemoryBytes: 20 << 20}, {Namespace: "shop", Name: "api-5d-a1", CPUMilli: 40, MemoryBytes: 30 << 20}}
	if !reflect.DeepEqual(c.Busiest, busiest) {
		t.Errorf("busiest Pods %+v, want %+v", c.Busiest, busiest)
	}
}

func TestTheDashboardSaysWhichListsItCouldNotRead(t *testing.T) {
	s := councilSnapshot()
	s.Denied = []string{"services", "events", "persistentvolumes"}
	s.Missing = []string{"endpointslices"}
	r, err := live.Open(s, nil, live.Usage{})
	if err != nil {
		t.Fatal(err)
	}
	c := r.Council()
	if want := []string{"endpointslices", "events", "services"}; !reflect.DeepEqual(c.Unread, want) {
		t.Errorf("unread lists %v, want %v", c.Unread, want)
	}
	if len(c.Busiest) != 0 {
		t.Errorf("no metrics-server answered, yet the busiest Pods are %+v", c.Busiest)
	}
}

func flag(b bool) *bool { return &b }

func TestALiveRealmOpensItsDashboard(t *testing.T) {
	r, err := live.Open(councilSnapshot(), nil, live.Usage{})
	if err != nil {
		t.Fatal(err)
	}
	if reveals := r.GameState().Reveals; !slices.Contains(reveals, "council") {
		t.Errorf("a live realm reveals %v: the Dashboard stays closed", reveals)
	}
}
