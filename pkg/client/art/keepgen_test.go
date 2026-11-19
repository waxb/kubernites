package art

import (
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func keepPixels(img *image.NRGBA) int {
	n := 0
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] > 0 {
			n++
		}
	}
	return n
}

func keepDiff(a, b *image.NRGBA) float64 {
	if a.Rect != b.Rect {
		return 1
	}
	diff, seen := 0, 0
	for i := 0; i < len(a.Pix); i += 4 {
		if a.Pix[i+3] == 0 && b.Pix[i+3] == 0 {
			continue
		}
		seen++
		if a.Pix[i] != b.Pix[i] || a.Pix[i+1] != b.Pix[i+1] || a.Pix[i+2] != b.Pix[i+2] || a.Pix[i+3] != b.Pix[i+3] {
			diff++
		}
	}
	return float64(diff) / float64(max(seen, 1))
}

func assertKeepFits(t *testing.T, k KeepLook, img *image.NRGBA, r Region) {
	t.Helper()
	w, h := img.Rect.Dx(), img.Rect.Dy()
	low, left, right, top := -1, w, -1, h
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if img.NRGBAAt(x, y).A > 96 {
				low, top = max(low, y), min(top, y)
				left, right = min(left, x), max(right, x)
			}
		}
	}
	switch {
	case w != keepFoot*TileWidth+4 || r.AnchorX != w/2 || r.AnchorY != h-2:
		t.Errorf("%s: canvas %dx%d anchor %d,%d", k.Kind(), w, h, r.AnchorX, r.AnchorY)
	case top < 2 || left < 1 || right > w-2:
		t.Errorf("%s age %d: art touches the canvas edge (top %d, left %d, right %d)", k.Kind(), k.Age, top, left, right)
	case low < r.AnchorY-6 || low > r.AnchorY+1:
		t.Errorf("%s age %d: the base rests on row %d, anchor %d", k.Kind(), k.Age, low, r.AnchorY)
	case keepPixels(img) < w*h/6:
		t.Errorf("%s age %d: only %d pixels painted", k.Kind(), k.Age, keepPixels(img))
	}
}

func TestEveryKeepSizeAndClassBuildsDiffersAndFits(t *testing.T) {
	for age := range AgeTiers {
		seen := map[string]*image.NRGBA{}
		for _, sz := range KeepSizes {
			for _, class := range KeepClasses {
				k := KeepLook{Size: sz, Class: class, Age: age, Seed: 0x5eed}
				img, r := KeepSprite(k, "ready")
				assertKeepFits(t, k, img, r)
				for other, prev := range seen {
					if d := keepDiff(img, prev); d < 0.04 {
						t.Errorf("age %d: %s and %s differ in only %.1f%% of pixels", age, k.Kind(), other, d*100)
					}
				}
				seen[k.Kind()] = img
			}
		}
	}
}

func TestKeepsVaryByNameAndStayStable(t *testing.T) {
	a := KeepLookOf("stone", "compute", "stone-keep-1", 2)
	b := KeepLookOf("stone", "compute", "stone-keep-2", 2)
	imgA, _ := KeepSprite(a, "ready")
	again, _ := KeepSprite(a, "ready")
	imgB, _ := KeepSprite(b, "ready")
	if keepDiff(imgA, again) != 0 {
		t.Fatal("the same keep painted twice differs")
	}
	if keepDiff(imgA, imgB) < 0.02 {
		t.Fatal("two keeps with different names look the same")
	}
	unique := map[uint64]bool{}
	for i := range 12 {
		img, _ := KeepSprite(KeepLookOf("great", "rich", fmt.Sprintf("great-keep-%d", i), 3), "ready")
		unique[pixelHash(img)] = true
	}
	if len(unique) < 12 {
		t.Fatalf("12 great keeps show only %d faces", len(unique))
	}
}

func pixelHash(img *image.NRGBA) uint64 {
	h := uint64(14695981039346656037)
	for _, b := range img.Pix {
		h = (h ^ uint64(b)) * 1099511628211
	}
	return h
}

