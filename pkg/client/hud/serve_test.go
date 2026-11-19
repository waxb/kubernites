package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func crewCarries(f *Frame, on bool) {
	f.State.Freight.Kinds[0].DepotOpen = on
	f.State.Freight.Kinds[0].Crew, f.State.Freight.Kinds[0].ServedRate = 0, 0
	if on {
		f.State.Freight.Kinds[0].Crew, f.State.Freight.Kinds[0].ServedRate = 3, 1
	}
}

func TestServeIsRetiredInEveryState(t *testing.T) {
	h := New(nil, Backend{})
	for _, gate := range []bool{true, false} {
		for _, crew := range []bool{true, false} {
			f := fullFrame(t)
			f.State.Pilgrims.GateOpen = gate
			crewCarries(&f, crew)
			main, more := h.MainCard(f)
			if _, ok := HitID(append(main, more...), "cmd:serve"); ok {
				t.Errorf("gate %v, crew %v: SERVE is back", gate, crew)
			}
		}
	}
}

func TestAReopenedGateBeckonsAtThePile(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	crewCarries(&f, true)
	h.Update(Input{}, f)
	if h.Beckons(f) {
		t.Error("a carrying crew needs no hands")
	}
	crewCarries(&f, false)
	h.Update(Input{}, f)
	f.State.RealmTime += GateSettle
	h.Update(Input{}, f)
	if !h.Beckons(f) {
		t.Error("the reopened gate points at the boxes on the quay")
	}
	goal := f
	goal.GoalAt = &[2]float64{40, 20}
	if h.Beckons(goal) {
		t.Error("a second arrow at the quay beside the step's own: SHOW ME goes to one place, the eye to another")
	}
	f.State.Freight.Kinds[0].Waiting = 0
	if h.Beckons(f) {
		t.Error("an empty quay has nothing to click")
	}
}

func TestHandsNeededFollowsTheParcelsCrew(t *testing.T) {
	f := fullFrame(t)
	crewCarries(&f, true)
	if HandsNeeded(f.State.Freight) {
		t.Error("a carrying porters crew frees your hands")
	}
	crewCarries(&f, false)
	if !HandsNeeded(f.State.Freight) {
		t.Error("with no crew the parcels need your hands")
	}
	if !HandsNeeded(view.FreightView{}) {
		t.Error("an empty harbor still takes hands")
	}
}

func TestCrewTakeoverKeepsHandsWelcome(t *testing.T) {
	h := New(nil, Backend{})
	h.SetStaging(false)
	f := fullFrame(t)
	crewCarries(&f, false)
	h.Update(Input{}, f)
	crewCarries(&f, true)
	h.Update(Input{}, f)
	if len(h.Toasts()) != 1 || h.Toasts()[0].Text != CrewTakesToast || h.Toasts()[0].Bad {
		t.Fatalf("takeover toast = %+v", h.Toasts())
	}
	if strings.Contains(strings.ToLower(CrewTakesToast), "no longer") {
		t.Error("the takeover must not say your hands are done")
	}
	crewCarries(&f, false)
	h.Update(Input{}, f)
	f.Clock += NoticeGap
	f.State.RealmTime += GateSettle
	h.Update(Input{}, f)
	if !h.GateReopened() || len(h.Toasts()) != 2 || !h.Toasts()[1].Bad || h.Toasts()[1].Text != HandsNeededToast {
		t.Fatalf("a lost crew pulses SERVE and warns: %+v", h.Toasts())
	}
}

func TestHandValueIsTheNextBox(t *testing.T) {
	k := view.KindView{Kind: "parcel", Value: 2, Cohorts: []view.CohortView{{Count: 3, Remaining: 20, TTL: 20}}}
	fv := view.FreightView{Kinds: []view.KindView{k}}
	if v := HandValue(fv); v != 2.5 {
		t.Errorf("a fresh parcel is worth %v, want 2.5", v)
	}
	fv.Kinds[0].Cohorts = append(fv.Kinds[0].Cohorts, view.CohortView{Count: 1, Remaining: 0, TTL: 20, Crowned: true})
	if v := HandValue(fv); v != 4 {
		t.Errorf("a stale crowned parcel goes first and is worth %v, want 4", v)
	}
	if v := HandValue(view.FreightView{}); v != 0 {
		t.Errorf("no parcels are worth %v", v)
	}
	if got := GoldLabel(12.4); got != "+12 gold" {
		t.Errorf("label %q", got)
	}
	if got := GoldLabel(2.5); got != "+2.5 gold" {
		t.Errorf("label %q", got)
	}
}

func TestPileTipDrawsBesideTheCursor(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.Tip, f.TipAt = "+3 MiB\nClick the parcels", [2]int{400, 300}
	h.Update(Input{MX: 400, MY: 300}, f)
	r, ok := h.WorldTipRect(f)
	if !ok || r.Contains(400, 300) || r.W < 40 {
		t.Fatalf("pile tip at %+v, %v", r, ok)
	}
	if r.X < 0 || r.Y < TopH || r.X+r.W > Width || r.Y+r.H > ConsoleY {
		t.Fatalf("pile tip %+v leaves the map", r)
	}
	f.Tip = ""
	if _, ok := h.WorldTipRect(f); ok {
		t.Fatal("no tip, no box")
	}
}

func TestHandCoinCarriesItsGold(t *testing.T) {
	h := New(nil, Backend{})
	h.Update(Input{}, fullFrame(t))
	h.LaunchHandCoin(300, 300, 12)
	if len(h.Coins.Flights) != 1 || h.Coins.Flights[0].Label != "+12 gold" {
		t.Fatalf("hand token = %+v", h.Coins.Flights)
	}
	h.LaunchCoin(300, 300, 5)
	if h.Coins.Flights[1].Label != "" {
		t.Fatal("rate tokens stay unlabelled")
	}
}
