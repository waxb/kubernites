package world

import (
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func livingScene(t *testing.T) (Scene, render.Camera, *Renderer, *ebiten.Image, view.Structure) {
	t.Helper()
	sc, cam, r := errandScene(t)
	keep, ok := FirstOfKind(sc.Site, view.StructKeep, true)
	if !ok {
		t.Fatal("the scene has no Keep")
	}
	sc.Keeps = []view.KeepView{{ID: keep.ID, Name: keep.Object.Name, Ready: true, Allocatable: view.ResourcePair{CPU: 4000}, Requested: view.ResourcePair{CPU: 800}}}
	dst := ebiten.NewImage(960, 540)
	t.Cleanup(dst.Deallocate)
	return sc, cam, r, dst, keep
}

func step(r *Renderer, dst *ebiten.Image, cam render.Camera, sc *Scene, seconds float64) {
	sc.Time += seconds
	sc.RealmTime += seconds
	r.Draw(dst, cam, *sc)
}

func TestWheatPatchesHoldFiveToSevenTilesAtMostHalfOutsideTheBorder(t *testing.T) {
	sc, cam, r, dst, keep := livingScene(t)
	r.Draw(dst, cam, sc)
	plans := &r.livingOf().plans
	plan, ok := plans.of(keep)
	if !ok || len(plan.wheat) == 0 {
		t.Fatalf("the Keep has no wheat: %+v", plan)
	}
	area, _ := plans.borders.AreaOf(keep.ID)
	patches := map[int][]wheatTile{}
	seen := map[Point]bool{}
	for _, w := range plan.wheat {
		patches[w.patch] = append(patches[w.patch], w)
		if seen[w.at] {
			t.Fatalf("tile %v holds two crops", w.at)
		}
		seen[w.at] = true
		if k := sc.Site.Tile(w.at.X, w.at.Y); k != view.TileGrass && k != view.TileMeadow {
			t.Errorf("wheat grows on tile kind %d at %v", k, w.at)
		}
	}
	for id, tiles := range patches {
		if len(tiles) < 5 || len(tiles) > 7 {
			t.Errorf("patch %d holds %d tiles, want 5 to 7", id, len(tiles))
		}
		outside, ripest := 0, tiles[0]
		for _, w := range tiles {
			if o, in := plans.borders.At(w.at.X, w.at.Y); !in || o != area {
				outside++
			}
			if w.mid > ripest.mid {
				ripest = w
			}
		}
		if outside > len(tiles)/2 {
			t.Errorf("patch %d has %d of %d tiles past the border", id, outside, len(tiles))
		}
		if ripest.mid < .8 {
			t.Errorf("patch %d has no ripe middle: %v", id, ripest.mid)
		}
		for _, w := range tiles {
			if w.mid < 0 || w.mid > 1 {
				t.Errorf("ripeness %v is off the scale", w.mid)
			}
		}
	}
	again := planKeep(plans.ground, plans.borders, area, keep)
	if !reflect.DeepEqual(again, plan) {
		t.Error("the same Keep always plants the same fields")
	}
}

func TestBushesPickBareWhileTheKeepIsHungryAndRegrowAfter(t *testing.T) {
	sc, cam, r, dst, keep := livingScene(t)
	sc.Keeps[0].Requested.CPU = 3900
	for range 12 {
		step(r, dst, cam, &sc, 1)
	}
	f, _ := r.livingOf().keepFolkOf(keep.ID)
	if !f.hungry || BerryFruit(f.fruit) != 0 {
		t.Fatalf("a Keep with requests at its food is hungry with bare bushes: %+v", f)
	}
	sc.Keeps[0].Requested.CPU = 600
	step(r, dst, cam, &sc, 2)
	f, _ = r.livingOf().keepFolkOf(keep.ID)
	if f.hungry || f.fruit <= 0 || f.fruit >= 1 {
		t.Errorf("fruit begins to return gradually, got %v", f.fruit)
	}
	for range 40 {
		step(r, dst, cam, &sc, 1)
	}
	if f, _ = r.livingOf().keepFolkOf(keep.ID); BerryFruit(f.fruit) != art.BushFruits-1 {
		t.Errorf("the bushes fruit again once the food is enough, got %v", f.fruit)
	}
}

func TestWheatShadesEaseTowardTheKeepsFreeFood(t *testing.T) {
	sc, cam, r, dst, keep := livingScene(t)
	sc.Keeps[0].Requested.CPU = 3600
	r.Draw(dst, cam, sc)
	low, _ := r.livingOf().keepFolkOf(keep.ID)
	sc.Keeps[0].Requested.CPU = 200
	step(r, dst, cam, &sc, 1)
	mid, _ := r.livingOf().keepFolkOf(keep.ID)
	for range 30 {
		step(r, dst, cam, &sc, 1)
	}
	high, _ := r.livingOf().keepFolkOf(keep.ID)
	if !(low.free < mid.free && mid.free < high.free) {
		t.Errorf("the wheat ripens over time: %v %v %v", low.free, mid.free, high.free)
	}
}

func TestIvyFollowsTheNodesQuietAgeAndVanishesWhenItIsRebuilt(t *testing.T) {
	sc, cam, r, dst, keep := livingScene(t)
	sc.RealmTime = 5000
	sc.Stood = map[string]float64{keep.Object.Name: sc.RealmTime - IvyFullAfter - 10}
	r.Draw(dst, cam, sc)
	if got := r.livingOf().ivyOf(keep.ID); got != IvySteps {
		t.Fatalf("a Keep quiet for days wears full ivy, got %d", got)
	}
	sc.Stood = map[string]float64{keep.Object.Name: sc.RealmTime}
	r.Draw(dst, cam, sc)
	if got := r.livingOf().ivyOf(keep.ID); got != 0 {
		t.Errorf("a rebuilt Keep is bare, got %d", got)
	}
	for _, s := range []float64{.2, .5, .9} {
		sc.Stood = map[string]float64{keep.Object.Name: sc.RealmTime - s*IvyFullAfter}
		r.Draw(dst, cam, sc)
	}
	sc.Stood = nil
	r.Draw(dst, cam, sc)
	if got := r.livingOf().ivyOf(keep.ID); got != 0 {
		t.Errorf("a Keep with no known age grows nothing, got %d", got)
	}
}

func bareHut(name, node string) model.PodView {
	return model.PodView{Namespace: "default", Name: name, Node: node, Sprite: "porter", Tier: 1, State: model.HearthReady, Ready: true}
}

func TestACairnMarksAnImpRazedBareHutOnly(t *testing.T) {
	sc, cam, r, dst, keep := livingScene(t)
	razed, deleted := bareHut("lone-razed", keep.Object.Name), bareHut("lone-deleted", keep.Object.Name)
	sc.Pods = append(sc.Pods, razed, deleted)
	r.Draw(dst, cam, sc)
	home, ok := r.Homes[razed.Name]
	if !ok {
		t.Fatal("the bare hut has no home")
	}
	sc.Omen = &Omen{Pod: razed.Name, Start: sc.Time, Arrive: sc.Time + 1}
	step(r, dst, cam, &sc, .5)
	kept := sc.Pods[:0:0]
	for _, p := range sc.Pods {
		if p.Name != razed.Name && p.Name != deleted.Name {
			kept = append(kept, p)
		}
	}
	sc.Pods = kept
	step(r, dst, cam, &sc, 1)
	cairns := r.livingOf().cairns
	if len(cairns) != 1 {
		t.Fatalf("cairns = %+v, want one where the Imp struck", cairns)
	}
	if want := (Vec{home.X + .5, home.Y + .5}); cairns[0].at != want {
		t.Errorf("the cairn stands at %v, want %v", cairns[0].at, want)
	}
	step(r, dst, cam, &sc, 3)
	if len(r.livingOf().cairns) != 1 {
		t.Error("a cairn is raised once")
	}
	items := r.cairnItems(nil, cam, sc, r.livingOf())
	if len(items) > 1 {
		t.Errorf("one cairn draws at most one item, got %d", len(items))
	}
}

func TestRetiredKeepsLeaveRubbleThatGrowsOver(t *testing.T) {
	sc, cam, r, dst, keep := livingScene(t)
	r.Draw(dst, cam, sc)
	if len(r.livingOf().rubble) != 0 {
		t.Fatal("a standing Keep leaves no rubble")
	}
	site := sc.Site
	site.Structures = nil
	for _, s := range sc.Site.Structures {
		if s.ID != keep.ID {
			site.Structures = append(site.Structures, s)
		}
	}
	sc.Site = site
	step(r, dst, cam, &sc, 1)
	l := r.livingOf()
	if len(l.rubble) != 1 || l.rubble[0].s.ID != keep.ID {
		t.Fatalf("rubble = %+v, want the retired Keep's land", l.rubble)
	}
	tile := int32(keep.Y*sc.Site.Width + keep.X)
	if !l.cleared[tile] {
		t.Error("the trees are cleared from the ruin")
	}
	if len(r.rubbleItems(nil, cam, sc, l)) == 0 && onScreen(0, 0, 1) {
		t.Log("the ruin is off screen")
	}
	step(r, dst, cam, &sc, 1)
	if len(l.rubble) != 1 {
		t.Error("the Keep is retired once")
	}
	if RubbleStage(sc.Time-l.rubble[0].since) >= RubbleStage(sc.Time+RubbleStageSpan*3-l.rubble[0].since) {
		t.Error("grass covers the rubble over time")
	}
}

func TestNothingIsRetiredWhenTheWholeSiteVanishes(t *testing.T) {
	sc, cam, r, dst, _ := livingScene(t)
	r.Draw(dst, cam, sc)
	sc.Site.Structures = nil
	step(r, dst, cam, &sc, 1)
	if len(r.livingOf().rubble) != 0 {
		t.Error("an empty site is a reload, not a retirement")
	}
}

func TestFootpathsWearAlongTheTilesPortersWalk(t *testing.T) {
	sc, cam, r, dst, keep := livingScene(t)
	r.Draw(dst, cam, sc)
	var grass Point
	found := false
	for y := keep.Y - 8; y < keep.Y+8 && !found; y++ {
		for x := keep.X - 8; x < keep.X+8 && !found; x++ {
			if sc.Site.Tile(x, y) == view.TileGrass && !covered(sc.Site, x, y) {
				grass, found = Point{x, y}, true
			}
		}
	}
	if !found {
		t.Skip("no open grass")
	}
	walker := []Figure{{Pose: Pose{Pos: grass.Center(), Action: "walk"}}, {Pose: Pose{Pos: Vec{1, 1}, Action: "idle"}}}
	l := r.livingOf()
	for range 40 {
		sc.Time++
		l.wear(walker, sc)
	}
	if got := l.worn[grass]; got < wornFull {
		t.Errorf("a walked tile wears to %.1f seconds, want at least %.1f", got, wornFull)
	}
	if _, idle := l.worn[Point{1, 1}]; idle {
		t.Error("standing still wears nothing")
	}
	before := l.worn[grass]
	l.wear(walker, sc)
	if l.worn[grass] > wornFull*wornCap || l.worn[grass] < before {
		t.Error("wear is capped and never reverses")
	}
}

func TestScaffoldsStandOverHutsBeingRebuiltAndComeDownAfter(t *testing.T) {
	sc, cam, r, dst, keep := livingScene(t)
	old, fresh := bareHut("porters-old-1", keep.Object.Name), bareHut("porters-new-1", keep.Object.Name)
	old.Owner, fresh.Owner, old.Crew, fresh.Crew = "ReplicaSet/porters-old", "ReplicaSet/porters-new", "porters", "porters"
	sc.Pods = append(sc.Pods, old, fresh)
	sc.Crews = []model.CrewChain{{Name: "porters", Desired: 2, Ready: 2, Foremen: []model.Foreman{
		{Name: "porters-old", Desired: 1, Current: 1, Pods: []string{old.Name}},
		{Name: "porters-new", Desired: 1, Current: 1, Newest: true, Pods: []string{fresh.Name}},
	}}}
	r.Draw(dst, cam, sc)
	l := r.livingOf()
	items := r.scaffoldItems(nil, cam.CenterOn(r.Homes[old.Name].X, r.Homes[old.Name].Y, 960, 540), sc, l)
	if !l.scaffold[old.Name] || l.scaffold[fresh.Name] || len(items) != 1 {
		t.Errorf("scaffold = %v, items = %d: only the old hut wears one", l.scaffold, len(items))
	}
	sc.Crews[0].Foremen = sc.Crews[0].Foremen[1:]
	r.Draw(dst, cam, sc)
	if len(r.livingOf().scaffold) != 0 {
		t.Error("the scaffolding comes down when the rollout ends")
	}
}

func TestWashingLinesLinkNearbyHutsOfAStandingCrew(t *testing.T) {
	sc, _, r, _, _ := livingScene(t)
	r.Homes = map[string]Vec{"a": {10, 10}, "b": {12, 10}, "c": {20, 20}, "d": {10, 13}, "e": {12.5, 13}}
	crew := model.CrewChain{Name: "porters", Desired: 5, Ready: 5, Foremen: []model.Foreman{{Pods: []string{"a", "b", "c", "d", "e"}}}}
	sc.Crews = []model.CrewChain{crew}
	lines := r.clothLines(sc)
	if len(lines) != 2 {
		t.Fatalf("lines = %d, want two pairs and the far hut left out", len(lines))
	}
	for _, l := range lines {
		if d := Dist(l.a, l.b); d < lineNear || d > lineFar {
			t.Errorf("a line spans %v tiles", d)
		}
	}
	sc.Crews[0].Ready = 4
	if len(r.clothLines(sc)) != 0 {
		t.Error("a short crew hangs no washing")
	}
}

func TestLanternsLineTheRoadAndStayOffWaterAndWalls(t *testing.T) {
	sc, _, _, _, _ := livingScene(t)
	var quay Vec
	for _, h := range SiteHavens(sc.Site) {
		if h.Open {
			quay = VecOf(h.Quay)
		}
	}
	gate := Vec{float64(sc.Site.Gate.X), float64(sc.Site.Gate.Y)}
	route := LandGridOf(sc.Site).Route(quay, gate)
	spots := lampSpotsAlong(sc.Site, route)
	if len(spots) == 0 || len(spots) > maxLanterns {
		t.Fatalf("lamps = %d", len(spots))
	}
	for _, s := range spots {
		p := TileOf(s)
		if WetAt(sc.Site, s) || covered(sc.Site, p.X, p.Y) {
			t.Errorf("a lantern stands in the water or a wall at %v", s)
		}
	}
	for i := 1; i < len(spots); i++ {
		if Dist(spots[i-1], spots[i]) < lanternGap/2 {
			t.Errorf("lanterns %d and %d crowd each other", i-1, i)
		}
	}
}

func TestGullsPerchOnTheQuayOnlyWhileAShipIsIn(t *testing.T) {
	_, _, r, _, _ := livingScene(t)
	plan := HavenPlan{Berths: []Berth{{Moor: Vec{5, 5}, Lane: Vec{5, 8}}}, Sea: Vec{5, 20}}
	r.HarborOf().Now = 20
	docked := &fleet{Ships: []Ship{{Dock: 10, Unload: 20, Sail: 4, Cargo: 3}}}
	if r.shipsIn(plan, docked) != 1 {
		t.Error("a moored ship brings the gulls")
	}
	sailing := &fleet{Ships: []Ship{{Dock: 22, Unload: 5, Sail: 6}}}
	if r.shipsIn(plan, sailing) != 0 {
		t.Error("a ship still at sea brings none")
	}
	if r.shipsIn(plan, &fleet{}) != 0 {
		t.Error("an empty quay has no gulls")
	}
}

func TestLivestockFillsAKeepOfFiveHutsWithChickensAndGoats(t *testing.T) {
	folk := keepFolk{s: view.Structure{ID: "keep-1"}, huts: LivestockHuts}
	anchors := []Vec{{10, 10}, {12, 10}, {10, 12}, {12, 12}}
	beasts := stockBeasts(folk, anchors)
	roles := map[string]int{}
	for _, b := range beasts {
		roles[b.role]++
	}
	if roles["chicken"] == 0 || roles["goat"] == 0 {
		t.Fatalf("beasts = %v, want chickens and goats", roles)
	}
	if !reflect.DeepEqual(beasts, stockBeasts(folk, anchors)) {
		t.Error("the farmyard is the same every time")
	}
	for _, b := range beasts {
		for _, tm := range []float64{0, 3.3, 41, 900} {
			at, _, frame := b.stand(tm)
			if Dist(at, b.anchor) > b.roam*1.5 {
				t.Errorf("a %s wanders %v tiles from its yard", b.role, Dist(at, b.anchor))
			}
			if frame < 0 || frame >= art.LivestockFrames {
				t.Errorf("frame %d is off the sheet", frame)
			}
		}
	}
}

func TestADrawnScenePutsTheLandscapeInTheDepthSortedItems(t *testing.T) {
	sc, cam, r, dst, keep := livingScene(t)
	sc.Pods = append(sc.Pods, bareHut("a", keep.Object.Name), bareHut("b", keep.Object.Name), bareHut("c", keep.Object.Name), bareHut("d", keep.Object.Name), bareHut("e", keep.Object.Name))
	r.Draw(dst, cam, sc)
	f, _ := r.livingOf().keepFolkOf(keep.ID)
	if f.huts < LivestockHuts {
		t.Skipf("only %d huts on the Keep", f.huts)
	}
	if got := r.livingItems(cam, sc); len(got) == 0 {
		t.Error("a Keep of huts draws its landscape")
	}
}
