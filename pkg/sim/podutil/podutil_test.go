package podutil

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func podWith(req, lim corev1.ResourceList) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "default"},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name:      "lantern",
			Image:     "lantern:1.0",
			Resources: corev1.ResourceRequirements{Requests: req, Limits: lim},
		}}},
	}
}

func rl(cpu, mem string) corev1.ResourceList {
	out := corev1.ResourceList{}
	if cpu != "" {
		out[corev1.ResourceCPU] = resource.MustParse(cpu)
	}
	if mem != "" {
		out[corev1.ResourceMemory] = resource.MustParse(mem)
	}
	return out
}

func TestQoSClass(t *testing.T) {
	cases := []struct {
		name string
		pod  *corev1.Pod
		want corev1.PodQOSClass
	}{
		{"none", podWith(nil, nil), corev1.PodQOSBestEffort},
		{"requests only", podWith(rl("100m", "64Mi"), nil), corev1.PodQOSBurstable},
		{"equal", podWith(rl("100m", "64Mi"), rl("100m", "64Mi")), corev1.PodQOSGuaranteed},
		{"limits only", podWith(nil, rl("200m", "128Mi")), corev1.PodQOSGuaranteed},
		{"cpu missing limit", podWith(rl("100m", "64Mi"), rl("", "64Mi")), corev1.PodQOSBurstable},
	}
	for _, tc := range cases {
		if got := QoSClass(tc.pod); got != tc.want {
			t.Errorf("%s: got %s want %s", tc.name, got, tc.want)
		}
	}
}

func TestRequestsDefaultToLimits(t *testing.T) {
	cpu, mem := Requests(podWith(rl("250m", ""), rl("", "128Mi")))
	if cpu != 250 || mem != 128*1024*1024 {
		t.Fatalf("cpu=%d mem=%d", cpu, mem)
	}
	cpuLim, memLim := Limits(podWith(nil, rl("", "128Mi")))
	if cpuLim != 0 || memLim != 128*1024*1024 {
		t.Fatalf("limits cpu=%d mem=%d", cpuLim, memLim)
	}
}

func TestConditionsAreImmutableUpdates(t *testing.T) {
	orig := []corev1.PodCondition{{Type: corev1.PodScheduled, Status: corev1.ConditionTrue}}
	now := metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	next := SetCondition(orig, corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionTrue}, now)
	if len(orig) != 1 || len(next) != 2 {
		t.Fatalf("orig=%d next=%d", len(orig), len(next))
	}
	again := SetCondition(next, corev1.PodCondition{Type: corev1.PodReady, Status: corev1.ConditionFalse}, now)
	if GetCondition(again, corev1.PodReady).Status != corev1.ConditionFalse || GetCondition(next, corev1.PodReady).Status != corev1.ConditionTrue {
		t.Fatal("set condition mutated input")
	}
	if GetCondition(orig, corev1.ContainersReady) != nil {
		t.Fatal("expected nil for missing condition")
	}
}

func TestReadyActiveTerminal(t *testing.T) {
	p := podWith(nil, nil)
	if IsReady(p) || !IsActive(p) || IsTerminal(p) {
		t.Fatal("fresh pod state wrong")
	}
	p.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}}
	if !IsReady(p) {
		t.Fatal("expected ready")
	}
	p.Status.Phase = corev1.PodFailed
	if IsActive(p) || !IsTerminal(p) {
		t.Fatal("failed pod should be terminal")
	}
	q := podWith(nil, nil)
	now := metav1.Now()
	q.DeletionTimestamp = &now
	if IsActive(q) {
		t.Fatal("deleting pod should not be active")
	}
}

func TestControllerRef(t *testing.T) {
	p := podWith(nil, nil)
	if ControllerRef(p) != nil {
		t.Fatal("expected nil")
	}
	yes := true
	p.OwnerReferences = []metav1.OwnerReference{{Kind: "ReplicaSet", Name: "lantern", UID: "u1", Controller: &yes}}
	if ref := ControllerRef(p); ref == nil || ref.Name != "lantern" {
		t.Fatalf("ref = %v", ref)
	}
}

func TestStatusReason(t *testing.T) {
	p := podWith(nil, nil)
	p.Status.Phase = corev1.PodPending
	if got := StatusReason(p); got != "Pending" {
		t.Fatalf("got %s", got)
	}
	p.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "lantern", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}}}
	p.Status.Phase = corev1.PodRunning
	if got := StatusReason(p); got != "CrashLoopBackOff" {
		t.Fatalf("got %s", got)
	}
	p.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "lantern", State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"}}}}
	if got := StatusReason(p); got != "OOMKilled" {
		t.Fatalf("got %s", got)
	}
	p.Status.Phase = corev1.PodFailed
	p.Status.Reason = "Evicted"
	if got := StatusReason(p); got != "Evicted" {
		t.Fatalf("got %s", got)
	}
	now := metav1.Now()
	q := podWith(nil, nil)
	q.Status.Phase = corev1.PodRunning
	q.DeletionTimestamp = &now
	if got := StatusReason(q); got != "Terminating" {
		t.Fatalf("got %s", got)
	}
	r := podWith(nil, nil)
	r.Status.Phase = corev1.PodRunning
	r.Status.ContainerStatuses = []corev1.ContainerStatus{{Name: "lantern", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}}}
	if got := StatusReason(r); got != "Running" {
		t.Fatalf("got %s", got)
	}
	if ready, total := ReadyCount(r); ready != 1 || total != 1 {
		t.Fatalf("ready=%d total=%d", ready, total)
	}
	if RestartCount(r) != 0 {
		t.Fatal("restarts")
	}
}
