package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func bustlingHarbor(t testing.TB) (Scene, render.Camera, *Renderer) {
	sc, _ := sceneFor(t, .5)
	sc.Pods, sc.Ghost = nil, nil
	havens := append([]view.Haven(nil), sc.Site.Havens...)
	for i := range havens {
		havens[i].Open, havens[i].Tier, havens[i].Share = true, 2, 1/float64(len(havens))
	}
	sc.Site.Havens = havens
	sc.ArrivalRate, sc.Queue = 6, 20
	sc.RealmTime = sc.Time
	sc.Freight = synthFreight(sc.RealmTime, 2, 6)
	r := NewRenderer(testLib())
	r.HarborOf().Replan(sc.Site)
	p := r.HarborOf().Plans[0]
	cam := render.NewCamera(480, 60)
	return sc, cam.CenterOn(p.Quay.X, p.Quay.Y, 960, 540), r
}

func TestTheLifeLayerDrawsAtEveryHourAndZoom(t *testing.T) {
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	sc, cam := sceneFor(t, .5)
	sc.Metrics.Nodes = map[string]view.NodeMetrics{"lone-tower": {CPUMilli: 900, CPUAllocatableMilli: 1900}}
	if k, ok := FirstOfKind(sc.Site, view.StructKeep, false); ok {
		cam = cam.CenterOn(float64(k.X), float64(k.Y), 960, 540)
	}
	r := NewRenderer(testLib())
	for _, c := range []struct{ phase, zoom float64 }{{.5, .5}, {fogPeak, 1}, {.95, 2}} {
		sc.DayPhase, cam.Zoom = c.phase, c.zoom
		for i := 0; i < 2; i++ {
			sc.Time += 7
			sc.Served += 30
			r.Draw(dst, cam, sc)
		}
	}
	sc.Reduce = true
	r.Draw(dst, cam, sc)
}

func TestTownsfolkThinOutAtNight(t *testing.T) {
	if VisibleFolk(12, true) >= VisibleFolk(12, false) || VisibleFolk(0, true) != 0 {
		t.Error("most townsfolk go home at night")
	}
}

func TestHarborBustleSwingsTheCraneAndSendsOutBoats(t *testing.T) {
	sc, cam, r := bustlingHarbor(t)
	l := r.syncLife(sc)
	h := r.HarborOf()
	seenDock, seenBoat := false, false
	for i := 0; i < 400 && !(seenDock && seenBoat); i++ {
		sc.Time += .25
		sc.RealmTime += .25
		sc.Freight = synthFreight(sc.RealmTime, 2, 6)
		h.Step(sc)
		seenDock = seenDock || len(r.dockItems(nil, cam, sc)) > 0
		seenBoat = seenBoat || len(r.boatItems(nil, cam, sc, l)) > 0
	}
	if !seenDock {
		t.Error("the crane works the quay while ships unload")
	}
	if !seenBoat {
		t.Error("fishing boats work the water off the harbor")
	}
}

func TestAConstructionSiteKeepsItsCratePile(t *testing.T) {
	site := lifeSite()
	site.Structures = append(site.Structures, view.Structure{ID: "codex-1", Kind: view.StructCodex, X: 12, Y: 8, W: 2, H: 2, State: view.StateConstruction})
	r := NewRenderer(testLib())
	cam := frictionCam(site)
	items := r.haulerItems(nil, cam, Scene{Site: site, Time: 3})
	if len(items) != 1 {
		t.Fatalf("a crate pile per site and no hauler, got %d", len(items))
	}
	drawAll(items)
	if p := PileSpot(site.Structures[3]); Occupied(site, int(p.X), int(p.Y)) {
		t.Errorf("the pile stands inside a footprint at %v", p)
	}
}
