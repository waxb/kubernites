package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func pickSite() view.Site {
	site := testSite()
	site.Structures = []view.Structure{
		{ID: "quartermaster", Kind: view.StructQuartermaster, X: 9, Y: 9, W: 2, H: 2, State: view.StateReady},
		{ID: "chancery", Kind: view.StructChancery, X: 10, Y: 11, W: 2, H: 2, State: view.StateReady},
	}
	return site
}

func drawnPicks(t testing.TB, site view.Site, zoom float64) *Renderer {
	t.Helper()
	r := NewRenderer(testLib())
	cam := render.NewCamera(480, 60).CenterOn(10.5, 10.5, 960, 540)
	cam.Zoom = zoom
	cam = cam.CenterOn(10.5, 10.5, 960, 540)
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.Draw(dst, cam, Scene{Site: site, DayPhase: .5, Time: 1})
	return r
}

func hitNamed(t testing.TB, r *Renderer, id string) Hit {
	t.Helper()
	for _, h := range r.HitList {
		if h.ID == id {
			return h
		}
	}
	t.Fatalf("no hit %q among %d", id, len(r.HitList))
	return Hit{}
}

func scanRect(r ui.Rect, each func(x, y int)) {
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			each(x, y)
		}
	}
}

func TestTheChanceryNoLongerHidesTheBuildingBehindIt(t *testing.T) {
	for _, zoom := range []float64{1, 2} {
		r := drawnPicks(t, pickSite(), zoom)
		front, back := hitNamed(t, r, "chancery"), hitNamed(t, r, "quartermaster")
		through, blocked := 0, 0
		scanRect(front.Rect, func(x, y int) {
			if !r.Covers(back, x, y, 0) {
				return
			}
			hit, ok := r.Pick(x, y)
			switch {
			case !ok:
				t.Fatalf("zoom %v: nothing picked at %d,%d over the Quartermaster", zoom, x, y)
			case r.Covers(front, x, y, 0):
				blocked++
			case hit.ID != back.ID:
				t.Fatalf("zoom %v: the Chancery's empty corner at %d,%d picked %q", zoom, x, y, hit.ID)
			default:
				through++
			}
		})
		if through == 0 || blocked == 0 {
			t.Fatalf("zoom %v: %d pixels click through, %d are the Chancery's own", zoom, through, blocked)
		}
	}
}

func TestPickingFollowsTheOpaquePixels(t *testing.T) {
	r := drawnPicks(t, pickSite(), 2)
	front := hitNamed(t, r, "chancery")
	clear := 0
	scanRect(front.Rect, func(x, y int) {
		if _, ok := r.Pick(x, y); !ok {
			clear++
		}
	})
	if clear == 0 {
		t.Fatal("every pixel of the Chancery's box picks something: picking still uses its rect")
	}
	shape := front.Shapes[0]
	if !r.Lib.Covers(shape.Name, shape.X, shape.Y, shape.Zoom, shape.Flip, int(shape.X), int(shape.Y)-4, PickPad) {
		t.Fatal("the Chancery's own foot is not under it")
	}
}

func TestRepeatedClicksReachEveryObjectUnderTheCursor(t *testing.T) {
	r := drawnPicks(t, pickSite(), 2)
	front, back := hitNamed(t, r, "chancery"), hitNamed(t, r, "quartermaster")
	found := false
	scanRect(front.Rect, func(x, y int) {
		if found || !r.Covers(front, x, y, 0) || !r.Covers(back, x, y, 0) {
			return
		}
		all := r.PickAll(x, y)
		if len(all) != 2 || all[0].ID != "chancery" || all[1].ID != "quartermaster" {
			t.Fatalf("under %d,%d: %+v", x, y, all)
		}
		found = true
	})
	if !found {
		t.Fatal("no pixel shows both buildings")
	}
}

func TestPickOrderPutsGleamsAndBoxesFirstThenWhatIsOnTop(t *testing.T) {
	area := ui.Rect{X: 0, Y: 0, W: 50, H: 50}
	r := &Renderer{HitList: []Hit{
		{Kind: HitGolden, ID: "g", Rect: area},
		{Kind: HitWalker, ID: "w", Rect: area},
		{Kind: HitPile, ID: "p", Rect: area},
		{Kind: HitStructure, ID: "s", Rect: area},
		{Kind: HitPile, ID: "p", Rect: area},
		{Kind: HitPile, ID: "p", Rect: area, Serial: 1},
	}}
	all := r.PickAll(10, 10)
	want := []string{"g", "p", "p", "s", "w"}
	if len(all) != len(want) {
		t.Fatalf("picked %+v", all)
	}
	for i, id := range want {
		if all[i].ID != id {
			t.Fatalf("pick %d = %q, want %q (%+v)", i, all[i].ID, id, all)
		}
	}
	if all[1].Serial != 1 {
		t.Fatal("the topmost pile is not first")
	}
	if got := r.PickAll(60, 60); len(got) != 0 {
		t.Fatalf("outside everything: %+v", got)
	}
}

