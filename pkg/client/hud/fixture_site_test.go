package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheFixtureSiteIsBuiltOnceAndSharedByEveryFrame(t *testing.T) {
	a, b := testSite(t), testSite(t)
	if len(a.Tiles) == 0 || &a.Tiles[0] != &b.Tiles[0] {
		t.Fatal("each frame founded its own realm: the fixture site must be built once per run")
	}
}

func TestAFrameThatAddsAStructureNeverLeaksItIntoTheNextFrame(t *testing.T) {
	first := testSite(t)
	before := len(first.Structures)
	first.Structures = append(first.Structures, view.Structure{ID: "leak", Kind: view.StructKeep})
	if again := testSite(t); len(again.Structures) != before {
		t.Fatalf("a structure added to one frame showed up in the next: %d, want %d", len(again.Structures), before)
	}
}
