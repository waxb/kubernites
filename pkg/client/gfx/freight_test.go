package gfx

import (
	"math"
	"reflect"
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

func TestBoxNamesClampAndWrap(t *testing.T) {
	checks := [][2]string{
		{BoxName(9, "ledger", 5), "box/3/ledger/1"},
		{BoxName(-2, "nope", -1), "box/0/parcel/3"},
		{BoxSpoiledName(1, "crate", 4), "box/1/crate/0/spoiled"},
		{BoxSealName(7), "box/seal/3"},
		{BoxCrownName(-1), "box/crown/0"},
		{BoxFreshName(2, 9), "box/fresh/2/3"},
		{BoxFreshName(2, -9), "box/fresh/2/0"},
		{BoxPileName("dispatch", 8), "boxpile/dispatch/3"},
		{BoxPileName("x", -3), "boxpile/parcel/0"},
		{CarryName(5, "crate", "up", 6), "carry/3/crate/se/2"},
		{CarryName(0, "ledger", "sw", -1), "carry/0/ledger/sw/3"},
		{CarryName(1, "sealed", "se", 0), "carry/1/sealed/se/0"},
		{SpoilName(7), "fx/spoil/1"},
		{KindIcon("dispatch"), "kind-dispatch"},
		{KindIcon("?"), "kind-parcel"},
	}
	for _, c := range checks {
		if c[0] != c[1] {
			t.Errorf("got %s, want %s", c[0], c[1])
		}
	}
}

func TestFreshLevelsFollowTheRemainingTime(t *testing.T) {
	for _, c := range []struct {
		fresh float64
		level int
	}{{1, 0}, {0.8, 0}, {0.7, 1}, {0.5, 2}, {0.3, 2}, {0.2, 3}, {0, 3}, {-1, 3}, {2, 0}, {math.NaN(), 3}} {
		if got := FreshLevel(c.fresh); got != c.level {
			t.Errorf("FreshLevel(%v) = %d, want %d", c.fresh, got, c.level)
		}
	}
}

func TestBoxLayersComposeInDrawOrder(t *testing.T) {
	cases := []struct {
		look BoxLook
		want []string
	}{
		{BoxLook{Size: 1, Kind: "parcel", Variant: 2, Fresh: 1}, []string{"box/1/parcel/2", "box/fresh/1/0"}},
		{BoxLook{Size: 3, Kind: "ledger", Sealed: true, Priority: true, Fresh: 0.1},
			[]string{"box/3/ledger/0", "box/seal/3", "box/crown/3", "box/fresh/3/3"}},
		{BoxLook{Size: 0, Kind: "crate", Variant: 1, Sealed: true, Spoiled: true}, []string{"box/0/crate/1/spoiled", "box/seal/0"}},
	}
	for _, c := range cases {
		if got := BoxLayers(c.look); !reflect.DeepEqual(got, c.want) {
			t.Errorf("BoxLayers(%+v) = %v, want %v", c.look, got, c.want)
		}
	}
}

func TestBoxPileLevelsRiseWithTheQueue(t *testing.T) {
	want := map[int]int{-3: -1, 0: -1, 1: 0, 3: 0, 4: 1, 9: 1, 10: 2, 24: 2, 25: 3, 900: 3}
	for waiting, level := range want {
		if got := BoxPileLevel(waiting); got != level {
			t.Errorf("BoxPileLevel(%d) = %d, want %d", waiting, got, level)
		}
	}
	if BoxLift(0) >= BoxLift(3) || BoxLift(9) != BoxLift(3) || BoxLift(-1) != BoxLift(0) {
		t.Error("box lifts must grow with the size and clamp")
	}
}

func TestTradeNamesAndFallbacks(t *testing.T) {
	if !IsTrade("porter") || IsTrade("lamplighter") || TradeKind("hauler") != "crate" || TradeKind("sealwright") != "" {
		t.Fatal("trade table")
	}
	got := TradeHoldNames("porter", 3, "sw", 5)
	if got[0] != "walker/porter/t1v3/hold/sw/1" || got[1] != "walker/porter/hold/sw/1" || got[2] != "walker/porter/walk/sw/1" {
		t.Errorf("porter hold chain = %v", got)
	}
	if got := TradeHoldNames("baker", 1, "se", 0); got[0] != "walker/baker/t1v1/carry/se/0" {
		t.Errorf("non-trade roles fall back to carry: %v", got)
	}
	ledger := view.Structure{Kind: view.DepotStructure("ledger"), W: 2, H: 1, Variant: view.FaceNE, State: view.StateDark}
	if got := BuildingNames(ledger); !reflect.DeepEqual(got, []string{"building/stall-ledger-ne/dark", "building/stall-ledger-ne/ready"}) {
		t.Errorf("depot chain = %v", got)
	}
	if HelperArtKind("cert-manager") != "seal-smithy" || HelperArtKind("CoreDNS") != "stall-herald" || HelperIcon("coredns") != "herald" || HelperNames("coredns", "dark")[0] != "building/stall-herald-sw/dark" || HelperArtKind("nginx") != "" {
		t.Error("helper art kinds")
	}
	if HelperNames("nginx", "ready") != nil || HelperNames("squid", "dark")[1] != "building/cache-store/ready" || HelperIcon("keycloak") != "key-gate" {
		t.Error("helper names")
	}
}

func freightContract() []string {
	names := []string{}
	for size := 0; size < BoxSizes; size++ {
		names = append(names, BoxSealName(size), BoxCrownName(size))
		for l := 0; l < FreshLevels; l++ {
			names = append(names, BoxFreshName(size, l))
		}
		for _, k := range BoxKinds {
			for v := 0; v < BoxVariants; v++ {
				names = append(names, BoxName(size, k, v), BoxSpoiledName(size, k, v))
			}
			for f := 0; f < 4; f++ {
				names = append(names, CarryName(size, k, "se", f), CarryName(size, k, "sw", f))
			}
		}
	}
	for _, k := range BoxKinds {
		for l := 0; l < BoxPileLevels; l++ {
			names = append(names, BoxPileName(k, l))
		}
		for _, face := range art.StallFaces {
			names = append(names, "building/"+art.StallArtKind(art.StallForBox(k), face)+"/construction-1")
		}
		names = append(names, "icon/"+KindIcon(k))
	}
	for f := 0; f < 6; f++ {
		names = append(names, SpoilName(f))
	}
	for _, p := range []string{"cert-manager", "coredns", "keycloak", "goldpinger", "squid", "rebalancer"} {
		names = append(names, HelperNames(p, "dark")...)
		names = append(names, "icon/"+HelperIcon(p), "portrait/"+HelperArtKind(p))
	}
	return names
}

func TestFreightNamesResolveInTheLibrary(t *testing.T) {
	lib := Load(art.Build())
	names := freightContract()
	for _, role := range TradeRoles {
		for v := -2; v < 2*art.WalkerLooks; v++ {
			for _, action := range []string{"idle", "walk", "work", "carry", HoldAction} {
				names = append(names, WalkerVariantNames(role, v, action, "sw", v)[0])
			}
			names = append(names, TradeHoldNames(role, v, "se", v)[0])
		}
		names = append(names, "portrait/"+role)
	}
	if missing := Missing(lib, names); len(missing) > 0 {
		t.Fatalf("%d freight sprites missing, first %s", len(missing), missing[0])
	}
	if w, h := lib.Size(CarryName(0, "parcel", "se", 0)); w != art.CarryWidth || h != art.CarryHeight {
		t.Errorf("carry overlay is %dx%d", w, h)
	}
}
