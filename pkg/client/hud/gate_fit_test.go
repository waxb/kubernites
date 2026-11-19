package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestTheQuayCardsLinesFitInBothModes(t *testing.T) {
	f := fullFrame(t)
	for _, mode := range []string{"story", "names"} {
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		}
		for _, l := range GateLines(f) {
			if w := (ui.Font{}).Width(l, 1); w > CardBox.W-16 {
				t.Errorf("%s: %q is %d px, the card holds %d", mode, ui.Name(l), w, CardBox.W-16)
			}
		}
		ui.SetNamer(nil)
	}
}
