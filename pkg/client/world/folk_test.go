package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func lifeSite() view.Site {
	const n = 24
	tiles := make([]view.TileKind, n*n)
	for i := range tiles {
		tiles[i] = view.TileGrass
	}
	for y := 0; y < n; y++ {
		tiles[y*n], tiles[y*n+1] = view.TileWater, view.TileShallows
	}
	for y := 13; y <= 20; y++ {
		for x := 3; x <= 9; x++ {
			tiles[y*n+x] = view.TilePlaza
		}
	}
	for x := 10; x <= 19; x++ {
		tiles[17*n+x] = view.TileRoad
	}
	for y := 9; y <= 17; y++ {
		tiles[y*n+19] = view.TileRoad
	}
	return view.Site{
		Width: n, Height: n, Tiles: tiles, Elevation: make([]uint8, n*n), Plaza: view.Point{X: 6, Y: 17}, Gate: view.Point{X: 3, Y: 21},
		Structures: []view.Structure{
			{ID: "k", Kind: view.StructKeep, Variant: "tower", X: 16, Y: 5, W: 3, H: 3, Plot: 4, State: view.StateReady, Object: view.ObjectRef{Kind: "Node", Name: "lone-tower"}},
			{ID: "g", Kind: view.StructGuildhall, X: 4, Y: 14, W: 3, H: 3, State: view.StateReady},
			{ID: "t", Kind: view.StructTreasury, X: 20, Y: 16, W: 1, H: 1, State: view.StateReady},
		},
	}
}

func TestRoadGraphWalksRoadsAndPlazaOnly(t *testing.T) {
	site := lifeSite()
	g := roadGraphOf(site)
	if !g.walk(12, 17) || !g.walk(19, 12) || !g.walk(8, 19) {
		t.Error("roads and plaza are walkable")
	}
	if g.walk(12, 12) || g.walk(0, 5) || g.walk(5, 15) || g.walk(-1, 3) {
		t.Error("grass, water, footprints and the void are not walkable")
	}
	if len(g.tiles) == 0 {
		t.Fatal("no walkable tiles")
	}
}

func TestStrollsFollowTheRoadsAndPingPong(t *testing.T) {
	site := lifeSite()
	g := roadGraphOf(site)
	folk := planFolk(site, g, 12)
	if len(folk) != 12 {
		t.Fatalf("planned %d strolls", len(folk))
	}
	again := planFolk(site, g, 12)
	for i, s := range folk {
		if len(s.path) < 2 || s.speed <= 0 || s.length <= 0 {
			t.Fatalf("stroll %d is degenerate: %+v", i, s)
		}
		if s.role != again[i].role || s.path[len(s.path)-1] != again[i].path[len(again[i].path)-1] {
			t.Fatal("strolls are deterministic")
		}
		for _, p := range s.path {
			if !g.walk(int(p.X+.5), int(p.Y+.5)) {
				t.Fatalf("stroll %d leaves the road at %v", i, p)
			}
		}
		start := s.At(-s.phase)
		if Dist(start.Pos, s.path[0]) > .01 {
			t.Errorf("stroll %d starts at %v, not %v", i, start.Pos, s.path[0])
		}
		far := s.At(-s.phase + s.rest + s.length/s.speed + s.rest/2)
		if Dist(far.Pos, s.path[len(s.path)-1]) > .01 || far.Action == "walk" {
			t.Errorf("stroll %d pauses at the far end: %+v", i, far)
		}
		back := s.At(-s.phase + s.cycle())
		if Dist(back.Pos, start.Pos) > .01 {
			t.Errorf("stroll %d loops back home", i)
		}
	}
}

func TestStrollPoseWalksBetweenPauses(t *testing.T) {
	s := newStroll("carrier", 0, []Vec{{0, 0}, {4, 0}}, 1, 2, 0)
	if p := s.At(1); p.Action != "idle" || p.Pos != (Vec{0, 0}) {
		t.Errorf("pause at home: %+v", p)
	}
	if p := s.At(4); p.Action != "carry" || Dist(p.Pos, Vec{2, 0}) > 1e-9 || p.Dir != "se" {
		t.Errorf("carries water out: %+v", p)
	}
	if p := s.At(9); p.Action != "walk" || Dist(p.Pos, Vec{3, 0}) > 1e-9 || p.Dir != "sw" {
		t.Errorf("walks back empty: %+v", p)
	}
	child := newStroll("child", 1, []Vec{{0, 0}, {1, 0}}, 1, 2, 0)
	if p := child.At(1); p.Action != "work" {
		t.Errorf("children play while they pause: %+v", p)
	}
	if one := newStroll("farmer", 0, []Vec{{2, 2}}, 1, 1, 0); one.At(3).Pos != (Vec{2, 2}) {
		t.Error("a one-point stroll stands still")
	}
}

func TestFolkScaleWithTheRealm(t *testing.T) {
	if FolkCount(0, 5, 0) != 0 {
		t.Error("no Keep, no townsfolk")
	}
	quiet, busy := FolkCount(1, 0, 1), FolkCount(20, 3, 3)
	if quiet < 1 || busy <= quiet || busy > maxFolk {
		t.Errorf("quiet %d busy %d", quiet, busy)
	}
	if FolkCount(500, 50, 9) != maxFolk {
		t.Error("capped")
	}
	if FolkBudget(.5) >= FolkBudget(1) || FolkBudget(1) >= FolkBudget(2) {
		t.Error("closer zoom shows more townsfolk")
	}
}

func TestDeclaredHearthsAndServiceRate(t *testing.T) {
	sc := Scene{Pods: []model.PodView{
		{Namespace: "default", Name: "a", State: model.HearthReady},
		{Namespace: "default", Name: "b", State: model.HearthReady, Static: true},
		{Namespace: "kube-system", Name: "c", State: model.HearthReady, System: true},
		{Namespace: "default", Name: "d", State: model.HearthPending},
	}, Metrics: view.MetricsSnapshot{Pods: map[string]view.PodMetrics{"default/a": {ServedRate: .5}, "default/b": {ServedRate: .25}}}}
	var c census
	got := c.of(sc)
	if got.declared != 1 {
		t.Errorf("declared hearths = %d", got.declared)
	}
	if got.rate != .75 {
		t.Errorf("service rate = %v", got.rate)
	}
	sc.Metrics = view.MetricsSnapshot{Pods: map[string]view.PodMetrics{"default/a": {ServedRate: 1}}}
	if c.of(sc).rate != 1 {
		t.Error("a fresh metrics snapshot refreshes the census")
	}
}

func TestStallsStandOnFreePlazaTiles(t *testing.T) {
	site := lifeSite()
	spots := StallSpots(site, 4)
	if len(spots) != 4 {
		t.Fatalf("stalls = %v", spots)
	}
	for i, s := range spots {
		x, y := int(s.X), int(s.Y)
		if site.Tile(x, y) != view.TilePlaza || NearAnyStructure(site, x, y) {
			t.Errorf("stall %d at %v is not on free plaza", i, s)
		}
		for _, o := range spots[:i] {
			if Dist(s, o) < stallGap {
				t.Errorf("stalls %v and %v crowd each other", s, o)
			}
		}
	}
	if StallCount(0) != 0 || StallCount(.2) != 1 || StallCount(10) != maxStalls || StallCount(1) < StallCount(.2) {
		t.Error("stalls follow trade")
	}
	if len(StallSpots(view.Site{}, 3)) != 0 {
		t.Error("no plaza, no stalls")
	}
}
