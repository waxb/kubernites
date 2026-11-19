package podutil

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func GetCondition(conds []corev1.PodCondition, t corev1.PodConditionType) *corev1.PodCondition {
	for i := range conds {
		if conds[i].Type == t {
			c := conds[i]
			return &c
		}
	}
	return nil
}

func SetCondition(conds []corev1.PodCondition, cond corev1.PodCondition, now metav1.Time) []corev1.PodCondition {
	out := make([]corev1.PodCondition, 0, len(conds)+1)
	replaced := false
	for _, c := range conds {
		if c.Type != cond.Type {
			out = append(out, c)
			continue
		}
		next := cond
		next.LastTransitionTime = c.LastTransitionTime
		if c.Status != cond.Status || c.LastTransitionTime.IsZero() {
			next.LastTransitionTime = now
		}
		out = append(out, next)
		replaced = true
	}
	if !replaced {
		next := cond
		next.LastTransitionTime = now
		out = append(out, next)
	}
	return out
}

func IsReady(pod *corev1.Pod) bool {
	for i := range pod.Status.Conditions {
		if pod.Status.Conditions[i].Type == corev1.PodReady {
			return pod.Status.Conditions[i].Status == corev1.ConditionTrue
		}
	}
	return false
}

func IsTerminal(pod *corev1.Pod) bool {
	return pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded
}

func IsActive(pod *corev1.Pod) bool {
	return !IsTerminal(pod) && pod.DeletionTimestamp == nil
}

func ControllerUID(obj metav1.Object) (types.UID, bool) {
	refs := obj.GetOwnerReferences()
	for i := range refs {
		if refs[i].Controller != nil && *refs[i].Controller {
			return refs[i].UID, true
		}
	}
	return "", false
}

func ControllerRef(obj metav1.Object) *metav1.OwnerReference {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Controller != nil && *ref.Controller {
			r := ref
			return &r
		}
	}
	return nil
}
