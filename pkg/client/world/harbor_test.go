package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func tieredSite(t testing.TB, tier uint8) (Scene, render.Camera) {
	sc, _ := harborScene(t)
	base := SiteHavens(sc.Site)
	if len(base) == 0 {
		t.Fatal("no haven on the lively realm")
	}
	h := base[0]
	h.Tier = tier
	closed := h
	closed.ID, closed.Open, closed.Quay.Y = "haven-9", false, h.Quay.Y+12
	sc.Site.Havens = []view.Haven{h, closed}
	sc.ArrivalRate, sc.Queue = 3, 12
	sc.Freight = synthFreight(sc.RealmTime, max(int(tier), 1), 5)
	r := NewRenderer(nil)
	return sc, cameraOn(sc, r)
}

func TestHarborDrawsEveryTierWithAndWithoutArt(t *testing.T) {
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	for _, lib := range []*gfx.Library{nil, gfx.Load(art.Build())} {
		for tier := uint8(0); tier <= 3; tier++ {
			sc, cam := tieredSite(t, tier)
			r := NewRenderer(lib)
			for i := 0; i < 90; i++ {
				sc.Time += .1
				sc.RealmTime += .1
				sc.Freight = synthFreight(sc.RealmTime, max(int(tier), 1), 5)
				sc.Beckon, sc.Hint = true, i%2 == 0
				sc.Hover = PileID(i % 4)
				r.Draw(dst, cam, sc)
			}
			h := r.HarborOf()
			if len(h.Plans) != 2 || len(h.Fleets) != 2 {
				t.Fatalf("plans %d fleets %d", len(h.Plans), len(h.Fleets))
			}
			if len(h.Fleets[0].Ships) == 0 {
				t.Errorf("tier %d: no ships at an open haven", tier)
			}
			if len(h.Fleets[1].Ships) != 0 {
				t.Errorf("a closed haven has ships")
			}
			if len(h.Yards) != len(FreightKinds) {
				t.Errorf("tier %d: %d yards on the quay", tier, len(h.Yards))
			}
			for i := 0; i < 120; i++ {
				sc.Time += .1
				r.Draw(dst, cam, sc)
			}
			if _, _, ok := r.FrontBox(cam, sc.Site); !ok {
				t.Errorf("tier %d: no parcel to carry once the quay settles", tier)
			}
			sc.Site.Havens[0].TierAt, sc.RealmTime = 50, 51
			sc.Time += .1
			r.Draw(dst, cam, sc)
			sc.Reduce, sc.Time = true, sc.Time+.1
			r.Draw(dst, cam, sc)
			far := cam
			far.Zoom = .5
			r.Draw(dst, far, sc)
		}
	}
}

func TestHarborSurvivesAnEmptySite(t *testing.T) {
	r := NewRenderer(nil)
	dst := ebiten.NewImage(64, 64)
	defer dst.Deallocate()
	r.Draw(dst, render.NewCamera(0, 0), Scene{Queue: 3, ArrivalRate: 1, Time: 4, Freight: synthFreight(4, 1, 3)})
	if _, _, ok := r.FrontBox(render.NewCamera(0, 0), view.Site{}); ok {
		t.Error("a parcel to carry with no harbor")
	}
	if _, ok := HarborFocus(view.Site{}); ok {
		t.Error("a harbor focus on an empty site")
	}
}

func TestWeightsFollowShareOverOpenHavens(t *testing.T) {
	cases := []struct {
		plans []HavenPlan
		want  []float64
	}{
		{[]HavenPlan{{Open: true, Share: .7}, {Open: true, Share: .3}}, []float64{.7, .3}},
		{[]HavenPlan{{Open: true}, {Open: false, Share: 5}, {Open: true}}, []float64{.5, 0, .5}},
		{[]HavenPlan{{Open: false}}, []float64{0}},
	}
	for _, c := range cases {
		got := Weights(c.plans)
		for i := range c.want {
			if got[i] != c.want[i] {
				t.Errorf("weights %+v = %v, want %v", c.plans, got, c.want)
				break
			}
		}
	}
}

func hamletHomes(site view.Site, pods []model.PodView) map[string]Vec {
	h := NewHamlet()
	h.Place(site, pods, 0)
	return h.Homes
}

