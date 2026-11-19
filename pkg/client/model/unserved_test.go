package model

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func ledgersNobodyCarries() view.KindView {
	return view.KindView{Kind: "ledger", Name: "ledgers", Open: true, Waiting: 20, ArrivalRate: 3.2, Value: 12.5,
		Depot: "ledgers", CrewName: "clerks", Losses: view.LossView{NoDepot: 1.4}}
}

func TestUnservedCountsCargoNoCrewCarries(t *testing.T) {
	fv := view.FreightView{Kinds: []view.KindView{
		{Kind: "parcel", Name: "parcels", Open: true, Waiting: 2, ArrivalRate: 2, ServedRate: 2, Value: 1, DepotOpen: true, Crew: 4},
		ledgersNobodyCarries(),
	}}
	got := UnservedKinds(fv)
	if len(got) != 1 {
		t.Fatalf("unserved = %+v", got)
	}
	u := got[0]
	if u.Kind != "ledger" || u.Rate != 1.4 || u.Gold != 17.5 || u.Cause != UnservedNoDepot || u.Crew != "clerks" || u.Depot != "ledgers" {
		t.Fatalf("ledgers = %+v", u)
	}
	if !u.Stranded() {
		t.Fatal("ledgers with no depot are not stranded")
	}
}

func TestUnservedTakesTheLossWhenItOutrunsThePile(t *testing.T) {
	k := ledgersNobodyCarries()
	k.Waiting, k.Losses = 0, view.LossView{NoDepot: 2, QuayFull: 1.5, Blocked: 9}
	if got := UnservedRate(k); got != 3.5 {
		t.Fatalf("rate = %v, want the boxes lost on the quay, not those dropped in flight", got)
	}
}

func TestAKeptUpCrewLeavesNothingUnserved(t *testing.T) {
	k := view.KindView{Kind: "parcel", Name: "parcels", Open: true, Waiting: 0.4, ArrivalRate: 2, ServedRate: 1.2, Value: 1, DepotOpen: true, Crew: 2}
	if got := UnservedKinds(view.FreightView{Kinds: []view.KindView{k}}); len(got) != 0 {
		t.Fatalf("a crew with an empty pile is short = %+v", got)
	}
}

func TestTooFewCarriersAreShortNotStranded(t *testing.T) {
	k := view.KindView{Kind: "parcel", Name: "parcels", Open: true, Waiting: 9, ArrivalRate: 2, ServedRate: 1.2, Value: 1, DepotOpen: true, Crew: 2,
		Losses: view.LossView{Spoiled: 0.3}}
	got := UnservedKinds(view.FreightView{Kinds: []view.KindView{k}})
	if len(got) != 1 || got[0].Cause != UnservedFew || got[0].Stranded() {
		t.Fatalf("short crew = %+v", got)
	}
	if got[0].Rate < 0.79 || got[0].Rate > 0.81 {
		t.Fatalf("short rate = %v", got[0].Rate)
	}
}

func TestADepotWithNoCrewStrandsItsBoxes(t *testing.T) {
	k := ledgersNobodyCarries()
	k.DepotOpen, k.Losses = true, view.LossView{NoCrew: 3}
	got := UnservedKinds(view.FreightView{Kinds: []view.KindView{k}})
	if len(got) != 1 || got[0].Cause != UnservedNoCrew || !got[0].Stranded() {
		t.Fatalf("empty depot = %+v", got)
	}
}

func TestTheDoorPorterCountsAsACrewForParcels(t *testing.T) {
	k := view.KindView{Kind: "parcel", Name: "parcels", Open: true, Waiting: 4, ArrivalRate: 1, ServedRate: 0.4, Value: 1, Door: true,
		Losses: view.LossView{Spoiled: 0.6}}
	got := UnservedKinds(view.FreightView{Kinds: []view.KindView{k}})
	if len(got) != 1 || got[0].Stranded() || got[0].Cause != UnservedFew {
		t.Fatalf("door porter = %+v", got)
	}
}

func TestTheWorstLossComesFirst(t *testing.T) {
	a := ledgersNobodyCarries()
	b := view.KindView{Kind: "crate", Name: "crates", Open: true, Waiting: 5, ArrivalRate: 1, Value: 100, CrewName: "haulers", Depot: "crates"}
	got := UnservedKinds(view.FreightView{Kinds: []view.KindView{a, b}})
	if len(got) != 2 || got[0].Kind != "crate" || got[1].Kind != "ledger" {
		t.Fatalf("order = %+v", got)
	}
	if w, ok := WorstUnserved(view.FreightView{Kinds: []view.KindView{a, b}}); !ok || w.Kind != "crate" {
		t.Fatalf("worst = %+v", w)
	}
}

func TestATrickleIsNotAnAlarm(t *testing.T) {
	k := ledgersNobodyCarries()
	k.Waiting, k.ArrivalRate, k.Losses = 0.5, 0.02, view.LossView{NoDepot: 0.02}
	if got := UnservedKinds(view.FreightView{Kinds: []view.KindView{k}}); len(got) != 0 {
		t.Fatalf("a trickle = %+v", got)
	}
}

func TestASpoilingFigureNeverRunsAboveWhatIsLost(t *testing.T) {
	k := view.KindView{Kind: "parcel", Name: "parcels", Open: true, Waiting: 30, ArrivalRate: 5.2, ServedRate: 1.4, SpoiledRate: 2.3, Value: 1, DepotOpen: true, Crew: 3,
		Losses: view.LossView{Spoiled: 2.3}}
	got := UnservedKinds(view.FreightView{Kinds: []view.KindView{k}})
	if len(got) != 1 || got[0].Queued || got[0].Rate != 2.3 {
		t.Fatalf("berths capture: the harbor says LOST 2.3/S and the card says SPOILING %+v", got)
	}
	k = ledgersNobodyCarries()
	k.Losses = view.LossView{}
	got = UnservedKinds(view.FreightView{Kinds: []view.KindView{k}})
	if len(got) != 1 || !got[0].Queued || got[0].Rate != 3.2 {
		t.Fatalf("ledgers pile up before they spoil, and the card reads %+v", got)
	}
	if text := got[0]; text.Gold != 40 {
		t.Fatalf("the backing-up ledgers put %v requests a second at risk", text.Gold)
	}
}
