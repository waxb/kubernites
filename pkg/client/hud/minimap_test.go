package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

func islesSite() view.Site {
	s := view.Site{Width: 4, Height: 2, Tiles: []view.TileKind{
		view.TileGrass, view.TileGrass, view.TileWater, view.TileGrass,
		view.TileGrass, view.TileRoad, view.TileWater, view.TileGrass,
	}}
	s.Islands = []view.Island{{ID: "home", Owned: true}, {ID: "far"}}
	s.Region = []uint8{1, 1, 0, 2, 1, 1, 0, 2}
	s.Wear = make([]uint16, 8)
	s.Wear[1] = 60000
	return s
}

func TestMinimapShadesUncharteredIslesAndWornRoads(t *testing.T) {
	s := islesSite()
	grass := world.TileColor(view.TileGrass)
	if got := MinimapColor(s, 0, 0); got != grass {
		t.Errorf("owned grass = %v, want %v", got, grass)
	}
	if got := MinimapColor(s, 3, 0); got == grass || got.R >= grass.R {
		t.Errorf("an unchartered isle is dimmed: %v", got)
	}
	if got := MinimapColor(s, 1, 0); got == grass {
		t.Error("a worn path shows on the minimap")
	}
	if got := MinimapColor(s, 2, 0); got != world.TileColor(view.TileWater) {
		t.Errorf("sea = %v", got)
	}
	if got := MinimapColor(s, 1, 1); got != world.TileColor(view.TileRoad) {
		t.Errorf("a paved road keeps its colour: %v", got)
	}
	img := MinimapPixels(s)
	if img.Bounds().Dx() != s.Width+s.Height+1 {
		t.Errorf("minimap size = %v", img.Bounds())
	}
}

func TestMinimapCacheRebuildsOnlyOnChange(t *testing.T) {
	h := New(nil, Backend{})
	s := islesSite()
	first := h.minimapImage(s)
	if h.minimapImage(s) != first {
		t.Error("the same site reuses its image")
	}
	next := s
	next.Tiles = append([]view.TileKind(nil), s.Tiles...)
	if h.minimapImage(next) != first {
		t.Error("a copy with the same content reuses its image")
	}
	charted := next
	charted.Islands = []view.Island{{ID: "home", Owned: true}, {ID: "far", Owned: true}}
	charted.Tiles = append([]view.TileKind(nil), s.Tiles...)
	if h.minimapImage(charted) == first {
		t.Error("chartering an isle redraws the minimap")
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	f := fullFrame(t)
	f.Site = charted
	f.Site.Structures = []view.Structure{{Kind: view.StructKeep, X: 1, Y: 0, W: 1, H: 1, Object: view.ObjectRef{Name: "lone-tower"}}}
	h.drawMinimap(dst, f)
}

func TestMinimapFitsTheLand(t *testing.T) {
	site := testSite(t)
	v := FitMinimap(site)
	inner := Minimap.Inset(5)
	full := min(float64(inner.W)/float64(site.Width+site.Height), float64(inner.H)/float64(site.Width+site.Height)*2)
	if v.S < full {
		t.Errorf("fitting the land never shrinks the map: %v < %v", v.S, full)
	}
	for i, tile := range site.Tiles {
		if tile == view.TileWater {
			continue
		}
		x, y := v.FromGrid(float64(i%site.Width), float64(i/site.Width))
		if !inner.Inset(-2).Contains(int(x), int(y)) {
			t.Fatalf("land tile %d at %v,%v leaves the minimap", i, x, y)
		}
	}
	for _, is := range site.Islands {
		x, y := v.FromGrid(float64(is.Center.X), float64(is.Center.Y))
		gx, gy := v.ToGrid(int(x+.5), int(y+.5))
		if abs(gx-float64(is.Center.X)) > 2 || abs(gy-float64(is.Center.Y)) > 2 {
			t.Errorf("island %s round trip = %.1f,%.1f", is.ID, gx, gy)
		}
	}
	sea := view.Site{Width: 4, Height: 4, Tiles: make([]view.TileKind, 16)}
	if s := FitMinimap(sea); s.S <= 0 {
		t.Error("an all-sea map still fits the box")
	}
}
