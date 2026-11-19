package hud

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	systemPodsHint  = "The Citadel's huts live in kube-system: try get pods -n kube-system, or -A for all."
	defaultWritHint = "The Stewards write each fief's default writ once the Stewards' Hall stands: build the Stewards' Hall, then speak again."
)

var (
	noneFound = regexp.MustCompile(`^No resources found in (\S+) namespace\.`)
	notFound  = regexp.MustCompile(`\(NotFound\): (\S+) "([^"]+)" not found`)
	noWrit    = regexp.MustCompile(`serviceaccount "default" not found`)
)

func ConsoleHints(cmd string, res view.CommandResult, v Vocabulary, seen bool) ([]string, bool) {
	tokens := strings.Fields(cmd)
	scope, words := scopeOf(tokens), positional(tokens)
	listing := len(words) == 2 && words[0] == "get" && isPods(words[1])
	if listing && (scope.all || scope.namespace == systemNS) {
		seen = true
	}
	var out []string
	if hint := missingNamespace(res.Output, v); hint != "" {
		out = append(out, hint)
	}
	if hint := elsewhere(res.Output, scope, v); hint != "" {
		out = append(out, hint)
	}
	if noWrit.MatchString(res.Output) {
		out = append(out, defaultWritHint)
	}
	if listing && res.OK && !seen && !scope.all && scope.namespace == defaultNS {
		out = append(out, systemPodsHint)
	}
	return out, seen
}

func isPods(word string) bool {
	kind, ok := canonicalResource(word)
	return ok && kind == "pods"
}

func missingNamespace(output string, v Vocabulary) string {
	m := noneFound.FindStringSubmatch(output)
	spaces := v.Names["namespaces"]
	if m == nil || len(spaces) == 0 || slices.Contains(spaces, m[1]) {
		return ""
	}
	for _, ns := range spaces {
		if strings.Contains(ns, m[1]) {
			return fmt.Sprintf("No namespace %q: did you mean -n %s? (get namespaces lists them all)", m[1], ns)
		}
	}
	return fmt.Sprintf("No namespace %q. Namespaces: %s", m[1], strings.Join(spaces, ", "))
}

func elsewhere(output string, scope nameScope, v Vocabulary) string {
	m := notFound.FindStringSubmatch(output)
	if m == nil {
		return ""
	}
	for _, ns := range sortedSpaces(v.Spaced[m[1]]) {
		if ns != scope.namespace && slices.Contains(v.Spaced[m[1]][ns], m[2]) {
			return fmt.Sprintf("%s lives in the %s namespace: add -n %s", m[2], ns, ns)
		}
	}
	return ""
}

func sortedSpaces(spaces map[string][]string) []string {
	out := make([]string, 0, len(spaces))
	for ns := range spaces {
		out = append(out, ns)
	}
	slices.Sort(out)
	return out
}

func (h *HUD) hintStone(cmd string, res view.CommandResult) {
	hints, seen := ConsoleHints(cmd, res, h.vocab, h.Stone.systemSeen)
	h.Stone.systemSeen = seen
	if len(hints) > 0 {
		h.Stone.Append(LineHint, hints...)
	}
	h.glossStone(SplitOutput(res.Output))
}

func (h *HUD) StoneLines() []StoneLine {
	return slices.Clone(h.Stone.Output)
}

func ConsoleText(l StoneLine) string {
	switch {
	case l.Kind == LineRaw || l.Kind == LineRawError || l.Kind == LineRawEcho:
		return l.Text
	case ui.StoryNaming() && (l.Kind == LineOutput || l.Kind == LineError):
		return l.Text
	case ui.StoryNaming():
		return ui.Name(l.Text)
	case l.Kind == LineHint:
		return ui.Name(l.Text)
	case l.Kind == LineEcho || tabular(l.Text) || l.Kind == LineOutput && keyValue(l.Text):
		return l.Text
	}
	return nameOutsideQuotes(l.Text)
}

var keyLine = regexp.MustCompile(`^\s*(- )?[A-Za-z][A-Za-z0-9_./-]*:(\s|$)`)

func keyValue(text string) bool {
	return keyLine.MatchString(text)
}

func tabular(text string) bool {
	return strings.Contains(strings.TrimLeft(text, " "), "  ")
}

func nameOutsideQuotes(text string) string {
	parts := strings.Split(text, `"`)
	for i := 0; i < len(parts); i += 2 {
		parts[i] = ui.Name(parts[i])
	}
	return strings.Join(parts, `"`)
}
