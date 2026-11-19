package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func spoilingLedgers(sc Scene, value float64) Scene {
	next := sc
	kinds := append([]view.KindView(nil), sc.Freight.Kinds...)
	kinds[1].ArrivalRate, kinds[1].Value, kinds[1].Losses = 3.2, value, view.LossView{NoDepot: 3.2}
	next.Freight.Kinds = kinds
	return next
}

func TestAPriceKeepsItsPlaceWhenTheBeaconBesideItMovesOrGoesAndItsAmountChanges(t *testing.T) {
	sc, r := harborScene(t)
	cam := cameraOn(sc, r)
	sc = spoilingLedgers(sc, 12.5)
	at, _ := YardFocus(sc.Site, "ledger")
	sc.Goal = &at
	first := r.UnservedMarks(cam, sc)
	if len(first) != 1 {
		t.Fatalf("marks = %+v", first)
	}
	home := first[0].Box
	nudged := Vec{X: at.X + .15, Y: at.Y - .15}
	for i, goal := range []*Vec{nil, &nudged, &at} {
		sc.Goal = goal
		sc = spoilingLedgers(sc, 12.5+float64(i))
		got := r.UnservedMarks(cam, sc)
		if len(got) != 1 || got[0].Box.X != home.X || got[0].Box.Y != home.Y {
			t.Fatalf("step %d: the label moved from %+v to %+v", i, home, got)
		}
		for _, b := range r.BeaconRects(cam, sc) {
			if Overlaps(got[0].Box, b) {
				t.Fatalf("step %d: the label %+v sits under the beacon %+v", i, got[0].Box, b)
			}
		}
	}
}
