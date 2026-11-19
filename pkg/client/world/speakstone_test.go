package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const stoneSamples = 24

func TestTheSpeakingStoneStandsInTheMiddleOfThePlaza(t *testing.T) {
	site := lifeSite()
	stone, ok := StoneTile(site)
	if !ok || stone != (Point{site.Plaza.X, site.Plaza.Y}) {
		t.Fatalf("stone tile = %v, %v", stone, ok)
	}
	if site.OnForum(stone.X, stone.Y) {
		t.Fatal("the Stone stands on the Forum")
	}
	bare := site
	bare.Plaza = view.Point{X: 12, Y: 12}
	if _, ok := StoneTile(bare); ok {
		t.Fatal("a Stone stands where there is no plaza")
	}
}

func TestNothingWalksThroughTheSpeakingStone(t *testing.T) {
	site := lifeSite()
	stone, _ := StoneTile(site)
	roads, land := roadGraphOf(site), LandGridOf(site)
	if roads.walk(stone.X, stone.Y) || land.walk(stone) {
		t.Fatal("folk or carriers may step onto the Stone")
	}
	for _, s := range planFolk(site, roads, maxFolk) {
		for _, p := range s.path {
			if TileOf(p) == stone {
				t.Fatalf("a stroll crosses the Stone: %v", s.path)
			}
		}
	}
	for _, leg := range [][2]Vec{{{3, 17}, {9, 17}}, {{6, 13}, {6, 20}}, {{3, 13}, {9, 20}}, {{9, 13}, {3, 20}}} {
		route := land.Route(leg[0], leg[1])
		for i := 1; i < len(route); i++ {
			for k := 0; k <= stoneSamples; k++ {
				if TileOf(Lerp(route[i-1], route[i], float64(k)/stoneSamples)) == stone {
					t.Fatalf("a carrier from %v to %v walks through the Stone: %v", leg[0], leg[1], route)
				}
			}
		}
	}
}

func TestNoBoothStandsWhereTheStoneWouldHideIt(t *testing.T) {
	site := lifeSite()
	stone, _ := StoneTile(site)
	at := func(d [2]int) Vec { return Point{stone.X + d[0], stone.Y + d[1]}.Center() }
	for _, d := range [][2]int{{-2, -2}, {-1, -2}, {-2, -1}, {-3, -2}, {-2, -3}, {1, 2}, {2, 1}, {2, 2}, {2, 3}, {3, 2}} {
		if clearOfStone(site, at(d)) {
			t.Errorf("a booth %v from the Stone would be covered by it or cover it", d)
		}
	}
	for _, d := range [][2]int{{-3, -3}, {3, 3}, {-3, 3}, {3, -3}, {3, 0}, {0, 3}, {-3, 0}, {0, -3}, {2, 0}, {0, 2}, {-2, 0}, {0, -2}} {
		if !clearOfStone(site, at(d)) {
			t.Errorf("a booth %v from the Stone stands clear of it", d)
		}
	}
	if len(StallSpots(site, maxStalls)) != maxStalls {
		t.Fatalf("the plaza keeps room for %d booths", maxStalls)
	}
	tight := lifeSite()
	tight.Structures = nil
	for i, k := range tight.Tiles {
		if k == view.TilePlaza {
			tight.Tiles[i] = view.TileGrass
		}
	}
	for _, d := range [][2]int{{0, 0}, {-2, -2}, {2, 0}} {
		p := Point{stone.X + d[0], stone.Y + d[1]}
		tight.Tiles[p.Y*tight.Width+p.X] = view.TilePlaza
	}
	if got := StallSpots(tight, maxStalls); len(got) != 1 || TileOf(got[0]) != (Point{stone.X + 2, stone.Y}) {
		t.Fatalf("booth spots beside the Stone = %v", got)
	}
}

func TestTheStoneStandsDormantAndIsPickedOnlyOnceAwake(t *testing.T) {
	site := lifeSite()
	at, _ := StoneSpot(site)
	cam := render.NewCamera(480, 60).CenterOn(at.X, at.Y, ViewWidth, ViewHeight)
	r := NewRenderer(gfxLibrary())
	dst := ebiten.NewImage(ViewWidth, ViewHeight)
	defer dst.Deallocate()
	sc := Scene{Site: site, Time: 1, Real: 1, Metrics: view.MetricsSnapshot{Pods: map[string]view.PodMetrics{}}}
	r.Draw(dst, cam, sc)
	if stoneHit(r) != nil || !r.stone.seen || r.stone.awake {
		t.Fatalf("the dormant stone opens like the awake one, or is not drawn: %+v", r.stone)
	}
	sx0, sy0 := ScreenPointOf(cam, site, at)
	if got, ok := r.Pick(int(sx0), int(sy0)-10); !ok || got.Kind != HitDormantStone {
		t.Fatalf("a click on the dormant Stone picks %+v: it should answer that it sleeps", got)
	}
	sc.Stone, sc.Real = true, 2
	r.Draw(dst, cam, sc)
	if stoneHit(r) == nil {
		t.Fatal("the awakened Stone is not drawn as clickable")
	}
	sx, sy := ScreenPointOf(cam, site, at)
	if got, ok := r.Pick(int(sx), int(sy)-10); !ok || got.Kind != HitStone {
		t.Fatalf("clicking the Stone picks %+v", got)
	}
	if p, waking := r.StoneWaking(2 + stoneWakeSecs/2); !waking || p < .4 || p > .6 {
		t.Fatalf("halfway through its awakening the Stone reads %v, %v", p, waking)
	}
	if _, waking := r.StoneWaking(2 + stoneWakeSecs + .1); waking {
		t.Fatal("the awakening never ends")
	}
}

func TestALoadedRealmsStoneDoesNotAwakenAgain(t *testing.T) {
	site := lifeSite()
	at, _ := StoneSpot(site)
	cam := render.NewCamera(480, 60).CenterOn(at.X, at.Y, ViewWidth, ViewHeight)
	r := NewRenderer(gfxLibrary())
	dst := ebiten.NewImage(ViewWidth, ViewHeight)
	defer dst.Deallocate()
	r.Draw(dst, cam, Scene{Site: site, Time: 1, Real: 1, Stone: true, Metrics: view.MetricsSnapshot{Pods: map[string]view.PodMetrics{}}})
	if _, waking := r.StoneWaking(1); waking {
		t.Fatal("a Stone already awake when the realm loads plays its awakening")
	}
}

func stoneHit(r *Renderer) *Hit {
	for _, h := range r.HitList {
		if h.Kind == HitStone {
			return &h
		}
	}
	return nil
}
