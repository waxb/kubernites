package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func ResultTitle(cmd string) string {
	if !ui.StoryNaming() {
		return ChipLabel(cmd)
	}
	words := strings.Fields(strings.TrimPrefix(strings.TrimSpace(cmd), "kubectl "))
	if len(words) > 0 && words[0] == lexicon.StoryCommand {
		words = words[1:]
	}
	if len(words) == 0 {
		return "The Stone speaks"
	}
	switch words[0] {
	case "get", "describe":
		return kindTitle(words[1:])
	case "logs":
		return "Chronicle: " + lastName(words[1:])
	case "top":
		return "The Omens"
	case "api-resources":
		return "What the Stone reads"
	case "explain":
		return "About " + kindTitle(words[1:])
	case "rollout":
		return "The roll-out"
	}
	return "The Stone speaks"
}

func kindTitle(rest []string) string {
	args := []string{}
	for _, w := range rest {
		if !strings.HasPrefix(w, "-") {
			args = append(args, w)
		}
	}
	if len(args) == 0 {
		return "The Stone speaks"
	}
	if strings.Contains(args[0], ",") {
		return "The realm"
	}
	noun := storyNoun(args[0])
	if len(args) > 1 {
		return strings.TrimSuffix(noun, "s") + " " + args[1]
	}
	return "The " + noun
}

func lastName(rest []string) string {
	for i := len(rest) - 1; i >= 0; i-- {
		if !strings.HasPrefix(rest[i], "-") {
			return rest[i]
		}
	}
	return "the hut"
}

var kindAliases = map[string]string{
	"pod": "pods", "po": "pods", "node": "nodes", "no": "nodes", "deployment": "deployments", "deploy": "deployments",
	"service": "services", "svc": "services", "replicaset": "replicasets", "rs": "replicasets", "daemonset": "daemonsets",
	"ds": "daemonsets", "event": "events", "ev": "events", "namespace": "namespaces", "ns": "namespaces", "configmap": "configmaps",
	"cm": "configmaps", "hpa": "horizontalpodautoscalers", "horizontalpodautoscaler": "horizontalpodautoscalers",
	"endpointslice": "endpointslices", "limitrange": "limitranges", "limits": "limitranges", "priorityclass": "priorityclasses",
	"pc": "priorityclasses", "statefulset": "statefulsets", "sts": "statefulsets",
}

func storyNoun(token string) string {
	plural, _, _ := strings.Cut(strings.ToLower(token), ".")
	if alias, ok := kindAliases[plural]; ok {
		plural = alias
	}
	if noun, ok := apiNouns[plural]; ok {
		return noun
	}
	return lexicon.StoryText(plural)
}
