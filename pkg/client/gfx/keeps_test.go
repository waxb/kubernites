package gfx

import (
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

var keepLib = Load(art.Set{})

func keepFixture(state string) view.Structure {
	return view.Structure{ID: "keep-4", Kind: view.StructKeep, Variant: "great", Class: "memory", State: state, Progress: .5, Object: view.ObjectRef{Kind: "Node", Name: "great-keep-2"}}
}

func TestKeepsPaintTheirOwnSpritesOnFirstUse(t *testing.T) {
	lib := Load(art.Set{})
	names := KeepNames(keepFixture(view.StateReady), 2)
	if !strings.HasPrefix(names[0], "building/keep-great-memory/ready/2/") || names[len(names)-1] != "building/keep-great/ready" {
		t.Fatalf("keep names %v", names)
	}
	if !lib.Has(names[0]) {
		t.Fatal("the keep was not painted")
	}
	_, r, ok := lib.Lookup(names[0])
	head, hok := lib.Headroom(names[0])
	if !ok || r.W != 3*art.TileWidth+4 || !hok || head <= 0 {
		t.Fatalf("region %+v headroom %d", r, head)
	}
	built := KeepNames(keepFixture(view.StateConstruction), 2)
	if !strings.Contains(built[0], "/construction-1/") || !lib.Has(built[0]) {
		t.Fatalf("a keep in scaffold draws %v", built)
	}
	for _, bad := range []string{"building/keep-great-memory/ready/2/zz", "building/keep-stone/ready", "decor/tree-oak/0"} {
		if lib.Has(bad) {
			t.Errorf("%s resolved", bad)
		}
	}
	var none *Library
	if none.Has(names[0]) {
		t.Fatal("a nil library found a keep")
	}
}

func TestKeepsPaintOffTheDrawThreadWhenAsked(t *testing.T) {
	lib := Load(art.Set{})
	lib.SetAsync(true)
	lib.SetAsync(true)
	name := KeepNames(keepFixture(view.StateReady), 3)[0]
	if lib.Has(name) {
		t.Fatal("an async library painted on the draw thread")
	}
	deadline := time.Now().Add(20 * time.Second)
	for !lib.Has(name) {
		if time.Now().After(deadline) {
			t.Fatal("the keep was never painted")
		}
		time.Sleep(5 * time.Millisecond)
	}
	keepLib.SetAsync(false)
	var none *Library
	none.SetAsync(true)
}

func TestKeepLooksComeFromTheStructure(t *testing.T) {
	k := KeepLookOf(view.Structure{Variant: "tower", Object: view.ObjectRef{Name: testTowerNode}}, 9)
	if k.Size != "tower" || k.Class != "balanced" || k.Age != art.AgeTiers-1 || k.Seed != art.NameSeed(testTowerNode) {
		t.Fatalf("tower look %+v", k)
	}
}

const testTowerNode = "lone-tower"
