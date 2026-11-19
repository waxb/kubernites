package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func drawAll(items []Drawable) {
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	for _, it := range items {
		it.Draw(dst)
	}
}

func frictionCam(site view.Site) render.Camera {
	k, _ := FirstOfKind(site, view.StructKeep, false)
	return render.NewCamera(480, 270).CenterOn(float64(k.X), float64(k.Y)+4, 960, 540)
}

func TestWardStandsAtAWardedKeepUntilTheStewardLiftsIt(t *testing.T) {
	site := lifeSite()
	r := NewRenderer(testLib())
	cam := frictionCam(site)
	k, _ := FirstOfKind(site, view.StructKeep, false)
	sc := Scene{Site: site, Time: 10, Frictions: []Friction{{Kind: FrictionWarded, Object: NodeObject(k.Object.Name)}}}
	var sets frictionSets
	var c census
	sets.fill(sc, c.of(sc))
	if n := len(r.wardItems(nil, cam, sc, sets.warded)); n != 1 {
		t.Fatalf("ward signs = %d", n)
	}
	sc.Frictions = nil
	if n := len(r.wardItems(nil, cam, sc, nil)); n != 0 {
		t.Fatalf("no ward, no sign: %d", n)
	}
	spot := WardSpot(site, k)
	e := Errand{Kind: ErrandUnward, Role: "steward", From: Vec{spot.X - 4, spot.Y}, To: Vec{spot.X + .35, spot.Y + .35}, Start: 0}
	e.Life = ErrandLife(e.Kind, e.From, e.To)
	walk := (e.Life - unwardAct) / 2
	sc.Errands = []Errand{e}
	sc.Time = walk / 2
	if lift, ok := WardLift(sc, spot); !ok || lift != 0 {
		t.Errorf("the ward stands while the Steward walks over: %v %v", lift, ok)
	}
	sc.Time = walk + unwardAct/2
	if lift, ok := WardLift(sc, spot); !ok || lift <= 0 || lift >= 1 {
		t.Errorf("the Steward lifts the ward: %v %v", lift, ok)
	}
	items := r.wardItems(nil, cam, sc, nil)
	if len(items) != 1 {
		t.Fatalf("the lifting ward is drawn: %d", len(items))
	}
	drawAll(items)
	sc.Time = e.Life - .1
	if _, ok := WardLift(sc, spot); ok {
		t.Error("once lifted the ward is gone")
	}
}

func TestFrictionItemsDrawTheProblems(t *testing.T) {
	site := lifeSite()
	site.Structures = append(site.Structures, view.Structure{ID: "a", Kind: view.StructArchive, X: 12, Y: 12, W: 3, H: 3, State: view.StateReady})
	r := NewRenderer(testLib())
	r.Homes = map[string]Vec{"idle": {12, 8}, "camp": {7, 18}}
	sc := Scene{Site: site, Time: 5, Queue: 3, ArrivalRate: 1, Ledger: Ledger{Open: true, Share: .95},
		Pods: []model.PodView{
			{Namespace: "default", Name: "idle", State: model.HearthReady, Sprite: "porter", Tier: 1},
			{Namespace: "default", Name: "camp", State: model.HearthPending, Sprite: "porter", Tier: 1},
		},
		Metrics: view.MetricsSnapshot{Pods: map[string]view.PodMetrics{"default/idle": {}}},
		Errands: []Errand{{Kind: ErrandDecree, Role: "scribe", From: Vec{8, 8}, To: Vec{14, 8}, Start: 4.5, Life: 6}},
	}
	cam := frictionCam(site)
	items := r.frictionItems(nil, cam, sc)
	if len(items) != 3 {
		t.Fatalf("friction drawables = %d, want homeless, notice scroll and scroll pile, and no idle bubble", len(items))
	}
	drawAll(items)
	if _, ok := r.Camp("camp"); !ok {
		t.Error("the homeless hut's camp is remembered")
	}
	if _, ok := r.Camp("idle"); ok {
		t.Error("only Pending huts camp")
	}
}

