package view

import (
	"math"
	"math/rand/v2"
)

const PlazaRadius = 3

const TowerClearing = 7

const plazaFromGate = 13

const TowerFromPlaza = 6

const worldSalt = 0x6c8e9cf570932bd5

const worldStream = 0x9e3779b97f4a7c15

type noise struct {
	grid [][]float64
	cell float64
}

func newNoise(rng *rand.Rand, cell float64, span int) noise {
	n := int(math.Ceil(float64(span)/cell)) + 2
	grid := make([][]float64, n)
	for y := range grid {
		grid[y] = make([]float64, n)
		for x := range grid[y] {
			grid[y][x] = rng.Float64()
		}
	}
	return noise{grid: grid, cell: cell}
}

func smooth(t float64) float64 {
	return t * t * (3 - 2*t)
}

func (n noise) at(x, y float64) float64 {
	limit := float64(len(n.grid) - 2)
	gx, gy := min(max(x/n.cell, 0), limit), min(max(y/n.cell, 0), limit)
	x0, y0 := int(gx), int(gy)
	tx, ty := smooth(gx-float64(x0)), smooth(gy-float64(y0))
	a, b := n.grid[y0][x0], n.grid[y0][x0+1]
	c, d := n.grid[y0+1][x0], n.grid[y0+1][x0+1]
	top := a + (b-a)*tx
	bottom := c + (d-c)*tx
	return top + (bottom-top)*ty
}

type Landmarks struct {
	GateX, GateY   int `json:"-"`
	PlazaX, PlazaY int `json:"-"`
	TowerX, TowerY int `json:"-"`
}

func WorldRNG(seed uint64) *rand.Rand {
	return rand.New(rand.NewPCG(seed^worldSalt, seed*worldStream+1))
}

type WorldLayout struct {
	Width, Height int        `json:"-"`
	Home          IsleSlot   `json:"-"`
	Outer         []IsleSlot `json:"-"`
	noSpots       bool
	shaped        bool
}

func GenerateLayout(seed uint64, plan WorldLayout) (Site, Landmarks) {
	rng := WorldRNG(seed)
	n := plan.Width * plan.Height
	site := Site{Width: plan.Width, Height: plan.Height, Tiles: make([]TileKind, n), Elevation: make([]uint8, n)}
	isles := layIslands(rng, plan)
	owner := shapeIslands(&site, rng, isles, plan.shaped)
	marks := placeLandmarks(site, isles[0])
	raiseHills(&site, rng, marks, owner, isles)
	carveCivic(&site, marks)
	sculptTown(&site, marks)
	relax(&site)
	site.Region = regionsOf(site, owner)
	site.Islands = islandsOf(isles)
	site.Gate, site.Plaza, site.TowerSpot = Point{X: marks.GateX, Y: marks.GateY}, Point{X: marks.PlazaX, Y: marks.PlazaY}, Point{X: marks.TowerX, Y: marks.TowerY}
	if !plan.noSpots {
		site.Spots = seedSpots(&site, rng)
	}
	return site, marks
}

func (s Site) span() int {
	return max(s.Width, s.Height)
}

func IsLand(t TileKind) bool {
	return t != TileWater && t != TileShallows
}

func placeLandmarks(site Site, home isleSpec) Landmarks {
	c := home.island.Center
	gateX := c.X - int(home.radius)
	for x := max(0, gateX); x < c.X; x++ {
		if IsLand(site.Tile(x, c.Y)) {
			gateX = x + 1
			break
		}
	}
	plazaX := gateX + plazaFromGate
	return Landmarks{GateX: gateX, GateY: c.Y, PlazaX: plazaX, PlazaY: c.Y, TowerX: plazaX + TowerFromPlaza, TowerY: c.Y}
}

func carveCivic(site *Site, m Landmarks) {
	set := func(x, y int, t TileKind) {
		if i, ok := site.Index(x, y); ok {
			site.Tiles[i] = t
			site.Elevation[i] = 0
		}
	}
	for y := m.TowerY - TowerClearing; y <= m.TowerY+TowerClearing; y++ {
		for x := m.TowerX - TowerClearing; x <= m.TowerX+TowerClearing; x++ {
			if dx, dy := x-m.TowerX, y-m.TowerY; dx*dx+dy*dy <= TowerClearing*TowerClearing {
				set(x, y, clearGround(site.Tile(x, y)))
			}
		}
	}
	for y := m.PlazaY - PlazaRadius; y <= m.PlazaY+PlazaRadius; y++ {
		for x := m.PlazaX - PlazaRadius; x <= m.PlazaX+PlazaRadius; x++ {
			set(x, y, TilePlaza)
		}
	}
	for x := m.GateX - 1; x < m.PlazaX-PlazaRadius; x++ {
		set(x, m.GateY, TileRoad)
	}
}

func clearGround(t TileKind) TileKind {
	switch t {
	case TileForest, TileRock, TileWater, TileShallows, TileSand:
		return TileGrass
	}
	return t
}
