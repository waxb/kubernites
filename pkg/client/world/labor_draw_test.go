package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func busyNodes(site view.Site, share float64) map[string]view.NodeMetrics {
	out := map[string]view.NodeMetrics{}
	for _, k := range Keeps(site) {
		out[k.Object.Name] = view.NodeMetrics{CPUMilli: 1900 * share, CPUAllocatableMilli: 1900, MemoryBytes: float64(3<<30) * share, MemoryAllocatableBytes: 3 << 30}
	}
	return out
}

func laborScene(t testing.TB, share float64) (Scene, render.Camera, *Renderer) {
	sc, cam := sceneFor(t, .5)
	sc.Ghost, sc.Transients = nil, nil
	sc.Metrics.Nodes = busyNodes(sc.Site, share)
	if k, ok := FirstOfKind(sc.Site, view.StructKeep, false); ok {
		cam = cam.CenterOn(float64(k.X)+2, float64(k.Y)-2, 960, 540)
	}
	return sc, cam, NewRenderer(gfx.Load(art.Build()))
}

func workers(items []Drawable) int {
	n := 0
	for _, it := range items {
		if it.stand.on {
			n++
		}
	}
	return n
}

func TestABusyKeepsLandShowsFelledTreesAndCutStoneWithNoFigure(t *testing.T) {
	sc, cam, r := laborScene(t, 1)
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
	l := r.laborOf()
	if len(l.sites) == 0 || len(l.sites[0].trees) == 0 || len(l.sites[0].rocks) == 0 {
		t.Fatalf("the Lone Tower's land holds no work: %+v", l.sites)
	}
	seen, scarred := 0, 0
	for step := 0; step < 40; step++ {
		sc.Time += 3
		items := r.laborItems(cam, sc)
		seen = max(seen, workers(items))
		scarred = max(scarred, len(l.hidden))
	}
	sh := l.shifts[l.sites[0].id]
	if sh[0].active != 2 || sh[1].active != 2 {
		t.Fatalf("a busy Lone Tower keeps %d woodcutters and %d masons", sh[0].active, sh[1].active)
	}
	if seen != 0 || scarred == 0 {
		t.Fatalf("figures %d (no figure that is no container), felled or cut tiles %d", seen, scarred)
	}
	for _, i := range l.hidden {
		if !l.hiddenAt(int32(i)) || l.hiddenAt(-1) {
			t.Fatal("hidden tiles disagree")
		}
	}
}

func TestAnIdleKeepSendsNobodyOut(t *testing.T) {
	sc, cam, r := laborScene(t, 0)
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
	for step := 0; step < 10; step++ {
		sc.Time += 5
		if items := r.laborItems(cam, sc); workers(items) != 0 || len(r.laborOf().hidden) != 0 {
			t.Fatalf("an idle keep shows %d workers and %d scars", workers(items), len(r.laborOf().hidden))
		}
	}
}

func TestLaborIsTheSameForTheSameTime(t *testing.T) {
	sc, cam, a := laborScene(t, .8)
	_, _, b := laborScene(t, .8)
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	a.Draw(dst, cam, sc)
	b.Draw(dst, cam, sc)
	sc.Time += 250
	ia, ib := a.laborItems(cam, sc), b.laborItems(cam, sc)
	if len(ia) != len(ib) || len(a.laborOf().hidden) != len(b.laborOf().hidden) {
		t.Fatalf("two renderers disagree: %d/%d items, %d/%d scars", len(ia), len(ib), len(a.laborOf().hidden), len(b.laborOf().hidden))
	}
	for i := range ia {
		if ia[i].Depth != ib[i].Depth {
			t.Fatalf("item %d at depth %v and %v", i, ia[i].Depth, ib[i].Depth)
		}
	}
}

func BenchmarkLaborItems(b *testing.B) {
	sc, cam, r := busyTown(b, 300, 1)
	sc.Metrics.Nodes = busyNodes(sc.Site, 1)
	dst := ebiten.NewImage(960, 540)
	r.Draw(dst, cam, sc)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.Time += 1.0 / 60
		r.laborItems(cam, sc)
	}
}

func TestLandOutOfViewIsCulled(t *testing.T) {
	cam := render.NewCamera(480, 60).CenterOn(40, 40, 960, 540)
	near := laborSite{lo: Point{35, 35}, hi: Point{45, 45}}
	inside := laborSite{lo: Point{0, 0}, hi: Point{90, 90}}
	far := laborSite{lo: Point{100, 100}, hi: Point{110, 110}}
	if !landInView(cam, near) || !landInView(cam, inside) || landInView(cam, far) {
		t.Fatal("land culling misjudged the view")
	}
}
