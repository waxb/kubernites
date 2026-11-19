package art

import (
	"image/color"
	"strings"
	"testing"
)

func opaquePixels(a Atlas, r Region) int {
	n := 0
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			if a.Image.NRGBAAt(x, y).A > 0 {
				n++
			}
		}
	}
	return n
}

func TestNoFullyTransparentSprites(t *testing.T) {
	for name, a := range atlases(built()) {
		for n, r := range a.Regions {
			if opaquePixels(a, r) == 0 {
				t.Errorf("%s: %s is fully transparent", name, n)
			}
		}
	}
}

func TestSizesAndAnchors(t *testing.T) {
	s := built()
	cases := []struct {
		atlas  Atlas
		prefix string
		w, h   int
		ax, ay int
	}{
		{s.Terrain, "sea/", SeaWidth, SeaHeight, 0, 0},
		{s.Icons, "icon/", IconSize, IconSize, 8, 8},
		{s.Portraits, "portrait/", PortraitSize, PortraitSize, 24, 24},
	}
	for _, c := range cases {
		for n, r := range c.atlas.Regions {
			if strings.HasPrefix(n, c.prefix) && (r.W != c.w || r.H != c.h || r.AnchorX != c.ax || r.AnchorY != c.ay) {
				t.Errorf("%s: got %+v", n, r)
			}
		}
	}
}

func paintedSize(kind string) size {
	if k, ok := CanonicalKeep(kind, 0); ok {
		return keepSize(k)
	}
	return buildingSize(kind)
}

func TestBuildingAnchorsAtFootprintBottom(t *testing.T) {
	a := built().Buildings
	for _, k := range BuildingKinds {
		want := paintedSize(k)
		for _, st := range BuildingStates {
			if Unbuilt[k] && strings.HasPrefix(st, "construction") {
				continue
			}
			r := a.Regions[buildingName(k, st)]
			if r.W != Footprints[k]*TileWidth+4 || r.W != want.w || r.H != want.h || r.AnchorX != r.W/2 || r.AnchorY != want.ay {
				t.Errorf("%s/%s: region %+v does not match footprint %d", k, st, r, Footprints[k])
			}
		}
	}
	if s := buildingSize("codex"); s.h-s.ay != buildingFoot {
		t.Errorf("a building keeps %d rows of footing below its anchor, got %d", buildingFoot, s.h-s.ay)
	}
}

func TestSouthWestMirrorsSouthEast(t *testing.T) {
	a := built().Walkers
	for _, role := range WalkerRoles {
		se, sw := a.Regions[walkerName(role, "walk", "se", 1)], a.Regions[walkerName(role, "walk", "sw", 1)]
		for y := 0; y < se.H; y++ {
			for x := 0; x < se.W; x++ {
				if a.Image.NRGBAAt(se.X+x, se.Y+y) != a.Image.NRGBAAt(sw.X+se.W-1-x, sw.Y+y) {
					t.Fatalf("%s: sw is not a mirror of se at %d,%d", role, x, y)
				}
			}
		}
	}
}

func TestAnimationFramesDiffer(t *testing.T) {
	s := built()
	pairs := [][2]string{
		{SeaName(0), SeaName(1)},
		{walkerName("builder", "walk", "se", 0), walkerName("builder", "walk", "se", 1)},
		{walkerName("builder", "work", "se", 0), walkerName("builder", "work", "se", 2)},
		{effectName("fire", 0), effectName("fire", 1)},
		{effectName("boat", 0), effectName("boat", 1)},
	}
	for _, p := range pairs {
		a := s.Terrain
		switch {
		case strings.HasPrefix(p[0], "walker/"):
			a = s.Walkers
		case strings.HasPrefix(p[0], "fx/"):
			a = s.Effects
		}
		if samePixels(a, a.Regions[p[0]], a.Regions[p[1]]) {
			t.Errorf("%s and %s are identical", p[0], p[1])
		}
	}
}

func samePixels(a Atlas, x, y Region) bool {
	for yy := 0; yy < x.H; yy++ {
		for xx := 0; xx < x.W; xx++ {
			if a.Image.NRGBAAt(x.X+xx, x.Y+yy) != a.Image.NRGBAAt(y.X+xx, y.Y+yy) {
				return false
			}
		}
	}
	return true
}

func TestDarkVariantIsDimmer(t *testing.T) {
	a := built().Buildings
	for _, k := range BuildingKinds {
		if luma(a, a.Regions[buildingName(k, "dark")]) >= luma(a, a.Regions[buildingName(k, "ready")]) {
			t.Errorf("%s: dark variant is not dimmer than ready", k)
		}
	}
}

func luma(a Atlas, r Region) float64 {
	sum, n := 0.0, 0
	for y := r.Y; y < r.Y+r.H; y++ {
		for x := r.X; x < r.X+r.W; x++ {
			if c := a.Image.NRGBAAt(x, y); c.A > 0 {
				sum += float64(grey(c).R)
				n++
			}
		}
	}
	return sum / float64(max(n, 1))
}

func TestUISlicesFit(t *testing.T) {
	for n, r := range built().UI.Regions {
		if r.Slice <= 0 || 2*r.Slice >= r.W+1 || 2*r.Slice >= r.H+1 {
			t.Errorf("%s: slice %d does not fit %dx%d", n, r.Slice, r.W, r.H)
		}
	}
}

func TestPaletteHelpers(t *testing.T) {
	c := color.NRGBA{200, 100, 50, 255}
	if got := mix(c, ink, 0); got != c {
		t.Errorf("mix at 0 = %v", got)
	}
	if got := withAlpha(c, 10).A; got != 10 {
		t.Errorf("withAlpha = %d", got)
	}
	if d := darken(c, 0.5); d.R != 100 {
		t.Errorf("darken = %v", d)
	}
	if g := grey(c); g.R != g.G || g.G != g.B {
		t.Errorf("grey = %v", g)
	}
}

func TestAppIcon(t *testing.T) {
	img, again := AppIcon(), AppIcon()
	if img.Bounds().Dx() != AppIconSize || img.Bounds().Dy() != AppIconSize {
		t.Fatalf("AppIcon is %v, want %dx%d", img.Bounds(), AppIconSize, AppIconSize)
	}
	a := built().Icons
	r := a.Regions[AppIconName]
	for y := 0; y < AppIconSize; y++ {
		for x := 0; x < AppIconSize; x++ {
			if img.NRGBAAt(x, y) != again.NRGBAAt(x, y) || img.NRGBAAt(x, y) != a.Image.NRGBAAt(r.X+x, r.Y+y) {
				t.Fatalf("AppIcon differs at %d,%d between calls or from the atlas", x, y)
			}
		}
	}
	for _, p := range [][2]int{{0, 0}, {AppIconSize - 1, 0}, {0, AppIconSize - 1}, {AppIconSize - 1, AppIconSize - 1}} {
		if img.NRGBAAt(p[0], p[1]).A != 0 {
			t.Errorf("corner %v is not transparent", p)
		}
	}
	for y := 4; y < AppIconSize-4; y++ {
		for x := 4; x < AppIconSize-4; x++ {
			if img.NRGBAAt(x, y).A != 255 {
				t.Fatalf("centre pixel %d,%d is not opaque", x, y)
			}
		}
	}
	if img.NRGBAAt(15, 15) != tkRed || img.NRGBAAt(16, 15) != tkRed {
		t.Errorf("hub is not red: %v", img.NRGBAAt(15, 15))
	}
}
