package lexicon

import (
	"regexp"
	"strings"
)

var storyManifestKeys = map[string]string{
	"apiVersion": "book", "kind": "decree", "metadata": "about", "spec": "plan", "status": "report",
	"namespace": "fief", "labels": "badges", "annotations": "marginalia", "replicas": "count",
	"selector": "looksFor", "containers": "workers", "initContainers": "squires", "image": "trade",
	"requests": "pledge", "limits": "binding", "cpu": "food", "memory": "housing",
	"ephemeral-storage": "cellar", "ports": "doors", "port": "door", "targetPort": "hutDoor",
	"containerPort": "workerDoor", "nodeSelector": "keepChoice", "nodeName": "keep",
	"tolerations": "charms", "taints": "wards", "minReplicas": "floor", "maxReplicas": "ceiling",
	"scaleTargetRef": "keeps", "readinessProbe": "shutters", "livenessProbe": "bell", "httpGet": "ask",
	"periodSeconds": "everySeconds", "failureThreshold": "missesAllowed", "timeoutSeconds": "patienceSeconds",
	"initialDelaySeconds": "waitSeconds",
}

var storyManifestKinds = map[string]string{
	"Deployment": "crew", "ReplicaSet": "shift", "Pod": "hut", "Service": "depot", "ConfigMap": "scroll",
	"Secret": "sealed-scroll", "DaemonSet": "shrine", "StatefulSet": "order", "HorizontalPodAutoscaler": "horn",
	"PersistentVolume": "vault", "PersistentVolumeClaim": "deed", "StorageClass": "vault-kind",
	"Namespace": "fief", "Job": "quest", "CronJob": "ritual", "ServiceAccount": "writ",
}

var storyManifestBooks = map[string]string{
	"v1": "realm/v1", "apps/v1": "crews/v1", "autoscaling/v2": "horns/v2", "autoscaling/v1": "horns/v1",
	"batch/v1": "quests/v1", "storage.k8s.io/v1": "vaults/v1", "policy/v1": "rules/v1",
}

var (
	kubeManifestKeys  = invert(storyManifestKeys, map[string]string{"badges": "labels"})
	kubeManifestKinds = invert(storyManifestKinds, nil)
	kubeManifestBooks = invert(storyManifestBooks, nil)
	manifestKeyLine   = regexp.MustCompile(`^(\s*)((?:- )?)([A-Za-z_][A-Za-z0-9_.-]*):(\s.*)?$`)
	manifestContent   = map[string]bool{"labels": true, "matchLabels": true, "annotations": true, "data": true, "nodeSelector": true, "badges": true, "marginalia": true, "keepChoice": true}
)

type manifestScope struct {
	indent int
	key    string
}

type manifestLine struct {
	head, dash, key, rest, parent string
}

func StoryManifest(text string) string {
	return walkManifest(text, storyManifestLine)
}

func ManifestFromStory(text string) string {
	return walkManifest(text, kubeManifestLine)
}

func walkManifest(text string, rename func(manifestLine) (string, string)) string {
	lines := strings.Split(text, "\n")
	stack := []manifestScope{}
	for i, line := range lines {
		m := manifestKeyLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		at := len(m[1]) + len(m[2])
		for len(stack) > 0 && stack[len(stack)-1].indent >= at {
			stack = stack[:len(stack)-1]
		}
		parent := ""
		if len(stack) > 0 {
			parent = stack[len(stack)-1].key
		}
		stack = append(stack, manifestScope{indent: at, key: m[3]})
		if manifestContent[parent] {
			continue
		}
		key, rest := rename(manifestLine{head: m[1], dash: m[2], key: m[3], rest: m[4], parent: parent})
		lines[i] = m[1] + m[2] + key + ":" + rest
	}
	return strings.Join(lines, "\n")
}

func storyManifestLine(l manifestLine) (string, string) {
	key := l.key
	if story, ok := storyManifestKeys[key]; ok && (key != "status" || l.parent == "") {
		key = story
	}
	if l.key == "matchLabels" {
		key = "badges"
	}
	return key, mapManifestValue(l.key, l.parent, l.rest, storyManifestValue)
}

func kubeManifestLine(l manifestLine) (string, string) {
	key := l.key
	if kube, ok := kubeManifestKeys[key]; ok && (key != "report" || l.parent == "") {
		key = kube
	}
	if l.key == "badges" && l.parent == "looksFor" {
		key = "matchLabels"
	}
	return key, mapManifestValue(key, l.parent, l.rest, kubeManifestValue)
}

func mapManifestValue(key, parent, rest string, convert func(key, parent, value string) string) string {
	value, comment, _ := strings.Cut(rest, " #")
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return rest
	}
	quote := ""
	if len(trimmed) > 1 && (trimmed[0] == '"' || trimmed[0] == '\'') && trimmed[len(trimmed)-1] == trimmed[0] {
		quote, trimmed = trimmed[:1], trimmed[1:len(trimmed)-1]
	}
	next := convert(key, parent, trimmed)
	if next == trimmed {
		return rest
	}
	out := " " + quote + next + quote
	if comment != "" || strings.Contains(rest, " #") {
		out += " #" + comment
	}
	return out
}

func storyManifestValue(key, parent, value string) string {
	switch {
	case key == "apiVersion":
		return lookupOr(storyManifestBooks, value)
	case key == "kind":
		return lookupOr(storyManifestKinds, value)
	case key == "cpu":
		return rationsOf(value)
	case key == "memory" || key == "ephemeral-storage":
		return mebiOf(value)
	case key == "name" && parent == "resource":
		return lookupOr(map[string]string{"cpu": "food", "memory": "housing"}, value)
	case key == "message":
		return StoryText(value)
	}
	return value
}

func kubeManifestValue(key, parent, value string) string {
	switch {
	case key == "apiVersion":
		return lookupOr(kubeManifestBooks, value)
	case key == "kind":
		return lookupOr(kubeManifestKinds, value)
	case key == "cpu" && digitsOnly(value):
		return value + "m"
	case (key == "memory" || key == "ephemeral-storage") && digitsOnly(value):
		return value + "Mi"
	case key == "name" && parent == "resource":
		return lookupOr(map[string]string{"food": "cpu", "housing": "memory"}, value)
	}
	return value
}

func lookupOr(table map[string]string, value string) string {
	if mapped, ok := table[value]; ok {
		return mapped
	}
	return value
}

func digitsOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func IsStoryManifest(text string) bool {
	head := strings.TrimLeft(text, " \n\t")
	return strings.HasPrefix(head, "book:") || strings.HasPrefix(head, "---\nbook:")
}

func StoryKeyOf(key string) string {
	if story, ok := storyManifestKeys[key]; ok {
		return story
	}
	return key
}

func StoryKind(kind string) (string, bool) {
	name, ok := storyManifestKinds[kind]
	return name, ok
}
