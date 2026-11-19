package art

import (
	"image"
	"testing"
)

func overHut(b Atlas, qos string, age int, layers ...string) sheetCell {
	return sheetCell{a: b, r: b.Regions[HearthLookName(1, "ready", qos, age, 0)], layers: layers}
}

func art2VaultRows(b Atlas) [][]sheetCell {
	rows := [][]sheetCell{}
	for _, st := range VaultStates {
		names := []string{}
		for l := 0; l < VaultShelfLevels; l++ {
			names = append(names, VaultName(st, l))
		}
		rows = append(rows, cellsOf(b, names, true))
	}
	links := []string{MountDotName()}
	for _, red := range []bool{false, true} {
		for p := 0; p < ChainPhases; p++ {
			links = append(links, ChainLinkName(red, p))
		}
	}
	return append(rows, cellsOf(b, links, true))
}

func art2ProbeRows(s Set) [][]sheetCell {
	b, rows := s.Buildings, [][]sheetCell{}
	for _, q := range HearthQoS {
		row := []sheetCell{}
		for age := 0; age < AgeTiers; age++ {
			row = append(row, overHut(b, q, age, ShutterName(q, age, false)), overHut(b, q, age, ShutterName(q, age, true)))
			for f := 0; f < BellFrames; f++ {
				row = append(row, overHut(b, q, age, BellName(q, age, f)))
			}
		}
		rows = append(rows, row)
	}
	marks := []string{}
	for f := 0; f < HourglassFrames; f++ {
		marks = append(marks, HourglassName(f))
	}
	for n := 1; n <= KnockMax; n++ {
		marks = append(marks, KnockName(n))
	}
	walk := []string{}
	for _, dir := range WalkerDirs {
		for f := 0; f < 4; f++ {
			walk = append(walk, walkerName("watchman", "walk", dir, f))
		}
	}
	return append(rows, append(cellsOf(b, marks, true), cellsOf(s.Walkers, walk, true)...))
}

func art2IncidentRows(s Set) [][]sheetCell {
	b := s.Buildings
	badges := []string{}
	for _, n := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 19, 21, 42, 77, 99} {
		badges = append(badges, RestartBadgeName(n))
	}
	rings := []string{}
	for st := 0; st <= len(BackoffWaits); st++ {
		rings = append(rings, BackoffRingName(st))
	}
	for f := 0; f < IncidentFrames; f++ {
		rings = append(rings, IncidentMarkerName(f))
	}
	icons := []string{}
	for _, k := range ClueKinds {
		icons = append(icons, ClueIconName(k, false), ClueIconName(k, true))
	}
	for _, k := range TimelineKinds {
		icons = append(icons, TimelineIconName(k))
	}
	chests := []string{}
	for _, k := range TreasureKinds {
		chests = append(chests, TreasureChestName(k, false), TreasureChestName(k, true))
	}
	return [][]sheetCell{cellsOf(b, badges, true), cellsOf(b, rings, true), cellsOf(s.Icons, icons, false), cellsOf(b, chests, true)}
}

func art2Rows(s Set) [][]sheetCell {
	rows := append(art2VaultRows(s.Buildings), art2ProbeRows(s)...)
	return append(rows, art2IncidentRows(s)...)
}

func TestWriteArt2Sheets(t *testing.T) {
	dir := sheetDir(t)
	rows := art2Rows(built())
	writeScaled(t, dir, "art2-day", rows, grassBackground, 1, 2)
	for _, k := range []int{1, 2} {
		writeSheet(t, dir, "art2-night-"+string(rune('0'+k))+"x.png", nightTint(drawSheet(rows, k, grassBackground)))
	}
	writeSheet(t, dir, "art2-scene-1x.png", art2Scene(built()))
	writeSheet(t, dir, "art2-scene-2x.png", upscale(art2Scene(built()), 2))
	writeSheet(t, dir, "art2-scene-night-2x.png", upscale(nightTint(art2Scene(built())), 2))
}

func art2Scene(s Set) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 300, 150))
	for y := 0; y < 150; y++ {
		for x := 0; x < 300; x++ {
			img.SetNRGBA(x, y, grassBackground(x, y))
		}
	}
	b := s.Buildings
	place(img, b, HearthLookName(2, "guttering", "freeman", 1, 0), 30, 76)
	place(img, b, RestartBadgeName(4), 30, 22)
	place(img, b, BackoffRingName(3), 48, 26)
	place(img, b, IncidentMarkerName(1), 14, 34)
	place(img, b, HearthLookName(1, "ready", "vagrant", 1, 0), 90, 76, ShutterName("vagrant", 1, false))
	place(img, b, HourglassName(1), 90, 34)
	place(img, b, HearthLookName(1, "ready", "vagrant", 1, 1), 124, 76, ShutterName("vagrant", 1, true))
	place(img, b, HearthLookName(1, "ready", "landed", 2, 0), 176, 76, BellName("landed", 2, 1))
	place(img, s.Walkers, walkerName("watchman", "walk", "sw", 1), 164, 80)
	place(img, b, KnockName(2), 176, 26)
	place(img, b, TreasureChestName("dispatch", true), 214, 72)
	place(img, b, TreasureChestName("sealed", false), 236, 72)
	for i, st := range VaultStates {
		place(img, b, VaultName(st, 2), 40+i*56, 142)
	}
	stamp(img, b, 50, 118, 110, 96, ChainStep, func(i int) string { return ChainLinkName(false, i) })
	stamp(img, b, 240, 134, 290, 100, ChainStep, func(i int) string { return ChainLinkName(true, i) })
	return img
}
