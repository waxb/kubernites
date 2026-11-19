package soulstone

import "testing"

func TestRavenIsASmallSteadyLogShipper(t *testing.T) {
	p, ok := Lookup("raven:1.0")
	if !ok || !p.Familiar || p.Proxy || p.System {
		t.Fatalf("raven = %+v, %v", p, ok)
	}
	if p.Profession != "Raven" || p.Sprite != "raven" || p.Repository != "raven" || p.Tag != "1.0" {
		t.Fatalf("raven identity = %q %q %q %q", p.Profession, p.Sprite, p.Repository, p.Tag)
	}
	busy := p.IdleMilli + p.CPUPerPilgrimMilliS*1
	if p.IdleMilli < 20 || busy > 50 {
		t.Fatalf("raven cpu idle %vm, busy %vm", p.IdleMilli, busy)
	}
	if mem := p.MemBaseMi + p.MemPerRpsMi*1; p.MemBaseMi < 56 || mem > 72 {
		t.Fatalf("raven memory %vMi", mem)
	}
	if p.Cost <= 0 || p.DutyLine == "" || len(p.StartLines) == 0 || p.CrashLine == "" {
		t.Fatalf("raven incomplete: %+v", p)
	}
}

func TestHoundIsAProxyWhoseOverheadGrowsWithPilgrims(t *testing.T) {
	p, ok := Lookup("hound:1.0")
	if !ok || !p.Familiar || !p.Proxy || p.System {
		t.Fatalf("hound = %+v, %v", p, ok)
	}
	if p.Profession != "Hound" || p.Sprite != "hound" {
		t.Fatalf("hound identity = %q %q", p.Profession, p.Sprite)
	}
	if p.CPUPerPilgrimMilliS <= 0 || p.CPUPerPilgrimMilliS >= 600 {
		t.Fatalf("sidecar overhead per request = %v", p.CPUPerPilgrimMilliS)
	}
	if low, high := p.MemBaseMi, p.MemBaseMi+p.MemPerRpsMi*1.7; low < 64 || high > 132 {
		t.Fatalf("hound memory %v..%vMi", low, high)
	}
	raven, _ := Lookup("raven:1.0")
	if p.Cost <= raven.Cost || p.DutyLine == "" {
		t.Fatalf("hound cost %v raven %v", p.Cost, raven.Cost)
	}
}

func TestFamiliarsListsOnlyFamiliarsByCost(t *testing.T) {
	list := Familiars()
	if len(list) != 2 || list[0].Image != "raven:1.0" || list[1].Image != "hound:1.0" {
		t.Fatalf("familiars = %+v", list)
	}
	for _, p := range Playable() {
		if p.Familiar && p.Image != "raven:1.0" && p.Image != "hound:1.0" {
			t.Fatalf("unexpected familiar %s", p.Image)
		}
	}
	if porter, _ := Lookup("porter:1.0"); porter.Familiar || porter.Proxy {
		t.Fatal("a porter is no familiar")
	}
	if name, sprite := ProfessionOf("hound:1.0"); name != "Hound" || sprite != "hound" {
		t.Fatalf("hound profession = %s %s", name, sprite)
	}
}
