package world

import (
	"math"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestADebutWaitsForTheTradesRevealAndEndsWhenItsCrewCarries(t *testing.T) {
	fv := view.FreightView{Kinds: []view.KindView{{Kind: "parcel", Crew: 3, Open: true}, {Kind: "ledger", Open: true}, {Kind: "crate"}}}
	fv.Seals.ArrivalRate = .2
	if got := DebutsOf([]string{"serve", "gold"}, fv); len(got) != 0 {
		t.Fatalf("debuts before any trade arrives = %+v", got)
	}
	got := DebutsOf([]string{"clerks"}, fv)
	if len(got) != 1 || got[0].Kind != "ledger" || got[0].Role != "clerk" {
		t.Fatalf("clerk debut = %+v", got)
	}
	fv.Kinds[1].Crew = 1
	if got := DebutsOf([]string{"clerks"}, fv); len(got) != 0 {
		t.Fatalf("a debut outlives the crew = %+v", got)
	}
	if got := DebutsOf([]string{"sealwrights"}, fv); len(got) != 1 || got[0].Role != "sealwright" {
		t.Fatalf("sealwright debut = %+v", got)
	}
	fv.Seals.Crew = 2
	if got := DebutsOf([]string{"sealwrights"}, fv); len(got) != 0 {
		t.Fatalf("sealwrights at work still debut = %+v", got)
	}
}

func TestTheFirstWorkerWalksOffTheShipToItsPile(t *testing.T) {
	sc, r := harborScene(t)
	cam := cameraOn(sc, r)
	sc.Debuts = []Debut{{Kind: "ledger", Role: "clerk"}}
	h := r.HarborOf()
	plank := h.Plans[0].Berths[0].Plank
	yard, _ := YardFocus(sc.Site, "ledger")
	start, ok := r.DebutPose(sc, sc.Debuts[0])
	if !ok || start.Action != "walk" || math.Hypot(start.Pos.X-plank.X, start.Pos.Y-plank.Y) > .5 {
		t.Fatalf("the debut starts at %+v, the plank is %v", start, plank)
	}
	sc.Time += debutWalk + 1
	end, _ := r.DebutPose(sc, sc.Debuts[0])
	if end.Action == "walk" || math.Hypot(end.Pos.X-yard.X, end.Pos.Y-yard.Y) > debutAside+1 {
		t.Fatalf("after its walk the clerk stands at %+v, the yard is %v", end, yard)
	}
	if items := r.debutItems(cam, sc); len(items) != 1 {
		t.Fatalf("debut drawables = %d", len(items))
	}
}

func TestTheFirstWorkerWalksOnToWhereItsStallGoes(t *testing.T) {
	sc, r := harborScene(t)
	cameraOn(sc, r)
	yard, _ := YardFocus(sc.Site, "ledger")
	stall := Vec{X: yard.X + 6, Y: yard.Y + 5}
	sc.Debuts = []Debut{{Kind: "ledger", Role: "clerk", Stall: stall, HasStall: true}}
	if _, ok := r.DebutPose(sc, sc.Debuts[0]); !ok {
		t.Fatal("no debut pose")
	}
	sc.Time += debutWalk / 2
	mid, _ := r.DebutPose(sc, sc.Debuts[0])
	if mid.Action != "walk" {
		t.Fatalf("halfway the clerk %+v has stopped walking", mid)
	}
	sc.Time += debutWalk * 4
	end, _ := r.DebutPose(sc, sc.Debuts[0])
	if end.Action == "walk" || math.Hypot(end.Pos.X-stall.X, end.Pos.Y-stall.Y) > debutAside+1 {
		t.Fatalf("the clerk waits at %+v, its stall goes at %v", end, stall)
	}
}

func TestTheFirstWorkerWalksOffWhereItsCargoLands(t *testing.T) {
	sc, r := harborScene(t)
	base := SiteHavens(sc.Site)[0]
	second := base
	second.ID, second.Quay.Y = "haven-2", base.Quay.Y+12
	sc.Site.Havens = []view.Haven{base, second}
	cameraOn(sc, r)
	h := r.HarborOf()
	far := 1
	if len(h.Plans) < 2 || len(h.Plans[far].Berths) == 0 {
		t.Fatalf("the second port has no berth: %d plans", len(h.Plans))
	}
	h.Yards = []Yard{{Haven: far, Kind: 1, Boxes: []QuayBox{{}}}}
	sc.Debuts = []Debut{{Kind: "ledger", Role: "clerk"}}
	start, ok := r.DebutPose(sc, sc.Debuts[0])
	plank := h.Plans[far].Berths[0].Plank
	if !ok || math.Hypot(start.Pos.X-plank.X, start.Pos.Y-plank.Y) > .5 {
		t.Fatalf("the clerk starts at %+v, the ledgers land at %v", start.Pos, plank)
	}
}

func TestNoDebutWhereItsCargoNoLongerLands(t *testing.T) {
	fv := view.FreightView{Kinds: []view.KindView{{Kind: "parcel", Crew: 3, Open: true}, {Kind: "ledger"}, {Kind: "crate"}}}
	if got := DebutsOf([]string{"clerks", "sealwrights"}, fv); len(got) != 0 {
		t.Fatalf("a new run's harbor lands no ledger and no seal, yet = %+v", got)
	}
}
