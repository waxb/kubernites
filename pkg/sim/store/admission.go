package store

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"

	"github.com/waxb/kubernites/pkg/sim/podutil"
)

const DefaultTolerationSeconds int64 = 300

func (s *Store) admit(obj, old Object) error {
	protect(obj, old)
	switch o := obj.(type) {
	case *corev1.Pod:
		return s.admitPod(o, old)
	case *schedulingv1.PriorityClass:
		return s.validateGlobalDefault(o, old)
	case *appsv1.DaemonSet:
		admitDaemonSet(o, old)
	case *corev1.PersistentVolumeClaim:
		s.admitClaim(o, old)
	}
	return nil
}

func (s *Store) admitPod(pod *corev1.Pod, old Object) error {
	if old == nil {
		s.defaultPodResources(pod)
	}
	pod.Spec.ServiceAccountName = defaultServiceAccount(pod)
	if old == nil {
		if err := s.admitServiceAccount(pod); err != nil {
			return err
		}
	}
	if err := s.admitPodPriority(pod, old); err != nil {
		return err
	}
	pod.Spec.Tolerations = withDefaultTolerationSeconds(pod.Spec.Tolerations)
	if old == nil {
		return s.validatePodResources(pod)
	}
	return nil
}

func withDefaultTolerationSeconds(tolerations []corev1.Toleration) []corev1.Toleration {
	toleratesNotReady, toleratesUnreachable := false, false
	for _, t := range tolerations {
		noExecute := t.Effect == corev1.TaintEffectNoExecute || t.Effect == ""
		if noExecute && (t.Key == corev1.TaintNodeNotReady || t.Key == "") {
			toleratesNotReady = true
		}
		if noExecute && (t.Key == corev1.TaintNodeUnreachable || t.Key == "") {
			toleratesUnreachable = true
		}
	}
	out := append([]corev1.Toleration(nil), tolerations...)
	if !toleratesNotReady {
		out = append(out, defaultToleration(corev1.TaintNodeNotReady))
	}
	if !toleratesUnreachable {
		out = append(out, defaultToleration(corev1.TaintNodeUnreachable))
	}
	return out
}

func defaultToleration(key string) corev1.Toleration {
	seconds := DefaultTolerationSeconds
	return corev1.Toleration{Key: key, Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute, TolerationSeconds: &seconds}
}

func defaultServiceAccount(p *corev1.Pod) string {
	if _, mirror := p.Annotations[corev1.MirrorPodAnnotationKey]; mirror || p.Spec.ServiceAccountName != "" {
		return p.Spec.ServiceAccountName
	}
	return "default"
}

func (s *Store) admitServiceAccount(pod *corev1.Pod) error {
	if podutil.IsMirrorPod(pod) {
		return nil
	}
	name := pod.Spec.ServiceAccountName
	if _, ok := s.objects[Key{Kind: KindServiceAccount, Namespace: pod.Namespace, Name: name}]; ok {
		return nil
	}
	return podForbidden(pod, fmt.Sprintf("error looking up service account %s/%s: serviceaccount %q not found", pod.Namespace, name, name))
}
