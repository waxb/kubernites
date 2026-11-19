package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheHarborCardsNextPriceIsNeverCut(t *testing.T) {
	box := PortraitBox()
	x, w := box.X+box.W+8, CardBox.X+CardBox.W-box.X-box.W-14
	for _, ships := range []string{"caravels", "galleons", "carracks"} {
		for _, cost := range []float64{720, 200000, 1.25e6} {
			l := portBar(view.PortView{Open: true, Level: 2, Ships: "cogs", UpgradeCost: cost, Next: &view.PortNextView{Level: 3, Ships: ships}}, 530)
			for _, mode := range []string{"story", "names"} {
				if mode == "names" {
					ui.SetNamer(names.Kubernetes(64))
				}
				if room := BarRoom(l.Text, l.Suffix, x, w); len([]rune(l.Suffix)) > room {
					t.Errorf("%s: %q leaves room for %d characters of its suffix %q", mode, ui.Name(l.Text), room, l.Suffix)
				}
				ui.SetNamer(nil)
			}
		}
	}
}
