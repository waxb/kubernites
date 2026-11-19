package world

import (
	"math"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestShipClassFollowsTheHarborTier(t *testing.T) {
	cases := map[int]string{0: "cog", 1: "cog", 2: "caravel", 3: "galleon", 4: "carrack", 9: "carrack"}
	for tier, want := range cases {
		if got := ShipClassOf(tier); got != want {
			t.Errorf("tier %d sails %s, want %s", tier, got, want)
		}
	}
}

func TestUnloadTimeFollowsTheCargo(t *testing.T) {
	cases := map[int]float64{0: minUnload, 4: minUnload, 12: 12 * unloadEach, 20: 20 * unloadEach, 90: maxUnload}
	for cargo, want := range cases {
		if got := UnloadTime(cargo); math.Abs(got-want) > 1e-9 {
			t.Errorf("unload(%d) = %v, want %v", cargo, got, want)
		}
	}
}

func TestShipVariantIsStablePerHavenAndSerial(t *testing.T) {
	seen := map[int]bool{}
	for serial := 0; serial < 64; serial++ {
		v := ShipVariant("haven-1", serial)
		if v < 0 || v >= 8 {
			t.Fatalf("variant %d out of range", v)
		}
		if v != ShipVariant("haven-1", serial) {
			t.Fatal("variant not stable")
		}
		seen[v] = true
	}
	if len(seen) < 6 {
		t.Errorf("ships barely vary: %v", seen)
	}
}

func dockedHarbor(t *testing.T, berths int) *Harbor {
	t.Helper()
	sc, _ := harborScene(t)
	h := &Harbor{}
	h.Replan(sc.Site)
	if len(h.Plans) == 0 || len(h.Plans[0].Berths) == 0 {
		t.Fatal("no berth on the harbor fixture")
	}
	for i := range h.Plans {
		h.Plans[i].Berths = h.Plans[i].Berths[:min(berths, len(h.Plans[i].Berths))]
	}
	return h
}

func freightAt(ships uint64, last, cargo, next float64, tier int) view.FreightView {
	fv := view.FreightView{Docking: view.DockingView{Ships: ships, Last: last, LastCargo: cargo, Next: next}}
	fv.Tier = tier
	return fv
}

func TestShipsSailInForTheNextDockingAndLandWhenItHappens(t *testing.T) {
	h := dockedHarbor(t, 1)
	w := []float64{1}
	h.schedule(freightAt(1, 0, 12, 20, 1), 1, w)
	ships := h.Fleets[0].Ships
	if len(ships) != 2 || ships[0].Serial != 1 || !ships[0].Docked || ships[0].Cargo != 12 || ships[0].Dock != 0 {
		t.Fatalf("after the first docking: %+v", ships)
	}
	next := ships[1]
	if next.Serial != 2 || next.Docked || next.Dock != 20 || next.Class != "cog" || next.Arrive() >= 20 {
		t.Fatalf("the next ship: %+v", next)
	}
	h.schedule(freightAt(1, 0, 12, 20, 1), 2, w)
	if len(h.Fleets[0].Ships) != 2 {
		t.Errorf("a ship expected twice: %+v", h.Fleets[0].Ships)
	}
	h.schedule(freightAt(2, 20.1, 7, 41, 1), 20.2, w)
	var landed Ship
	for _, s := range h.Fleets[0].Ships {
		if s.Serial == 2 {
			landed = s
		}
	}
	if !landed.Docked || landed.Dock != 20.1 || landed.Cargo != 7 || landed.Unload != UnloadTime(7) {
		t.Errorf("the expected ship did not land: %+v", landed)
	}
	if _, s, ok := h.shipOfLanding(20.1); !ok || s.Serial != 2 {
		t.Errorf("landing lookup %+v %v", s, ok)
	}
	if _, _, ok := h.shipOfLanding(33); ok {
		t.Error("a landing nobody made")
	}
}

func TestShipsLeaveAndTheFleetResetsOnReload(t *testing.T) {
	h := dockedHarbor(t, 3)
	w := []float64{1}
	h.schedule(freightAt(4, 100, 5, 130, 1), 100, w)
	if len(h.Fleets[0].Ships) != 2 {
		t.Fatalf("ships %+v", h.Fleets[0].Ships)
	}
	h.schedule(freightAt(4, 100, 5, 130, 1), 128, w)
	for _, s := range h.Fleets[0].Ships {
		if s.Serial == 4 {
			t.Errorf("a long gone ship survives: %+v", s)
		}
	}
	h.schedule(freightAt(1, 0, 12, 18, 1), 1, w)
	if h.docks.seen != 1 {
		t.Errorf("a reloaded realm keeps %d dockings", h.docks.seen)
	}
	for _, s := range h.Fleets[0].Ships {
		if s.Serial > 2 {
			t.Errorf("a ship from before the reload: %+v", s)
		}
	}
}

func TestATierClimbSendsAHeraldOfTheNewClass(t *testing.T) {
	h := dockedHarbor(t, 3)
	w := []float64{1}
	h.schedule(freightAt(3, 50, 6, 70, 1), 51, w)
	h.schedule(freightAt(3, 50, 6, 70, 2), 52, w)
	var herald Ship
	for _, s := range h.Fleets[0].Ships {
		if s.Herald {
			herald = s
		}
	}
	if herald.Class != "caravel" || herald.Dock <= 52 || herald.Cargo != 0 || herald.Arrive() > 52 {
		t.Fatalf("herald %+v", herald)
	}
	if pose, ok := ShipAt(herald, h.Plans[0].Berths[herald.Berth], h.Plans[0].Sea, herald.Dock+1); !ok || pose.Unloading {
		t.Errorf("a herald docks without unloading: %+v %v", pose, ok)
	}
	h.schedule(freightAt(3, 50, 6, 70, 2), 53, w)
	heralds := 0
	for _, s := range h.Fleets[0].Ships {
		if s.Herald {
			heralds++
		}
	}
	if heralds != 1 {
		t.Errorf("%d heralds for one climb", heralds)
	}
}

func TestShipsTakeAFreeBerth(t *testing.T) {
	f := &fleet{haven: "haven-0"}
	a := f.launch(Ship{Serial: 1, Dock: 10, Unload: 4, Wait: 2}, 3, SailSeconds)
	b := f.launch(Ship{Serial: 2, Dock: 12, Unload: 4, Wait: 2}, 3, SailSeconds)
	c := f.launch(Ship{Serial: 3, Dock: 30, Unload: 4, Wait: 2}, 3, SailSeconds)
	if a.Berth != 0 || b.Berth != 1 || c.Berth != 0 {
		t.Errorf("berths %d %d %d", a.Berth, b.Berth, c.Berth)
	}
	one := &fleet{haven: "haven-0"}
	one.launch(Ship{Serial: 1, Dock: 10, Unload: 4, Wait: 2}, 1, SailSeconds)
	if s := one.launch(Ship{Serial: 2, Dock: 11}, 1, SailSeconds); s.Berth != 0 {
		t.Errorf("a single berth: %d", s.Berth)
	}
	if HavenFor(3, nil) != 0 || HavenFor(3, []float64{0, 1}) != 1 {
		t.Error("haven choice")
	}
}

func TestShipSailsInDocksAndSailsAway(t *testing.T) {
	b := Berth{Moor: Vec{4, 10}, Plank: Vec{4, 10.8}, Lane: Vec{1, 10}}
	sea := Vec{-6, 10}
	s := Ship{Serial: 1, Cargo: 4, Class: "cog", Dock: 10, Unload: 5, Wait: 2, Sail: SailSeconds}
	early, ok := ShipAt(s, b, sea, s.Arrive()+.1)
	if !ok || Dist(early.Pos, sea) > 1 || early.Alpha >= 1 || !early.Moving || early.Dir != "se" {
		t.Errorf("a ship appears at sea, fading in: %+v", early)
	}
	docked, _ := ShipAt(s, b, sea, s.Dock+1)
	if docked.Pos != b.Moor || docked.Moving || !docked.Unloading || docked.Alpha != 1 {
		t.Errorf("docked pose %+v", docked)
	}
	if waiting, _ := ShipAt(s, b, sea, s.Dock+s.Unload+.5); waiting.Unloading || waiting.Moving {
		t.Errorf("after unloading the ship waits: %+v", waiting)
	}
	leaving, _ := ShipAt(s, b, sea, s.Leave()+SailSeconds*.9)
	if !leaving.Moving || leaving.Alpha >= .5 || Dist(leaving.Pos, sea) > Dist(b.Moor, sea) {
		t.Errorf("a leaving ship fades out at sea: %+v", leaving)
	}
	if _, ok := ShipAt(s, b, sea, s.Gone()+.1); ok {
		t.Error("a gone ship is still drawn")
	}
	if _, ok := ShipAt(s, b, sea, s.Arrive()-.1); ok {
		t.Error("a ship is drawn before it sails")
	}
	mid1, _ := ShipAt(s, b, sea, s.Arrive()+2)
	mid2, _ := ShipAt(s, b, sea, s.Arrive()+2+1.0/60)
	if Dist(mid1.Pos, mid2.Pos) > .2 {
		t.Errorf("the ship jumps: %v -> %v", mid1.Pos, mid2.Pos)
	}
}

func TestSailTimeFollowsTheDistanceToSea(t *testing.T) {
	cases := []struct{ dist, want float64 }{{0, SailSeconds}, {12, SailSeconds}, {21, 7}, {27, sailMost}, {60, sailMost}}
	for _, c := range cases {
		if got := SailTime(c.dist); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("sail over %v tiles = %v, want %v", c.dist, got, c.want)
		}
	}
	f := &fleet{haven: "haven-0"}
	s := f.launch(Ship{Serial: 1, Dock: 10}, 1, 8)
	if s.Sail != 8 || s.Arrive() != 2 || s.Gone() != s.Leave()+8 {
		t.Errorf("ship sail %+v", s)
	}
}