func TestKeepStatesAndNames(t *testing.T) {
	k := KeepLookOf("citadel", "memory", "citadel-keep-1", 1)
	ready, _ := KeepSprite(k, "ready")
	dark, _ := KeepSprite(k, "dark")
	if keepGlow(dark) >= keepGlow(ready) {
		t.Error("a dark keep is not dimmer")
	}
	var prev *image.NRGBA
	for stage := range 3 {
		img, r := KeepSprite(k, stateName(stage))
		if keepPixels(img) == 0 || r.AnchorY != img.Rect.Dy()-2 || keepDiff(img, ready) < 0.1 || prev != nil && keepDiff(img, prev) < 0.1 {
			t.Errorf("construction %d paints %d pixels, too like its neighbours", stage, keepPixels(img))
		}
		prev = img
	}
	for _, state := range BuildingStates {
		name := k.Name(state)
		got, st, ok := ParseKeepName(name)
		if !ok || got != k || st != state {
			t.Errorf("%s parses as %+v %q %v", name, got, st, ok)
		}
	}
	for _, bad := range []string{"building/keep-stone/ready", "building/keep-stone-compute/ready/2", "building/keep-moat-compute/ready/2/0000beef",
		"building/keep-stone-lava/ready/2/0000beef", "building/keep-stone-compute/smashed/2/0000beef", "building/keep-stone-compute/ready/9/0000beef",
		"building/keep-stone-compute/ready/x/0000beef", "building/keep-stone-compute/ready/1/zz", "building/codex/ready/1/1"} {
		if _, _, ok := ParseKeepName(bad); ok {
			t.Errorf("%s parsed", bad)
		}
	}
	if l := KeepLookOf("keep", "odd", "x", 7); l.Size != "stone" || l.Class != "balanced" || l.Age != AgeTiers-1 {
		t.Errorf("fallback look = %+v", l)
	}
}

func keepGlow(img *image.NRGBA) float64 {
	sum, n := 0.0, 0
	for i := 0; i < len(img.Pix); i += 4 {
		if img.Pix[i+3] > 0 {
			sum += float64(img.Pix[i]) + float64(img.Pix[i+1]) + float64(img.Pix[i+2])
			n++
		}
	}
	return sum / float64(max(n, 1))
}

func TestKeepGallerySheet(t *testing.T) {
	dir := os.Getenv("KEEP_GALLERY")
	if dir == "" {
		t.Skip("set KEEP_GALLERY to a directory to write the keep gallery")
	}
	for age := range AgeTiers {
		if err := writeKeepSheet(filepath.Join(dir, fmt.Sprintf("keeps-age%d.png", age)), age, 0x5eed); err != nil {
			t.Fatal(err)
		}
	}
}

func writeKeepSheet(path string, age int, seed uint32) error {
	cellW, cellH := 110, 190
	sheet := image.NewNRGBA(image.Rect(0, 0, cellW*len(KeepClasses), cellH*len(KeepSizes)))
	draw.Draw(sheet, sheet.Rect, image.NewUniform(grassR[1]), image.Point{}, draw.Src)
	for row, sz := range KeepSizes {
		for col, class := range KeepClasses {
			img, _ := KeepSprite(KeepLook{Size: sz, Class: class, Age: age, Seed: seed + uint32(row*7+col)}, "ready")
			at := image.Pt(col*cellW+5, row*cellH+cellH-img.Rect.Dy()-4)
			draw.Draw(sheet, img.Rect.Add(at), img, image.Point{}, draw.Over)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", path, err)
	}
	defer f.Close()
	return png.Encode(f, sheet)
}

func TestKeepSmokeRisesFromItsChimney(t *testing.T) {
	for _, sz := range KeepSizes {
		for _, class := range KeepClasses {
			for age := range AgeTiers {
				k := KeepLook{Size: sz, Class: class, Age: age, Seed: 0xbeef}
				img, r := KeepSprite(k, "ready")
				dx, dy := KeepVent(k)
				x, y := r.AnchorX+int(dx), r.AnchorY+int(dy)
				hit := false
				for yy := y; yy <= y+4; yy++ {
					for xx := x - 2; xx <= x+2; xx++ {
						hit = hit || img.NRGBAAt(xx, yy).A > 0
					}
				}
				if !hit || dy > -20 {
					t.Errorf("%s age %d: smoke at %d,%d leaves no chimney", k.Kind(), age, x, y)
				}
			}
		}
	}
}