func TestADoorPorterBoxedInByTheCitadelStillHasAHome(t *testing.T) {
	sc, _ := sceneFor(t, .5)
	pods := []model.PodView{
		{Namespace: "default", Name: "porter-lone-tower", Node: view.TowerName, Static: true},
		{Namespace: "default", Name: "a", Node: view.TowerName},
	}
	homes := hamletHomes(sc.Site, pods)
	door, ok := homes["porter-lone-tower"]
	if !ok || door == homes["a"] {
		t.Fatalf("homes %v", homes)
	}
	tile := Point{int(door.X), int(door.Y)}
	if Occupied(sc.Site, tile.X, tile.Y) || !Homeable(sc.Site.Tile(tile.X, tile.Y)) {
		t.Errorf("the door porter lives on %v", tile)
	}
}

func TestDoorPorterCarriesParcelsToThePorch(t *testing.T) {
	sc, _ := sceneFor(t, .5)
	sc.Site.Structures = withoutKind(sc.Site.Structures, view.StructGuildhall)
	sc.Freight = synthFreight(20, 1, 6)
	door := model.PodView{Namespace: "default", Name: "porter-lone-tower", Node: view.TowerName, Static: true, Sprite: "porter", State: model.HearthReady}
	sc.Pods = []model.PodView{door}
	sc.ArrivalRate = .3
	r := NewRenderer(nil)
	r.HarborOf().Replan(sc.Site)
	r.Homes = map[string]Vec{door.Name: {40, 60}}
	keep, _ := KeepForNode(sc.Site, view.TowerName)
	h, ok := r.HaulOf(sc, door)
	if !ok || h.Drop != Porch(keep) || h.Kind != 0 {
		t.Fatalf("door haul %+v %v", h, ok)
	}
	sc.Metrics.Pods[PodKey(door)] = view.PodMetrics{ServedRate: .25, Throttled: true}
	carried := false
	for i := 0; i < 400 && !carried; i++ {
		sc.Time += .25
		f, ok := r.FigureFor(sc, door, RouteOf(sc.Site))
		if !ok {
			t.Fatal("no door porter figure")
		}
		carried = f.Load != nil && f.Load.Kind == "parcel" && f.Pose.Carry
		if f.Pose.Carry != (f.Load != nil) {
			t.Fatalf("a carry pose without a box: %+v", f)
		}
	}
	if !carried {
		t.Error("the door porter never carries a parcel")
	}
	if _, ok := r.DoorFigure(sc, model.PodView{Node: "nowhere"}); ok {
		t.Error("a door worker without a Keep")
	}
	idle := door
	idle.Static = false
	if _, ok := r.HaulOf(sc, idle); ok {
		t.Error("a porter with no depot and no door hauls")
	}
}

func withoutKind(ss []view.Structure, kind string) []view.Structure {
	out := []view.Structure{}
	for _, s := range ss {
		if s.Kind != kind {
			out = append(out, s)
		}
	}
	return out
}

func TestHandPipsShowPledgedAndUsedHands(t *testing.T) {
	cases := []struct {
		m    view.NodeMetrics
		want []Pip
	}{
		{view.NodeMetrics{}, nil},
		{view.NodeMetrics{CPUAllocatableMilli: 2000, CPURequestedMilli: 1500, CPUMilli: 400}, []Pip{{1, 1, .4}, {1, .5, 0}}},
		{view.NodeMetrics{CPUAllocatableMilli: 1900, CPURequestedMilli: 1900, CPUMilli: 1900}, []Pip{{1, 1, 1}, {.9, .9, .9}}},
		{view.NodeMetrics{CPUAllocatableMilli: 3000, CPURequestedMilli: 0, CPUMilli: 5000}, []Pip{{1, 0, 1}, {1, 0, 1}, {1, 0, 1}}},
	}
	for _, c := range cases {
		got := HandPips(c.m)
		if len(got) != len(c.want) {
			t.Fatalf("%+v: %d pips, want %d", c.m, len(got), len(c.want))
		}
		for i := range got {
			if !near(got[i].Cap, c.want[i].Cap) || !near(got[i].Pledged, c.want[i].Pledged) || !near(got[i].Used, c.want[i].Used) {
				t.Errorf("%+v pip %d = %+v, want %+v", c.m, i, got[i], c.want[i])
			}
		}
	}
	if PipSize(.5) != 5 || PipSize(1) != 7 || PipSize(2) != 11 {
		t.Error("pip size")
	}
}

func near(a, b float64) bool { return a-b < 1e-9 && b-a < 1e-9 }

func TestKeepPipsWaitForTheChancery(t *testing.T) {
	sc, _ := sceneFor(t, .5)
	sc.Metrics.Nodes = map[string]view.NodeMetrics{view.TowerName: {CPUAllocatableMilli: 1900}}
	if !PipsShown(sc) {
		t.Fatal("a bootstrapped realm shows its Keeps' Hands")
	}
	sc.Site.Structures = withoutKind(sc.Site.Structures, view.StructChancery)
	if PipsShown(sc) {
		t.Error("Hands shown before the Chancery answers")
	}
	sc, _ = sceneFor(t, .5)
	sc.Metrics.Nodes = nil
	if PipsShown(sc) {
		t.Error("Hands shown without metrics")
	}
}

