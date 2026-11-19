package hud

import (
	"slices"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

const (
	chipEllipsis = "..."
	drainVerb    = "drain"
)

var chipShorthands = []struct{ long, short string }{
	{"deployment/", "deploy/"}, {"deployment ", "deploy "}, {"horizontalpodautoscaler", "hpa"},
}

func ChipFit(cmd string, width int) string {
	if ui.StoryNaming() {
		return fitChip(strings.TrimPrefix(lexicon.StoryText(ChipLabel(lexicon.CommandHead(cmd))), lexicon.StoryCommand+" "), width)
	}
	label := ChipLabel(cmd)
	if len([]rune(label)) <= width {
		return label
	}
	for _, s := range chipShorthands {
		label = strings.ReplaceAll(label, s.long, s.short)
	}
	return fitChip(strings.TrimPrefix(label, "kubectl "), width)
}

func fitChip(label string, width int) string {
	if short, ok := drainFit(label, width); ok {
		return short
	}
	return fitLabel(label, width)
}

func drainIgnoreK8s() string { return "--ignore-daemonsets" }

var drainIgnores = []string{drainIgnoreK8s(), lexicon.StoryText(drainIgnoreK8s())}

func drainFit(label string, width int) (string, bool) {
	words := strings.Fields(label)
	if len([]rune(label)) <= width || len(words) < 3 || words[0] != drainVerb {
		return "", false
	}
	added := slices.DeleteFunc(slices.Clone(words[2:]), func(w string) bool { return slices.Contains(drainIgnores, w) })
	if len(added) == len(words)-2 || len(added) == 0 {
		return "", false
	}
	for ; len(added) > 0; added = added[1:] {
		if short := strings.Join(append(append(slices.Clone(words[:2]), chipEllipsis), added...), " "); len([]rune(short)) <= width {
			return short, true
		}
	}
	return "", false
}

func fitLabel(label string, width int) string {
	if len([]rune(label)) <= width {
		return label
	}
	if short, ok := withoutTailFlags(label, width); ok {
		return short
	}
	runes := []rune(label)
	if width <= len(chipEllipsis)+2 {
		return string(runes[:max(width, 0)])
	}
	head := (width - len(chipEllipsis)) / 2
	tail := width - len(chipEllipsis) - head
	return string(runes[:head]) + chipEllipsis + string(runes[len(runes)-tail:])
}

func withoutTailFlags(label string, width int) (string, bool) {
	words := strings.Fields(label)
	for len(words) > 1 && strings.HasPrefix(words[len(words)-1], "-") {
		words = words[:len(words)-1]
		if short := strings.Join(words, " ") + " " + chipEllipsis; len([]rune(short)) <= width {
			return short, true
		}
	}
	return "", false
}
