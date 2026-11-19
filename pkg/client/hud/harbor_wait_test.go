package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheHarborBoardPricesStrandedDispatches(t *testing.T) {
	f := fullFrame(t)
	f.Selection = Selection{Kind: SelHaven}
	f.State.Keeps = []view.KeepView{{Name: "stone-keep-2", Stranded: 3, StrandedGold: 41000}, {Name: "lone-tower"}}
	f.Site.Structures = append(f.Site.Structures, view.Structure{ID: "k9", Kind: view.StructKeep, Label: "Morrow Keep", Object: view.ObjectRef{Kind: "Node", Name: "stone-keep-2"}})
	if !containsLine(HarborCard(f).Lines, "DISPATCHES WAITING AT MORROW KEEP: NO WARDEN THERE, 41K GOLD/S") {
		t.Fatalf("lines = %+v", HarborCard(f).Lines)
	}
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	if !containsLine(HarborCard(f).Lines, "courier Pods on stone-keep-2 have no local post endpoint: -$41K/s") {
		t.Fatalf("NAMES lines = %+v", HarborCard(f).Lines)
	}
}

func TestTheHarborBoardNamesAWaitingTreasure(t *testing.T) {
	f := fullFrame(t)
	f.Selection = Selection{Kind: SelHaven}
	f.State.Treasure = []view.TreasureView{{Kind: "dispatch", Boxes: "dispatches", Crew: "couriers"}}
	if !containsLine(HarborCard(f).Lines, "Dispatches wait at sea until their depot stands: their treasure waits") {
		t.Fatalf("lines = %+v", HarborCard(f).Lines)
	}
}
