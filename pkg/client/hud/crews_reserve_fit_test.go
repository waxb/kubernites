package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestACrewsRowShowsItsWholeReserve(t *testing.T) {
	rr := ui.Rect{X: 0, Y: 0, W: crewsW - 2*crewsPad, H: crewsRowH}
	for _, mode := range []string{"story", "names"} {
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		}
		for _, word := range []string{"", "BOTTLENECK", "NO DEPOT"} {
			x, w := crewsReserveSpan(rr, word)
			shown := crewsReserveText(reserveLine(1200, 1152<<20), reserveShort(1200, 1152<<20), w)
			if clipped := ui.Clip(shown, ui.MaxChars(w, 1)); strings.HasSuffix(clipped, "...") {
				t.Errorf("%s, state %q: the reserve reads %q", mode, word, clipped)
			}
			if word != "" && x+w > crewsStateAt(rr, word) {
				t.Errorf("%s: the reserve runs into the state word %q", mode, word)
			}
			if word != "" && crewsStateAt(rr, word)+(ui.Font{}).Width(word, 1) > rr.X+rr.W-2*crewsBtnW-4 {
				t.Errorf("%s: the state word %q runs under the row's buttons", mode, word)
			}
		}
		ui.SetNamer(nil)
	}
}
