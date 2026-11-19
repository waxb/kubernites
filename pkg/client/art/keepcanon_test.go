package art

import (
	"strings"
	"testing"
)

func TestCanonicalKeepsReplaceTheOldKeepPainters(t *testing.T) {
	set := built()
	for kind, size := range keepKindSizes {
		k, ok := CanonicalKeep(kind, 2)
		if !ok || k.Size != size || k.Class != "balanced" || k.Age != 2 || k.Seed != NameSeed(KeepPreviewNode(size)) {
			t.Fatalf("%s: canonical %+v", kind, k)
		}
		for _, name := range []string{BuildingLookName(kind, "ready", 3, 0), buildingName(kind, "dark"), buildingName(kind, "construction-1")} {
			r, ok := set.Buildings.Regions[name]
			if want := keepSize(k); !ok || r.W != want.w || r.AnchorY != want.ay {
				t.Errorf("%s: region %+v %v", name, r, ok)
			}
		}
	}
	if _, ok := CanonicalKeep("codex", 1); ok {
		t.Fatal("only Keeps have a canonical look")
	}
	if KeepPreviewNode("tower") != towerNode || KeepPreviewNode("great") != "great-keep-preview" {
		t.Fatal("preview nodes")
	}
}

func TestLegacyKeepSpritesAreTheProceduralKeep(t *testing.T) {
	set := built()
	k, _ := CanonicalKeep("keep-great", 0)
	want, _ := KeepSprite(k, "ready")
	r := set.Buildings.Regions[buildingName("keep-great", "ready")]
	for y := 0; y < r.H; y++ {
		for x := 0; x < r.W; x++ {
			if set.Buildings.Image.NRGBAAt(r.X+x, r.Y+y) != want.NRGBAAt(x, y) {
				t.Fatalf("pixel %d,%d differs from the procedural keep", x, y)
			}
		}
	}
}

func TestKeepPortraitNamesRoundTrip(t *testing.T) {
	k := KeepLookOf("citadel", "rich", "citadel-keep-3", 2)
	name := k.PortraitName()
	back, ok := ParseKeepPortraitName(name)
	if !ok || back != k || !strings.HasPrefix(name, "portrait/keep-citadel-rich/2/") {
		t.Fatalf("%s parsed to %+v %v", name, back, ok)
	}
	for _, bad := range []string{"portrait/keep-stone", "portrait/keep-stone/2", "portrait/keep-stone-rich/2", "portrait/keep-moat-rich/2/00000001",
		"portrait/keep-stone-lava/2/00000001", "portrait/keep-stone-rich/7/00000001", "portrait/keep-stone-rich/x/00000001", "portrait/keep-stone-rich/1/zz",
		"portrait/keepstone/1/00000001", "portrait/codex/2"} {
		if _, ok := ParseKeepPortraitName(bad); ok {
			t.Errorf("%s parsed", bad)
		}
	}
}

func TestKeepPortraitFramesTheKeep(t *testing.T) {
	img, r := KeepPortrait(KeepLookOf("great", "compute", "great-keep-1", 3))
	if r.W != PortraitSize || r.H != PortraitSize || img.Rect.Dx() != PortraitSize || r.AnchorX != PortraitSize/2 {
		t.Fatalf("portrait region %+v", r)
	}
	if keepPixels(img) != PortraitSize*PortraitSize {
		t.Fatal("a portrait fills its framed square")
	}
	plain, _ := KeepPortrait(KeepLookOf("great", "memory", "great-keep-1", 3))
	if keepDiff(img, plain) == 0 {
		t.Fatal("the class shows in the portrait")
	}
}

func TestKeepBuildingPortraitsFollowTheAge(t *testing.T) {
	set := built()
	a, b := set.Portraits.Regions[portraitName("keep-stone")], set.Portraits.Regions[portraitName(AgedPortrait("keep-stone", 3))]
	if a.W == 0 || b.W == 0 {
		t.Fatal("keep portraits missing")
	}
	same := true
	for y := 0; y < a.H && same; y++ {
		for x := 0; x < a.W; x++ {
			if set.Portraits.Image.NRGBAAt(a.X+x, a.Y+y) != set.Portraits.Image.NRGBAAt(b.X+x, b.Y+y) {
				same = false
				break
			}
		}
	}
	if same {
		t.Fatal("the Empire keep portrait must differ from the Dev one")
	}
}

func TestAgedScenesCarryTheirAge(t *testing.T) {
	if got := AgedScene(SceneAgeUp, 9); got != "age-up:age3" {
		t.Fatalf("scene %s", got)
	}
	cases := map[string]int{"age2": 2, "parcel:age1:crown": 1, "": ageDark, "ageless": ageDark, "age-4": ageDark}
	for arg, want := range cases {
		if got := (&vgPaint{arg: arg}).argAge(); got != want {
			t.Errorf("arg %q: age %d, want %d", arg, got, want)
		}
	}
	sh := testSheet(t)
	for age := 0; age < AgeTiers; age++ {
		for _, scene := range []string{SceneAgeUp, SceneReforge, SceneDiskPressure, SceneNoSpace, SceneNoCrew + ":crate"} {
			if missing := sh.VignetteMissing(AgedScene(scene, age), 7, 320, 0); len(missing) > 0 {
				t.Errorf("%s at age %d misses %v", scene, age, missing)
			}
		}
		if !sh.Has(vgKeep(age)) || !sh.Has(vgAgedHut(2, age, 5)) {
			t.Errorf("age %d: the vignette sheet lacks the aged keep or hut", age)
		}
	}
}
