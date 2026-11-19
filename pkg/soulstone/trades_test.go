package soulstone

import (
	"slices"
	"testing"
)

func TestEveryTradeServesItsKindAndListensOn8080(t *testing.T) {
	cases := map[string]string{
		"porter:1.0": KindParcel, "clerk:1.0": KindLedger, "hauler:1.0": KindCrate,
		"courier:1.0": KindDispatch, "sealwright:1.0": KindSealed,
	}
	for image, kind := range cases {
		p, ok := Lookup(image)
		if !ok || p.System || p.Familiar {
			t.Fatalf("%s = %+v, %v", image, p, ok)
		}
		if p.Trade != p.Repository || !p.Serves(kind) || p.Port != TradePort {
			t.Fatalf("%s trade %q kinds %v port %d", image, p.Trade, p.Kinds, p.Port)
		}
		if p.Sprite != p.Trade || p.Profession == "" || len(p.StartLines) == 0 || p.CrashLine == "" {
			t.Fatalf("%s identity incomplete: %+v", image, p)
		}
	}
}

func TestPorterKeepsTheFoundingPace(t *testing.T) {
	p, _ := Lookup("porter:1.0")
	if p.CPUPerPilgrimMilliS != 1200 || p.ThreadMilli != 1000 || p.IdleMilli != 4 || p.MemBaseMi != 40 || p.MemPerRpsMi != 3 || p.MTBFSeconds != baseMTBF || p.Cost != 0 {
		t.Fatalf("porter:1.0 = %+v", p)
	}
	upgrade, _ := Lookup("porter:1.1")
	if upgrade.MTBFSeconds <= p.MTBFSeconds || upgrade.CPUPerPilgrimMilliS != p.CPUPerPilgrimMilliS || upgrade.Cost <= 0 {
		t.Fatalf("porter:1.1 is no upgrade: %+v", upgrade)
	}
	lean, _ := Lookup("porter:2.0")
	if lean.CPUPerPilgrimMilliS*2 != p.CPUPerPilgrimMilliS || lean.Cursed || lean.Cost <= upgrade.Cost {
		t.Fatalf("porter:2.0 is not lean: %+v", lean)
	}
	flawed, _ := Lookup("porter:2.1")
	if !flawed.Cursed || flawed.CurseSeconds <= 0 || flawed.MTBFSeconds != 0 {
		t.Fatalf("porter:2.1 does not lie: %+v", flawed)
	}
}

func TestTradeAppetitesDifferBySpecialty(t *testing.T) {
	porter, _ := Lookup("porter:1.0")
	hauler, _ := Lookup("hauler:1.0")
	clerk, _ := Lookup("clerk:1.0")
	sealwright, _ := Lookup("sealwright:1.0")
	courier, _ := Lookup("courier:1.0")
	if hauler.BoxMemory(SizeContainer) <= porter.BoxMemory(SizeContainer) || hauler.MemBaseMi <= porter.MemBaseMi {
		t.Fatal("a hauler should be the memory-heavy trade")
	}
	for _, p := range []Profile{porter, clerk, courier, sealwright} {
		if hauler.BoxCPU(SizeContainer)/hauler.BoxCPU(SizeParcel) >= p.BoxCPU(SizeContainer)/p.BoxCPU(SizeParcel) {
			t.Fatalf("a hauler should carry a big crate with less extra effort than %s", p.Image)
		}
	}
	if sealwright.BoxCPU(SizeParcel) <= courier.BoxCPU(SizeParcel) || sealwright.ThreadMilli <= courier.ThreadMilli {
		t.Fatal("a sealwright should be the CPU-heavy trade")
	}
	if clerk.MemBaseMi <= hauler.MemBaseMi {
		t.Fatal("a clerk holds the ledger in memory")
	}
	if courier.MemBaseMi >= hauler.MemBaseMi || courier.BoxMemory(SizeContainer) >= hauler.BoxMemory(SizeContainer) {
		t.Fatal("a courier should be light on Lore")
	}
}

