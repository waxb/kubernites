package podutil

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func ContainerEphemeralRequest(c corev1.Container) int64 {
	if q, ok := requestOf(c, corev1.ResourceEphemeralStorage); ok {
		return q.Value()
	}
	return 0
}

func EphemeralRequest(pod *corev1.Pod) int64 {
	return effectiveScalar(pod, ContainerEphemeralRequest)
}

func EphemeralLimit(pod *corev1.Pod) (int64, bool) {
	q, found := EphemeralLimitQuantity(pod)
	return q.Value(), found
}

func EphemeralLimitQuantity(pod *corev1.Pod) (resource.Quantity, bool) {
	var total resource.Quantity
	found := false
	for _, c := range append(append([]corev1.Container(nil), pod.Spec.Containers...), Sidecars(pod)...) {
		q, ok := c.Resources.Limits[corev1.ResourceEphemeralStorage]
		if !ok || q.IsZero() {
			continue
		}
		if !found {
			total, found = q.DeepCopy(), true
			continue
		}
		total.Add(q)
	}
	return total, found
}

func effectiveScalar(pod *corev1.Pod, amount func(corev1.Container) int64) int64 {
	var total int64
	for _, c := range pod.Spec.Containers {
		total += amount(c)
	}
	var sidecars, initUse int64
	for i := range pod.Spec.InitContainers {
		c := pod.Spec.InitContainers[i]
		use := amount(c) + sidecars
		if IsRestartableInitContainer(&c) {
			total += amount(c)
			sidecars = use
		}
		initUse = max(initUse, use)
	}
	return max(total, initUse)
}