func TestKeepPipsDrawOnlyWithMetrics(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	if k, ok := FirstOfKind(sc.Site, view.StructKeep, false); ok {
		cam = cam.CenterOn(float64(k.X), float64(k.Y), 960, 540)
	}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	for _, lib := range []*gfx.Library{nil, gfx.Load(art.Build())} {
		r := NewRenderer(lib)
		sc.Metrics.Nodes = nil
		r.DrawKeepPips(dst, cam, sc)
		sc.Metrics.Nodes = map[string]view.NodeMetrics{view.TowerName: {CPUAllocatableMilli: 12000, CPURequestedMilli: 5500, CPUMilli: 3000}}
		r.DrawKeepPips(dst, cam, sc)
		sc.Metrics.Augury = true
		r.DrawKeepPips(dst, cam, sc)
	}
}

func TestShipsAtAHeadBerthPullStraightOut(t *testing.T) {
	p := HavenPlan{Quay: Vec{10, 10}, Out: Vec{-1, 0}, Side: Vec{0, 1}, Head: Vec{7, 10}, Pier: []Vec{{9, 10}, {8, 10}, {7, 10}}}
	b := BerthAt(p, 0, 6)
	if b.Plank.X >= p.Head.X || b.Moor.X >= b.Plank.X || b.Lane.Y != 10 || b.Lane.X >= 3 {
		t.Fatalf("head berth %+v", b)
	}
	flank := BerthAt(p, 1, 6)
	if flank.Moor != (Vec{7, 11}) || flank.Lane.Y != 11 || flank.Lane.X >= 3 {
		t.Errorf("flank berth %+v", flank)
	}
	s := Ship{Class: "cog", Dock: 10, Unload: 4, Wait: 2, Sail: SailSeconds}
	pose, ok := ShipAt(s, b, Vec{0, 10}, s.Leave()+1)
	if !ok || pose.Pos.X >= b.Moor.X || pose.Dir != "nw" {
		t.Errorf("leaving from the head berth: %+v", pose)
	}
	if got := AwayPath(Berth{Moor: Vec{1, 1}, Plank: Vec{1, 1}, Lane: Vec{0, 1}}, Vec{-5, 1}); len(got) != 3 || got[1] != (Vec{0, 1}) {
		t.Errorf("a ship leaves by its lane: %v", got)
	}
	if pos, _ := PathAt([]Vec{{1, 1}, {1, 1}}, .5); pos != (Vec{1, 1}) {
		t.Errorf("a zero-length path stays put: %v", pos)
	}
}

func TestShipHeadingsRoundTrip(t *testing.T) {
	for _, d := range []string{"se", "sw", "ne", "nw"} {
		if ShipHeading(HeadingVec(d)) != d {
			t.Errorf("heading round trip %s", d)
		}
	}
}

func TestHarborKeepsDecorOffTheQuay(t *testing.T) {
	site := coastSite(40, 32, 8)
	site.Havens = []view.Haven{westHaven(3, true)}
	hb := BuildHabitat(site, LandOf(site))
	p := PlanHaven(site, site.Havens[0])
	for _, d := range p.Deck {
		if hb.BlockAt(d.X, d.Y) != BlockFootprint {
			t.Errorf("decor may grow on the deck at %v", d)
		}
	}
	for _, v := range p.Pier {
		if tt := TileOf(v); hb.BlockAt(tt.X, tt.Y) != BlockFootprint {
			t.Errorf("reeds may grow through the pier at %v", tt)
		}
	}
	a, b := StructureSig(site), StructureSig(view.Site{Havens: []view.Haven{westHaven(2, true)}})
	if a == b {
		t.Error("the habitat does not notice haven changes")
	}
}

func TestHarborGuardsAndSignatures(t *testing.T) {
	h := &Harbor{}
	if p := h.Plan(3); p.ID != "" {
		t.Errorf("an empty harbor plans %+v", p)
	}
	a := view.Haven{ID: "h", Tier: 1, Pier: 2005}
	b := view.Haven{ID: "h", Tier: 3, Pier: 5}
	if PlanSig(view.Site{}, []view.Haven{a}) == PlanSig(view.Site{}, []view.Haven{b}) {
		t.Error("distinct havens share a signature")
	}
}
