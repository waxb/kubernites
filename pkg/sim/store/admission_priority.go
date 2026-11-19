package store

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
)

const (
	podsResource            = "pods"
	priorityClassesResource = "priorityclasses.scheduling.k8s.io"
)

type ForbiddenError struct {
	Resource string
	Name     string
	Reason   string
}

func (e *ForbiddenError) Error() string {
	return fmt.Sprintf("%s %q is forbidden: %s", e.Resource, e.Name, e.Reason)
}

type admittedPriority struct {
	className string
	value     int32
	policy    *corev1.PreemptionPolicy
}

func (s *Store) admitPodPriority(pod *corev1.Pod, old Object) error {
	if prev, ok := old.(*corev1.Pod); ok {
		keepPriority(pod, prev)
		return nil
	}
	resolved, reason := s.establishPriority(pod.Spec.PriorityClassName)
	if reason != "" {
		return podForbidden(pod, reason)
	}
	if pod.Spec.Priority != nil && *pod.Spec.Priority != resolved.value {
		return podForbidden(pod, fmt.Sprintf("the integer value of priority (%d) must not be provided in pod spec; priority admission controller computed %d from the given PriorityClass name", *pod.Spec.Priority, resolved.value))
	}
	if resolved.policy != nil && pod.Spec.PreemptionPolicy != nil && *pod.Spec.PreemptionPolicy != *resolved.policy {
		return podForbidden(pod, fmt.Sprintf("the string value of PreemptionPolicy (%s) must not be provided in pod spec; priority admission controller computed %s from the given PriorityClass name", *pod.Spec.PreemptionPolicy, *resolved.policy))
	}
	value := resolved.value
	pod.Spec.PriorityClassName = resolved.className
	pod.Spec.Priority = &value
	if resolved.policy != nil {
		pod.Spec.PreemptionPolicy = resolved.policy
	}
	return nil
}

func podForbidden(pod *corev1.Pod, reason string) error {
	return &ForbiddenError{Resource: podsResource, Name: pod.Name, Reason: reason}
}

func keepPriority(pod, prev *corev1.Pod) {
	if pod.Spec.Priority == nil && prev.Spec.Priority != nil {
		value := *prev.Spec.Priority
		pod.Spec.Priority = &value
	}
	if pod.Spec.PreemptionPolicy == nil && prev.Spec.PreemptionPolicy != nil {
		pod.Spec.PreemptionPolicy = copyPolicy(prev.Spec.PreemptionPolicy)
	}
}

func (s *Store) establishPriority(className string) (admittedPriority, string) {
	if className == "" {
		if dpc := s.defaultPriorityClass(); dpc != nil {
			return admittedPriority{className: dpc.Name, value: dpc.Value, policy: copyPolicy(dpc.PreemptionPolicy)}, ""
		}
		preempt := corev1.PreemptLowerPriority
		return admittedPriority{policy: &preempt}, ""
	}
	pc, ok := Peek[*schedulingv1.PriorityClass](s, KindPriorityClass, "", className)
	if !ok {
		return admittedPriority{}, fmt.Sprintf("no PriorityClass with name %v was found", className)
	}
	return admittedPriority{className: className, value: pc.Value, policy: copyPolicy(pc.PreemptionPolicy)}, ""
}

func (s *Store) defaultPriorityClass() *schedulingv1.PriorityClass {
	var out *schedulingv1.PriorityClass
	for _, key := range s.keysOf(KindPriorityClass) {
		pc, ok := s.objects[key].(*schedulingv1.PriorityClass)
		if ok && pc.GlobalDefault && (out == nil || out.Value > pc.Value) {
			out = pc
		}
	}
	return out
}

func (s *Store) validateGlobalDefault(pc *schedulingv1.PriorityClass, old Object) error {
	if !pc.GlobalDefault {
		return nil
	}
	dpc := s.defaultPriorityClass()
	if dpc == nil || (old != nil && dpc.Name == pc.Name) {
		return nil
	}
	return &ForbiddenError{Resource: priorityClassesResource, Name: pc.Name,
		Reason: fmt.Sprintf("PriorityClass %v is already marked as default. Only one default can exist", dpc.Name)}
}

func copyPolicy(p *corev1.PreemptionPolicy) *corev1.PreemptionPolicy {
	if p == nil {
		return nil
	}
	out := *p
	return &out
}
