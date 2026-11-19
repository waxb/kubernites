package store

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/sim/limitranger"
)

type InvalidError struct {
	Kind   string
	Name   string
	Causes []string
}

func (e *InvalidError) Error() string {
	cause := strings.Join(e.Causes, ", ")
	if len(e.Causes) > 1 {
		cause = "[" + cause + "]"
	}
	return fmt.Sprintf("%s %q is invalid: %s", e.Kind, e.Name, cause)
}

func (s *Store) limitRanges(namespace string) []*corev1.LimitRange {
	out := []*corev1.LimitRange{}
	for _, key := range s.keysOf(KindLimitRange) {
		if lr, ok := s.objects[key].(*corev1.LimitRange); ok && key.Namespace == namespace {
			out = append(out, lr)
		}
	}
	return out
}

func (s *Store) defaultPodResources(pod *corev1.Pod) {
	limitranger.RequestsFromLimits(&pod.Spec)
	limitranger.Mutate(pod, s.limitRanges(pod.Namespace))
}

func (s *Store) validatePodResources(pod *corev1.Pod) error {
	causes := []string{}
	for _, part := range []struct {
		field string
		list  []corev1.Container
	}{{"spec.containers", pod.Spec.Containers}, {"spec.initContainers", pod.Spec.InitContainers}} {
		for i, c := range part.list {
			causes = append(causes, requestAboveLimit(fmt.Sprintf("%s[%d].resources.requests", part.field, i), c.Resources)...)
		}
	}
	if len(causes) > 0 {
		return &InvalidError{Kind: KindPod, Name: pod.Name, Causes: causes}
	}
	if err := limitranger.Validate(pod, s.limitRanges(pod.Namespace)); err != nil {
		return podForbidden(pod, err.Error())
	}
	return nil
}

func requestAboveLimit(field string, r corev1.ResourceRequirements) []string {
	out := []string{}
	for _, name := range []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory} {
		req, hasReq := r.Requests[name]
		lim, hasLim := r.Limits[name]
		if hasReq && hasLim && req.Cmp(lim) > 0 {
			out = append(out, fmt.Sprintf("%s: Invalid value: %q: must be less than or equal to %s limit of %s", field, req.String(), name, lim.String()))
		}
	}
	return out
}
