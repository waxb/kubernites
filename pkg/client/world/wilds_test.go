package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func wildSite(n int, kind view.TileKind) view.Site {
	site := view.Site{Width: n, Height: n, Tiles: make([]view.TileKind, n*n), Elevation: make([]uint8, n*n),
		Region: make([]uint8, n*n), Power: make([]uint8, n*n), Wear: make([]uint16, n*n)}
	for i := range site.Tiles {
		site.Tiles[i] = kind
		site.Region[i] = 1
	}
	site.Islands = []view.Island{{ID: "home", Owned: true, Center: view.Point{X: n / 2, Y: n / 2}}}
	return site
}

func keepAt(x, y, plot int) view.Structure {
	return view.Structure{ID: "k", Kind: view.StructKeep, Variant: "stone", X: x, Y: y, W: 3, H: 3, Plot: plot, State: view.StateReady}
}

func TestHabitatMarksTownPlotsAndVeils(t *testing.T) {
	site := wildSite(30, view.TileGrass)
	site.Structures = []view.Structure{keepAt(10, 10, 3)}
	site.Wear[25*30+25] = 9000
	hb := BuildHabitat(site, LandOf(site))
	switch {
	case hb.BlockAt(11, 11) != BlockFootprint || hb.BlockAt(13, 11) != blockNear || hb.BlockAt(20, 20) != blockFree:
		t.Error("footprint and margin")
	case !hb.inPlot(13, 12) || hb.inPlot(14, 12) || hb.inPlot(20, 12) || hb.townAt(13, 12) != 1:
		t.Error("a keep clears only its yard")
	case hb.townAt(14, 12) <= stumpTown || hb.townAt(14, 12) >= 1 || hb.townAt(15, 12) >= stumpTown || hb.townAt(16, 12) != 0 || hb.townAt(28, 2) != 0:
		t.Error("the town fades just beyond the yard")
	case !hb.settled(16, 11) || !hb.settled(13, 12) || hb.settled(20, 12) || hb.settled(-1, 3):
		t.Error("the keep's land is settled")
	case hb.townAt(26, 25) < 0.3 || hb.townAt(25, 25) < 0.8:
		t.Error("worn road disturbs the wild")
	case !hb.veiled(13, 13) || hb.veiled(9, 9) || hb.veiled(-1, 5):
		t.Error("veil lies in front of the keep")
	case hb.townAt(-1, 0) != 0 || hb.BlockAt(99, 0) != blockFree || hb.inPlot(-5, -5):
		t.Error("outside the map")
	}
	if StructureSig(site) == StructureSig(wildSite(30, view.TileGrass)) {
		t.Error("structure signature ignores structures")
	}
}

func count(items []scenery, prefix string) int {
	n := 0
	for _, s := range items {
		if len(s.name) >= len(prefix) && s.name[:len(prefix)] == prefix {
			n++
		}
	}
	return n
}

func allScenery(site view.Site) []scenery {
	hb := BuildHabitat(site, LandOf(site))
	out := []scenery{}
	for y := 0; y < site.Height; y++ {
		for x := 0; x < site.Width; x++ {
			out = append(out, sceneryFor(site, hb, x, y)...)
		}
	}
	return out
}

func TestForestsGrowAndTownsClearThem(t *testing.T) {
	wild := wildSite(24, view.TileForest)
	trees := count(allScenery(wild), "decor/tree-")
	if trees < 24*24 {
		t.Fatalf("a forest of 576 tiles grew only %d trees", trees)
	}
	town := wild
	town.Structures = []view.Structure{keepAt(8, 8, 5)}
	cleared := trees - count(allScenery(town), "decor/tree-")
	if cleared < 20 || cleared > 110 {
		t.Errorf("a keep cleared %d of %d trees, want only its yard", cleared, trees)
	}
	hb := BuildHabitat(town, LandOf(town))
	for _, p := range [][2]int{{8, 14}, {14, 9}, {5, 6}, {7, 13}} {
		if !hb.settled(p[0], p[1]) || count(sceneryFor(town, hb, p[0], p[1]), "decor/tree-") == 0 {
			t.Errorf("the forest at %v inside the keep's land was felled", p)
		}
	}
	if count(allScenery(town), "decor/stump") == 0 {
		t.Error("the yard's edge leaves no stumps")
	}
	species := map[string]bool{}
	for _, s := range allScenery(wildSite(80, view.TileForest)) {
		species[s.name[:len(s.name)-2]] = true
	}
	for _, want := range []string{"decor/tree-oak", "decor/tree-pine", "decor/tree-birch", "decor/tree-spruce", "decor/tree-maple", "decor/tree-old"} {
		if !species[want] {
			t.Errorf("no %s in a large forest", want)
		}
	}
}

