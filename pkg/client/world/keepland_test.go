package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func meadowSite(w, h int) view.Site {
	site := view.Site{Width: w, Height: h, Tiles: make([]view.TileKind, w*h)}
	for i := range site.Tiles {
		site.Tiles[i] = view.TileGrass
	}
	return site
}

func patchOf(t *testing.T, patches []KeepPatch, keep string) KeepPatch {
	t.Helper()
	for _, p := range patches {
		if p.Keep == keep {
			return p
		}
	}
	t.Fatalf("no patch for %s", keep)
	return KeepPatch{}
}

func TestAKeepsLandFollowsItsHousingAndItsFieldsItsFood(t *testing.T) {
	site := meadowSite(60, 40)
	site.Structures = []view.Structure{
		{ID: "tower", Kind: view.StructKeep, Variant: "tower", X: 10, Y: 10, W: 2, H: 2, State: view.StateReady, CPU: 2, MemoryGiB: 4},
		{ID: "stone", Kind: view.StructKeep, Variant: "stone", X: 40, Y: 20, W: 3, H: 3, State: view.StateReady, CPU: 4, MemoryGiB: 16},
		{ID: "codex", Kind: view.StructCodex, X: 12, Y: 12, W: 2, H: 2, State: view.StateReady},
	}
	patches := KeepPatches(site, map[string]Vec{"porter-a": {X: 9.5, Y: 12.5}})
	tower, stone := patchOf(t, patches, "tower"), patchOf(t, patches, "stone")
	if stone.Reach-1.5 != 2*(tower.Reach-1) || len(stone.Land) <= len(tower.Land) {
		t.Fatalf("16 GiB of housing does not double the reach of 4: %.1f vs %.1f, %d vs %d tiles", stone.Reach, tower.Reach, len(stone.Land), len(tower.Land))
	}
	if len(tower.Fields) != 2 || len(stone.Fields) != 4 {
		t.Fatalf("fields %d and %d, want one per thousand rations", len(tower.Fields), len(stone.Fields))
	}
	for i, f := range stone.Fields {
		want := "wheat"
		if i < berryPlots {
			want = "berries"
		}
		if f.Crop != want {
			t.Errorf("plot %d grows %s, want %s", i, f.Crop, want)
		}
	}
	for _, p := range patches {
		for _, f := range p.Fields {
			for _, s := range site.Structures {
				if Covers(s, f.At.X, f.At.Y) {
					t.Errorf("%s plants a field under %s at %v", p.Keep, s.ID, f.At)
				}
			}
			if f.At == (view.Point{X: 9, Y: 12}) {
				t.Errorf("%s plants a field on a hut", p.Keep)
			}
		}
	}
}

func TestNoLandOnWaterAndNoneForAKeepUnderConstruction(t *testing.T) {
	site := meadowSite(30, 30)
	for y := range 30 {
		for x := 16; x < 30; x++ {
			site.Tiles[y*30+x] = view.TileWater
		}
	}
	site.Structures = []view.Structure{
		{ID: "k", Kind: view.StructKeep, X: 14, Y: 14, W: 2, H: 2, State: view.StateReady, CPU: 2, MemoryGiB: 4},
		{ID: "new", Kind: view.StructKeep, X: 4, Y: 4, W: 2, H: 2, State: view.StateConstruction, CPU: 4, MemoryGiB: 8},
	}
	patches := KeepPatches(site, nil)
	if len(patches) != 1 {
		t.Fatalf("%d patches, want only the standing Keep's", len(patches))
	}
	for _, p := range patches[0].Land {
		if site.Tile(p.At.X, p.At.Y) == view.TileWater {
			t.Fatalf("land on water at %v", p)
		}
	}
}

func TestTouchingLandsShareOneOutline(t *testing.T) {
	site := meadowSite(60, 40)
	site.Structures = []view.Structure{
		{ID: "a", Kind: view.StructKeep, Variant: "stone", X: 20, Y: 20, W: 3, H: 3, State: view.StateReady, Territory: 5},
		{ID: "b", Kind: view.StructKeep, Variant: "stone", X: 28, Y: 20, W: 3, H: 3, State: view.StateReady, Territory: 5},
	}
	l := BordersOf(site)
	if len(l.Areas) != 2 {
		t.Fatalf("%d areas", len(l.Areas))
	}
	shared := 0
	for i, a := range l.Areas {
		for _, p := range a.Border {
			if sharedEdge(l.Owner, l.Width, i, p) {
				shared++
			}
		}
	}
	if shared == 0 {
		t.Fatal("the fixture's lands do not touch")
	}
	for _, m := range l.Marks {
		p := view.Point{X: m.X, Y: m.Y}
		if owner, ok := l.At(p.X, p.Y); ok {
			if sharedEdge(l.Owner, l.Width, owner, p) {
				t.Errorf("a border mark at %v stands where the two lands meet", p)
			}
		}
	}
}
