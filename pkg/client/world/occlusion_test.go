package world

import (
	"image"
	"image/color"
	"maps"
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func solidMask(w, h int) gfx.Mask {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, color.NRGBA{A: 255})
		}
	}
	return gfx.MaskOf(img, art.Region{W: w, H: h})
}

func TestHiddenShareCountsTheFocusedPixelsTheFrontCovers(t *testing.T) {
	back, front := solidMask(4, 4), solidMask(2, 4)
	for _, c := range []struct {
		ox, oy int
		want   float64
	}{{0, 0, .5}, {2, 0, .5}, {3, 0, .25}, {4, 0, 0}, {0, 2, .25}, {-2, 0, 0}} {
		if got := HiddenShare(back, front, c.ox, c.oy); got != c.want {
			t.Errorf("front at %d,%d hides %.2f of the back, want %.2f", c.ox, c.oy, got, c.want)
		}
	}
	if got := HiddenShare(gfx.Mask{}, front, 0, 0); got != 0 {
		t.Errorf("an empty sprite is hidden by %.2f", got)
	}
}

func plazaSite() view.Site {
	const n = 60
	tiles := make([]view.TileKind, n*n)
	for i := range tiles {
		tiles[i] = view.TileGrass
	}
	return view.Site{
		Width: n, Height: n, Tiles: tiles, Elevation: make([]uint8, n*n), Plaza: view.Point{X: 30, Y: 65 - 40}, Gate: view.Point{X: 17, Y: 25},
		Structures: []view.Structure{
			{ID: "keep-3", Kind: view.StructKeep, Variant: "tower", X: 34, Y: 24, W: 3, H: 3, State: view.StateReady, Class: "balanced", Object: view.ObjectRef{Kind: "Node", Name: "lone-tower"}},
			{ID: "codex-4", Kind: view.StructCodex, X: 34, Y: 27, W: 2, H: 2, State: view.StateReady},
			{ID: "chancery-5", Kind: view.StructChancery, X: 37, Y: 25, W: 2, H: 2, State: view.StateReady},
			{ID: "stewards-hall-8", Kind: view.StructStewards, X: 39, Y: 25, W: 2, H: 2, State: view.StateReady},
			{ID: "herald-10", Kind: view.StructHerald, Variant: view.FaceNW, X: 39, Y: 30, W: 1, H: 2, State: view.StateReady},
		},
	}
}

func seeThrough(r *Renderer, cam render.Camera, sc Scene) []string {
	items := r.placeStructures(cam, sc)
	return slices.Sorted(maps.Keys(Occluders(items, sc.Selected, r.hides(cam.Scale()))))
}

func plazaCamera(zoom float64) render.Camera {
	cam := render.NewCamera(480, 60)
	cam.Zoom = zoom
	return cam.CenterOn(36, 26, 960, 540)
}

func TestOnlyABuildingThatHidesTheSelectedTowerTurnsSeeThrough(t *testing.T) {
	r := NewRenderer(testLib())
	for age := range 4 {
		sc := Scene{Site: plazaSite(), Selected: "keep-3", Age: age}
		got := seeThrough(r, plazaCamera(1), sc)
		if slices.Contains(got, "herald-10") || slices.Contains(got, "stewards-hall-8") {
			t.Errorf("tier %d: the Herald's stall and the Stewards' Hall hide none of the Tower, yet turn see-through: %v", age, got)
		}
		if age == 0 && !slices.Equal(got, []string{"chancery-5"}) {
			t.Errorf("age 0: only the Chancery hides the Tower, see-through: %v", got)
		}
	}
}

func TestTheSeeThroughHoldsStillWhileTheCameraGlides(t *testing.T) {
	r := NewRenderer(testLib())
	site := plazaSite()
	for _, focus := range []string{"keep-3", "codex-4", "chancery-5", "herald-10"} {
		sc := Scene{Site: site, Selected: focus}
		for _, zoom := range render.ZoomLevels {
			want := seeThrough(r, plazaCamera(zoom), sc)
			for i := range 240 {
				cam := plazaCamera(zoom).Pan(float64(i)*.173, float64(i)*.091)
				if got := seeThrough(r, cam, sc); !slices.Equal(got, want) {
					t.Fatalf("%s at zoom %g, pan %d: see-through %v, then %v", focus, zoom, i, want, got)
				}
			}
		}
	}
}

func TestABuildingUnderConstructionStillSeesThroughWhatHidesIt(t *testing.T) {
	r := NewRenderer(testLib())
	site := plazaSite()
	site.Structures[0].State, site.Structures[0].Progress = view.StateConstruction, .4
	got := seeThrough(r, plazaCamera(1), Scene{Site: site})
	if !slices.Contains(got, "chancery-5") || slices.Contains(got, "herald-10") || slices.Contains(got, "stewards-hall-8") {
		t.Errorf("the Tower rising behind the Chancery: see-through %v", got)
	}
}
