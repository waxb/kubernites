package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func wornSite(n int, wear map[[2]int]uint16) view.Site {
	w := make([]uint16, n*n)
	for p, v := range wear {
		w[p[1]*n+p[0]] = v
	}
	tiles := make([]view.TileKind, n*n)
	for i := range tiles {
		tiles[i] = view.TileGrass
	}
	return view.Site{Width: n, Height: n, Tiles: tiles, Elevation: make([]uint8, n*n), Wear: w}
}

func TestTheTownPlanKeepsATrunkAndItsBranchAndDropsFringeBlobsAndStubs(t *testing.T) {
	wear := map[[2]int]uint16{}
	for x := 2; x < 18; x++ {
		wear[[2]int{x, 10}] = 3000
	}
	for y := 3; y < 10; y++ {
		wear[[2]int{9, y}] = 3000
	}
	for x := 5; x < 8; x++ {
		for y := 12; y < 15; y++ {
			wear[[2]int{x, y}] = 500
		}
	}
	wear[[2]int{14, 11}] = 500
	wear[[2]int{15, 11}] = 500
	wear[[2]int{13, 14}] = 500
	site := wornSite(24, wear)
	plan := TownPlan(site)
	at := func(x, y int) uint8 { return plan[y*site.Width+x] }
	for x := 2; x < 18; x++ {
		if at(x, 10) != RoadDirt {
			t.Fatalf("the trunk lost its tile at %d,10", x)
		}
	}
	for y := 3; y < 10; y++ {
		if at(9, y) != RoadDirt {
			t.Fatalf("the branch lost its tile at 9,%d", y)
		}
	}
	if at(14, 11) != RoadNone || at(15, 11) != RoadNone {
		t.Error("a shoulder beside the trunk survived")
	}
	if at(13, 14) != RoadNone {
		t.Error("a stray tile that leads nowhere survived")
	}
	blob := 0
	for x := 5; x < 8; x++ {
		for y := 12; y < 15; y++ {
			if at(x, y) != RoadNone {
				blob++
			}
		}
	}
	if blob >= 9 {
		t.Error("a trodden blob stayed a blob")
	}
}
