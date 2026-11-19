package model

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	FamiliarRaven = "raven"
	FamiliarHound = "hound"
	oomReason     = "OOMKilled"
)

func FamiliarsOf(p *corev1.Pod) []string {
	var out []string
	for _, c := range p.Spec.InitContainers {
		if c.RestartPolicy != nil && *c.RestartPolicy == corev1.ContainerRestartPolicyAlways {
			out = append(out, FamiliarKind(c.Image))
		}
	}
	return out
}

func FamiliarNamesOf(p *corev1.Pod) []string {
	var out []string
	for _, c := range p.Spec.InitContainers {
		if c.RestartPolicy != nil && *c.RestartPolicy == corev1.ContainerRestartPolicyAlways {
			out = append(out, c.Name)
		}
	}
	return out
}

func FamiliarKind(image string) string {
	name := image[strings.LastIndex(image, "/")+1:]
	if i := strings.IndexAny(name, ":@"); i >= 0 {
		name = name[:i]
	}
	return name
}

func LastExit(p *corev1.Pod) string {
	for _, cs := range p.Status.ContainerStatuses {
		if t := cs.LastTerminationState.Terminated; t != nil && t.Reason != "" {
			return t.Reason
		}
	}
	return ""
}

func (p PodView) OOM() bool {
	return p.Reason == oomReason || p.LastExit == oomReason
}
