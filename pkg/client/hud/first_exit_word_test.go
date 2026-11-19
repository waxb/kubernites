package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestBogdansPodThatCrashedOnceIsNotCalledCrashLoopBackOff(t *testing.T) {
	defer ui.SetNamer(nil)
	once := model.PodView{Name: "porter", State: model.HearthGuttering, Reason: "Error"}
	looping := model.PodView{Name: "porter", State: model.HearthGuttering, Reason: "CrashLoopBackOff", Restarts: 2}
	ui.SetNamer(lexicon.TrueNames)
	for _, word := range []string{lexicon.TrueNames(HearthWord(once)), hutStateWord(once)} {
		if strings.Contains(strings.ToUpper(word), "CRASHLOOP") {
			t.Errorf("bogdan ch1 00:45: a Pod after one crash reads %q; Kubernetes restarts it at once and shows Error, CrashLoopBackOff only while the back-off delay runs after repeated crashes", word)
		}
	}
	if got := lexicon.TrueNames(HearthWord(once)); got != "ERROR" {
		t.Errorf("a Pod after one crash reads %q, want ERROR as kubectl get pods shows it", got)
	}
	if got := lexicon.TrueNames(HearthWord(looping)); !strings.Contains(strings.ToUpper(got), "CRASHLOOP") {
		t.Errorf("a Pod in back-off reads %q, want CrashLoopBackOff", got)
	}
	ui.SetStoryNamer(lexicon.StoryText)
	if got := HearthWord(once); got != "STOPPED" {
		t.Errorf("story mode: a hut whose worker exited once reads %q, want STOPPED", got)
	}
}
