package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
)

func TestTheWholeQuaySelectsItsHarbor(t *testing.T) {
	site := archipelago(t)
	r := NewRenderer(testLib())
	at, ok := HarborFocus(site)
	if !ok {
		t.Skip("no haven")
	}
	cam := render.NewCamera(480, 60).CenterOn(at.X, at.Y, 960, 540)
	sc := Scene{Site: site}
	sc.Freight.Waiting = 0
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
	h := r.HarborOf()
	if len(h.deckOrder) == 0 {
		t.Skip("no open quay deck")
	}
	hits := 0
	for _, d := range h.deckOrder {
		sx, sy := ScreenPointOf(cam, site, d.Center())
		for _, hit := range r.PickAll(int(sx), int(sy)-2) {
			if hit.Kind == HitHaven {
				hits++
				break
			}
		}
	}
	if hits*10 < len(h.deckOrder)*9 {
		t.Fatalf("only %d of %d quay planks select the harbor", hits, len(h.deckOrder))
	}
}
