package world

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func livelyRealm(t testing.TB) (view.Site, []model.PodView) {
	t.Helper()
	var site view.Site
	var pods []model.PodView
	for path, into := range map[string]any{"testdata/lively-site.json": &site, "testdata/lively-pods.json": &pods} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, into); err != nil {
			t.Fatal(err)
		}
	}
	return site, pods
}

func sceneFor(t testing.TB, phase float64) (Scene, render.Camera) {
	t.Helper()
	site, pods := livelyRealm(t)
	pods = append(pods,
		model.PodView{Namespace: "default", Name: "gutter", Node: "lone-tower", Sprite: "porter", Tier: 2, State: model.HearthGuttering},
		model.PodView{Namespace: "default", Name: "gone", Sprite: "porter", Tier: 3, State: model.HearthBanished, Terminating: true},
		model.PodView{Namespace: "default", Name: "kindle", Node: "lone-tower", Sprite: "porter", Tier: 1, State: model.HearthKindling},
	)
	sel := ""
	if len(site.Structures) > 0 {
		sel = site.Structures[0].ID
	}
	sc := Scene{
		Site: site, Pods: pods, ArrivalRate: 1, Queue: 4, DayPhase: phase, Time: 12.5, Selected: sel,
		Ghost:      &Ghost{Kind: view.StructKeep, X: 10, Y: 10, W: 3, H: 3, Valid: false},
		Transients: []Transient{{Kind: "imp", At: Vec{20, 20}, Start: 12, Life: 2}, {Kind: "dust", At: Vec{21, 20}, Start: 12, Life: 1}, {Kind: "void", Start: 0, Life: 1}},
		Metrics:    view.MetricsSnapshot{Pods: map[string]view.PodMetrics{}},
	}
	cam := render.NewCamera(480, 60).CenterOn(float64(site.Gate.X), float64(site.Gate.Y), 960, 540)
	return sc, cam
}

func TestRendererDrawsAndPicks(t *testing.T) {
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	for _, lib := range []*gfx.Library{nil, gfx.Load(art.Build())} {
		for _, phase := range []float64{.5, .95} {
			sc, cam := sceneFor(t, phase)
			if k, ok := FirstOfKind(sc.Site, view.StructKeep, false); ok {
				cam = cam.CenterOn(float64(k.X), float64(k.Y), 960, 540)
			}
			r := NewRenderer(lib)
			r.Draw(dst, cam, sc)
			r.DrawLamps(dst, cam, sc)
			if len(r.HitList) == 0 {
				t.Fatal("no hit targets recorded")
			}
			h := r.HitList[len(r.HitList)-1]
			got, ok := r.Pick(h.Rect.X+h.Rect.W/2, h.Rect.Y+h.Rect.H/2)
			if !ok || got.ID == "" {
				t.Errorf("pick failed for %+v", h)
			}
			if _, ok := r.Pick(-500, -500); ok {
				t.Error("picked nothing")
			}
			if _, ok := r.Home("gutter"); !ok {
				t.Error("guttering hearth has no home")
			}
		}
	}
	r := NewRenderer(nil)
	r.Draw(dst, render.NewCamera(0, 0), Scene{})
}

func TestMinimapAndHelpers(t *testing.T) {
	sc, _ := sceneFor(t, .5)
	img := MinimapImage(sc.Site, 1)
	if w, h := img.Bounds().Dx(), img.Bounds().Dy(); w != sc.Site.Width+sc.Site.Height+1 || h < sc.Site.Height {
		t.Errorf("minimap size %dx%d", w, h)
	}
	if MinimapImage(view.Site{}, 1).Bounds().Dx() != 1 {
		t.Error("empty minimap")
	}
	if TileColor(view.TileKind(250)) != TileColor(view.TileGrass) {
		t.Error("unknown tile color falls back to grass")
	}
	if ComponentStructure(model.PodView{Name: "kube-apiserver-lone-tower"}) != view.StructChancery || ComponentStructure(model.PodView{Name: "x"}) != "" {
		t.Error("ComponentStructure mismatch")
	}
	if Darkness(.5) != 0 || Darkness(0) < .5 {
		t.Errorf("darkness noon %v midnight %v", Darkness(.5), Darkness(0))
	}
	g := GhostStructure(sc.Site, Ghost{Kind: view.StructKeep, X: 1, Y: 2})
	if g.Variant != "stone" || g.W != 1 {
		t.Errorf("ghost keep = %+v", g)
	}
	if GhostStructure(view.Site{}, Ghost{Kind: view.StructKeep}).Variant != "tower" {
		t.Error("first keep ghost is the tower")
	}
	if Itoa(0) != "0" || Itoa(-42) != "-42" || Itoa(1234) != "1234" {
		t.Error("itoa")
	}
}

func TestVoyage(t *testing.T) {
	sc, _ := sceneFor(t, .5)
	home := Vec{20, 20}
	landing, ok := EastLanding(sc.Site, 20)
	if !ok || landing.X <= home.X {
		t.Fatalf("landing = %v %v", landing, ok)
	}
	if v := VoyageAt(sc.Site, home, 1); !v.Sailing {
		t.Error("early voyage sails")
	}
	if v := VoyageAt(sc.Site, home, VoyageSeconds*.8); v.Sailing || v.Walker.Action != "walk" {
		t.Errorf("late voyage walks: %+v", v)
	}
	if v := VoyageAt(sc.Site, home, VoyageSeconds+1); !v.Arrived {
		t.Error("voyage arrives")
	}
	water := view.Site{Width: 2, Height: 2, Tiles: []view.TileKind{view.TileWater, view.TileWater, view.TileWater, view.TileWater}}
	if _, ok := EastLanding(water, 0); ok {
		t.Error("no landing on open sea")
	}
}

func TestOccluders(t *testing.T) {
	back := Placed{S: view.Structure{ID: "b", State: view.StateConstruction}, Rect: ui.Rect{X: 0, Y: 0, W: 10, H: 10}, Depth: 1}
	front := Placed{S: view.Structure{ID: "f", State: view.StateReady}, Rect: ui.Rect{X: 5, Y: 5, W: 10, H: 10}, Depth: 2}
	far := Placed{S: view.Structure{ID: "x", State: view.StateReady}, Rect: ui.Rect{X: 50, Y: 50, W: 5, H: 5}, Depth: 3}
	always := func(Placed, Placed) bool { return true }
	got := Occluders([]Placed{back, front, far}, "", always)
	if !got["f"] || got["x"] || got["b"] {
		t.Errorf("occluders = %v", got)
	}
	if len(Occluders([]Placed{front, far}, "", always)) != 0 {
		t.Error("nothing focused, nothing faded")
	}
	if len(Occluders([]Placed{back, front, far}, "", func(Placed, Placed) bool { return false })) != 0 {
		t.Error("a building that hides none of the focused one stays solid")
	}
}
