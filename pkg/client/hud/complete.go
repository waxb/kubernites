package hud

import (
	"sort"
	"strings"
)

var stoneVerbs = []string{
	"api-resources", "apply", "cluster-info", "cordon", "create", "delete", "describe", "edit", "explain", "get",
	"help", "label", "logs", "patch", "replace", "run", "scale", "set", "top", "uncordon", "version",
}

type resourceWord struct{ plural, singular, short string }

var resourceWords = []resourceWord{
	{"pods", "pod", "po"}, {"nodes", "node", "no"}, {"replicasets", "replicaset", "rs"}, {"services", "service", "svc"},
	{"endpointslices", "endpointslice", ""}, {"events", "event", "ev"}, {"namespaces", "namespace", "ns"},
	{"configmaps", "configmap", "cm"},
}

var typedVerbs = map[string]bool{"get": true, "describe": true, "delete": true, "explain": true, "label": true, "scale": true, "edit": true}

var valueFlags = map[string]bool{"-n": true, "--namespace": true, "-o": true, "--output": true, "-l": true, "--selector": true, "-c": true, "--container": true}

var stoneFlags = []string{"-A", "--all-namespaces", "-n", "--namespace", "-o", "--output", "-l", "--selector", "--show-labels", "-c", "--container"}

var outputFormats = []string{"wide", "yaml", "json", "name"}

type Vocabulary struct {
	Names  map[string][]string
	Spaced map[string]map[string][]string
}

type nameScope struct {
	namespace string
	all       bool
}

type Completion struct {
	Token      string
	Ghost      string
	Candidates []string
}

func canonicalResource(word string) (string, bool) {
	for _, r := range resourceWords {
		if word == r.plural || word == r.singular || (r.short != "" && word == r.short) {
			return r.plural, true
		}
	}
	return "", false
}

func allResourceWords() []string {
	out := []string{}
	for _, r := range resourceWords {
		out = append(out, r.plural, r.singular)
		if r.short != "" {
			out = append(out, r.short)
		}
	}
	return out
}

func positional(tokens []string) []string {
	out := []string{}
	skip := false
	for i, t := range tokens {
		switch {
		case skip:
			skip = false
		case i == 0 && t == "kubectl":
		case strings.HasPrefix(t, "-"):
			skip = valueFlags[t]
		default:
			out = append(out, t)
		}
	}
	return out
}

func (v Vocabulary) wordsAt(words []string, scope nameScope) []string {
	switch len(words) {
	case 0:
		return stoneVerbs
	case 1:
		switch verb := words[0]; {
		case typedVerbs[verb]:
			return allResourceWords()
		case verb == "logs":
			return v.namesIn("pods", scope)
		case verb == "top":
			return []string{"pods", "pod", "nodes", "node"}
		case verb == "cordon" || verb == "uncordon":
			return v.Names["nodes"]
		case verb == "help":
			return stoneVerbs
		}
	case 2:
		if kind, ok := canonicalResource(words[1]); ok && (typedVerbs[words[0]] || words[0] == "top") {
			return v.namesIn(kind, scope)
		}
	}
	return nil
}

func (v Vocabulary) namesIn(kind string, scope nameScope) []string {
	spaces, ok := v.Spaced[kind]
	if !ok || scope.all {
		return v.Names[kind]
	}
	return spaces[scope.namespace]
}

func Complete(line string, v Vocabulary) Completion {
	if strings.TrimSpace(line) == "" {
		return Completion{}
	}
	tokens := strings.Fields(line)
	if strings.HasSuffix(line, " ") {
		return completeAfter(tokens[len(tokens)-1], v)
	}
	cur := tokens[len(tokens)-1]
	candidates := matching(v.wordsFor(tokens[:len(tokens)-1], cur), cur)
	c := Completion{Token: cur, Candidates: candidates}
	if len(candidates) > 0 {
		c.Ghost = strings.TrimPrefix(CommonPrefix(candidates), cur)
	}
	return c
}

func completeAfter(last string, v Vocabulary) Completion {
	if !valueFlags[last] {
		return Completion{}
	}
	return Completion{Candidates: matching(v.flagValues(last), "")}
}

func (v Vocabulary) wordsFor(prev []string, cur string) []string {
	switch {
	case len(prev) > 0 && valueFlags[prev[len(prev)-1]]:
		return v.flagValues(prev[len(prev)-1])
	case strings.HasPrefix(cur, "-") && strings.Contains(cur, "="):
		flag, _, _ := strings.Cut(cur, "=")
		return prefixed(flag+"=", v.flagValues(flag))
	case strings.HasPrefix(cur, "-"):
		return stoneFlags
	}
	return v.wordsAt(positional(prev), scopeOf(prev))
}

func (v Vocabulary) flagValues(flag string) []string {
	switch flag {
	case "-n", "--namespace":
		return v.Names["namespaces"]
	case "-o", "--output":
		return outputFormats
	}
	return nil
}

func prefixed(prefix string, words []string) []string {
	out := make([]string, 0, len(words))
	for _, w := range words {
		out = append(out, prefix+w)
	}
	return out
}

func scopeOf(tokens []string) nameScope {
	scope := nameScope{namespace: "default"}
	for i, t := range tokens {
		name, value, attached := strings.Cut(t, "=")
		switch {
		case name == "-A" || name == "--all-namespaces":
			scope.all = value != "false"
		case (name == "-n" || name == "--namespace") && attached:
			scope.namespace = value
		case (name == "-n" || name == "--namespace") && i+1 < len(tokens):
			scope.namespace = tokens[i+1]
		case strings.HasPrefix(t, "-n") && !strings.HasPrefix(t, "--") && len(t) > 2:
			scope.namespace = t[2:]
		}
	}
	return scope
}

func matching(words []string, prefix string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, w := range words {
		if strings.HasPrefix(w, prefix) && w != prefix && !seen[w] {
			seen[w] = true
			out = append(out, w)
		}
	}
	sort.Strings(out)
	return out
}

func CommonPrefix(words []string) string {
	if len(words) == 0 {
		return ""
	}
	prefix := words[0]
	for _, w := range words[1:] {
		for !strings.HasPrefix(w, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}
