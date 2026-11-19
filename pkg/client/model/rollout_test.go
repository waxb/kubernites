package model

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func rollingCrew(ready int, from string) CrewChain {
	return CrewChain{Namespace: "default", Name: "porters", Image: "porter:2.0", Desired: 6, Depot: "parcels", Foremen: []Foreman{
		{Name: "porters-new", Revision: 2, Desired: ready + 1, Current: ready + 1, Ready: ready, Image: "porter:2.0", Newest: true},
		{Name: "porters-old", Revision: 1, Desired: 6 - ready, Current: 6 - ready, Ready: 6 - ready, Image: from},
	}}
}

func rolledCrew() CrewChain {
	c := rollingCrew(6, "porter:1.0")
	c.Foremen = []Foreman{c.Foremen[0], {Name: "porters-old", Revision: 1, Image: "porter:1.0"}}
	c.Foremen[0].Desired, c.Foremen[0].Current = 6, 6
	return c
}

func parcelIncome(served float64) view.FreightView {
	return view.FreightView{Kinds: []view.KindView{{Kind: "parcel", Depot: "parcels", ServedRate: served, Value: 2}}}
}

func TestARolloutIsWatchedFromItsStartToItsGain(t *testing.T) {
	w, done := Rollouts{}.Step([]CrewChain{rollingCrew(1, "porter:1.0")}, parcelIncome(1))
	if len(done) != 0 || w.Gain(rollingCrew(1, "porter:1.0"), parcelIncome(1)) != 0 {
		t.Fatalf("at the start: %+v %+v", w, done)
	}
	w, _ = w.Step([]CrewChain{rollingCrew(3, "porter:1.0")}, parcelIncome(1.5))
	if got := w.Gain(rollingCrew(3, "porter:1.0"), parcelIncome(1.5)); got != 1 {
		t.Fatalf("gain halfway = %v", got)
	}
	w, done = w.Step([]CrewChain{rolledCrew()}, parcelIncome(2))
	if len(done) != 1 || done[0].Crew != "porters" || done[0].Image != "porter:2.0" || done[0].From != "porter:1.0" || done[0].Huts != 6 || done[0].Gain != 2 {
		t.Fatalf("done = %+v", done)
	}
	if len(w) != 0 {
		t.Fatalf("a finished rollout is still watched: %+v", w)
	}
}

func TestARolloutOfTheSameClassEndsQuietly(t *testing.T) {
	w, _ := Rollouts{}.Step([]CrewChain{rollingCrew(2, "porter:2.0")}, parcelIncome(1))
	finished := rolledCrew()
	finished.Foremen[1].Image = "porter:2.0"
	if _, done := w.Step([]CrewChain{finished}, parcelIncome(1)); len(done) != 0 {
		t.Fatalf("a restart of the same class was celebrated: %+v", done)
	}
}

func TestAGoneCrewDropsItsWatch(t *testing.T) {
	w, _ := Rollouts{}.Step([]CrewChain{rollingCrew(2, "porter:1.0")}, parcelIncome(1))
	if w, done := w.Step(nil, parcelIncome(1)); len(w) != 0 || len(done) != 0 {
		t.Fatalf("a deleted crew: %+v %+v", w, done)
	}
}

func TestTheRolloutBannerCountsEveryHutRebuiltNotOnlyTheReadyOnes(t *testing.T) {
	w, _ := Rollouts{}.Step([]CrewChain{rollingCrew(1, "porter:1.0")}, parcelIncome(1))
	finished := rolledCrew()
	finished.Foremen[0].Ready = 3
	_, done := w.Step([]CrewChain{finished}, parcelIncome(2))
	if len(done) != 1 || done[0].Huts != 6 {
		t.Fatalf("the banner counts %+v: all 6 huts were rebuilt, 3 of them are ready", done)
	}
}
