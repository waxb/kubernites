package world

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
)

func focusedScene(t testing.TB) (Scene, *Renderer, *ebiten.Image, render.Camera) {
	sc, cam := sceneFor(t, .5)
	for _, p := range sc.Pods {
		if p.State == model.HearthReady && !p.System {
			sc.Focus = PersonKey(p.Person(0))
			break
		}
	}
	r := NewRenderer(gfx.Load(art.Build()))
	dst := ebiten.NewImage(960, 540)
	r.Draw(dst, cam, sc)
	return sc, r, dst, cam
}

func hitKeys(hits []Hit) []string {
	out := []string{}
	for _, h := range hits {
		out = append(out, fmt.Sprintf("%s %s %v", h.Kind, h.ID, h.Rect))
	}
	return out
}

func TestInsetCentresTheWorker(t *testing.T) {
	sc, r, dst, cam := focusedScene(t)
	defer dst.Deallocate()
	mark, ok := r.Focus()
	if !ok || !mark.Seen {
		t.Fatalf("focus = %+v", mark)
	}
	ic := InsetCamera(sc.Site, mark.Pos, InsetW, InsetH)
	if x, y := ScreenPointOf(ic, sc.Site, mark.Pos); math.Abs(x-InsetW/2) > 0.5 || math.Abs(y-InsetH*insetFeet) > 0.5 {
		t.Errorf("the worker stands at %v,%v in the inset", x, y)
	}
	if ic.Scale() != InsetZoom {
		t.Errorf("the inset is zoomed in: %v", ic.Scale())
	}
	inset := ebiten.NewImage(InsetW, InsetH)
	defer inset.Deallocate()
	if !r.DrawInset(inset, sc) {
		t.Fatal("a selected worker has an inset view")
	}
	sc.Focus = ""
	r.Draw(dst, cam, sc)
	if r.DrawInset(inset, sc) {
		t.Error("no selected worker, no inset")
	}
}

func TestInsetLeavesTheMainFrameAlone(t *testing.T) {
	sc, r, dst, cam := focusedScene(t)
	defer dst.Deallocate()
	inset := ebiten.NewImage(InsetW, InsetH)
	defer inset.Deallocate()
	r.Draw(dst, cam, sc)
	want, wantDecks, wantFocus := hitKeys(r.HitList), hitKeys(r.deckHits), r.focus
	r.DrawInset(inset, sc)
	r.Draw(dst, cam, sc)
	if got := hitKeys(r.HitList); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("picking changed after the inset: %d hits, want %d", len(got), len(want))
	}
	if got := hitKeys(r.deckHits); strings.Join(got, "|") != strings.Join(wantDecks, "|") {
		t.Error("deck picking changed after the inset")
	}
	if r.focus != wantFocus {
		t.Errorf("focus changed: %+v, want %+v", r.focus, wantFocus)
	}
}

func BenchmarkDrawInsetBesideTown(b *testing.B) {
	sc, cam, r := busyTown(b, 300, 1)
	for _, p := range sc.Pods {
		if p.State == model.HearthReady {
			sc.Focus = PersonKey(p.Person(0))
			break
		}
	}
	dst := ebiten.NewImage(960, 540)
	inset := ebiten.NewImage(InsetW, InsetH)
	r.Draw(dst, cam, sc)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.Time += 1.0 / 60
		r.DrawInset(inset, sc)
	}
}
