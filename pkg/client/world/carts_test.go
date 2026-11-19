package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestCartsRollInStepWithIncome(t *testing.T) {
	var y cartYard
	y.step(0, 100, 10)
	if len(y.rolls) != 0 {
		t.Fatal("priming never dispatches the backlog")
	}
	y.step(1, 102, 10)
	if len(y.rolls) != 0 {
		t.Fatal("a cart waits for a full load")
	}
	y.step(7, 112, 10)
	if len(y.rolls) != 1 || y.rolls[0].load != 0 {
		t.Fatalf("a small load rolls out: %+v", y.rolls)
	}
	y.step(9, 200, 10)
	if len(y.rolls) != 1 {
		t.Fatal("carts keep a gap between them")
	}
	y.step(14, 200, 10)
	if len(y.rolls) != 2 || y.rolls[1].load != 2 {
		t.Fatalf("a rich stretch heaps the cart: %+v", y.rolls)
	}
	y.step(60, 200, 10)
	if len(y.rolls) != 0 {
		t.Fatal("finished carts leave")
	}
	y.step(50, 300, 10)
	if len(y.rolls) != 0 || y.banked != 300 {
		t.Fatal("time running backwards resets the yard")
	}
	y.step(70, 400, 0)
	if len(y.rolls) != 0 || y.banked != 400 {
		t.Fatal("no road, no carts, and the backlog is not hoarded")
	}
}

func TestCartsAreCapped(t *testing.T) {
	var y cartYard
	y.step(0, 0, 100)
	for i := 1; i < 40; i++ {
		y.step(float64(i)*cartEvery, float64(i)*100, 100)
	}
	if len(y.rolls) > maxCarts {
		t.Errorf("%d carts on the road", len(y.rolls))
	}
}

func TestCartLoadLevels(t *testing.T) {
	if CartLoadLevel(6) != 0 || CartLoadLevel(30) != 1 || CartLoadLevel(500) != 2 {
		t.Error("load levels")
	}
}

func TestCartRouteFollowsTheRoad(t *testing.T) {
	site := lifeSite()
	g := roadGraphOf(site)
	route := CartRoute(site, g)
	if len(route) < 3 {
		t.Fatalf("route = %v", route)
	}
	g0, _ := FirstOfKind(site, view.StructGuildhall, true)
	t0, _ := FirstOfKind(site, view.StructTreasury, true)
	if Dist(route[0], Porch(g0)) > 1.5 || Dist(route[len(route)-1], Porch(t0)) > 1.5 {
		t.Errorf("route runs %v .. %v", route[0], route[len(route)-1])
	}
	for _, p := range route[1 : len(route)-1] {
		if !g.walk(int(p.X+.5), int(p.Y+.5)) {
			t.Errorf("cart leaves the road at %v", p)
		}
	}
	bare := lifeSite()
	bare.Structures = bare.Structures[:2]
	if CartRoute(bare, roadGraphOf(bare)) != nil {
		t.Error("no store, no route")
	}
	island := lifeSite()
	for i := range island.Tiles {
		if island.Tiles[i] == view.TileRoad {
			island.Tiles[i] = view.TileGrass
		}
	}
	if r := CartRoute(island, roadGraphOf(island)); len(r) != 2 {
		t.Errorf("without a road the cart goes straight: %v", r)
	}
}

func TestCartPoseRollsAndFades(t *testing.T) {
	route := []Vec{{0, 0}, {4, 0}}
	run := cartRun{start: 10, life: 8, load: 1}
	if _, ok := run.pose(route, 9); ok {
		t.Error("not yet rolling")
	}
	p, ok := run.pose(route, 14)
	if !ok || Dist(p.pos, Vec{2, 0}) > 1e-9 || p.dir != "se" || p.alpha != 1 {
		t.Errorf("midway pose %+v", p)
	}
	if p, _ := run.pose(route, 10.1); p.alpha >= 1 {
		t.Error("carts fade in")
	}
	if _, ok := run.pose(route, 18.5); ok {
		t.Error("gone after its life")
	}
}
