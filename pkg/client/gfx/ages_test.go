package gfx

import (
	"reflect"
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

func TestAgeTierFollowsTheRoad(t *testing.T) {
	for age, want := range map[string]int{"dark": 0, "feudal": 1, "castle": 2, "empire": 3, "": 0, "stone": 0} {
		if got := AgeTier(age); got != want {
			t.Errorf("AgeTier(%q) = %d, want %d", age, got, want)
		}
	}
}

func TestBuildingLookNamesFallBackThroughLowerAges(t *testing.T) {
	got := BuildingLookNames("codex", "ready", 2, 1)
	want := []string{
		"building/codex/ready/2/1", "building/codex/ready/2/0", "building/codex/ready/1/1", "building/codex/ready/1/0",
		"building/codex/ready/0/1", "building/codex/ready",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("chain %v, want %v", got, want)
	}
	dark := BuildingLookNames("warhorn", "dark", 1, 0)
	want = []string{"building/warhorn/dark/1/0", "building/warhorn/dark", "building/warhorn/ready/1/0", "building/warhorn/ready"}
	if !reflect.DeepEqual(dark, want) {
		t.Fatalf("dark chain %v, want %v", dark, want)
	}
	if got := BuildingLookNames("codex", "ready", 9, -4); got[0] != "building/codex/ready/3/0" {
		t.Fatalf("out of range age and variant clamp: %v", got)
	}
}

func TestStructureLookNamesKeepTheLegacyTail(t *testing.T) {
	s := view.Structure{ID: "codex-4", Kind: view.StructCodex, State: view.StateReady}
	names := StructureLookNames(s, 3)
	v := LookVariant("codex-4")
	if names[0] != BuildingLookNames("codex", "ready", 3, v)[0] {
		t.Fatalf("first %s", names[0])
	}
	if names[len(names)-1] != "building/codex/ready" {
		t.Fatalf("the chain must end on the legacy sprite: %v", names)
	}
	depot := view.Structure{ID: "depot-ledger-11", Kind: view.DepotStructure("ledger"), W: 2, H: 1, Variant: view.FaceSW, State: view.StateReady}
	names = StructureLookNames(depot, 1)
	if names[0] != BuildingLookNames("stall-ledger-sw", "ready", 1, LookVariant("depot-ledger-11"))[0] || names[len(names)-1] != "building/stall-ledger-sw/ready" {
		t.Fatalf("a depot stands in its stall: %v", names)
	}
}

func TestLookVariantIsStableAndBounded(t *testing.T) {
	seen := map[int]bool{}
	for _, id := range []string{"a", "b", "c", "codex-4", "chancery-5", "porter-x1", "porter-x2", "porter-x3"} {
		v := LookVariant(id)
		if v != LookVariant(id) || v < 0 || v >= 3 {
			t.Fatalf("variant %d for %s", v, id)
		}
		seen[v] = true
	}
	if len(seen) < 2 {
		t.Fatal("ids should spread across variants")
	}
}

func TestHearthLookNames(t *testing.T) {
	got := HearthLookNames(2, "ready", "Guaranteed", 2, 1)
	head := []string{"hearth/2/ready/landed/2/1", "hearth/2/ready/landed/2/0", "hearth/2/ready/landed/1/1", "hearth/2/ready/landed/1/0",
		"hearth/2/ready/landed/0/1", "hearth/2/ready/landed"}
	if !reflect.DeepEqual(got[:len(head)], head) {
		t.Fatalf("head %v, want %v", got[:len(head)], head)
	}
	if got[len(got)-1] != "hearth/2/ready" {
		t.Fatalf("tail %v", got)
	}
	if plain := HearthLookNames(1, "pending", "", 3, 2); !reflect.DeepEqual(plain, HearthQoSNames(1, "pending", "")) {
		t.Fatalf("an unknown QoS keeps the old chain: %v", plain)
	}
}

func TestStructurePortraitFollowsTheAge(t *testing.T) {
	codex := view.Structure{Kind: view.StructCodex}
	cases := map[int]string{0: "portrait/codex", 2: "portrait/codex/2", 9: "portrait/codex/3"}
	for age, want := range cases {
		if got := StructurePortrait(codex, age); got != want {
			t.Errorf("age %d: %s, want %s", age, got, want)
		}
	}
	if got := StructurePortrait(view.Structure{Kind: "mystery"}, 2); got != "portrait/mystery" {
		t.Errorf("unknown kinds keep the plain portrait, got %s", got)
	}
	lib := Load(art.Build())
	for age := 0; age < art.AgeTiers; age++ {
		if name := StructurePortrait(codex, age); !lib.Has(name) {
			t.Errorf("missing %s", name)
		}
	}
}
