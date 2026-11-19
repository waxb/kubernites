package world

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func rollingPorters(ready int) model.CrewChain {
	return model.CrewChain{Namespace: "default", Name: "porters", Image: "porter:2.0", Desired: 6, Depot: "parcels", Foremen: []model.Foreman{
		{Name: "porters-new", Revision: 2, Desired: ready + 1, Current: ready + 1, Ready: ready, Image: "porter:2.0", Newest: true},
		{Name: "porters-old", Revision: 1, Desired: 6 - ready, Current: 6 - ready, Ready: 6 - ready, Image: "porter:1.0"},
	}}
}

func TestARolloutCountsTheHutsRebuiltAndTheIncomeGained(t *testing.T) {
	defer ui.SetNamer(nil)
	got, ok := RolloutText(rollingPorters(3), 0.84)
	if !ok || got != "CLASS 2.0: 3 OF 6 REBUILT, +0.84 GOLD/S" {
		t.Fatalf("rollout = %q %v", got, ok)
	}
	if got, _ := RolloutText(rollingPorters(1), 0); got != "CLASS 2.0: 1 OF 6 REBUILT" {
		t.Fatalf("rollout with no gain yet = %q", got)
	}
	ui.SetNamer(names.Kubernetes(64))
	if got, _ := RolloutText(rollingPorters(3), 0.84); !strings.Contains(got, "porter:2.0") || !strings.Contains(got, "3/6 UPDATED") || !strings.Contains(got, "+$0.84/S") {
		t.Fatalf("names rollout = %q", got)
	}
	steady := rollingPorters(6)
	steady.Foremen = steady.Foremen[:1]
	if _, ok := RolloutText(steady, 1); ok {
		t.Fatal("a finished rollout still reads as rolling")
	}
}

func TestTheRolloutLabelReadsTheIncomeGainedSinceItsStart(t *testing.T) {
	sc, r := harborScene(t)
	stall, ok := DepotOf(sc.Site, "parcels")
	if !ok {
		t.Fatal("the lively realm has no parcels depot")
	}
	at := Anchor(stall)
	cam := render.NewCamera(480, 60).CenterOn(at.X, at.Y, 960, 540)
	sc.Freight.Kinds[0].ServedRate, sc.Freight.Kinds[0].Value, sc.Freight.Kinds[0].Depot = 1.5, 2, "parcels"
	sc.Crews = []model.CrewChain{rollingPorters(3)}
	sc.Rollouts = model.Rollouts{"default/porters": {Rev: 2, From: "porter:1.0", Income: 2}}
	marks := r.rolloutMarks(cam, sc)
	if len(marks) != 1 || !strings.HasSuffix(marks[0].Text, "+1 GOLD/S") {
		t.Fatalf("marks = %+v", marks)
	}
}

func TestARolloutUnderAPanelStaysInSight(t *testing.T) {
	sc, r := harborScene(t)
	stall, ok := DepotOf(sc.Site, "parcels")
	if !ok {
		t.Fatal("the lively realm has no parcels depot")
	}
	at := Anchor(stall)
	cam := render.NewCamera(480, 60).CenterOn(at.X, at.Y, 960, 540)
	sc.Crews = []model.CrewChain{rollingPorters(3)}
	free := r.rolloutMarks(cam, sc)
	if len(free) != 1 {
		t.Fatalf("free rollout marks = %+v", free)
	}
	b := free[0].Box
	sc.Panels = []ui.Rect{{X: b.X - 60, Y: b.Y - 30, W: b.W + 120, H: 90}}
	moved := r.rolloutMarks(cam, sc)
	if len(moved) != 1 || Overlaps(moved[0].Box, sc.Panels[0]) {
		t.Fatalf("the rollout hides under the panel %+v: %+v", sc.Panels[0], moved)
	}
}

func TestAShuttersRolloutIsNamedSoAndSaysWhyItWaits(t *testing.T) {
	defer ui.SetNamer(nil)
	c := rollingPorters(0)
	c.Image = "porter:1.0"
	c.Foremen[0].Image, c.Foremen[0].Shutters = "porter:1.0", true
	c.Foremen[0].Pods = []string{"porters-new-a"}
	got, ok := RolloutTextWaiting(c, 0, 0)
	if !ok || got != "SHUTTERS: 0 OF 6 REBUILT, THE FIRST IS STARTING" {
		t.Fatalf("a Shutters roll-out reads %q", got)
	}
	if got, _ := RolloutTextWaiting(c, 0, 1); got != "SHUTTERS: 0 OF 6 REBUILT, 1 WAIT FOR ROOM" {
		t.Fatalf("a roll-out stalled for room reads %q", got)
	}
	pods := []model.PodView{{Namespace: "default", Name: "porters-new-a", State: model.HearthPending}}
	if n := newestWaiting(c, pods); n != 1 {
		t.Fatalf("waiting huts of the new shift = %d", n)
	}
}
