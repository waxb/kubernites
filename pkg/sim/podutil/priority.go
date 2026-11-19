package podutil

import corev1 "k8s.io/api/core/v1"

const SystemCriticalPriority = int32(2000000000)

func Priority(pod *corev1.Pod) int32 {
	if pod.Spec.Priority != nil {
		return *pod.Spec.Priority
	}
	return 0
}

func IsCriticalPod(pod *corev1.Pod) bool {
	if IsStaticPod(pod) || IsMirrorPod(pod) {
		return true
	}
	return pod.Spec.Priority != nil && *pod.Spec.Priority >= SystemCriticalPriority
}
