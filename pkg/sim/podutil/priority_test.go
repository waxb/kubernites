package podutil

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPriorityAndCriticality(t *testing.T) {
	critical, user := SystemCriticalPriority, int32(1000000000)
	cases := []struct {
		pod      *corev1.Pod
		priority int32
		critical bool
	}{
		{&corev1.Pod{}, 0, false},
		{&corev1.Pod{Spec: corev1.PodSpec{Priority: &user}}, user, false},
		{&corev1.Pod{Spec: corev1.PodSpec{Priority: &critical}}, critical, true},
		{&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{ConfigSourceKey: "file"}}}, 0, true},
		{&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{corev1.MirrorPodAnnotationKey: "h"}}}, 0, true},
	}
	for i, tc := range cases {
		if got := Priority(tc.pod); got != tc.priority {
			t.Errorf("%d: priority = %d", i, got)
		}
		if got := IsCriticalPod(tc.pod); got != tc.critical {
			t.Errorf("%d: critical = %v", i, got)
		}
	}
}
