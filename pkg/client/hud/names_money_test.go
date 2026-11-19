package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestNamesModeSaysMoneyWhereTheStorySaysGold(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	f := Frame{}
	f.State.Archive = &view.ArchiveView{Unsettled: 120, Seals: 4, SettleIn: 30}
	texts := []string{PortLine(view.PortView{OpenCost: 150}), archiveRowNote(f)}
	for _, l := range archiveCard(f).Lines {
		texts = append(texts, l.Text)
	}
	for _, text := range texts {
		if strings.Contains(strings.ToLower(text), "gold") {
			t.Errorf("NAMES mode reads %q", text)
		}
	}
}
