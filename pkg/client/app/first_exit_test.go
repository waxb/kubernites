package app

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestAFirstExitIsNotCalledACrashLoop(t *testing.T) {
	_, first, ok := hearthAlert(model.PodView{Name: "porter-1", State: model.HearthGuttering, Reason: "Error"})
	if !ok || strings.Contains(first, "guttering") || strings.Contains(lexicon.TrueNames(first), "crash-looping") {
		t.Errorf("a worker that exited once, with 0 restarts, reads %q (NAMES %q): crash-looping means repeated restarts with back-off", first, lexicon.TrueNames(first))
	}
	_, again, _ := hearthAlert(model.PodView{Name: "porter-1", State: model.HearthGuttering, Reason: "CrashLoopBackOff", Restarts: 2})
	if !strings.Contains(again, "guttering") {
		t.Errorf("a worker restarting again and again still gutters: %q", again)
	}
}
