package hud

import (
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

const labelMemory = 4096

var storyLabel = lexicon.StoryLabeler(labelMemory)

func Label(text string) string {
	if ui.Naming() {
		return text
	}
	return storyLabel(text)
}

func Glossed(name, k8s string) (string, string) {
	if !ui.Naming() {
		return name, k8s
	}
	named := strings.ToLower(ui.Name(name))
	for _, article := range []string{"the ", "a ", "an "} {
		named = strings.TrimPrefix(named, article)
	}
	if strings.HasPrefix(strings.ToLower(k8s), strings.TrimSuffix(named, "s")) {
		return k8s, ""
	}
	return name, k8s
}

func Unechoed(item string) string {
	name, k8s, ok := strings.Cut(item, ": ")
	if !ok {
		return item
	}
	if title, gloss := Glossed(name, k8s); gloss == "" {
		return title
	}
	return item
}

func GlossTip(name, k8s, text string) string {
	if title, gloss := Glossed(name, k8s); gloss == "" {
		return title + ": " + text
	}
	return name + " (" + view.K8sTerm(k8s) + "): " + text
}

func GlossLine(name, k8s string) string {
	title, gloss := Glossed(name, k8s)
	if gloss == "" {
		return title
	}
	return title + " (" + view.FlatGloss(gloss) + ")"
}
