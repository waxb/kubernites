package limitranger

import (
	"fmt"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const Annotation = "kubernetes.io/limit-ranger"

func DefaultRequirements(lr *corev1.LimitRange) corev1.ResourceRequirements {
	out := corev1.ResourceRequirements{Requests: corev1.ResourceList{}, Limits: corev1.ResourceList{}}
	for _, item := range lr.Spec.Limits {
		if item.Type != corev1.LimitTypeContainer {
			continue
		}
		for name, q := range item.DefaultRequest {
			out.Requests[name] = q.DeepCopy()
		}
		for name, q := range item.Default {
			out.Limits[name] = q.DeepCopy()
		}
	}
	return out
}

func RequestsFromLimits(spec *corev1.PodSpec) {
	for _, list := range [][]corev1.Container{spec.Containers, spec.InitContainers} {
		for i := range list {
			requestsFromLimits(&list[i])
		}
	}
}

func requestsFromLimits(c *corev1.Container) {
	if c.Resources.Limits == nil {
		return
	}
	requests := corev1.ResourceList{}
	for name, q := range c.Resources.Requests {
		requests[name] = q.DeepCopy()
	}
	for name, q := range c.Resources.Limits {
		if _, set := requests[name]; !set {
			requests[name] = q.DeepCopy()
		}
	}
	c.Resources.Requests = requests
}

func mergeContainer(c *corev1.Container, defaults corev1.ResourceRequirements, prefix string) []string {
	limits, requests := copyList(c.Resources.Limits), copyList(c.Resources.Requests)
	setLimits, setRequests := fill(limits, defaults.Limits), fill(requests, defaults.Requests)
	c.Resources.Limits, c.Resources.Requests = limits, requests
	notes := []string{}
	if len(setRequests) > 0 {
		notes = append(notes, strings.Join(setRequests, ", ")+fmt.Sprintf(" request for %s %s", prefix, c.Name))
	}
	if len(setLimits) > 0 {
		notes = append(notes, strings.Join(setLimits, ", ")+fmt.Sprintf(" limit for %s %s", prefix, c.Name))
	}
	return notes
}

func copyList(in corev1.ResourceList) corev1.ResourceList {
	out := corev1.ResourceList{}
	for name, q := range in {
		out[name] = q.DeepCopy()
	}
	return out
}

func fill(into, defaults corev1.ResourceList) []string {
	set := []string{}
	for name, q := range defaults {
		if _, found := into[name]; !found {
			into[name] = q.DeepCopy()
			set = append(set, string(name))
		}
	}
	sort.Strings(set)
	return set
}

func mutateOne(pod *corev1.Pod, lr *corev1.LimitRange) {
	defaults := DefaultRequirements(lr)
	notes := []string{}
	for i := range pod.Spec.Containers {
		notes = append(notes, mergeContainer(&pod.Spec.Containers[i], defaults, "container")...)
	}
	for i := range pod.Spec.InitContainers {
		notes = append(notes, mergeContainer(&pod.Spec.InitContainers[i], defaults, "init container")...)
	}
	if len(notes) == 0 {
		return
	}
	annotations := map[string]string{}
	for k, v := range pod.Annotations {
		annotations[k] = v
	}
	annotations[Annotation] = "LimitRanger plugin set: " + strings.Join(notes, "; ")
	pod.Annotations = annotations
}

func Mutate(pod *corev1.Pod, ranges []*corev1.LimitRange) {
	for _, lr := range ranges {
		mutateOne(pod, lr)
	}
}

func DefaultedSpec(spec corev1.PodSpec, ranges []*corev1.LimitRange) corev1.PodSpec {
	pod := &corev1.Pod{Spec: *spec.DeepCopy()}
	RequestsFromLimits(&pod.Spec)
	Mutate(pod, ranges)
	return pod.Spec
}

func quantityOf(list corev1.ResourceList, name corev1.ResourceName) (resource.Quantity, bool) {
	q, ok := list[name]
	return q, ok
}

func DefaultedRequest(pod *corev1.Pod, name corev1.ResourceName) bool {
	note, ok := pod.Annotations[Annotation]
	if !ok {
		return false
	}
	for _, part := range strings.Split(strings.TrimPrefix(note, "LimitRanger plugin set: "), "; ") {
		names, rest, found := strings.Cut(part, " request for ")
		if !found || rest == "" {
			continue
		}
		for _, n := range strings.Split(names, ", ") {
			if n == string(name) {
				return true
			}
		}
	}
	return false
}
