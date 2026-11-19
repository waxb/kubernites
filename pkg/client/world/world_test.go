package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func testSite() view.Site {
	const n = 20
	tiles := make([]view.TileKind, n*n)
	elev := make([]uint8, n*n)
	for i := range tiles {
		tiles[i] = view.TileGrass
	}
	for x := 0; x < n; x++ {
		tiles[x] = view.TileWater
	}
	tiles[5*n+12] = view.TileForest
	tiles[6*n+12] = view.TileRock
	tiles[7*n+12] = view.TileMeadow
	elev[10*n+10] = 2
	return view.Site{
		Width: n, Height: n, Tiles: tiles, Elevation: elev, Plaza: view.Point{X: 3, Y: 15}, Gate: view.Point{X: 1, Y: 18},
		Structures: []view.Structure{
			{ID: "k", Kind: view.StructKeep, Variant: "tower", X: 8, Y: 8, W: 3, H: 3, Plot: 4, State: view.StateReady, Object: view.ObjectRef{Kind: "Node", Name: "lone-tower"}},
			{ID: "g", Kind: view.StructGuildhall, X: 2, Y: 12, W: 3, H: 3, State: view.StateReady},
			{ID: "c", Kind: view.StructCodex, X: 13, Y: 8, W: 2, H: 2, State: view.StateConstruction},
		},
	}
}

func TestFootprintHelpers(t *testing.T) {
	s := testSite().Structures[0]
	lo, hi := Footprint(s)
	if lo != (Point{8, 8}) || hi != (Point{10, 10}) || !Covers(s, 10, 9) || Covers(s, 11, 9) {
		t.Fatal("footprint wrong")
	}
	if Anchor(s) != (Vec{10.5, 10.5}) || Door(s).X <= Anchor(s).X {
		t.Fatal("anchor/door wrong")
	}
	if (Point{2, 3}).Center() != (Vec{2, 3}) {
		t.Fatal("center wrong")
	}
}

func TestLookups(t *testing.T) {
	site := testSite()
	if k, ok := KeepForNode(site, "lone-tower"); !ok || k.ID != "k" {
		t.Fatal("keep lookup failed")
	}
	if _, ok := KeepForNode(site, "nope"); ok {
		t.Fatal("unexpected keep")
	}
	if _, ok := FirstOfKind(site, view.StructCodex, true); ok {
		t.Fatal("codex under construction should not be ready")
	}
	if _, ok := FirstOfKind(site, view.StructCodex, false); !ok {
		t.Fatal("codex not found")
	}
}

func TestPendingSpotsGatherOnThePlaza(t *testing.T) {
	site := testSite()
	if p := PendingSpot(site, 5); p.X >= 8 {
		t.Fatalf("pending spot = %v", p)
	}
	if p := PendingSpot(view.Site{Plaza: view.Point{X: 4, Y: 4}}, 0); p != (Vec{4, 4}) {
		t.Fatalf("pending without keep = %v", p)
	}
}

func TestWalkerPoses(t *testing.T) {
	home, work, bank := Vec{0, 0}, Vec{6, 0}, Vec{6, 6}
	actions := map[string]bool{}
	carried := false
	for i := 0; i < 400; i++ {
		p := WalkerPose(float64(i)*.1, 7, home, work, bank, true)
		actions[p.Action] = true
		carried = carried || p.Carry
		if p.Pos.X < -.01 || p.Pos.X > 6.01 || p.Pos.Y < -.01 || p.Pos.Y > 6.01 {
			t.Fatalf("pose out of route: %+v", p)
		}
	}
	if !actions["walk"] || !actions["work"] || !actions["carry"] || !carried {
		t.Fatalf("actions = %v", actions)
	}
	idleActions := map[string]bool{}
	for i := 0; i < 200; i++ {
		p := WalkerPose(float64(i)*.1, 3, home, work, bank, false)
		idleActions[p.Action] = true
		if Dist(p.Pos, home) > 1 {
			t.Fatalf("idle wandered too far: %+v", p)
		}
	}
	if !idleActions["idle"] || !idleActions["walk"] {
		t.Fatalf("idle actions = %v", idleActions)
	}
	if Facing(Vec{0, 0}, Vec{1, 0}) != "se" || Facing(Vec{0, 0}, Vec{0, 1}) != "sw" || Frame(1.9, 2) != 3 {
		t.Fatal("facing/frame wrong")
	}
	if Lerp(Vec{0, 0}, Vec{2, 2}, 2) != (Vec{2, 2}) {
		t.Fatal("lerp should clamp")
	}
}
