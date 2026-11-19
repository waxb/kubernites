package podutil

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const unbounded = -1 << 40

func IsRestartableInitContainer(c *corev1.Container) bool {
	return c != nil && c.RestartPolicy != nil && *c.RestartPolicy == corev1.ContainerRestartPolicyAlways
}

func Sidecars(pod *corev1.Pod) []corev1.Container {
	var out []corev1.Container
	for i := range pod.Spec.InitContainers {
		if IsRestartableInitContainer(&pod.Spec.InitContainers[i]) {
			out = append(out, pod.Spec.InitContainers[i])
		}
	}
	return out
}

func requestOf(c corev1.Container, name corev1.ResourceName) (resource.Quantity, bool) {
	if q, ok := c.Resources.Requests[name]; ok {
		return q, true
	}
	q, ok := c.Resources.Limits[name]
	return q, ok
}

func requestAmounts(c corev1.Container) [2]int64 {
	var out [2]int64
	if q, ok := requestOf(c, corev1.ResourceCPU); ok {
		out[0] = q.MilliValue()
	}
	if q, ok := requestOf(c, corev1.ResourceMemory); ok {
		out[1] = q.Value()
	}
	return out
}

func declaredLimits(c corev1.Container) [2]int64 {
	var out [2]int64
	if q, ok := c.Resources.Limits[corev1.ResourceCPU]; ok {
		out[0] = q.MilliValue()
	}
	if q, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
		out[1] = q.Value()
	}
	return out
}

func limitAmounts(c corev1.Container) [2]int64 {
	out := [2]int64{unbounded, unbounded}
	if q, ok := c.Resources.Limits[corev1.ResourceCPU]; ok {
		out[0] = q.MilliValue()
	}
	if q, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
		out[1] = q.Value()
	}
	return out
}

func addAmounts(a, b [2]int64) [2]int64 {
	var out [2]int64
	for i := range out {
		if a[i] < 0 || b[i] < 0 {
			out[i] = unbounded
			continue
		}
		out[i] = a[i] + b[i]
	}
	return out
}

func maxAmounts(a, b [2]int64) [2]int64 {
	return [2]int64{max(a[0], b[0]), max(a[1], b[1])}
}

func effective(pod *corev1.Pod, amounts func(corev1.Container) [2]int64) [2]int64 {
	var total [2]int64
	for _, c := range pod.Spec.Containers {
		total = addAmounts(total, amounts(c))
	}
	var sidecars, initUse [2]int64
	for _, c := range pod.Spec.InitContainers {
		use := addAmounts(amounts(c), sidecars)
		if IsRestartableInitContainer(&c) {
			total = addAmounts(total, amounts(c))
			sidecars = use
		}
		initUse = maxAmounts(initUse, use)
	}
	return maxAmounts(total, initUse)
}

func ContainerRequests(cs ...corev1.Container) (cpuMilli int64, memBytes int64) {
	var total [2]int64
	for _, c := range cs {
		total = addAmounts(total, requestAmounts(c))
	}
	return total[0], total[1]
}

func ContainerLimits(cs ...corev1.Container) (cpuMilli int64, memBytes int64) {
	var total [2]int64
	for _, c := range cs {
		total = addAmounts(total, limitAmounts(c))
	}
	return clampUnbounded(total[0]), clampUnbounded(total[1])
}

func Requests(pod *corev1.Pod) (cpuMilli int64, memBytes int64) {
	total := effective(pod, requestAmounts)
	return total[0], total[1]
}

func Limits(pod *corev1.Pod) (cpuMilli int64, memBytes int64) {
	total := effective(pod, declaredLimits)
	return total[0], total[1]
}

func clampUnbounded(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}

func effectiveRequests(c corev1.Container) corev1.ResourceList {
	out := corev1.ResourceList{}
	for name, q := range c.Resources.Limits {
		out[name] = q.DeepCopy()
	}
	for name, q := range c.Resources.Requests {
		out[name] = q.DeepCopy()
	}
	return out
}

func QoSClass(pod *corev1.Pod) corev1.PodQOSClass {
	anySet := false
	guaranteed := true
	containers := append(append([]corev1.Container(nil), pod.Spec.InitContainers...), pod.Spec.Containers...)
	for _, c := range containers {
		if declaresCompute(c) {
			anySet = true
		}
		req := effectiveRequests(c)
		for _, name := range []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory} {
			lim, hasLim := c.Resources.Limits[name]
			r, hasReq := req[name]
			if !hasLim || !hasReq || lim.Cmp(r) != 0 {
				guaranteed = false
			}
		}
	}
	switch {
	case !anySet:
		return corev1.PodQOSBestEffort
	case guaranteed:
		return corev1.PodQOSGuaranteed
	}
	return corev1.PodQOSBurstable
}

func declaresCompute(c corev1.Container) bool {
	for _, name := range []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory} {
		_, req := c.Resources.Requests[name]
		_, lim := c.Resources.Limits[name]
		if req || lim {
			return true
		}
	}
	return false
}
