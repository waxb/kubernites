package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func selectedKeep(t *testing.T, sc Scene) view.Structure {
	t.Helper()
	for _, s := range sc.Site.Structures {
		if IsKeepKind(s.Kind) && s.State == view.StateReady {
			return s
		}
	}
	t.Fatal("the lively realm has no ready Keep")
	return view.Structure{}
}

func labelOver(t *testing.T, r *Renderer, sx float64) ui.Rect {
	t.Helper()
	for _, b := range r.Drawn() {
		if b.Kind == DrawnLabel && float64(b.Rect.X) <= sx && sx <= float64(b.Rect.X+b.Rect.W) {
			return b.Rect
		}
	}
	t.Fatalf("no label drawn over x=%v: %+v", sx, r.Drawn())
	return ui.Rect{}
}

func TestASelectedBuildingsLabelSitsAboveItsDrawnTopAtEveryZoom(t *testing.T) {
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	for _, zoom := range render.ZoomLevels {
		r := NewRenderer(gfx.Load(art.Build()))
		sc, cam := sceneFor(t, .5)
		sc.Ghost = nil
		keep := selectedKeep(t, sc)
		sc.Selected = keep.ID
		a := Anchor(keep)
		cam.Zoom = zoom
		cam = cam.CenterOn(a.X, a.Y, 960, 540)
		r.Draw(dst, cam, sc)
		hit, ok := r.HitOf(HitStructure, keep.ID)
		if !ok || len(hit.Shapes) == 0 {
			t.Fatalf("zoom %v: the Keep %s was not drawn", zoom, keep.ID)
		}
		up, ok := r.Lib.Headroom(hit.Shapes[0].Name)
		if !ok {
			t.Fatalf("zoom %v: no headroom for %s", zoom, hit.Shapes[0].Name)
		}
		top := hit.Shapes[0].Y - float64(up)*zoom
		sx, _ := ScreenPointOf(cam, sc.Site, a)
		label := labelOver(t, r, sx)
		if float64(label.Y+label.H) > top {
			t.Errorf("zoom %v: the label %+v reaches below the Keep's drawn top at y=%.0f", zoom, label, top)
		}
	}
}

func TestABuildingsLabelStacksAboveItsKeepsHandsMeter(t *testing.T) {
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r := NewRenderer(gfx.Load(art.Build()))
	sc, cam := sceneFor(t, .5)
	sc.Ghost = nil
	keep := selectedKeep(t, sc)
	sc.Selected = keep.ID
	sc.Site.Structures = append(sc.Site.Structures, view.Structure{ID: "chancery", Kind: view.StructChancery, X: 2, Y: 2, W: 2, H: 2, State: view.StateReady, Progress: 1})
	sc.Metrics.Nodes = map[string]view.NodeMetrics{keep.Object.Name: {CPUAllocatableMilli: 4000, CPURequestedMilli: 2000}}
	a := Anchor(keep)
	cam = cam.CenterOn(a.X, a.Y, 960, 540)
	r.Draw(dst, cam, sc)
	pips := r.keepPips(cam, sc)
	if len(pips) == 0 {
		t.Fatalf("the Keep %s (node %q) shows no hands meter", keep.ID, keep.Object.Name)
	}
	sx, _ := ScreenPointOf(cam, sc.Site, a)
	label := labelOver(t, r, sx)
	for _, k := range pips {
		if Overlaps(label, k.plaque()) {
			t.Errorf("the label %+v covers the hands meter %+v", label, k.plaque())
		}
	}
}

func TestALabelWithNoRoomAboveGoesBelowItsBuildingNeverAcrossIt(t *testing.T) {
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r := NewRenderer(gfx.Load(art.Build()))
	sc, cam := sceneFor(t, .5)
	sc.Ghost = nil
	keep := selectedKeep(t, sc)
	sc.Selected = keep.ID
	a := Anchor(keep)
	cam = cam.CenterOn(a.X, a.Y, 960, 540)
	r.Draw(dst, cam, sc)
	hit, _ := r.HitOf(HitStructure, keep.ID)
	sh := hit.Shapes[0]
	up, down, _ := r.Lib.Extent(sh.Name)
	top, bottom := sh.Y-float64(up)*sh.Zoom, sh.Y+float64(down)*sh.Zoom
	sc.Panels = []ui.Rect{{X: 0, Y: 0, W: 960, H: int(top) - 4}}
	r.Draw(dst, cam, sc)
	sx, _ := ScreenPointOf(cam, sc.Site, a)
	label := labelOver(t, r, sx)
	if float64(label.Y) < bottom {
		t.Errorf("a panel holds the room above the Keep (drawn from y=%.0f to %.0f), and its label %+v lies across it instead of below it", top, bottom, label)
	}
}

func TestABuildingsLabelStandsClearOfTheBeaconAboveIt(t *testing.T) {
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r := NewRenderer(gfx.Load(art.Build()))
	sc, cam := sceneFor(t, .5)
	sc.Ghost = nil
	low := lowestStructure(t, r, sc, cam)
	sc.Selected = low.ID
	a := Anchor(low)
	sc.Goal = &a
	sc.Reduce = true
	cam = cam.CenterOn(a.X, a.Y, 960, 540)
	r.Draw(dst, cam, sc)
	sx, sy := ScreenPointOf(cam, sc.Site, a)
	arrow := beaconArrow(sx, sy, cam.Scale())
	label := labelOver(t, r, sx)
	if Overlaps(label, arrow) {
		t.Errorf("the label %+v lies across the beacon %+v that points at %s", label, arrow, low.ID)
	}
}

func lowestStructure(t *testing.T, r *Renderer, sc Scene, cam render.Camera) view.Structure {
	t.Helper()
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	var low view.Structure
	least := 0
	for _, s := range sc.Site.Structures {
		a := Anchor(s)
		r.Draw(dst, cam.CenterOn(a.X, a.Y, 960, 540), sc)
		hit, ok := r.HitOf(HitStructure, s.ID)
		if !ok || len(hit.Shapes) == 0 {
			continue
		}
		if up, _, ok := r.Lib.Extent(hit.Shapes[0].Name); ok && (low.ID == "" || up < least) {
			low, least = s, up
		}
	}
	if low.ID == "" {
		t.Fatal("no structure of the lively realm was drawn")
	}
	return low
}
