package art

import (
	"image"
	"math"
	"testing"
)

func storageSheetRows(s Set) [][]sheetCell {
	b := s.Buildings
	posts := []string{}
	for _, st := range WardenPostStates {
		posts = append(posts, WardenPostName(st))
	}
	rows := [][]sheetCell{append(cellsOf(b, posts, true), cellsOf(b, []string{StrandedTagName(), DeedTagName()}, true)...)}
	for _, st := range VaultStates {
		names := []string{}
		for l := 0; l < VaultShelfLevels; l++ {
			names = append(names, VaultName(st, l))
		}
		rows = append(rows, cellsOf(b, names, true))
	}
	detach := []sheetCell{}
	for i := 0; i < VaultDetachBanners; i++ {
		detach = append(detach, sheetCell{a: b, r: b.Regions[VaultName("bound", 2)], layers: []string{VaultDetachName(i)}})
	}
	rows = append(rows, detach)
	links := []string{DeedScrollName(true), DeedScrollName(false), MountDotName()}
	for _, red := range []bool{false, true} {
		for p := 0; p < ChainPhases; p++ {
			links = append(links, ChainLinkName(red, p))
		}
	}
	for v := 0; v < LoosePageVariants; v++ {
		links = append(links, LoosePagesName(v))
	}
	rows = append(rows, cellsOf(b, links, false))
	icons := []string{}
	for _, id := range StoragePageIcons {
		icons = append(icons, PageIconName(id))
	}
	return append(rows, cellsOf(s.Icons, icons, false))
}

func nightTint(img *image.NRGBA) *image.NRGBA {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetNRGBA(x, y, dusk(img.NRGBAAt(x, y)))
		}
	}
	return img
}

func TestWriteStorageSheets(t *testing.T) {
	dir := sheetDir(t)
	rows := storageSheetRows(built())
	writeScaled(t, dir, "storage-day", rows, grassBackground, 1, 2)
	for _, k := range []int{1, 2} {
		writeSheet(t, dir, "storage-night-"+string(rune('0'+k))+"x.png", nightTint(drawSheet(rows, k, grassBackground)))
	}
	scene := storageScene(built())
	writeSheet(t, dir, "storage-scene-2x.png", upscale(scene, 2))
	writeSheet(t, dir, "storage-scene-night-2x.png", upscale(nightTint(storageScene(built())), 2))
}

func place(dst *image.NRGBA, a Atlas, name string, x, y int, layers ...string) {
	r, ok := a.Regions[name]
	if !ok {
		return
	}
	blitOver(dst, sheetCell{a: a, r: r, layers: layers}, x-r.AnchorX, y-r.AnchorY)
}

func stamp(dst *image.NRGBA, a Atlas, x0, y0, x1, y1 float64, step int, name func(i int) string) {
	n := int(math.Hypot(x1-x0, y1-y0)) / step
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(max(n, 1))
		place(dst, a, name(i), int(x0+(x1-x0)*t), int(y0+(y1-y0)*t))
	}
}

func storageScene(s Set) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 220, 120))
	for y := 0; y < 120; y++ {
		for x := 0; x < 220; x++ {
			img.SetNRGBA(x, y, grassBackground(x, y))
		}
	}
	b := s.Buildings
	stamp(img, b, 36, 92, 120, 96, ChainStep, func(i int) string { return ChainLinkName(false, i) })
	stamp(img, b, 36, 92, 70, 40, ChainStep, func(i int) string { return ChainLinkName(true, i) })
	place(img, b, VaultName("bound", 2), 30, 100)
	place(img, b, VaultName("released", 0), 70, 44)
	place(img, b, DeedScrollName(true), 78, 94)
	place(img, b, DeedScrollName(false), 54, 66)
	place(img, b, buildingName("keep-stone", "ready"), 140, 110)
	place(img, b, WardenPostName("running"), 124, 110)
	stamp(img, b, 152, 104, 190, 94, MountStep, func(int) string { return MountDotName() })
	place(img, b, hearthQoSName(1, "ready", "freeman"), 196, 96)
	place(img, b, DeedTagName(), 196, 58)
	place(img, b, hearthQoSName(1, "ready", "vagrant"), 176, 62)
	place(img, b, StrandedTagName(), 176, 34)
	place(img, b, LoosePagesName(1), 110, 30)
	place(img, b, VaultName("bound", 1), 100, 118, VaultDetachName(1))
	return img
}