func TestSceneryRespectsRoadsPlazaAndCharters(t *testing.T) {
	site := wildSite(20, view.TileRock)
	site.Tiles[5*20+5] = view.TilePlaza
	site.Tiles[6*20+6] = view.TileRoad
	site.Wear[7*20+7] = 3000
	site.Islands[0].Owned = false
	hb := BuildHabitat(site, LandOf(site))
	for _, p := range [][2]int{{5, 5}, {6, 6}, {7, 7}, {10, 10}, {9, 11}} {
		if got := sceneryFor(site, hb, p[0], p[1]); len(got) != 0 {
			t.Errorf("scenery at %v: %v", p, got)
		}
	}
	all := allScenery(site)
	if count(all, "decor/boulder") == 0 || count(all, "decor/rock") == 0 {
		t.Error("rocky ground has no boulders or rocks")
	}
	for _, s := range all {
		if !s.wild {
			t.Fatal("scenery on an unowned island is not marked wild")
		}
	}
}

func TestShoresGrowReedsAndNoLeyStoneStands(t *testing.T) {
	site := wildSite(40, view.TileSand)
	for y := 0; y < 40; y++ {
		for x := 20; x < 40; x++ {
			site.Tiles[y*40+x] = view.TileShallows
		}
	}
	if count(allScenery(site), "decor/reeds") == 0 {
		t.Error("no reeds along the shore")
	}
	meadow := wildSite(40, view.TileMeadow)
	for i := range meadow.Power {
		meadow.Power[i] = uint8(2 + i%2)
	}
	if n := count(allScenery(meadow), "decor/ley-stone"); n != 0 {
		t.Errorf("%d ley stones stand though the ley layer is retired", n)
	}
}

func TestShoreMaskAndRoadLevels(t *testing.T) {
	site := wildSite(10, view.TileGrass)
	site.Tiles[0*10+5] = view.TileWater
	site.Tiles[4*10+4] = view.TileRoad
	site.Tiles[3*10+3] = view.TilePlaza
	site.Wear[2*10+2] = 500
	site.Wear[2*10+3] = 3000
	site.Wear[2*10+4] = 60000
	if ShoreMask(site, 5, 1) != 1 || ShoreMask(site, 5, 0) != 0 || ShoreMask(site, 5, 5) != 0 {
		t.Error("shore mask")
	}
	want := map[[2]int]uint8{{4, 4}: RoadCobbled, {3, 3}: RoadNone, {5, 0}: RoadNone, {2, 2}: RoadTrodden, {3, 2}: RoadDirt, {4, 2}: RoadCobbled, {7, 7}: RoadNone}
	for p, lv := range want {
		if got := RoadLevel(site, p[0], p[1]); got != lv {
			t.Errorf("road at %v = %d, want %d", p, got, lv)
		}
	}
}

func TestSettledGrassKeepsRoomForHuts(t *testing.T) {
	site := wildSite(40, view.TileGrass)
	for y := 0; y < 40; y += 3 {
		site.Tiles[y*40+30] = view.TileForest
	}
	site.Structures = []view.Structure{keepAt(18, 18, 4)}
	hb := BuildHabitat(site, LandOf(site))
	wildTall := 0
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			items := sceneryFor(site, hb, x, y)
			if hb.settled(x, y) && site.Tile(x, y) == view.TileGrass && tallScenery(items) {
				t.Fatalf("tall scenery %v on settled grass at %d,%d", items, x, y)
			}
			if !hb.settled(x, y) && tallScenery(items) {
				wildTall++
			}
		}
	}
	if wildTall == 0 {
		t.Error("unsettled grass grows nothing tall")
	}
	if StructureSig(site) == StructureSig(func() view.Site {
		s := site
		s.Structures = []view.Structure{func() view.Structure { k := keepAt(18, 18, 4); k.Territory = 9; return k }()}
		return s
	}()) {
		t.Error("the structure signature ignores the keep's reach")
	}
}
