package model

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/waxb/kubernites/pkg/sim/store"
)

func pod(name, node, image string) *corev1.Pod {
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", UID: "u-" + "x"},
		Spec:       corev1.PodSpec{NodeName: node, Containers: []corev1.Container{{Name: "c", Image: image}}},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	p.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod"))
	return p
}

func waiting(p *corev1.Pod, reason string) *corev1.Pod {
	next := p.DeepCopy()
	next.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "c", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason}}}}
	return next
}

func TestCacheResetApplyAndLists(t *testing.T) {
	c := NewCache()
	a, b := pod("a", "n1", "porter:1.0"), pod("b", "n1", "porter:1.1")
	c.Reset([]store.Object{b, a, &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "n1"}}})
	if c.Len() != 3 || len(c.Pods()) != 2 || c.Pods()[0].Name != "a" || len(c.Nodes()) != 1 {
		t.Fatalf("lists wrong: %d pods", len(c.Pods()))
	}
	v := c.Version()
	c.Apply(store.Event{Type: store.Deleted, Object: a})
	c.Apply(store.Event{Type: store.Added, Object: &corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: "e", Namespace: "default"}}})
	c.Apply(store.Event{Type: store.Added, Object: &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: "default"}}})
	c.Apply(store.Event{Type: store.Added, Object: &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "r", Namespace: "default"}}})
	c.Apply(store.Event{Type: store.Added, Object: &discoveryv1.EndpointSlice{ObjectMeta: metav1.ObjectMeta{Name: "sl", Namespace: "default"}}})
	if len(c.Pods()) != 1 || len(c.Events()) != 1 || len(c.Services()) != 1 || len(c.ReplicaSets()) != 1 || len(c.Slices()) != 1 {
		t.Fatal("apply wrong")
	}
	if c.Version() <= v {
		t.Fatal("version not bumped")
	}
	if _, ok := c.Get("Pod", "default", "b"); !ok {
		t.Fatal("get failed")
	}
}

func TestHearthStates(t *testing.T) {
	base := pod("p", "n1", "porter:1.0")
	cases := map[string]*corev1.Pod{
		HearthReady:     base,
		HearthKindling:  waiting(base, "ContainerCreating"),
		HearthGuttering: waiting(base, "CrashLoopBackOff"),
		HearthGhost:     waiting(base, "ImagePullBackOff"),
		HearthPending:   pod("p", "", "porter:1.0"),
	}
	for want, p := range cases {
		if got := HearthState(p); got != want {
			t.Errorf("state = %s want %s", got, want)
		}
	}
	failed := base.DeepCopy()
	failed.Status.Phase = corev1.PodFailed
	failed.Status.Reason = "Evicted"
	if HearthState(failed) != HearthBanished {
		t.Fatal("evicted should be banished")
	}
	now := metav1.Now()
	gone := base.DeepCopy()
	gone.DeletionTimestamp = &now
	if HearthState(gone) != HearthBanished {
		t.Fatal("terminating should be banished")
	}
}

func TestTierAndView(t *testing.T) {
	if TierOf("porter:1.0") != 1 || TierOf("porter:1.1") != 2 || TierOf("porter:2.0") != 3 || TierOf("nope:9") != 1 {
		t.Fatal("tiers wrong")
	}
	yes := true
	p := pod("porter-abc", "n1", "porter:1.1")
	p.OwnerReferences = []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "porters", Controller: &yes}}
	v := ViewOf(p)
	if v.Sprite != "porter" || v.Tier != 2 || v.Owner != "ReplicaSet/porters" || v.System {
		t.Fatalf("view = %+v", v)
	}
	if ViewOf(&corev1.Pod{}).Sprite != "wanderer" {
		t.Fatal("containerless pod should be a wanderer")
	}
	c := NewCache()
	c.Reset([]store.Object{p})
	if len(c.PodViews()) != 1 || Hash("a") == Hash("b") {
		t.Fatal("pod views / hash wrong")
	}
}

func TestCacheNames(t *testing.T) {
	c := NewCache()
	a, b := &corev1.Pod{}, &corev1.Pod{}
	a.Name, a.Namespace, b.Name, b.Namespace = "b-pod", "default", "a-pod", "default"
	c.Reset([]store.Object{a, b})
	if got := c.Names(store.KindPod); len(got) != 2 || got[0] != "a-pod" {
		t.Fatalf("names = %v", got)
	}
	if len(c.Names(store.KindNode)) != 0 {
		t.Fatal("no nodes")
	}
}

func TestHorns(t *testing.T) {
	min, goal, util := int32(2), int32(70), int32(84)
	h := &autoscalingv2.HorizontalPodAutoscaler{}
	h.Name, h.Namespace = "porters", "default"
	h.Spec.ScaleTargetRef.Name = "porters"
	h.Spec.MinReplicas, h.Spec.MaxReplicas = &min, 10
	h.Spec.Metrics = []autoscalingv2.MetricSpec{{Resource: &autoscalingv2.ResourceMetricSource{Target: autoscalingv2.MetricTarget{AverageUtilization: &goal}}}}
	h.Status.CurrentReplicas, h.Status.DesiredReplicas = 2, 5
	h.Status.CurrentMetrics = []autoscalingv2.MetricStatus{{Resource: &autoscalingv2.ResourceMetricStatus{Current: autoscalingv2.MetricValueStatus{AverageUtilization: &util}}}}
	c := NewCache()
	c.Reset([]store.Object{h})
	got := c.Horns()
	if len(got) != 1 || got[0].Goal != 70 || got[0].Utilization != 84 || got[0].Min != 2 || !got[0].Sounding {
		t.Fatalf("horns = %+v", got)
	}
	h.Status.Conditions = []autoscalingv2.HorizontalPodAutoscalerCondition{{Type: autoscalingv2.ScalingActive, Status: corev1.ConditionFalse, Message: "no metrics"}}
	if v := HornOf(h); v.Sounding || v.Message != "no metrics" {
		t.Fatalf("silent horn = %+v", v)
	}
}