func TestAnExactPixelBeatsANeighboursMargin(t *testing.T) {
	r := drawnPicks(t, pickSite(), 1)
	front, back := hitNamed(t, r, "chancery"), hitNamed(t, r, "quartermaster")
	found := 0
	scanRect(front.Rect, func(x, y int) {
		if !r.Covers(back, x, y, 0) || r.Covers(front, x, y, 0) || !r.Covers(front, x, y, PickPad) {
			return
		}
		if hit, ok := r.Pick(x, y); !ok || hit.ID != back.ID {
			t.Fatalf("the Quartermaster's own pixel at %d,%d went to %q", x, y, hit.ID)
		}
		found++
	})
	if found == 0 {
		t.Fatal("no pixel of the Quartermaster lies in the Chancery's margin")
	}
}

func BenchmarkPickOverACrowdedView(b *testing.B) {
	site := pickSite()
	for i := 0; i < 36; i++ {
		site.Structures = append(site.Structures, view.Structure{ID: "q" + Itoa(i), Kind: view.StructQuartermaster,
			X: 2 + (i%6)*3, Y: 2 + (i/6)*3, W: 2, H: 2, State: view.StateReady})
	}
	r := drawnPicks(b, site, 1)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Pick(300+i%360, 150+i%240)
	}
}

type shadowCount struct{ own, lost, inFront int }

func shadowOf(r *Renderer, back, front Hit) shadowCount {
	var c shadowCount
	scanRect(back.Rect, func(x, y int) {
		if !r.Covers(back, x, y, 0) || r.Covers(front, x, y, 0) {
			return
		}
		if front.Rect.Contains(x, y) {
			c.inFront++
		}
		if hit, ok := r.Pick(x, y); ok && hit.ID == back.ID {
			c.own++
		} else {
			c.lost++
		}
	})
	return c
}

func TestAWarHornBesideADepotKeepsItsOwnPixels(t *testing.T) {
	site := testSite()
	site.Structures = []view.Structure{
		{ID: "horn", Kind: view.StructWarHorn, X: 9, Y: 9, W: 2, H: 2, State: view.StateReady},
		{ID: "depot", Kind: "depot-parcel", X: 11, Y: 9, W: 3, H: 3, State: view.StateReady, Object: view.ObjectRef{Kind: "Service", Name: "parcels"}},
	}
	for _, zoom := range []float64{1, 2} {
		r := drawnPicks(t, site, zoom)
		c := shadowOf(r, hitNamed(t, r, "horn"), hitNamed(t, r, "depot"))
		if c.lost != 0 || c.own == 0 || c.inFront == 0 {
			t.Fatalf("zoom %v: the War Horn keeps %d of its pixels and loses %d; %d lie in the depot's box", zoom, c.own, c.lost, c.inFront)
		}
	}
}

func TestAWalkerAtThePostTakesOnlyItsOwnPixels(t *testing.T) {
	site := testSite()
	site.Structures = []view.Structure{{ID: "post", Kind: view.StructWayfinder, X: 10, Y: 10, W: 1, H: 1, State: view.StateReady}}
	r := drawnPicks(t, site, 2)
	post := hitNamed(t, r, "post")
	names := r.LookNames(NameKey{Role: "wayfinder", Action: "idle", Dir: "se", Tier: 1})
	sx, sy := post.Shapes[0].X+4, post.Shapes[0].Y+2
	shown := r.ShownName(names)
	walker := Hit{Kind: HitWalker, ID: "kube-system/kube-proxy", Rect: SpriteRect(r.Lib, shown, sx, sy, 2, 12, 20), Shapes: ShapeAt(shown, sx, sy, 2)}
	r.HitList = append(r.HitList, walker)
	c := shadowOf(r, post, walker)
	if c.lost != 0 || c.own == 0 || c.inFront == 0 {
		t.Fatalf("the post keeps %d of its pixels and loses %d; %d lie in the walker's box", c.own, c.lost, c.inFront)
	}
	if hit, ok := r.Pick(int(sx), int(sy)-6); !ok || hit.ID != walker.ID {
		t.Fatalf("the walker's own body picked %+v", hit)
	}
}
