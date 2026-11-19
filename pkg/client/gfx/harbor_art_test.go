package gfx

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
)

func TestHarborContractNamesResolveInTheLibrary(t *testing.T) {
	lib := Load(art.Build())
	names := []string{}
	for _, class := range ShipClasses {
		for v := 0; v < ShipVariants; v++ {
			for _, d := range art.ShipDirs {
				for f := 0; f < art.ShipFrames; f++ {
					names = append(names, ShipName(class, v, d, f))
				}
			}
		}
	}
	for _, k := range art.HavenKinds {
		for v := 0; v < art.HavenVariants(k); v++ {
			names = append(names, HavenName(k, v))
		}
	}
	for f := 0; f < 6; f++ {
		names = append(names, EffectName("wake", f), EffectName("request", f))
	}
	if missing := Missing(lib, names); len(missing) > 0 {
		t.Fatalf("%d harbor sprites missing, first %s", len(missing), missing[0])
	}
	if len(ShipClasses) != len(art.ShipClasses) || ShipVariants != art.ShipVariants {
		t.Fatal("gfx and art disagree on the ship classes")
	}
	for i := range ShipClasses {
		if ShipClasses[i] != art.ShipClasses[i] {
			t.Fatalf("class %d: gfx %s, art %s", i, ShipClasses[i], art.ShipClasses[i])
		}
	}
	if w, h := lib.Size(ShipName("carrack", 9, "nw", 5)); w == 0 || h == 0 {
		t.Fatal("wrapped ship name does not resolve")
	}
}
