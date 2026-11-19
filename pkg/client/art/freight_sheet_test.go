package art

import (
	"fmt"
	"testing"
)

func boxRows(a Atlas, spoiled bool) [][]sheetCell {
	rows := [][]sheetCell{}
	for size := 0; size < BoxSizes; size++ {
		names := []string{}
		for _, k := range BoxKinds {
			for v := 0; v < BoxVariants; v++ {
				if spoiled {
					names = append(names, BoxSpoiledName(size, k, v))
				} else {
					names = append(names, BoxName(size, k, v))
				}
			}
		}
		rows = append(rows, cellsOf(a, names, false))
	}
	return rows
}

func markedBox(a Atlas, size int, kind string, v int, layers ...string) sheetCell {
	return sheetCell{a: a, r: a.Regions[BoxName(size, kind, v)], layers: layers}
}

func TestWriteFreightSheets(t *testing.T) {
	dir := sheetDir(t)
	s := built()
	writeScaled(t, dir, "boxes", boxRows(s.Buildings, false), sandBackground, 1, 2, 4)
	writeScaled(t, dir, "boxes-spoiled", boxRows(s.Buildings, true), sandBackground, 2)
	paper := [][]sheetCell{}
	for size := 0; size < BoxSizes; size++ {
		names := []string{}
		for v := 0; v < BoxVariants; v++ {
			names = append(names, PaperBoxName(size, v))
		}
		paper = append(paper, cellsOf(s.Buildings, names, false))
	}
	writeScaled(t, dir, "box-paper", paper, sandBackground, 1, 2, 4)
	marks := [][]sheetCell{}
	for size := 0; size < BoxSizes; size++ {
		row := []sheetCell{}
		for i, k := range BoxKinds {
			row = append(row, markedBox(s.Buildings, size, k, i, BoxSealName(size)))
			row = append(row, markedBox(s.Buildings, size, k, i, BoxCrownName(size)))
			row = append(row, markedBox(s.Buildings, size, k, i, BoxSealName(size), BoxCrownName(size), BoxFreshName(size, i)))
		}
		for l := 0; l < FreshLevels; l++ {
			row = append(row, markedBox(s.Buildings, size, "crate", 0, BoxFreshName(size, l)))
		}
		marks = append(marks, row)
	}
	writeScaled(t, dir, "box-marks", marks, sandBackground, 1, 2, 4)
	piles := [][]sheetCell{}
	for _, k := range BoxKinds {
		names := []string{}
		for l := 0; l < BoxPileLevels; l++ {
			names = append(names, BoxPileName(k, l))
		}
		piles = append(piles, cellsOf(s.Buildings, names, true))
	}
	writeScaled(t, dir, "box-piles", piles, sandBackground, 1, 2)
	spoil := []string{}
	for f := 0; f < SpoilFrames; f++ {
		spoil = append(spoil, effectName("spoil", f))
	}
	writeScaled(t, dir, "fx-spoil", [][]sheetCell{cellsOf(s.Effects, spoil, true)}, sandBackground, 4)
}

func TestWriteFreightScene(t *testing.T) {
	dir := sheetDir(t)
	for f := 0; f < 2; f++ {
		img := freightCanvas()
		drawPlaced(img, built(), freightGround(), freightOX, freightOY)
		drawPlaced(img, built(), freightItems(f), freightOX, freightOY)
		writeSheet(t, dir, fmt.Sprintf("freight-quay-%d-1x.png", f), img)
		writeSheet(t, dir, fmt.Sprintf("freight-quay-%d-2x.png", f), upscale(img, 2))
	}
}
