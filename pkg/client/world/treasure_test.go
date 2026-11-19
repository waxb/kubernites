package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
)

func TestAChestSitsOnTheYardOfItsCargo(t *testing.T) {
	site := archipelago(t)
	at, ok := YardFocus(site, "dispatch")
	if !ok {
		t.Skip("the archipelago has no haven")
	}
	r := NewRenderer(testLib())
	cam := render.NewCamera(480, 60).CenterOn(at.X, at.Y, 960, 540)
	sc := Scene{Site: site, Time: 2, Chests: []ChestMark{{Kind: "dispatch"}, {Kind: "ledger", Open: true, Age: .5}}}
	got := r.chestSprites(cam, sc)
	names := map[string]bool{}
	for _, s := range got {
		names[s.name] = true
		if !r.Lib.Has(s.name) {
			t.Errorf("%s is not in the atlas", s.name)
		}
	}
	if !names[art.TreasureChestName("dispatch", false)] || !names[art.TreasureChestName("ledger", true)] {
		t.Fatalf("sprites = %v", names)
	}
	sx, sy := ScreenPointOf(cam, site, Vec{at.X, at.Y})
	for _, s := range got {
		if s.name == art.TreasureChestName("dispatch", false) && (s.sx != sx || s.sy >= sy) {
			t.Fatalf("the dispatch chest at %v,%v is not above its yard at %v,%v", s.sx, s.sy, sx, sy)
		}
	}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.drawChests(dst, cam, sc)
}
