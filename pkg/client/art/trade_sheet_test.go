package art

import "testing"

func TestWriteTradeSheets(t *testing.T) {
	dir := sheetDir(t)
	a := built().Walkers
	rows := [][]sheetCell{}
	for _, role := range TradeRoles {
		names := []string{}
		for _, action := range append(append([]string{}, WalkerActions...), HoldAction) {
			for f := 0; f < WalkerFrames; f++ {
				names = append(names, walkerName(role, action, "se", f))
			}
		}
		rows = append(rows, cellsOf(a, names, false))
	}
	looks := [][]sheetCell{}
	for _, role := range TradeRoles {
		names := []string{}
		for v := 1; v < WalkerLooks; v++ {
			names = append(names, walkerLookName(role, 1, v, "walk", "se", 1), walkerLookName(role, 1, v, "carry", "sw", 1))
		}
		looks = append(looks, cellsOf(a, names, false))
	}
	writeScaled(t, dir, "trades", rows, grassBackground, 1, 2, 4)
	writeScaled(t, dir, "trade-looks", looks, grassBackground, 2, 4)
	writeScaled(t, dir, "trade-carry", carryRows(a), grassBackground, 1, 4)
}

func carryRows(a Atlas) [][]sheetCell {
	rows := [][]sheetCell{}
	for _, role := range TradeRoles {
		row := []sheetCell{}
		for size := 0; size < BoxSizes; size++ {
			for i, dir := range WalkerDirs {
				kind := BoxKinds[(size+i)%len(BoxKinds)]
				if k := TradeKinds[role]; k != "" && i == 0 {
					kind = k
				}
				name := CarryName(size, kind, dir, 1)
				row = append(row, sheetCell{a: a, r: a.Regions[name], under: []string{walkerName(role, HoldAction, dir, 1)}})
			}
		}
		rows = append(rows, row)
	}
	return rows
}
