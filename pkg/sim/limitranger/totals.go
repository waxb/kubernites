package limitranger

import corev1 "k8s.io/api/core/v1"

func podTotals(pod *corev1.Pod) (corev1.ResourceList, corev1.ResourceList) {
	requests := sumOf(pod, func(c corev1.Container) corev1.ResourceList { return c.Resources.Requests })
	limits := sumOf(pod, func(c corev1.Container) corev1.ResourceList { return c.Resources.Limits })
	if pod.Spec.Resources != nil {
		podLevel(requests, pod.Spec.Resources.Requests)
		podLevel(limits, pod.Spec.Resources.Limits)
	}
	return requests, limits
}

func sumOf(pod *corev1.Pod, of func(corev1.Container) corev1.ResourceList) corev1.ResourceList {
	total := corev1.ResourceList{}
	for _, c := range pod.Spec.Containers {
		addList(total, of(c))
	}
	sidecars, initMax := corev1.ResourceList{}, corev1.ResourceList{}
	for _, c := range pod.Spec.InitContainers {
		use := corev1.ResourceList{}
		if c.RestartPolicy != nil && *c.RestartPolicy == corev1.ContainerRestartPolicyAlways {
			addList(total, of(c))
			addList(sidecars, of(c))
			addList(use, sidecars)
		} else {
			addList(use, of(c))
			addList(use, sidecars)
		}
		maxList(initMax, use)
	}
	maxList(total, initMax)
	return total
}

func podLevel(into, pod corev1.ResourceList) {
	for _, name := range []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory} {
		if q, ok := pod[name]; ok {
			into[name] = q.DeepCopy()
		}
	}
}

func addList(into, add corev1.ResourceList) {
	for name, q := range add {
		if have, ok := into[name]; ok {
			have.Add(q)
			into[name] = have
			continue
		}
		into[name] = q.DeepCopy()
	}
}

func maxList(into, other corev1.ResourceList) {
	for name, q := range other {
		if have, ok := into[name]; !ok || q.Cmp(have) > 0 {
			into[name] = q.DeepCopy()
		}
	}
}
