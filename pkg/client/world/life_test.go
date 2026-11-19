package world

import (
	"math"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestLanternsBurnOnlyOnReadyOwners(t *testing.T) {
	site := lifeSite()
	site.Structures = append(site.Structures, view.Structure{ID: "k2", Kind: view.StructKeep, Variant: "stone", X: 12, Y: 3, W: 3, H: 3, State: view.StateDark, Object: view.ObjectRef{Kind: "Node", Name: "keep-2"}})
	lamps := LampSpots(site)
	lit, dark := 0, 0
	for _, l := range lamps {
		if l.lit {
			lit++
		} else {
			dark++
		}
	}
	if dark != lampsPerKeep || lit < lampsPerKeep+lampsPerGuild {
		t.Errorf("lamps lit %d dark %d: %+v", lit, dark, lamps)
	}
	for _, l := range lamps {
		if Occupied(site, int(math.Floor(l.at.X)), int(math.Floor(l.at.Y))) {
			t.Errorf("lamp inside a footprint at %v", l.at)
		}
	}
	if LampsLit(.5) || !LampsLit(0) || !LampsLit(.95) {
		t.Error("lamps are lit from dusk to dawn")
	}
}

func TestDawnFogComesAndGoes(t *testing.T) {
	if FogAmount(.5) != 0 || FogAmount(0) != 0 || FogAmount(.9) != 0 {
		t.Error("no fog at noon, midnight or dusk")
	}
	if FogAmount(fogPeak) < .99 {
		t.Errorf("fog peaks at dawn: %v", FogAmount(fogPeak))
	}
	if a := FogAmount((fogStart + fogPeak) / 2); a <= 0 || a >= 1 {
		t.Errorf("fog rolls in: %v", a)
	}
}

func TestCloudShadowsDriftDeterministically(t *testing.T) {
	a := CloudAt(3, 4, 10)
	if a != CloudAt(3, 4, 10) {
		t.Fatal("deterministic")
	}
	b := CloudAt(3, 4, 20)
	if Dist(a.center, b.center) < 1 || a.radius != b.radius {
		t.Errorf("clouds drift with the wind: %+v then %+v", a, b)
	}
	if CloudAt(3, 5, 10).center == a.center {
		t.Error("each cell holds its own cloud")
	}
}

func raisedHarbors(site view.Site, tier uint8) view.Site {
	site.Havens = append([]view.Haven(nil), site.Havens...)
	for i := range site.Havens {
		if site.Havens[i].Open {
			site.Havens[i].Tier = tier
		}
	}
	return site
}

func TestFishingBoatsWaitForTheHarborsFirstTier(t *testing.T) {
	site := archipelago(t)
	if got := PlanBoats(raisedHarbors(site, 0)); len(got) != 0 {
		t.Errorf("a harbor below its first tier sends %d boats", len(got))
	}
	if got := PlanBoats(raisedHarbors(site, 1)); len(got) == 0 {
		t.Error("the first tier sends a fishing boat")
	}
}

func TestFishingBoatsKeepToOpenWater(t *testing.T) {
	site := raisedHarbors(archipelago(t), 1)
	boats := PlanBoats(site)
	if len(boats) == 0 {
		t.Fatal("the open harbor sends out a fishing boat")
	}
	for _, b := range boats {
		for k := 0; k < 12; k++ {
			pos, _ := b.At(float64(k) * b.Period / 12)
			if !WetAt(site, pos) {
				t.Errorf("boat runs aground at %v", pos)
			}
		}
	}
	if len(PlanBoats(view.Site{})) != 0 {
		t.Error("no harbors, no boats")
	}
}

func TestDockhandsShuttleCrates(t *testing.T) {
	from, to := Vec{0, 0}, Vec{3, 0}
	out := DockShuttle(0.5, 0, from, to)
	if out.Action != "carry" || out.Dir != "se" {
		t.Errorf("carries a crate from the ship: %+v", out)
	}
	back := DockShuttle(dockTrip(from, to)+dockDrop+.5, 0, from, to)
	if back.Action != "walk" || back.Dir != "sw" {
		t.Errorf("walks back for the next crate: %+v", back)
	}
	drop := DockShuttle(dockTrip(from, to)+dockDrop/2, 0, from, to)
	if drop.Pos != to || drop.Action != "work" {
		t.Errorf("sets the crate down: %+v", drop)
	}
}

func TestCampWalkWaitsForTheClerk(t *testing.T) {
	from, to := Vec{0, 0}, Vec{4, 0}
	if p := CampWalk(from, to, 1, 2); p.Pos != from || p.Action != "idle" {
		t.Errorf("waits for the clerk: %+v", p)
	}
	if p := CampWalk(from, to, 2+1/campSpeed, 2); Dist(p.Pos, Vec{1, 0}) > 1e-9 || p.Action != "walk" {
		t.Errorf("walks to the plot: %+v", p)
	}
	if p := CampWalk(from, to, 100, 2); p.Pos != to || p.Action != "work" {
		t.Errorf("builds its hut: %+v", p)
	}
}

func TestCraneHoistsBetweenDeckAndHold(t *testing.T) {
	if HoistDrop(0) != hoistMin || math.Abs(HoistDrop(hoistTime/2)-(hoistMin+hoistSpan)) > 1e-9 {
		t.Errorf("hoist runs from %v to %v", HoistDrop(0), HoistDrop(hoistTime/2))
	}
}
