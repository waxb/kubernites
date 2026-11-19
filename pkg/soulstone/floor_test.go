package soulstone

import "testing"

func TestFloorHabitsFollowTheProfession(t *testing.T) {
	for _, img := range []string{"porter:1.0", "porter:1.1", "porter:2.0", "porter:2.1"} {
		h := FloorHabitsOf(img)
		if h.LogKiPerPilgrim <= 0 || h.HoardMiPerPilgrim != 0 {
			t.Fatalf("%s habits = %+v: porters log to stdout and hoard nothing", img, h)
		}
	}
	clerk := FloorHabitsOf("clerk:1.0")
	if clerk.LogKiPerPilgrim <= 0 || clerk.HoardMiPerPilgrim <= 0 {
		t.Fatalf("clerk habits = %+v: clerks keep a journal on the floor", clerk)
	}
	if h := FloorHabitsOf("raven:1.0"); h != (FloorHabits{ShipsJournals: true}) {
		t.Fatalf("raven habits = %+v: a raven ships journals and keeps none", h)
	}
	if FloorHabitsOf("hound:1.0").ShipsJournals {
		t.Fatal("a hound ships nothing")
	}
	if h := FloorHabitsOf("nonsense:9"); h != (FloorHabits{}) {
		t.Fatalf("unknown habits = %+v", h)
	}
}

func TestFloorHabitsResolveAliasesAndRegistryNames(t *testing.T) {
	if FloorHabitsOf("academy.kubernites.dev/clerk:1.0") != FloorHabitsOf("clerk:1.0") || FloorHabitsOf("porter") != FloorHabitsOf("porter:2.1") {
		t.Fatal("aliases resolve differently")
	}
}
