package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func grassSite(withKeep bool) view.Site {
	s := view.Site{Width: 8, Height: 8, Tiles: make([]view.TileKind, 64)}
	for i := range s.Tiles {
		s.Tiles[i] = view.TileGrass
	}
	s.Tiles[0] = view.TileWater
	if withKeep {
		s.Structures = []view.Structure{{ID: "k", Kind: view.StructKeep, Variant: "tower", X: 1, Y: 1, W: 3, H: 3}}
	}
	return s
}

func TestGhostTakesTheKeepVariant(t *testing.T) {
	bare, built := grassSite(false), grassSite(true)
	cases := []struct {
		site          view.Site
		kind, variant string
	}{
		{bare, view.StructKeep, "tower"}, {built, view.StructKeep, "stone"}, {built, "great-keep", "great"}, {bare, "citadel-keep", "citadel"},
	}
	for _, c := range cases {
		s := GhostStructure(c.site, Ghost{Kind: c.kind, W: 3, H: 3})
		if s.Kind != view.StructKeep || s.Variant != c.variant {
			t.Errorf("ghost %s = %s/%s, want keep/%s", c.kind, s.Kind, s.Variant, c.variant)
		}
	}
	if s := GhostStructure(built, Ghost{Kind: "codex"}); s.Kind != "codex" || s.Variant != "" {
		t.Errorf("a codex ghost stays a codex: %+v", s)
	}
}

func TestGhostOverlayDrawsWithoutALeyLayer(t *testing.T) {
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	site := grassSite(true)
	cam := render.NewCamera(480, 60).CenterOn(4, 4, 960, 540)
	x0, y0, x1, y1 := visibleTiles(dst, cam, site)
	if x0 != 0 || y0 != 0 || x1 != 7 || y1 != 7 {
		t.Errorf("visible = %d,%d..%d,%d", x0, y0, x1, y1)
	}
	r := NewRenderer(nil)
	r.drawOverlays(dst, cam, Scene{Site: site, Ghost: &Ghost{Kind: "great-keep", X: 3, Y: 3, W: 3, H: 3, Valid: true}})
	if len(site.Power) != 0 || site.PowerAt(3, 3) != 0 {
		t.Error("the retired ley layer is back")
	}
}