func TestBoxAppetiteGrowsWithSizeAndClamps(t *testing.T) {
	for _, p := range Trades() {
		if p.BoxCPU(SizeParcel) != p.CPUPerPilgrimMilliS {
			t.Fatalf("%s size-0 box cost %v != CPUPerPilgrimMilliS %v", p.Image, p.BoxCPU(SizeParcel), p.CPUPerPilgrimMilliS)
		}
		for size := SizeParcel + 1; size < SizeClasses; size++ {
			if p.BoxCPU(size) <= p.BoxCPU(size-1) || p.BoxMemory(size) <= p.BoxMemory(size-1) {
				t.Fatalf("%s appetite does not grow at size %d", p.Image, size)
			}
		}
		if p.BoxCPU(-3) != p.BoxCPU(SizeParcel) || p.BoxMemory(99) != p.BoxMemory(SizeContainer) {
			t.Fatalf("%s sizes are not clamped", p.Image)
		}
	}
	if (Profile{}).BoxCPU(1) != 0 {
		t.Fatal("a profile with no boxes has no appetite")
	}
}

func TestTradesListsOneBaseClassPerTradeByCost(t *testing.T) {
	var images []string
	for _, p := range Trades() {
		images = append(images, p.Image)
	}
	want := []string{"porter:1.0", "clerk:1.0", "hauler:1.0", "courier:1.0", "sealwright:1.0"}
	if !slices.Equal(images, want) {
		t.Fatalf("trades = %v, want %v", images, want)
	}
}

func TestDepotsMapKindsToServicesCrewsAndImages(t *testing.T) {
	want := []Depot{
		{Kind: KindParcel, Service: "parcels", Crew: "porters", Image: "porter:1.0", Port: 80, TargetPort: TradePort},
		{Kind: KindLedger, Service: "ledgers", Crew: "clerks", Image: "clerk:1.0", Port: 80, TargetPort: TradePort},
		{Kind: KindCrate, Service: "crates", Crew: "haulers", Image: "hauler:1.0", Port: 80, TargetPort: TradePort},
		{Kind: KindDispatch, Service: "dispatches", Crew: "couriers", Image: "courier:1.0", Port: 80, TargetPort: TradePort},
		{Kind: KindSealed, Service: "seals", Crew: "sealwrights", Image: "sealwright:1.0", Port: 80, TargetPort: TradePort},
	}
	got := Depots()
	if !slices.Equal(got, want) {
		t.Fatalf("depots = %+v", got)
	}
	got[0].Service = "mutated"
	if Depots()[0].Service != "parcels" {
		t.Fatal("Depots leaks its table")
	}
	for _, d := range want {
		p, ok := Lookup(d.Image)
		if !ok || !p.Serves(d.Kind) || d.Crew != p.Trade+"s" {
			t.Fatalf("depot %+v does not match its trade %+v", d, p)
		}
		if byKind, ok := DepotFor(d.Kind); !ok || byKind != d {
			t.Fatalf("DepotFor(%s) = %+v", d.Kind, byKind)
		}
		if bySvc, ok := DepotOfService(d.Service); !ok || bySvc != d {
			t.Fatalf("DepotOfService(%s) = %+v", d.Service, bySvc)
		}
		if byCrew, ok := DepotOfCrew(d.Crew); !ok || byCrew != d {
			t.Fatalf("DepotOfCrew(%s) = %+v", d.Crew, byCrew)
		}
	}
	if _, ok := DepotFor("gold"); ok {
		t.Fatal("unknown kind resolved")
	}
	if _, ok := DepotOfService("lantern"); ok {
		t.Fatal("unknown service resolved")
	}
	if _, ok := DepotOfCrew("lanterns"); ok {
		t.Fatal("unknown crew resolved")
	}
}
