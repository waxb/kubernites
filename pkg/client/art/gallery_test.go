package art

import (
	"fmt"
	"image/color"
	"os"
	"sort"
	"strings"
	"testing"
)

func agedKinds() []string {
	out := []string{}
	for k := range agedPainters {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func galleryKinds() []string {
	only := os.Getenv("ART_KINDS")
	if only == "" {
		return agedKinds()
	}
	return strings.Split(only, ",")
}

func meadowBackground(x, y int) color.NRGBA {
	switch n := hash(x, y, 9) % 23; {
	case n == 0:
		return grassR[0]
	case n < 3:
		return grassR[2]
	}
	return grassR[1]
}

func kindGrid(a Atlas, kind string, state string) [][]sheetCell {
	rows := [][]sheetCell{}
	for age := 0; age < AgeTiers; age++ {
		row := []sheetCell{}
		for v := 0; v < BuildingVariants; v++ {
			if r, ok := a.Regions[BuildingLookName(kind, state, age, v)]; ok {
				row = append(row, sheetCell{a: a, r: r})
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func chunks(kinds []string, n int) [][]string {
	out := [][]string{}
	for len(kinds) > n {
		out, kinds = append(out, kinds[:n]), kinds[n:]
	}
	return append(out, kinds)
}

func variantRows(a Atlas, kinds []string, age int) [][]sheetCell {
	rows := [][]sheetCell{}
	for _, group := range chunks(kinds, 4) {
		row := []sheetCell{}
		for _, kind := range group {
			for v := 0; v < BuildingVariants; v++ {
				row = append(row, sheetCell{a: a, r: a.Regions[BuildingLookName(kind, "ready", age, v)]})
			}
		}
		rows = append(rows, row)
	}
	return rows
}

func ageRows(a Atlas, kinds []string) [][]sheetCell {
	rows := [][]sheetCell{}
	for _, group := range chunks(kinds, 7) {
		for age := 0; age < AgeTiers; age++ {
			row := []sheetCell{}
			for _, kind := range group {
				row = append(row, sheetCell{a: a, r: a.Regions[BuildingLookName(kind, "ready", age, 0)]})
			}
			rows = append(rows, row)
		}
	}
	return rows
}

func TestAgedGallerySheets(t *testing.T) {
	dir := sheetDir(t)
	a := built().Buildings
	kinds := galleryKinds()
	for _, kind := range kinds {
		writeSheet(t, dir, "aged-"+kind+".png", drawSheet(kindGrid(a, kind, "ready"), 3, meadowBackground))
	}
	writeSheet(t, dir, "gallery-ages.png", drawSheet(ageRows(a, kinds), 1, meadowBackground))
	for age := 0; age < AgeTiers; age++ {
		writeSheet(t, dir, fmt.Sprintf("gallery-variants-age%d.png", age), drawSheet(variantRows(a, kinds, age), 1, meadowBackground))
	}
}
