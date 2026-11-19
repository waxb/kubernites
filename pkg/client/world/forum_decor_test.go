package world

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func propsFrom(d forumDecor, age int) map[string]int {
	out := map[string]int{}
	for _, p := range d.props {
		if age >= p.from && age <= p.upTo {
			out[p.kind]++
		}
	}
	return out
}

func TestTheForumDecorFollowsTheAges(t *testing.T) {
	d := planForumDecor(bareForumSite())
	if len(d.tiles) != 25 {
		t.Fatalf("the decor paves %d tiles, want 25", len(d.tiles))
	}
	cases := []struct {
		age  int
		want map[string]int
	}{
		{0, map[string]int{PropTerminus: 1}},
		{1, map[string]int{}},
		{tierUAT, map[string]int{PropStatue: 4, PropFountain: 1}},
		{tierProd, map[string]int{PropStatue: 4, PropFountain: 1, PropRostra: 1, PropArch: 1, PropColonnade: 1}},
	}
	for _, c := range cases {
		got := propsFrom(d, c.age)
		for kind, n := range c.want {
			if (kind == PropTerminus || kind == PropColonnade) && got[kind] < n || kind != PropTerminus && kind != PropColonnade && got[kind] != n {
				t.Errorf("age %d: %d %s, want %d (%v)", c.age, got[kind], kind, n, got)
			}
		}
		if c.age > 0 && got[PropTerminus] > 0 {
			t.Errorf("age %d keeps the tier 0 boundary stones", c.age)
		}
	}
	if !d.edges[Vec{13, 1.5}] && !d.edges[Vec{12.5, 2}] {
		t.Error("the colonnades replaced no kerb stone")
	}
	for _, p := range d.props {
		if p.kind == PropColonnade && (p.at == Vec{12.5, 2} || p.at == Vec{13, 1.5} || p.at == Vec{14, 1.5}) {
			t.Errorf("a colonnade hides the corner statue at %v", p.at)
		}
	}
}

func TestTheForumPaveNamesUpgradeByAge(t *testing.T) {
	d := planForumDecor(bareForumSite())
	roses, bands := 0, 0
	for i, tile := range d.tiles {
		for age := 0; age < art.AgeTiers; age++ {
			names := d.tileNames(i, age)
			if !strings.HasPrefix(names[0], "forum/pave/") || !strings.Contains(names[0], "/"+string(rune('0'+age))+"/") {
				t.Fatalf("tile %v at age %d paves %q", tile.at, age, names[0])
			}
			if age < tierUAT && len(names) != 1 {
				t.Fatalf("a lower-tier tile carries overlays %v", names)
			}
			for _, n := range names {
				roses += btoiT(age == tierProd && strings.HasPrefix(n, "forum/rose/"))
				bands += btoiT(strings.HasPrefix(n, "forum/band/"))
			}
		}
		if &d.tileNames(i, 1)[0] != &d.tileNames(i, 1)[0] {
			t.Fatal("tile names are not cached")
		}
	}
	if roses != 9 || bands == 0 {
		t.Fatalf("%d rose pieces and %d bands at Production", roses, bands)
	}
}

func btoiT(b bool) int {
	if b {
		return 1
	}
	return 0
}

func forumCamera() render.Camera {
	cam := render.NewCamera(ViewWidth/2, ViewHeight/2)
	cam.X, cam.Y = (15-4)*16, (15+4)*8
	return cam
}

func TestTheForumDrawsAtEveryAgeAndCullsOffScreen(t *testing.T) {
	site := bareForumSite()
	r := NewRenderer(gfx.Load(art.Build()))
	dst := ebiten.NewImage(ViewWidth, ViewHeight)
	defer dst.Deallocate()
	cam := forumCamera()
	for age := 0; age < art.AgeTiers; age++ {
		r.ages = nil
		sc := Scene{Site: site, Age: age, Time: 1}
		r.Draw(dst, cam, sc)
		if len(r.forumItems(cam, sc)) == 0 {
			t.Fatalf("age %d drew nothing on the Forum's edge", age)
		}
	}
	far := cam
	far.X += 5000
	if n := len(r.forumItems(far, Scene{Site: site, Age: tierProd, Time: 1})); n != 0 {
		t.Fatalf("%d Forum pieces drawn off screen", n)
	}
	if r.forumItems(cam, Scene{Site: landSite()}) != nil {
		t.Fatal("a realm without a Forum draws one")
	}
}

