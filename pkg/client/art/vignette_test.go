package art

import (
	"image"
	"slices"
	"sync"
	"testing"
)

var (
	vgSheetOnce sync.Once
	vgSheet     *Sheet
	vgAll       = []string{
		SceneOOMKilled, SceneCrashLoop, SceneImagePull, SceneEviction, SceneStorm, SceneTremor, SceneUnschedulable,
		SceneDrought, SceneAdoption, SceneDiskPressure, SceneFloorLimit, SceneNoSpace, ScenePreempted, SceneNoDepot,
		SceneNoCrew, SceneSpoiling, SceneQuayFull, SceneRushTide, SceneContract, SceneAgeUp, SceneReforge, SceneIncident,
		SceneEagerWatchman,
	}
	vgWidths = []int{VignetteMinW, 300, 328, 340, 388}
)

func testSheet(t *testing.T) *Sheet {
	t.Helper()
	vgSheetOnce.Do(func() { vgSheet = NewSheet(Build()) })
	return vgSheet
}

func differing(a, b *image.NRGBA) float64 {
	n := 0
	for i := 0; i < len(a.Pix); i += 4 {
		if a.Pix[i] != b.Pix[i] || a.Pix[i+1] != b.Pix[i+1] || a.Pix[i+2] != b.Pix[i+2] {
			n++
		}
	}
	return float64(n) / float64(len(a.Pix)/4)
}

func TestEverySceneIsRegistered(t *testing.T) {
	got := VignetteScenes()
	for _, s := range vgAll {
		if !slices.Contains(got, s) || !HasVignette(s) {
			t.Errorf("scene %s is not registered", s)
		}
	}
	if len(got) != len(vgAll) {
		t.Errorf("registered %v, expected %v", got, vgAll)
	}
	if !HasVignette("contract:ledger:crown") || HasVignette("no-such-woe") || SceneBase("no-depot:crate") != SceneNoDepot {
		t.Error("scene arguments follow a colon")
	}
}

func TestVignettesAreDeterministicAndFillTheirFrame(t *testing.T) {
	sh := testSheet(t)
	for _, s := range vgAll {
		for _, w := range vgWidths {
			a, b := sh.Vignette(s, 42, w, 3), sh.Vignette(s, 42, w, 3)
			if a.Bounds() != image.Rect(0, 0, w, VignetteH) {
				t.Fatalf("%s at %d: bounds %v", s, w, a.Bounds())
			}
			if differing(a, b) != 0 {
				t.Errorf("%s at %d is not deterministic", s, w)
			}
			for i := 3; i < len(a.Pix); i += 4 {
				if a.Pix[i] != 255 {
					t.Fatalf("%s at %d leaves a hole at %d", s, w, i/4)
				}
			}
		}
	}
}

func TestVignettesUseOnlyKnownSprites(t *testing.T) {
	sh := testSheet(t)
	for _, s := range append(append([]string(nil), vgAll...), "contract:crate:crown:sealed", "no-crew:dispatch", "no-depot:ledger") {
		for seed := uint64(1); seed <= 8; seed++ {
			if missing := sh.VignetteMissing(s, seed, 340, int(seed)); len(missing) > 0 {
				t.Errorf("%s seed %d misses %v", s, seed, missing)
			}
		}
		for frame := 0; frame < VignetteFrames; frame++ {
			if missing := sh.VignetteMissing(s, 3, VignetteMaxW, frame); len(missing) > 0 {
				t.Errorf("%s frame %d misses %v", s, frame, missing)
			}
		}
	}
}

func TestSeedsVaryDetailsButKeepThePicture(t *testing.T) {
	sh := testSheet(t)
	for _, s := range vgAll {
		base := sh.Vignette(s, 1, 340, 0)
		varied := false
		for seed := uint64(2); seed <= 6; seed++ {
			d := differing(base, sh.Vignette(s, seed, 340, 0))
			if d > 0.3 {
				t.Errorf("%s seed %d changes %.0f%% of the picture", s, seed, d*100)
			}
			varied = varied || d > 0
		}
		if !varied && s != SceneIncident && s != SceneFloorLimit {
			t.Errorf("%s never varies with its seed", s)
		}
	}
}

func TestEverySceneHasAnIdleLoop(t *testing.T) {
	sh := testSheet(t)
	for _, s := range vgAll {
		first := sh.Vignette(s, 5, 340, 0)
		moved := false
		for f := 1; f < VignetteFrames && !moved; f++ {
			moved = differing(first, sh.Vignette(s, 5, 340, f)) > 0
		}
		if !moved {
			t.Errorf("%s never moves", s)
		}
		if differing(first, sh.Vignette(s, 5, 340, VignetteFrames)) != 0 || differing(first, sh.Vignette(s, 5, 340, -VignetteFrames)) != 0 {
			t.Errorf("%s does not loop", s)
		}
	}
}

func TestUnknownScenesFallBackToTheIncident(t *testing.T) {
	sh := testSheet(t)
	if differing(sh.Vignette("never-heard-of", 9, 300, 1), sh.Vignette(SceneIncident, 9, 300, 1)) != 0 {
		t.Error("an unknown scene must paint the generic incident")
	}
	if VignetteWidth(10) != VignetteMinW || VignetteWidth(5000) != VignetteMaxW || VignetteWidth(300) != 300 {
		t.Error("widths are clamped")
	}
}

func TestThumbnailsStayInsideThePicture(t *testing.T) {
	for _, s := range append(append([]string(nil), vgAll...), "nonsense") {
		for _, w := range vgWidths {
			r := ThumbRect(s, w)
			if r.Dx() != ThumbW || r.Dy() != ThumbH || !r.In(image.Rect(0, 0, VignetteWidth(w), VignetteH)) {
				t.Errorf("%s at %d: thumb %v", s, w, r)
			}
		}
	}
}

func TestSheetKeepsOnlyWhatScenesNeed(t *testing.T) {
	sh := testSheet(t)
	if !sh.Has("walker/reeve/idle/se/0") || !sh.Has("ship/cog/0/sw/0") || sh.Has("kmap/parchment") || sh.Has("page/muster") {
		t.Error("the sheet holds scene sprites and nothing else")
	}
	if sh.Len() == 0 {
		t.Error("the sheet is empty")
	}
	var none *Sheet
	if none.Has("walker/reeve/idle/se/0") || none.Len() != 0 {
		t.Error("a nil sheet is empty")
	}
	img := none.Vignette(SceneStorm, 1, 220, 0)
	if img.Bounds().Dx() != VignetteMinW || len(none.VignetteMissing(SceneStorm, 1, 220, 0)) == 0 {
		t.Error("a nil sheet still paints the backdrop and reports what it lacks")
	}
}

func TestScenesAreRichInColour(t *testing.T) {
	sh := testSheet(t)
	for _, s := range vgAll {
		img := sh.Vignette(s, 2, 340, 0)
		seen := map[[3]uint8]bool{}
		for i := 0; i < len(img.Pix); i += 4 {
			seen[[3]uint8{img.Pix[i], img.Pix[i+1], img.Pix[i+2]}] = true
		}
		if len(seen) < 40 {
			t.Errorf("%s uses only %d colours", s, len(seen))
		}
	}
}
