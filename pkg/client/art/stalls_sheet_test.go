package art

import (
	"image/color"
	"testing"
)

func stallAtlas() Atlas {
	b := &atlasBuilder{}
	addStalls(b)
	return b.build()
}

func pavingBackground(x, y int) color.NRGBA {
	if (x/16+y/8)%2 == 0 {
		return plazaR[2]
	}
	return plazaR[1]
}

func stallRow(a Atlas, kind string, age int) []sheetCell {
	names := []string{}
	for _, face := range StallFaces {
		names = append(names, BuildingLookName(StallArtKind(kind, face), "ready", age, 0))
	}
	sw := StallArtKind(kind, "sw")
	names = append(names, BuildingLookName(sw, "ready", age, 1), BuildingLookName(sw, "ready", age, 2),
		BuildingLookName(sw, "dark", age, 0), BuildingLookName(sw, "construction-1", age, 0), BuildingLookName(sw, "construction-2", age, 0))
	return cellsOf(a, names, true)
}

func TestWriteStallSheets(t *testing.T) {
	dir := sheetDir(t)
	a := stallAtlas()
	for _, kind := range StallKinds {
		rows := [][]sheetCell{}
		for age := 0; age < AgeTiers; age++ {
			rows = append(rows, stallRow(a, kind, age))
		}
		writeScaled(t, dir, kind, rows, pavingBackground, 1, 3)
	}
	all := [][]sheetCell{}
	for age := 0; age < AgeTiers; age++ {
		row := []sheetCell{}
		for _, kind := range StallKinds {
			row = append(row, cellsOf(a, []string{BuildingLookName(StallArtKind(kind, "sw"), "ready", age, 0), BuildingLookName(StallArtKind(kind, "se"), "ready", age, 0)}, false)...)
		}
		all = append(all, row)
	}
	writeScaled(t, dir, "stalls-all", all, pavingBackground, 1, 2)
}