func TestAnAgeUpRepavesTheForumTileByTileInDust(t *testing.T) {
	site := bareForumSite()
	r := NewRenderer(nil)
	cam := forumCamera()
	r.forumItems(cam, Scene{Site: site, Age: 1, Time: 10})
	bursts, changed := 0, map[string]bool{}
	for step := 0.0; step <= AgeStagger+AgeWork+.5; step += .1 {
		sc := Scene{Site: site, Age: tierUAT, Time: 10.01 + step}
		r.forumItems(cam, sc)
		bursts += len(r.forumBursts(cam, sc, r.forumDecorOf(site)))
		for _, tile := range r.decor.tiles {
			if age, _ := r.agesOf().phase(tile.id); age == tierUAT {
				changed[tile.id] = true
			}
		}
	}
	if bursts == 0 || len(changed) != len(r.decor.tiles) {
		t.Fatalf("%d dust puffs, %d of %d tiles repaved", bursts, len(changed), len(r.decor.tiles))
	}
}

func TestTheMarketSquareGrowsGreenOverTime(t *testing.T) {
	site := landSite()
	site.Plaza = view.Point{X: 20, Y: 20}
	for y := 17; y <= 23; y++ {
		for x := 17; x <= 23; x++ {
			site.Tiles[y*site.Width+x] = view.TilePlaza
		}
	}
	if MarketGrown(0) != 0 || MarketGrown(MarketGrowth*2) != 1 || len(MarketTufts(site, 0)) != 0 {
		t.Fatal("the market square is green before any time passes")
	}
	half, full := MarketTufts(site, .5), MarketTufts(site, 1)
	if len(half) == 0 || len(full) <= len(half) {
		t.Fatalf("growth %d at half, %d at full", len(half), len(full))
	}
	for _, tuft := range full {
		dx, dy := tuft.At.X-site.Plaza.X, tuft.At.Y-site.Plaza.Y
		if site.Tile(tuft.At.X, tuft.At.Y) != view.TilePlaza || dx*dx+dy*dy <= marketHeartR2 || tuft.Variant >= art.MarketGrowths {
			t.Fatalf("a tuft at %v", tuft)
		}
	}
	r := NewRenderer(gfx.Load(art.Build()))
	dst := ebiten.NewImage(ViewWidth, ViewHeight)
	defer dst.Deallocate()
	r.drawMarketGround(dst, render.NewCamera(ViewWidth/2, ViewHeight/2), Scene{Site: site, RealmTime: MarketGrowth})
}

func TestErrandsWalkAroundTheCodex(t *testing.T) {
	site := landSite()
	codex := structureNamed(site, "codex")
	from, to := Vec{float64(codex.X) - 3, float64(codex.Y) + .5}, Vec{float64(codex.X) + 4, float64(codex.Y) + .5}
	e := Errand{Kind: ErrandDecree, Role: "scribe", From: from, To: to, Start: 0, Life: ErrandLife(ErrandDecree, from, to)}
	r := NewRenderer(nil)
	sc := Scene{Site: site}
	for tm := 0.0; tm <= e.Life; tm += .05 {
		step, ok := ErrandAt(e, to, tm)
		if !ok {
			continue
		}
		step = r.RoutedStep(sc, e, to, step)
		if x, y := step.Pose.Pos.X, step.Pose.Pos.Y; x > float64(codex.X)-.4 && x < float64(codex.X+codex.W)-.6 && y > float64(codex.Y)-.4 && y < float64(codex.Y+codex.H)-.6 {
			t.Fatalf("the scribe walks through the Codex at %.1f: %v", tm, step.Pose.Pos)
		}
	}
}

func TestTheForumIsPavingOnTheMapAndNoRoadOrPasture(t *testing.T) {
	site := bareForumSite()
	site.Wear = make([]uint16, len(site.Tiles))
	site.Wear[4*site.Width+15] = 60000
	if RoadLevel(site, 15, 4) != RoadNone || TileColor(view.TileForum) == TileColor(view.TileGrass) {
		t.Fatal("the Forum shows as a road or as grass on the map")
	}
	hb := BuildHabitat(site, LandOf(site))
	if roams(site, hb, Vec{15, 4}, 1) {
		t.Fatal("animals graze on the Forum")
	}
}
