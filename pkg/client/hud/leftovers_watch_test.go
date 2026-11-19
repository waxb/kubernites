package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheWatchStepNamesTheButtonTheCrewCardHas(t *testing.T) {
	f := Frame{Guide: Guide{Button: view.WatchButton}}
	text := "ready: set the Watch on clerks (" + view.ButtonToken + ")"
	if got := f.QuoteObjective(text); got != "ready: set the Watch on clerks (WATCH)" {
		t.Fatalf("story mode reads %q", got)
	}
	ui.SetNamer(func(s string) string { return s })
	defer ui.SetNamer(nil)
	if got := f.QuoteObjective(text); got != "ready: set the Watch on clerks (PROBES)" {
		t.Fatalf("names mode reads %q", got)
	}
}
