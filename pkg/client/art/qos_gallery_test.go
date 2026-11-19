package art

import (
	"fmt"
	"os"
	"strconv"
	"testing"
)

func hutGalleryRow(a Atlas, tier int, qos string, age int) []sheetCell {
	row := []sheetCell{}
	for v := 0; v < BuildingVariants; v++ {
		row = append(row, sheetCell{a: a, r: a.Regions[HearthLookName(tier, "ready", qos, age, v)]})
	}
	for _, st := range []string{"guttering", "ghost", "kindling", "banished", "pending"} {
		row = append(row, sheetCell{a: a, r: a.Regions[HearthLookName(tier, st, qos, age, 0)]})
	}
	return row
}

func TestHutGallerySheets(t *testing.T) {
	dir := sheetDir(t)
	a := built().Buildings
	tiers := []int{1, 2, 3}
	if n, err := strconv.Atoi(os.Getenv("ART_TIER")); err == nil {
		tiers = []int{n}
	}
	for _, tier := range tiers {
		rows := [][]sheetCell{}
		for _, q := range HearthQoS {
			for age := 0; age < AgeTiers; age++ {
				rows = append(rows, hutGalleryRow(a, tier, q, age))
			}
		}
		writeSheet(t, dir, fmt.Sprintf("huts-tier%d.png", tier), drawSheet(rows, 4, meadowBackground))
		town := [][]sheetCell{}
		for age := 0; age < AgeTiers; age++ {
			row := []sheetCell{}
			for _, q := range HearthQoS {
				for v := 0; v < BuildingVariants; v++ {
					row = append(row, sheetCell{a: a, r: a.Regions[HearthLookName(tier, "ready", q, age, v)]})
				}
			}
			town = append(town, row)
		}
		writeSheet(t, dir, fmt.Sprintf("huts-town-tier%d-1x.png", tier), drawSheet(town, 1, meadowBackground))
		writeSheet(t, dir, fmt.Sprintf("huts-town-tier%d-2x.png", tier), drawSheet(town, 2, meadowBackground))
	}
}
