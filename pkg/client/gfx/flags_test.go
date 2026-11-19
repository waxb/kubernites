package gfx

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
)

func TestTheLibraryKnowsWhereABuildingsFlagsFly(t *testing.T) {
	lib := Load(art.Build())
	if flags := lib.Flags("building/gate/ready"); len(flags) == 0 {
		t.Fatal("the gate's flags are unknown to the library")
	}
	if lib.Flags("no/such/sprite") != nil {
		t.Fatal("an unknown sprite has no flags")
	}
	m := lib.Flags("building/gate/ready")[0]
	if a, b := clothImage(m, 1), clothImage(m, 1); a != b {
		t.Fatal("cloth frames are not cached")
	}
}
