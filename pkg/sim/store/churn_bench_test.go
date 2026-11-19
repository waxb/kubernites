package store

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func churnStore(pods int) *Store {
	s := cluster("a", "b", "d")
	for i := 0; i < pods; i++ {
		s.Create(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("porter-%05d", i), Namespace: "default"}})
		s.Create(&corev1.Event{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("ev-%05d", i), Namespace: "default"}})
	}
	return s
}

func BenchmarkViewAfterChurn(b *testing.B) {
	for _, n := range []int{500, 2000, 10000} {
		b.Run(fmt.Sprintf("pods=%d", n), func(b *testing.B) {
			s := churnStore(n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				name := fmt.Sprintf("churn-%d", i)
				s.Create(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"}})
				View[*corev1.Pod](s, KindPod, "default")
				s.Delete(Key{Kind: KindPod, Namespace: "default", Name: name})
				View[*corev1.Pod](s, KindPod, "default")
			}
		})
	}
}
