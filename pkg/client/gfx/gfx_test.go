package gfx

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func testSet() art.Set {
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	regions := map[string]art.Region{
		"tile/grass/0":         {X: 0, Y: 0, W: 32, H: 16, AnchorX: 16, AnchorY: 8},
		"icon/gold":            {X: 0, Y: 16, W: 16, H: 16},
		"portrait/lamplighter": {X: 16, Y: 16, W: 16, H: 16},
		"ui/panel":             {X: 0, Y: 32, W: 24, H: 24, Slice: 8},
	}
	return art.Set{Terrain: art.Atlas{Image: img, Regions: regions}}
}

func TestLibraryLookupAndDraw(t *testing.T) {
	lib := Load(testSet())
	if !lib.Has("tile/grass/0") || lib.Has("tile/nope/0") {
		t.Fatal("has wrong")
	}
	if w, h := lib.Size("tile/grass/0"); w != 32 || h != 16 {
		t.Fatalf("size = %d,%d", w, h)
	}
	if w, _ := lib.Size("missing"); w != 0 {
		t.Fatal("missing size should be 0")
	}
	dst := ebiten.NewImage(100, 100)
	if !lib.Draw(dst, "tile/grass/0", 10, 10, 1, render.SpriteOptions{}) || lib.Draw(dst, "x", 0, 0, 1, render.SpriteOptions{}) {
		t.Fatal("draw result wrong")
	}
	if name, ok := lib.DrawFirst(dst, []string{"x", "tile/grass/0"}, 0, 0, 1, render.SpriteOptions{}); !ok || name != "tile/grass/0" {
		t.Fatal("draw first wrong")
	}
	if _, ok := lib.DrawFirst(dst, []string{"x"}, 0, 0, 1, render.SpriteOptions{}); ok {
		t.Fatal("draw first should fail")
	}
	if !lib.Icon(dst, "gold", 0, 0) || lib.Icon(dst, "nope", 0, 0) {
		t.Fatal("icon wrong")
	}
	if !lib.Portrait(dst, "lamplighter", ui.Rect{W: 48, H: 48}) || lib.Portrait(dst, "nope", ui.Rect{}) {
		t.Fatal("portrait wrong")
	}
	if !lib.NineSlice(dst, "panel", ui.Rect{W: 60, H: 40}) || lib.NineSlice(dst, "panel", ui.Rect{W: 4, H: 4}) || lib.NineSlice(dst, "x", ui.Rect{W: 60, H: 40}) {
		t.Fatal("nine slice wrong")
	}
	if len(Missing(lib, []string{"tile/grass/0", "x"})) != 1 || lib.String() == "" {
		t.Fatal("missing wrong")
	}
	var none *Library
	if none.Has("x") || none.NineSlice(dst, "panel", ui.Rect{}) || none.String() == "" {
		t.Fatal("nil library")
	}
}

func TestNineParts(t *testing.T) {
	parts := NineParts(art.Region{X: 0, Y: 0, W: 24, H: 24, Slice: 8}, ui.Rect{X: 10, Y: 10, W: 100, H: 50})
	if len(parts) != 9 {
		t.Fatalf("parts = %d", len(parts))
	}
	if parts[4].Dst != image.Rect(18, 18, 102, 52) {
		t.Fatalf("centre = %v", parts[4].Dst)
	}
}

func TestNames(t *testing.T) {
	keep := view.Structure{Kind: view.StructKeep, Variant: "tower", State: view.StateConstruction, Progress: .9}
	if BuildingName(keep) != "building/keep-tower/construction-2" {
		t.Fatalf("building = %s", BuildingName(keep))
	}
	stone := view.Structure{Kind: view.StructKeep, State: view.StateDark}
	if BuildingName(stone) != "building/keep-stone/dark" || BuildingNames(stone)[1] != "building/keep-stone/ready" {
		t.Fatal("stone keep wrong")
	}
	for variant, want := range map[string]string{"great": "keep-great", "citadel": "keep-citadel", "stone": "keep-stone", "": "keep-stone"} {
		if got := ArtKind(view.Structure{Kind: view.StructKeep, Variant: variant}); got != want {
			t.Errorf("keep variant %q = %s, want %s", variant, got, want)
		}
	}
	if BuildingState(view.Structure{State: view.StateReady}) != "ready" || ArtKind(view.Structure{Kind: "codex"}) != "codex" {
		t.Fatal("state/kind wrong")
	}
	if HearthName(7, "ready") != "hearth/3/ready" || WalkerName("baker", "walk", "se", 5) != "walker/baker/walk/se/1" {
		t.Fatal("hearth/walker wrong")
	}
	if len(WalkerNames("x", "carry", "sw", 0)) != 4 || EffectName("fire", 7) != "fx/fire/1" {
		t.Fatal("names wrong")
	}
	if DecorName("rock", 4) != "decor/rock/1" {
		t.Fatal("decor wrong")
	}
}

func TestContentTop(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 4, 6))
	img.SetNRGBA(1, 3, color.NRGBA{A: 255})
	if got := ContentTop(img, art.Region{W: 4, H: 6}); got != 3 {
		t.Fatalf("top = %d", got)
	}
	if ContentTop(image.NewNRGBA(image.Rect(0, 0, 2, 2)), art.Region{W: 2, H: 2}) != 2 {
		t.Fatal("empty region")
	}
	lib := Load(art.Build())
	if h, ok := lib.Headroom("building/stall-archive-sw/ready"); !ok || h <= 0 {
		t.Fatalf("archive headroom = %d %v", h, ok)
	}
	var none *Library
	if _, ok := none.Headroom("x"); ok {
		t.Fatal("nil library")
	}
	if _, ok := lib.Headroom("missing"); ok {
		t.Fatal("missing sprite")
	}
}
