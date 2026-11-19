package model

import (
	corev1 "k8s.io/api/core/v1"
)

func TaintSpec(t corev1.Taint) string {
	spec := t.Key
	if t.Value != "" {
		spec += "=" + t.Value
	}
	return spec + ":" + string(t.Effect)
}

func TaintSpecs(taints []corev1.Taint) []string {
	if len(taints) == 0 {
		return nil
	}
	out := make([]string, 0, len(taints))
	for _, t := range taints {
		out = append(out, TaintSpec(t))
	}
	return out
}
