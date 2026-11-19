package gfx

import (
	"image"
	"image/color"
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
)

func ringSet() art.Set {
	img := image.NewNRGBA(image.Rect(0, 0, 80, 20))
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			if x == 0 || y == 0 || x == 9 || y == 9 {
				img.SetNRGBA(x, y, color.NRGBA{200, 100, 50, 255})
			}
		}
	}
	img.SetNRGBA(20, 5, color.NRGBA{A: 90})
	img.SetNRGBA(30, 0, color.NRGBA{A: 255})
	regions := map[string]art.Region{
		"ring":   {X: 0, Y: 0, W: 10, H: 10, AnchorX: 5, AnchorY: 10},
		"shadow": {X: 20, Y: 0, W: 10, H: 10, AnchorX: 5, AnchorY: 10},
		"corner": {X: 30, Y: 0, W: 70 - 30, H: 10, AnchorX: 0, AnchorY: 10},
	}
	return art.Set{Buildings: art.Atlas{Image: img, Regions: regions}}
}

func TestAMaskHoldsOnlyTheOpaquePixels(t *testing.T) {
	set := ringSet()
	m := MaskOf(set.Buildings.Image, set.Buildings.Regions["ring"])
	if m.W != 10 || m.H != 10 || !m.At(0, 0) || !m.At(9, 5) || m.At(5, 5) || m.At(-1, 0) || m.At(10, 0) {
		t.Fatalf("ring mask %+v", m)
	}
	if !m.Near(5, 5, 4) || m.Near(5, 5, 3) || !m.Near(-2, 5, 2) || m.Near(-3, 5, 2) {
		t.Fatal("near reads the wrong neighbourhood")
	}
	if s := MaskOf(set.Buildings.Image, set.Buildings.Regions["shadow"]); s.Near(0, 5, 9) {
		t.Fatal("a faint shadow pixel counts as opaque")
	}
	if c := MaskOf(set.Buildings.Image, set.Buildings.Regions["corner"]); !c.At(0, 0) || c.At(1, 0) {
		t.Fatal("wide mask rows are misread")
	}
}

func TestCoversMapsTheScreenToTheSpriteAsDrawn(t *testing.T) {
	lib := Load(ringSet())
	sx, sy := 100.0, 100.0
	cases := []struct {
		x, y int
		zoom float64
		pad  float64
		flip bool
		want bool
	}{
		{95, 90, 1, 0, false, true},
		{100, 95, 1, 0, false, false},
		{100, 95, 1, 3, false, false},
		{100, 95, 1, 4, false, true},
		{104, 99, 1, 0, false, true},
		{94, 95, 1, 0, false, false},
		{94, 95, 1, 1, false, true},
		{90, 80, 2, 0, false, true},
		{100, 90, 2, 0, false, false},
		{109, 99, 2, 0, false, true},
		{95, 90, 1, 0, true, true},
		{130, 130, 1, 2, false, false},
	}
	for i, c := range cases {
		if got := lib.Covers("ring", sx, sy, c.zoom, c.flip, c.x, c.y, c.pad); got != c.want {
			t.Errorf("case %d (%d,%d zoom %v pad %v flip %v) = %v", i, c.x, c.y, c.zoom, c.pad, c.flip, got)
		}
	}
	if lib.Covers("missing", sx, sy, 1, false, 95, 90, 0) {
		t.Fatal("a missing sprite covers nothing")
	}
	var none *Library
	if none.Covers("ring", sx, sy, 1, false, 95, 90, 0) {
		t.Fatal("a nil library covers nothing")
	}
}

func TestCoversFlipsWithTheSprite(t *testing.T) {
	lib := Load(ringSet())
	if !lib.Covers("corner", 0, 10, 1, false, 0, 0, 0) || lib.Covers("corner", 0, 10, 1, false, 39, 0, 0) {
		t.Fatal("unflipped corner")
	}
	if lib.Covers("corner", 0, 10, 1, true, -40, 0, 0) || !lib.Covers("corner", 0, 10, 1, true, -1, 0, 0) {
		t.Fatal("flipped corner")
	}
}

func TestEverySpriteOfTheGameHasAMask(t *testing.T) {
	lib := Load(art.Build())
	for _, name := range []string{"building/chancery/ready", "building/codex/ready"} {
		if m, ok := lib.Mask(name); !ok || m.Empty() {
			t.Fatalf("%s has no mask", name)
		}
	}
	if _, ok := lib.Mask("building/nothing/ready"); ok {
		t.Fatal("a missing sprite has a mask")
	}
}

func BenchmarkMaskOfEveryAtlas(b *testing.B) {
	set := art.Build()
	atlases := []art.Atlas{set.Terrain, set.Buildings, set.Walkers, set.Effects, set.Icons, set.Portraits, set.UI}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, a := range atlases {
			for _, r := range a.Regions {
				MaskOf(a.Image, r)
			}
		}
	}
}

func BenchmarkCovers(b *testing.B) {
	lib := Load(art.Build())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lib.Covers("building/chancery/ready", 400, 300, 2, false, 380+i%40, 240+i%60, 2)
	}
}
