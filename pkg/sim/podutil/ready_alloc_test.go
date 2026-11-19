package podutil

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestReadinessAndControllerReadsDoNotAllocate(t *testing.T) {
	yes := true
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{{UID: "a"}, {UID: "rs", Controller: &yes}}},
		Status: corev1.PodStatus{Conditions: []corev1.PodCondition{
			{Type: corev1.PodScheduled, Status: corev1.ConditionTrue},
			{Type: corev1.PodReady, Status: corev1.ConditionTrue},
		}},
	}
	allocs := testing.AllocsPerRun(100, func() {
		if !IsReady(p) {
			t.Fatal("pod should be ready")
		}
		if uid, ok := ControllerUID(p); !ok || uid != "rs" {
			t.Fatal("controller uid should be rs")
		}
	})
	if allocs != 0 {
		t.Fatalf("allocs = %v", allocs)
	}
	p.Status.Conditions[1].Status = corev1.ConditionFalse
	p.OwnerReferences[1].Controller = nil
	if _, ok := ControllerUID(p); IsReady(p) || ok {
		t.Fatal("pod should be unready and have no controller")
	}
}
