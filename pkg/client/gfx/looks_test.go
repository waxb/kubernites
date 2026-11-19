package gfx

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
)

func TestWalkerVariantNamesResolve(t *testing.T) {
	lib := Load(art.Build())
	for _, role := range art.VariedRoles {
		if WalkerVariants(role) != art.WalkerLooks {
			t.Fatalf("%s variants = %d", role, WalkerVariants(role))
		}
		for tier := 1; tier <= 3; tier++ {
			for v := -3; v < 2*art.WalkerLooks; v++ {
				names := WalkerTierVariantNames(role, v, tier, "walk", "sw", 5)
				if !lib.Has(names[0]) {
					t.Fatalf("%s tier %d variant %d: %s missing", role, tier, v, names[0])
				}
			}
		}
	}
}

func TestWalkerVariantFallbacks(t *testing.T) {
	if got := WalkerVariantNames("imp", 3, "walk", "se", 1); len(got) != 4 || got[0] != "walker/imp/walk/se/1" {
		t.Errorf("unvaried role = %v", got)
	}
	if WalkerVariants("imp") != 1 || WalkerTiers("imp") != 1 || WalkerTiers("scribe") != 1 || WalkerTiers("lamplighter") != 3 {
		t.Error("variant counts")
	}
	if got := WalkerLookName("lamplighter", art.WalkerLooks, 1, "idle", "se", 0); got != "walker/lamplighter/idle/se/0" {
		t.Errorf("variant 0 of tier 1 is the base look, got %s", got)
	}
	if got := WalkerTierVariantNames("lamplighter", 2, 3, "idle", "se", 0); got[1] != "walker/lamplighter/t3v0/idle/se/0" || got[2] != "walker/lamplighter/idle/se/0" {
		t.Errorf("tier fallbacks = %v", got)
	}
	if got := WalkerLookName("scribe", 1, 3, "work", "sw", 2); got != "walker/scribe/t1v1/work/sw/2" {
		t.Errorf("scribe has one tier: %s", got)
	}
}
