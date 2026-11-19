package world

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
)

func TestHutsRiseFromTheGroundAndFadeAway(t *testing.T) {
	if RiseOf(neverBorn, 3) != 1 || RiseOf(2, 2) != 0 || math.Abs(RiseOf(2, 2+RiseSeconds/2)-.5) > 1e-9 {
		t.Error("a hut rises over riseSeconds and old huts stand whole")
	}
	if FadeOf(2, 2) != 1 || FadeOf(2, 2+FadeSeconds) != 0 || FadeOf(2, 1) != 1 {
		t.Error("a ruin fades over fadeSeconds")
	}
}

func lifecycleScene(t *testing.T) (Scene, render.Camera, *Renderer, *ebiten.Image) {
	t.Helper()
	sc, cam, r := peopleScene(t, 0, householdView("porters-7d9f-first", "lone-tower", porterFolk("porter")))
	dst := ebiten.NewImage(960, 540)
	t.Cleanup(dst.Deallocate)
	return sc, cam, r, dst
}

func TestANewHutRisesWithDustAndAGoneHutLeavesFadingRubble(t *testing.T) {
	sc, cam, r, dst := lifecycleScene(t)
	fresh := householdView("porters-7d9f-fresh", "lone-tower", porterFolk("porter"))
	sc.Pods = append(sc.Pods, fresh)
	sc.Time += .1
	r.Draw(dst, cam, sc)
	spot, ok := r.hamlet.SpotList[fresh.Name]
	if !ok || spot.Born != sc.Time {
		t.Fatalf("the new hut was born at %v, want %v (%v)", spot.Born, sc.Time, ok)
	}
	home := r.Homes[fresh.Name]
	cam = cam.CenterOn(home.X, home.Y, 960, 540)
	if _, dusty := r.riseDust(cam, sc, Vec{home.X + .5, home.Y + .5}, RiseOf(spot.Born, sc.Time)); !dusty {
		t.Error("a rising hut kicks up dust")
	}
	if _, dusty := r.riseDust(cam, sc, Vec{home.X + .5, home.Y + .5}, 1); dusty {
		t.Error("a standing hut raises no dust")
	}
	for _, it := range r.HearthItems(cam, sc) {
		it.Draw(dst)
	}
	sc.Pods = sc.Pods[:len(sc.Pods)-1]
	sc.Time += .1
	r.Draw(dst, cam, sc)
	if len(r.hamlet.gone) != 1 || r.hamlet.gone[0].spot.At != spot.At {
		t.Fatalf("gone huts = %+v", r.hamlet.gone)
	}
	rubble := r.hamletItems(nil, cam, sc, r.hamlet)
	if len(rubble) == 0 {
		t.Fatal("the gone hut leaves rubble")
	}
	for _, it := range rubble {
		it.Draw(dst)
	}
	sc.Time += FadeSeconds + .1
	r.Draw(dst, cam, sc)
	if len(r.hamlet.gone) != 0 || r.hamlet.occAt(spot.At) != occFree {
		t.Error("the rubble clears and frees its land")
	}
}

func TestGardensLieBesideHutsWhenCloseEnough(t *testing.T) {
	sc, cam, r, dst := lifecycleScene(t)
	sown := ""
	for name, s := range r.hamlet.SpotList {
		if s.Sown {
			sown = name
		}
	}
	if sown == "" {
		t.Fatal("no hut has a garden")
	}
	home := r.Homes[sown]
	cam = cam.CenterOn(home.X, home.Y, 960, 540)
	if got := r.gardenItems(nil, cam, sc, r.hamlet); len(got) != 0 {
		t.Fatalf("huts without a Housing Binding keep no garden: %d", len(got))
	}
	r.livingOf().bound = map[string]bool{sown: true}
	near := r.gardenItems(nil, cam, sc, r.hamlet)
	if len(near) == 0 {
		t.Fatal("no garden drawn near its hut")
	}
	for _, it := range near {
		it.Draw(dst)
	}
	far := cam
	far.Zoom = .5
	if got := r.hamletItems(nil, far, sc, r.hamlet); len(got) != 0 {
		t.Errorf("gardens hide when zoomed far out: %d", len(got))
	}
}

func TestGardenBedsArePaintedSoilWithSprouts(t *testing.T) {
	for k := -1; k <= gardenKinds; k++ {
		img := PaintGarden(k)
		b := img.Bounds()
		if b.Dx() != gardenW || b.Dy() != gardenH+gardenSprout {
			t.Fatalf("garden %d is %v", k, b)
		}
		top := img.NRGBAAt(gardenW/2, gardenSprout+1)
		if top.A == 0 {
			t.Errorf("garden %d has an empty middle", k)
		}
	}
	a, b := PaintGarden(0), PaintGarden(1)
	if string(a.Pix) == string(b.Pix) || a.Pix[3] != 0 {
		t.Error("garden kinds differ and leave the corners clear")
	}
	if !tallScenery([]scenery{{name: "decor/tree-oak/0"}}) || tallScenery([]scenery{{name: "decor/flowers/1"}, {name: "decor/rock/0", tiny: true}}) {
		t.Error("only tall scenery keeps huts away")
	}
}

func TestHutPlacementFollowsTheLandCover(t *testing.T) {
	sc, cam, r, dst := lifecycleScene(t)
	r.Draw(dst, cam, sc)
	h := r.hamletOf()
	if h.cover == nil || h.covSig == 0 {
		t.Fatal("the hamlet knows the land cover once the land is painted")
	}
	bare := NewRenderer(nil)
	bh := bare.hamletOf()
	bare.coverHamlet(bh, sc.Site)
	if bh.cover != nil {
		t.Error("without painted land nothing covers the ground")
	}
	if crewKey(model.PodView{Owner: "ReplicaSet/x"}) != "ReplicaSet/x" || crewKey(model.PodView{Depot: "parcels"}) != "parcels" {
		t.Error("a crew is known by its Deployment, owner or depot")
	}
}
