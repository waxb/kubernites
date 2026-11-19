package view

import (
	"strings"
)

func K8sTerm(k8s string) string {
	term, _, _ := strings.Cut(k8s, " (")
	return term
}

func FlatGloss(k8s string) string {
	term, rest, ok := strings.Cut(k8s, " (")
	if !ok || !strings.HasSuffix(rest, ")") {
		return k8s
	}
	return term + ": " + strings.TrimSuffix(rest, ")")
}
