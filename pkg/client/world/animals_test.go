package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func beasts(site view.Site) []critter {
	hb := BuildHabitat(site, LandOf(site))
	out := []critter{}
	for y := 0; y < site.Height; y++ {
		for x := 0; x < site.Width; x++ {
			out = append(out, crittersFor(site, hb, x, y)...)
		}
	}
	return out
}

func edgeOfWood(n int) view.Site {
	site := wildSite(n, view.TileMeadow)
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			switch {
			case x%12 < 3:
				site.Tiles[y*n+x] = view.TileForest
			case y%2 == 0:
				site.Tiles[y*n+x] = view.TileGrass
			}
		}
	}
	return site
}

func TestWildlifeLivesInTheWildsAndFleesTheTown(t *testing.T) {
	site := edgeOfWood(60)
	wild := beasts(site)
	roles := map[string]int{}
	for _, a := range wild {
		roles[a.role]++
	}
	for _, r := range []string{"deer", "stag", "rabbit", "fox", "boar", "sheep"} {
		if roles[r] == 0 {
			t.Errorf("no %s in the wilds: %v", r, roles)
		}
	}
	busy := site
	for i := 0; i < 5; i++ {
		busy.Structures = append(busy.Structures, keepAt(6+i*11, 28, 4))
	}
	for i := range busy.Wear {
		if i%60 == 30 {
			busy.Wear[i] = 9000
		}
	}
	near := func(list []critter) int {
		n := 0
		for _, a := range list {
			if a.home.Y > 22 && a.home.Y < 36 {
				n++
			}
		}
		return n
	}
	if before, after := near(wild), near(beasts(busy)); before < 10 || after*4 > before {
		t.Errorf("a busy town keeps %d of %d animals nearby", after, before)
	}
}

func TestAnimalsWanderDeterministicallyAroundHome(t *testing.T) {
	site := edgeOfWood(40)
	site.Structures = []view.Structure{keepAt(20, 20, 0)}
	hb := BuildHabitat(site, LandOf(site))
	moved, grazed := false, false
	for _, a := range beasts(site) {
		for ti := 0; ti < 40; ti++ {
			tt := float64(ti) * 1.3
			p := a.Pose(site, hb, tt, false)
			if p != a.Pose(site, hb, tt, false) {
				t.Fatal("pose is not deterministic")
			}
			if Dist(p.Pos, a.home) > a.kind.radius*1.5+0.01 {
				t.Fatalf("%s strayed %v from home", a.role, Dist(p.Pos, a.home))
			}
			if !roams(site, hb, p.Pos, a.kind.shy) && Dist(p.Pos, a.home) > 0.01 {
				t.Fatalf("%s entered forbidden ground at %v", a.role, p.Pos)
			}
			moved = moved || p.Action == "walk"
			grazed = grazed || p.Action == "work"
		}
		if r := a.Pose(site, hb, 5, true); r.Pos != a.home || r.Action != "idle" {
			t.Fatal("resting animals stay home")
		}
	}
	if !moved || !grazed {
		t.Errorf("moved %v grazed %v", moved, grazed)
	}
}

func TestBirdsCircleAndFishLeap(t *testing.T) {
	n := 160
	site := wildSite(n, view.TileWater)
	for y := 40; y < 120; y++ {
		for x := 40; x < 120; x++ {
			site.Tiles[y*n+x] = view.TileGrass
		}
	}
	gulls, finches, fish := 0, 0, 0
	var f flock
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if fl, ok := flockFor(site, x, y); ok {
				f = fl
				if fl.role == "gull" {
					gulls++
				} else {
					finches++
				}
			}
			if _, ok := fishFor(site, x, y); ok {
				fish++
			}
		}
	}
	if gulls == 0 || finches == 0 || fish == 0 {
		t.Errorf("gulls %d finches %d fish %d", gulls, finches, fish)
	}
	a, b := f.member(1, 0), f.member(3, 0)
	if a.pos == b.pos || a.alt < 20 || Dist(a.pos, f.center) > f.radius+0.01 {
		t.Errorf("flock member %+v then %+v", a, b)
	}
	if sign(-2) != -1 || sign(0) != 1 {
		t.Error("sign")
	}
}

func TestWildlifeAndChartersDraw(t *testing.T) {
	site := edgeOfWood(64)
	for y := 40; y < 64; y++ {
		for x := 0; x < 64; x++ {
			site.Tiles[y*64+x] = view.TileWater
		}
	}
	site.Islands = append(site.Islands, view.Island{ID: "far", Center: view.Point{X: 20, Y: 20}})
	site.Islands[0].Owned = false
	r := NewRenderer(gfx.Load(art.Build()))
	dst := ebiten.NewImage(960, 540)
	cam := render.NewCamera(480, 60).CenterOn(20, 30, 960, 540)
	cam.Zoom = .5
	for _, phase := range []float64{.5, 0} {
		sc := Scene{Site: site, DayPhase: phase}
		for f := 0; f < 20; f++ {
			sc.Time = float64(f) * .7
			r.Draw(dst, cam, sc)
		}
		if n := len(r.wildItems(cam, sc)); n == 0 {
			t.Errorf("no wildlife drawn at phase %v", phase)
		}
	}
	if len(r.charterItems(cam, Scene{Site: site})) != 2 {
		t.Error("each unowned island raises its marker")
	}
	site.Islands[0].Owned = true
	if len(r.charterItems(cam, Scene{Site: site})) != 1 {
		t.Error("owned islands have no marker")
	}
	if len(NewRenderer(nil).wildItems(cam, Scene{})) != 0 || len(NewRenderer(nil).sceneryItems(cam, Scene{})) != 0 {
		t.Error("a renderer without land draws no wildlife")
	}
}
