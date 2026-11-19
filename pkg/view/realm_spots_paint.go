package view

import (
	"math"
	"math/rand/v2"
	"sort"
)

var FoundingWood = []Point{{X: 1, Y: -8}, {X: 2, Y: -8}, {X: 3, Y: -8}, {X: 1, Y: -7}, {X: 2, Y: -7}, {X: 3, Y: -7}}

var FoundingRock = []Point{{X: 7, Y: 1}, {X: 8, Y: 1}, {X: 7, Y: 2}, {X: 8, Y: 2}}

type ringTile struct {
	i     int
	angle float64
	beach bool
}

func (sd *SpotSeeder) paint(fit spotFit) {
	c := fit.c
	turn := sd.rng.Float64() * 2 * math.Pi
	forests, rocks, open := []ringTile{}, []ringTile{}, []ringTile{}
	sd.site.EachInReach(c, sd.stone.Territory, func(i, _ int) {
		if sd.frozen[i] {
			return
		}
		sd.frozen[i] = true
		x, y, t := i%sd.site.Width, i/sd.site.Width, sd.site.Tiles[i]
		ring := max(Abs(x-c.X), Abs(y-c.Y))
		tile := ringTile{i: i, angle: math.Atan2(float64(y-c.Y), float64(x-c.X)), beach: t == TileSand}
		switch {
		case ring <= 1:
			if t == TileForest || t == TileRock {
				sd.site.Tiles[i] = TileGrass
			}
		case t == TileForest:
			forests = append(forests, tile)
		case t == TileRock:
			rocks = append(rocks, tile)
		case ring > spotYard && (paintableTile(t) || t == TileSand):
			open = append(open, tile)
		}
	})
	sow(sd.site, sd.rng, forests, turn, fit.DropWood, TileGrass)
	sow(sd.site, sd.rng, rocks, turn+math.Pi, fit.dropStone, TileGrass)
	open = sow(sd.site, sd.rng, open, turn+math.Pi, fit.AddWood, TileForest)
	sow(sd.site, sd.rng, open, turn, fit.AddStone, TileRock)
	sd.flatten(c)
}

func sow(site *Site, rng *rand.Rand, ring []ringTile, toward float64, n int, kind TileKind) []ringTile {
	keys := make([]float64, len(ring))
	order := make([]int, len(ring))
	for k, t := range ring {
		keys[k], order[k] = AngleGap(t.angle, toward)+rng.Float64()*spotJitter, k
		if t.beach && kind == TileForest {
			keys[k] = math.Inf(1)
		}
	}
	sort.SliceStable(order, func(a, b int) bool { return keys[order[a]] < keys[order[b]] })
	taken := make([]bool, len(ring))
	for _, k := range order[:min(n, len(order))] {
		if !math.IsInf(keys[k], 1) {
			site.Tiles[ring[k].i], taken[k] = kind, true
		}
	}
	rest := make([]ringTile, 0, len(ring))
	for k, t := range ring {
		if !taken[k] {
			rest = append(rest, t)
		}
	}
	return rest
}

func AngleGap(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 2*math.Pi)
	return min(d, 2*math.Pi-d)
}

func (sd *SpotSeeder) footprintLevels(c Point) (uint8, uint8) {
	low, high := uint8(math.MaxUint8), uint8(0)
	for y := c.Y - 1; y <= c.Y+1; y++ {
		for x := c.X - 1; x <= c.X+1; x++ {
			level := sd.site.ElevationAt(x, y)
			low, high = min(low, level), max(high, level)
		}
	}
	return low, high
}

func (sd *SpotSeeder) flatten(c Point) {
	low, high := sd.footprintLevels(c)
	if low == high {
		return
	}
	for y := c.Y - 1; y <= c.Y+1; y++ {
		for x := c.X - 1; x <= c.X+1; x++ {
			if i, ok := sd.site.Index(x, y); ok {
				sd.site.Elevation[i] = low
			}
		}
	}
	relax(sd.site)
}

func seaDistance(s Site) []int {
	dist := make([]int, len(s.Tiles))
	queue := []int{}
	for i, t := range s.Tiles {
		dist[i] = -1
		if !IsLand(t) {
			dist[i] = 0
			queue = append(queue, i)
		}
	}
	for head := 0; head < len(queue); head++ {
		i := queue[head]
		x, y := i%s.Width, i/s.Width
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			if j, ok := s.Index(x+d[0], y+d[1]); ok && dist[j] < 0 {
				dist[j] = dist[i] + 1
				queue = append(queue, j)
			}
		}
	}
	return dist
}

func sculptTown(site *Site, m Landmarks) {
	t := Point{X: m.TowerX, Y: m.TowerY}
	reach := TowerTerritory + townMargin
	for dy := -reach; dy <= reach; dy++ {
		for dx := -reach; dx <= reach; dx++ {
			if i, ok := site.Index(t.X+dx, t.Y+dy); ok && WithinReach(dx, dy, reach) && (site.Tiles[i] == TileForest || site.Tiles[i] == TileRock) {
				site.Tiles[i] = TileGrass
			}
		}
	}
	plantTown(site, t, FoundingWood, TileForest)
	plantTown(site, t, FoundingRock, TileRock)
}

func plantTown(site *Site, t Point, offsets []Point, kind TileKind) {
	for _, o := range offsets {
		if i, ok := site.Index(t.X+o.X, t.Y+o.Y); ok && paintableTile(site.Tiles[i]) {
			site.Tiles[i] = kind
		}
	}
}
