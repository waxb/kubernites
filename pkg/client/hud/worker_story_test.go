package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestTheWorkerLineNamesASystemWorkerByItsStoryName(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetStoryNamer(func(s string) string { return lexicon.StoryText(lexicon.StoryNames(s)) })
	p := model.PodView{Focus: "kube-apiserver", People: []model.Kubernite{{Name: "kube-apiserver"}}}
	if got := KuberniteLine(p); strings.Contains(got, "kube-apiserver") || !strings.Contains(got, "Chancery") {
		t.Errorf("'WORKER kube-apiserver (1 OF 1)': story mode reads %q (the worker card draws literally)", got)
	}
	p = model.PodView{Focus: "porter", People: []model.Kubernite{{Name: "porter"}}}
	if got := KuberniteLine(p); got != "WORKER porter (1 OF 1)" {
		t.Errorf("a trade's worker keeps its name: %q", got)
	}
}
