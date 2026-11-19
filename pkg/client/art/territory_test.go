package art

import (
	"image"
	"slices"
	"testing"
)

func TestBorderMarkersStandOnTheirAnchors(t *testing.T) {
	a := built().Buildings
	for v := 0; v < BorderStones; v++ {
		r := a.Regions[BorderStoneName(v)]
		if r.AnchorX != 6 || r.AnchorY != 13 {
			t.Errorf("stone %d: %+v", v, r)
		}
		if low := solidBounds(a, r).Max.Y - 1; low < r.AnchorY-1 || low > r.AnchorY+1 {
			t.Errorf("stone %d: foot on row %d", v, low)
		}
		for w := v + 1; w < BorderStones; w++ {
			if samePixels(a, r, a.Regions[BorderStoneName(w)]) {
				t.Errorf("stones %d and %d are identical", v, w)
			}
		}
	}
	stone := solidBounds(a, a.Regions[BorderStoneName(0)]).Dy()
	for f := 0; f < BorderFlags; f++ {
		r := a.Regions[BorderFlagName(f)]
		if r.AnchorX != 5 || r.AnchorY != 27 || solidBounds(a, r).Dy() < 2*stone {
			t.Errorf("flag %d: %+v", f, r)
		}
	}
	if samePixels(a, a.Regions[BorderFlagName(0)], a.Regions[BorderFlagName(1)]) {
		t.Error("the flag does not flutter")
	}
}

func TestGroundTintsTileWithoutSeams(t *testing.T) {
	a := built().Terrain
	for _, k := range GroundTints {
		r := a.Regions[GroundTintName(k)]
		if r.W != TileWidth || r.H != TileHeight || r.AnchorX != TileWidth/2 || r.AnchorY != TileHeight/2 {
			t.Fatalf("%s: %+v", k, r)
		}
		cover := map[image.Point]int{}
		for gx := 0; gx < 3; gx++ {
			for gy := 0; gy < 3; gy++ {
				ox, oy := 64+(gx-gy)*16-r.AnchorX, 16+(gx+gy)*8-r.AnchorY
				for y := 0; y < r.H; y++ {
					for x := 0; x < r.W; x++ {
						if c := a.Image.NRGBAAt(r.X+x, r.Y+y); c.A > 0 {
							if c != tintColors[k] {
								t.Fatalf("%s: pixel %v is not the tint", k, c)
							}
							cover[image.Pt(ox+x, oy+y)]++
						}
					}
				}
			}
		}
		for p, n := range cover {
			if n != 1 {
				t.Fatalf("%s: pixel %v is covered %d times", k, p, n)
			}
		}
		for y := 16 + 8; y < 16+32; y++ {
			if cover[image.Pt(64, y)] != 1 || cover[image.Pt(63, y)] != 1 {
				t.Fatalf("%s: gap in the middle column at row %d", k, y)
			}
		}
	}
}

func TestResourceDecorAndIcons(t *testing.T) {
	s := built()
	for _, k := range []string{"outcrop", "quarry"} {
		if !slices.Contains(DecorKinds, k) {
			t.Fatalf("%s is not decor", k)
		}
		for v := 0; v < DecorVariants; v++ {
			for w := v + 1; w < DecorVariants; w++ {
				if samePixels(s.Terrain, s.Terrain.Regions[decorName(k, v)], s.Terrain.Regions[decorName(k, w)]) {
					t.Errorf("%s variants %d and %d are identical", k, v, w)
				}
			}
		}
	}
	names := append(append([]string{}, IconNames...), FreightIcons...)
	for i, x := range FreightIcons {
		r, ok := s.Icons.Regions[iconName(x)]
		if !ok || r.W != IconSize || r.AnchorX != 8 {
			t.Fatalf("icon %s: %+v", x, r)
		}
		for _, y := range names {
			if y != x && samePixels(s.Icons, r, s.Icons.Regions[iconName(y)]) {
				t.Errorf("icons %s and %s are identical", x, y)
			}
		}
		_ = i
	}
}

func TestHelpersMatchTheBuildingStyle(t *testing.T) {
	a := built().Buildings
	for _, k := range HelperKinds {
		if !slices.Contains(BuildingKinds, k) || Footprints[k] < 2 {
			t.Fatalf("%s is not a building kind with a footprint", k)
		}
		r := a.Regions[buildingName(k, "ready")]
		b := contentBounds(a, r)
		if b.Min.Y < 1 || b.Min.X < 0 || b.Max.X > r.W {
			t.Errorf("%s: content %v touches the canvas top", k, b)
		}
		if h := contentHeight(a, r) - Footprints[k]*TileHeight; h > buildingHeight(k)+2 || h < buildingHeight(k)-16 {
			t.Errorf("%s rises %d px over its footprint, declared %d", k, h, buildingHeight(k))
		}
	}
	for i, k := range HelperKinds {
		for _, o := range HelperKinds[i+1:] {
			if Footprints[k] == Footprints[o] && differingShare(a, a.Regions[buildingName(k, "ready")], a.Regions[buildingName(o, "ready")]) < 0.3 {
				t.Errorf("%s and %s look alike", k, o)
			}
		}
	}
	for project, kind := range HelperArt {
		if !slices.Contains(BuildingKinds, kind) && !slices.Contains(StallKinds, kind) {
			t.Errorf("%s maps to unknown building %s", project, kind)
		}
	}
}

func TestNewBuildingsHavePortraits(t *testing.T) {
	a := built().Portraits
	for _, k := range append(append([]string{}, StallKinds...), HelperKinds...) {
		if _, ok := a.Regions[portraitName(k)]; !ok {
			t.Errorf("portrait/%s missing", k)
		}
	}
}

var freightLifts = map[string]int{
	"seal-smithy": 5, "key-gate": 7, "bell-watch": 8, "cache-store": 4, "caravan-yard": 3,
}

func TestFreightBuildingLiftsAreDocumented(t *testing.T) {
	if len(freightLifts) != len(freightHeights) {
		t.Fatalf("%d lifts for %d freight buildings", len(freightLifts), len(freightHeights))
	}
	for kind, px := range freightHeights {
		if _, clash := buildingHeights[kind]; clash {
			t.Errorf("%s is declared twice", kind)
		}
		if want := (px + 7) / 8; freightLifts[kind] != want {
			t.Errorf("%s is %dpx tall: documented lift %d, want %d", kind, px, freightLifts[kind], want)
		}
	}
}
