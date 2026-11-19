package art

import "testing"

func TestWriteDepotSheets(t *testing.T) {
	dir := sheetDir(t)
	s := built()
	for stem, kinds := range map[string][]string{"helpers": HelperKinds} {
		rows := [][]sheetCell{}
		for _, k := range kinds {
			names := []string{}
			for _, st := range BuildingStates {
				names = append(names, buildingName(k, st))
			}
			rows = append(rows, cellsOf(s.Buildings, names, true))
		}
		writeScaled(t, dir, stem, rows, grassBackground, 1, 2)
	}
	faces := [][]sheetCell{
		cellsOf(s.Portraits, prefixed("portrait/", TradeRoles), false),
		cellsOf(s.Portraits, prefixed("portrait/", append(append([]string{}, StallKinds...), HelperKinds...)), false),
	}
	writeScaled(t, dir, "freight-portraits", faces, grassBackground, 2)
}
