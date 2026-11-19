package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestTheSchedulerPageSpeaksKubernetesInNamesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	page := QuartermasterPage(fullFrame(t))
	for _, head := range page.Sections[0].Head {
		if shown := ui.Name(head); strings.Contains(shown, "EATEN") {
			t.Errorf("the Nodes table heads a column %q", shown)
		}
	}
	if got := housingRatio(Supply{Used: 0.4, Total: 3.5}); strings.Contains(got, "GI") || !strings.Contains(got, "Gi") {
		t.Errorf("memory reads %q, want the Gi unit", got)
	}
}