func TestRuinsStayWhileNobodyReplacesTheFallen(t *testing.T) {
	site := lifeSite()
	r := NewRenderer(testLib())
	cam := frictionCam(site)
	fallen := Errand{Kind: ErrandFallen, Pod: "gone", To: Vec{12.45, 8.8}, Start: 1, Life: FallLife, Tier: 2}
	sc := Scene{Site: site, Time: 2, Errands: []Errand{fallen}}
	if n := len(r.ruinItems(nil, cam, sc, false)); n != 0 {
		t.Errorf("replaced huts leave no ruin: %d", n)
	}
	sc.Time = 30
	sc.Errands = nil
	items := r.ruinItems(nil, cam, sc, true)
	if len(items) != 1 {
		t.Fatalf("the ruin stays while unreplaced: %d", len(items))
	}
	drawAll(items)
	r.Homes = map[string]Vec{"new": {12, 8}}
	if n := len(r.ruinItems(nil, cam, sc, true)); n != 0 {
		t.Errorf("a new hut clears the ruin: %d", n)
	}
}

func TestKindlingHearthsWalkFromTheirCampOrMarchFromTheHorn(t *testing.T) {
	site := lifeSite()
	site.Structures = append(site.Structures, view.Structure{ID: "h", Kind: view.StructWarHorn, X: 12, Y: 12, W: 2, H: 2, State: view.StateReady})
	r := NewRenderer(testLib())
	cam := frictionCam(site)
	camper := model.PodView{Namespace: "default", Name: "camper", State: model.HearthKindling, Sprite: "porter", Tier: 1}
	levy := model.PodView{Namespace: "default", Name: "levy", State: model.HearthKindling, Sprite: "porter", Tier: 1, Owner: "ReplicaSet/porters"}
	loner := model.PodView{Namespace: "default", Name: "loner", State: model.HearthKindling, Sprite: "porter", Tier: 1}
	r.lifeOf().camps["camper"] = camp{at: Vec{7, 18}, seen: 0}
	clerk := Errand{Kind: ErrandSchedule, Role: "quartermaster", Pod: "camper", From: Vec{14, 14}, To: Vec{7.5, 17.8}, Start: 0}
	clerk.Life = ErrandLife(clerk.Kind, clerk.From, clerk.To)
	sc := Scene{Site: site, Time: 1, Errands: []Errand{clerk}}
	seen := map[string]float64{}
	if at, ok := clerkArrival(sc, "camper"); !ok || at <= clerk.Start {
		t.Fatalf("clerk arrival %v %v", at, ok)
	}
	for _, p := range []model.PodView{camper, levy} {
		items, ok := r.arrivalItems(cam, sc, p, Vec{16, 12}, r.kindleElapsed(p.Name, sc.Time, seen), 0)
		if !ok || len(items) != 1 {
			t.Fatalf("%s arrives on foot: %v %d", p.Name, ok, len(items))
		}
		drawAll(items)
	}
	if _, ok := r.arrivalItems(cam, sc, loner, Vec{16, 12}, r.kindleElapsed(loner.Name, sc.Time, seen), 0); ok {
		t.Error("a hut that never camped still sails in")
	}
	if len(r.voyageItems(cam, sc, camper, Vec{16, 12}, seen)) != 1 {
		t.Error("the voyage hands over to the camp walk")
	}
	pair := camper
	pair.People = []model.Kubernite{{Name: "porter", Sprite: "porter"}, {Name: "loader", Sprite: "porter"}}
	if got := r.voyageItems(cam, sc, pair, Vec{16, 12}, seen); len(got) != 2 {
		t.Errorf("both people of a household walk in: %d", len(got))
	}
}

func TestFanLinesDrawOnTheGroundAfterTheFirstAnswer(t *testing.T) {
	site := lifeSite()
	r := NewRenderer(testLib())
	cam := frictionCam(site)
	r.Homes = map[string]Vec{"a": {12, 10}, "b": {14, 10}}
	pods := []model.PodView{{Namespace: "default", Name: "a", State: model.HearthReady}, {Namespace: "default", Name: "b", State: model.HearthReady}}
	sc := Scene{Site: site, Pods: pods, Time: 1, Metrics: view.MetricsSnapshot{Pods: map[string]view.PodMetrics{}}}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.drawLifeGround(dst, cam, sc)
	sc.Time = 2
	sc.Metrics.Pods = map[string]view.PodMetrics{"default/a": {ServedRate: .4}, "default/b": {ServedRate: .2}}
	r.drawLifeGround(dst, cam, sc)
	if !r.lifeOf().fan.visible("a", 3) || len(r.lifeOf().fanNow) != 2 {
		t.Fatal("the first answer lights the paths")
	}
	sc.Reduce = true
	r.drawLifeGround(dst, cam, sc)
}
