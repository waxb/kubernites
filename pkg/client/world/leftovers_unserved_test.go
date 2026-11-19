package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func TestAShortfallThatHasNotSpoiledAnythingIsNotCalledSpoiling(t *testing.T) {
	k := view.KindView{Kind: "parcel", Name: "parcels", CrewName: "porters", Depot: "parcels", DepotOpen: true, Crew: 2,
		Waiting: 6, ArrivalRate: 2.3, ServedRate: 1.9, SpoiledRate: 0, Value: 1}
	u, ok := model.UnservedOf(k)
	if !ok {
		t.Fatal("a backlog is still shown")
	}
	if got := UnservedLabel(u); got == "PARCELS SPOILING: -0.4 GOLD/S" || got != "PARCELS BACKING UP: 0.4 GOLD/S AT RISK" {
		t.Fatalf("label %q while the board says nothing is lost", got)
	}
	k.SpoiledRate = 0.4
	k.Losses.Spoiled = 0.4
	u, _ = model.UnservedOf(k)
	if got := UnservedLabel(u); got != "PARCELS SPOILING: -0.4 GOLD/S" {
		t.Fatalf("label %q once boxes spoil", got)
	}
}
