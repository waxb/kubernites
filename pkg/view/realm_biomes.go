package view

import (
	"math"
	"math/rand/v2"
	"slices"
)

const shallowsTop = 0.07

const rockHeight = 0.3

const forestCell = 6

const rockCell = 5

const meadowCell = 7

type biome struct {
	names      []string
	beachTiles float64
	forest     float64
	rock       float64
	meadow     float64
	hillArea   int
	leyLines   int
	wells      int
}

var homeBiome = biome{names: []string{"Hearthmoor", "Lanternholm", "Kingsmere", "Emberwick"},
	beachTiles: 1.3, forest: 0.14, rock: 0.04, meadow: 0.35, hillArea: 450, leyLines: 2, wells: 3}

var outerBiomes = []biome{
	{names: []string{"Oakhollow", "Wildwood", "Brackenfold"}, beachTiles: 1.2, forest: 0.5, rock: 0.03, meadow: 0.3, hillArea: 300, leyLines: 1, wells: 1},
	{names: []string{"Highcairn", "Stonemoor", "Wyrmcrag"}, beachTiles: 1, forest: 0.08, rock: 0.22, meadow: 0.3, hillArea: 120, leyLines: 2, wells: 3},
	{names: []string{"Ashcombe", "Eldmere", "Greendown"}, beachTiles: 1.3, forest: 0.08, rock: 0.01, meadow: 0.6, hillArea: 450, leyLines: 2, wells: 2},
	{names: []string{"Saltmere", "Gullsreach", "Sandholm"}, beachTiles: 2.6, forest: 0.06, rock: 0.01, meadow: 0.3, hillArea: 600, leyLines: 1, wells: 1},
}

type cuts struct {
	beach, forest, rock, meadow float64
}

type fields struct {
	forest, rock, meadow noise
}

func newFields(rng *rand.Rand, span int) fields {
	return fields{forest: newNoise(rng, forestCell, span), rock: newNoise(rng, rockCell, span), meadow: newNoise(rng, meadowCell, span)}
}

func (f fields) at(i, w int) (float64, float64, float64) {
	x, y := float64(i%w), float64(i/w)
	return f.forest.at(x, y), f.rock.at(x, y), f.meadow.at(x, y)
}

func (sp isleSpec) cutsFor(inland []int, height []float64, f fields, w int) cuts {
	c := cuts{beach: sp.beach()}
	rocky := []float64{}
	for _, i := range inland {
		if _, rv, _ := f.at(i, w); height[i] > rockHeight {
			rocky = append(rocky, rv)
		}
	}
	c.rock = quantile(rocky, sp.biome.rock*float64(len(inland))/float64(max(len(rocky), 1)))
	wooded, open := []float64{}, []float64{}
	for _, i := range inland {
		if fv, rv, _ := f.at(i, w); height[i] <= rockHeight || rv <= c.rock {
			wooded = append(wooded, fv)
		}
	}
	c.forest = quantile(wooded, sp.biome.forest*float64(len(inland))/float64(max(len(wooded), 1)))
	for _, i := range inland {
		if fv, rv, mv := f.at(i, w); (height[i] <= rockHeight || rv <= c.rock) && fv <= c.forest {
			open = append(open, mv)
		}
	}
	c.meadow = quantile(open, sp.biome.meadow)
	return c
}

func quantile(values []float64, share float64) float64 {
	if len(values) == 0 || share <= 0 {
		return math.Inf(1)
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	k := int(float64(len(sorted)) * (1 - min(share, 1)))
	return sorted[min(k, len(sorted)-1)]
}

func classify(h, forest, rock, meadow float64, c cuts) TileKind {
	switch {
	case h < 0:
		return TileWater
	case h < shallowsTop:
		return TileShallows
	case h < c.beach:
		return TileSand
	case rock > c.rock && h > rockHeight:
		return TileRock
	case forest > c.forest:
		return TileForest
	case meadow > c.meadow:
		return TileMeadow
	}
	return TileGrass
}
