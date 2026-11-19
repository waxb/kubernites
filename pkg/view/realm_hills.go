package view

import (
	"math"
	"math/rand/v2"
)

const hillMinR = 3

const hillSpanR = 4

const HillMaxPeak = 3

const hillTries = 300

const rimJitter = 1.6

const rimCell = 8

type hillock struct {
	x, y, radius, peak float64
}

func raiseHills(site *Site, rng *rand.Rand, m Landmarks, owner []uint8, isles []isleSpec) {
	rim := newNoise(rng, rimCell, site.span())
	hills := make([][]hillock, len(isles))
	for i, sp := range isles {
		hills[i] = pickHills(*site, rng, m, sp, owner, uint8(i+1))
	}
	for i, id := range owner {
		if id == 0 || !hillGround(site.Tiles[i]) {
			continue
		}
		site.Elevation[i] = uint8(hillLevel(hills[id-1], i%site.Width, i/site.Width, rim))
	}
}

func hillLevel(hills []hillock, x, y int, rim noise) int {
	level := 0
	jitter := (rim.at(float64(x), float64(y)) - 0.5) * rimJitter
	for _, h := range hills {
		d := max(0, math.Hypot(float64(x)-h.x, float64(y)-h.y)+jitter)
		if d < h.radius {
			level = max(level, int(math.Ceil(h.peak*(1-d/h.radius))))
		}
	}
	return level
}

func landOf(owner []uint8, id uint8) int {
	n := 0
	for _, o := range owner {
		if o == id {
			n++
		}
	}
	return n
}

func pickHills(site Site, rng *rand.Rand, m Landmarks, sp isleSpec, owner []uint8, id uint8) []hillock {
	want := landOf(owner, id) / sp.biome.hillArea
	box := sp.bounds(site.Width, site.Height)
	out := []hillock{}
	for tries := 0; len(out) < want && tries < hillTries; tries++ {
		x, y := box.X0+rng.IntN(box.X1-box.X0+1), box.Y0+rng.IntN(box.Y1-box.Y0+1)
		r := float64(hillMinR + rng.IntN(hillSpanR))
		i := y*site.Width + x
		if owner[i] != id || !hillGround(site.Tiles[i]) || id == 1 && nearCivic(x, y, r, m) {
			continue
		}
		peak := math.Min(HillMaxPeak, math.Max(1, math.Round(r/2)))
		out = append(out, hillock{x: float64(x), y: float64(y), radius: r, peak: peak})
	}
	return out
}

func hillGround(t TileKind) bool {
	return IsLand(t) && t != TileSand && t != TileRoad && t != TilePlaza
}

func nearCivic(x, y int, r float64, m Landmarks) bool {
	dist := func(ax, ay int) float64 { return math.Hypot(float64(x-ax), float64(y-ay)) }
	return dist(m.TowerX, m.TowerY) < r+TowerClearing || dist(m.PlazaX, m.PlazaY) < r+PlazaRadius+2 || dist(m.GateX, m.GateY) < r+3
}

func relax(site *Site) {
	for changed := true; changed; {
		changed = false
		for y := 0; y < site.Height; y++ {
			for x := 0; x < site.Width; x++ {
				i := y*site.Width + x
				limit := lowestNeighbour(*site, x, y) + 1
				if !hillGround(site.Tiles[i]) {
					limit = 0
				}
				if site.Elevation[i] > limit {
					site.Elevation[i] = limit
					changed = true
				}
			}
		}
	}
}

func lowestNeighbour(site Site, x, y int) uint8 {
	low := uint8(math.MaxUint8)
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, ny := x+d[0], y+d[1]
		if nx < 0 || ny < 0 || nx >= site.Width || ny >= site.Height {
			return 0
		}
		low = min(low, site.ElevationAt(nx, ny))
	}
	return low
}
