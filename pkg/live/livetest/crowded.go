package livetest

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/live"
)

func Crowded(zones, nodesPerZone, podsPerNode int) live.Snapshot {
	born := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	s := live.Snapshot{TakenAt: born.Add(time.Hour), Version: "v1.37.0"}
	s.Namespaces = []corev1.Namespace{
		{ObjectMeta: metav1.ObjectMeta{Name: "kube-system", UID: "00000000-0000-4000-8000-00000000000a", CreationTimestamp: metav1.NewTime(born)}},
		{ObjectMeta: metav1.ObjectMeta{Name: "shop", UID: "00000000-0000-4000-8000-00000000000b", CreationTimestamp: metav1.NewTime(born)}},
	}
	pod := 0
	for z := 0; z < zones; z++ {
		for n := 0; n < nodesPerZone; n++ {
			name := fmt.Sprintf("node-%c-%03d", 'a'+z, n)
			s.Nodes = append(s.Nodes, corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID(fmt.Sprintf("00000000-0000-4000-9000-%012d", z*1000+n)), CreationTimestamp: metav1.NewTime(born.Add(time.Duration(z*1000+n) * time.Second)),
					Labels: map[string]string{live.LabelZone: fmt.Sprintf("zone-%c", 'a'+z), live.LabelInstanceType: "e2-standard-4"}},
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4"), corev1.ResourceMemory: resource.MustParse("16Gi"), corev1.ResourcePods: resource.MustParse("110")},
					Conditions:  []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}},
				},
			})
			for p := 0; p < podsPerNode; p++ {
				s.Pods = append(s.Pods, corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("swarm-%06d", pod), Namespace: "shop", UID: types.UID(fmt.Sprintf("00000000-0000-4000-8000-%012d", pod)),
						CreationTimestamp: metav1.NewTime(born.Add(time.Duration(pod) * time.Millisecond))},
					Spec:   corev1.PodSpec{NodeName: name},
					Status: corev1.PodStatus{Phase: corev1.PodRunning},
				})
				pod++
			}
		}
	}
	return s
}
