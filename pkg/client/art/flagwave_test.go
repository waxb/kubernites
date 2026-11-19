package art

import (
	"image/color"
	"testing"
)

func TestBuildingsRecordWhereTheirFlagsFly(t *testing.T) {
	regions := built().Buildings.Regions
	r, ok := regions[buildingName("gate", "ready")]
	if !ok || len(r.Flags) == 0 {
		t.Fatalf("the gate's flags are not recorded: %+v", r.Flags)
	}
	if dark := regions[BuildingLookName("gate", "dark", 1, 0)]; len(dark.Flags) != 0 {
		t.Fatal("a dark building's flags hang still")
	}
	for _, f := range r.Flags {
		x, y := f.X+r.AnchorX, f.Y+r.AnchorY
		if x < -2 || y < -2 || x+FlagClothW > r.W+2 || y+FlagClothH > r.H+2 {
			t.Errorf("flag %+v falls outside its %dx%d sprite", f, r.W, r.H)
		}
		if f.Light == (color.NRGBA{}) || f.Dark == (color.NRGBA{}) {
			t.Errorf("flag %+v has no cloth colours", f)
		}
	}
}

func TestAKeepRecordsItsFlags(t *testing.T) {
	_, r := KeepSprite(KeepLookOf("citadel", "balanced", "citadel-keep-1", 1), "ready")
	if len(r.Flags) == 0 {
		t.Fatal("a ready citadel records no flag")
	}
	if _, dark := KeepSprite(KeepLookOf("citadel", "balanced", "citadel-keep-1", 1), "dark"); len(dark.Flags) != 0 {
		t.Fatal("a dark keep's flags hang still")
	}
}

func TestFlagClothFramesWaveInTheClothsOwnColours(t *testing.T) {
	light, dark := rgb(200, 60, 50), rgb(150, 40, 35)
	seen := map[string]bool{}
	for f := range FlagFrames {
		img := FlagClothFrame(light, dark, f)
		if b := img.Bounds(); b.Dx() != FlagClothW || b.Dy() != FlagClothH {
			t.Fatalf("frame %d is %v", f, b)
		}
		key := ""
		painted := 0
		for y := 0; y < FlagClothH; y++ {
			for x := 0; x < FlagClothW; x++ {
				c := img.NRGBAAt(x, y)
				if c.A == 0 {
					key += "."
					continue
				}
				painted++
				key += "#"
				if c != light && c != dark {
					t.Fatalf("frame %d paints %v, not the cloth", f, c)
				}
			}
		}
		if painted < 30 {
			t.Fatalf("frame %d paints only %d pixels", f, painted)
		}
		seen[key] = true
	}
	if len(seen) < FlagFrames/2 {
		t.Fatalf("only %d distinct shapes in %d frames: the cloth does not wave", len(seen), FlagFrames)
	}
}
