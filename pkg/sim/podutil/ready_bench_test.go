package podutil

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func BenchmarkIsReady(b *testing.B) {
	p := &corev1.Pod{Status: corev1.PodStatus{Conditions: []corev1.PodCondition{
		{Type: corev1.PodInitialized, Status: corev1.ConditionTrue},
		{Type: corev1.PodScheduled, Status: corev1.ConditionTrue},
		{Type: corev1.ContainersReady, Status: corev1.ConditionTrue},
		{Type: corev1.PodReady, Status: corev1.ConditionTrue},
	}}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !IsReady(p) {
			b.Fatal("pod should be ready")
		}
	}
}
