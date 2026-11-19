package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestTheCalamityLastResortNeverLandsOnTheCrewBoard(t *testing.T) {
	f := boardFrame()
	f.State.Reveals = append(append([]string(nil), allReveals...), "stone")
	f.State.Calamities = append(f.State.Calamities, noDepot)
	h, _ := openBoard(t, f)
	f = settled(h, f)
	f.Beacons = []ui.Rect{{X: 0, Y: TopH, W: Width, H: 260}}
	r, ok := h.CalamityRect(f)
	if !ok {
		t.Fatal("no calamity card")
	}
	for _, board := range h.CrewRects(f) {
		if Overlaps(r, board) {
			t.Errorf("the calamity card %+v lands on the crew board %+v", r, board)
		}
	}
}
