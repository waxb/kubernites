package view

import (
	"fmt"
	"math/rand/v2"
)

const spotYard = 2

const spotApart = 5

const spotShore = 8

const spotTownGap = 3

const spotJitter = 0.8

const spotLean = 3.0

const spotSalt = 0x3c6ef372fe94f82b

const townMargin = 3

type Spot struct {
	ID    string `json:"id"`
	Isle  string `json:"isle"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Class string `json:"class"`
}

type spotPlan struct {
	class       string
	cpu, memory int
	exact       bool
}

var BalancedPlan = spotPlan{class: ClassBalanced, cpu: 4, memory: 8, exact: true}

var ComputePlan = spotPlan{class: ClassCompute, cpu: 8, memory: 3}

var MemoryPlan = spotPlan{class: ClassMemory, cpu: 3, memory: 16}

var richPlan = spotPlan{class: ClassRich, cpu: 6, memory: 12}

var leanBalanced = spotPlan{class: ClassBalanced, cpu: 3, memory: 6}

var leanCompute = spotPlan{class: ClassCompute, cpu: 6, memory: 2}

var leanMemory = spotPlan{class: ClassMemory, cpu: 2, memory: 12}

var plansInTurn = [][]spotPlan{
	{BalancedPlan, ComputePlan, MemoryPlan},
	{MemoryPlan, ComputePlan, BalancedPlan},
	{ComputePlan, MemoryPlan, BalancedPlan},
	{MemoryPlan, BalancedPlan, ComputePlan},
	{leanBalanced, leanCompute, leanMemory},
	{leanMemory, leanCompute, leanBalanced},
	{leanCompute, leanMemory, leanBalanced},
	{leanMemory, leanBalanced, leanCompute},
}

type Span struct {
	Lo, Hi int `json:"-"`
}

type SpotSeeder struct {
	site    *Site
	rng     *rand.Rand
	sea     []int
	owner   []int16
	frozen  []bool
	stone   KeepType
	hands   []Span
	lore    []Span
	classes [][]string
	out     []Spot
}

type SpotLand struct {
	FrozenWood, frozenStone int `json:"-"`
	Wood, Stone             int `json:"-"`
	Ground, beach, overlap  int `json:"-"`
}

type spotFit struct {
	c                   Point
	AddWood, AddStone   int `json:"-"`
	DropWood, dropStone int `json:"-"`
	Cost                int `json:"-"`
}

func NewSpotSeeder(site *Site, rng *rand.Rand) *SpotSeeder {
	stone, _ := KeepTier(StructKeep, false)
	sd := &SpotSeeder{site: site, rng: rng, sea: seaDistance(*site), owner: site.OwnerGrid(), frozen: make([]bool, len(site.Tiles)), stone: stone, out: []Spot{}}
	sd.hands = spans(stone.MaxCPU, func(v int) int { return HandsFrom(v, stone) })
	sd.lore = spans(stone.MaxMemoryGiB, func(v int) int { return GibFrom(v, stone) })
	sd.classes = make([][]string, len(sd.hands))
	for c := range sd.hands {
		sd.classes[c] = make([]string, len(sd.lore))
		for g := range sd.lore {
			if Shortfall(Survey{CPU: c, MemoryGiB: g}, stone) == "" {
				sd.classes[c][g] = ClassOf(c, g, stone)
			}
		}
	}
	return sd
}

func spans(most int, of func(int) int) []Span {
	out := make([]Span, most+1)
	for k := range out {
		out[k] = Span{Lo: -1, Hi: -1}
	}
	for v := 0; out[most].Lo < 0; v++ {
		k := of(v)
		if out[k].Lo < 0 {
			out[k].Lo = v
		}
		out[k].Hi = v
	}
	out[most].Hi = -1
	return out
}

func seedSpots(site *Site, rng *rand.Rand) []Spot {
	sd := NewSpotSeeder(site, rng)
	for i := range site.Islands {
		sd.plantGuaranteed(i)
	}
	for left, order := 1+rng.IntN(2), rng.Perm(max(len(site.Islands)-1, 0)); left > 0 && len(order) > 0; order = order[1:] {
		if sd.plant(order[0]+1, richPlan) {
			left--
		}
	}
	if len(site.Islands) > 0 {
		sd.plant(0, BalancedPlan)
	}
	return sd.out
}

func (sd *SpotSeeder) plantGuaranteed(isle int) {
	for _, turn := range plansInTurn {
		trial := sd.fork()
		planted := 0
		for _, plan := range turn {
			planted += btoi(trial.plant(isle, plan))
		}
		if planted == len(turn) {
			sd.site.Tiles, sd.site.Elevation, sd.frozen, sd.out = trial.site.Tiles, trial.site.Elevation, trial.frozen, trial.out
			return
		}
	}
	for _, plan := range plansInTurn[0] {
		sd.plant(isle, plan)
	}
}

func (sd *SpotSeeder) fork() *SpotSeeder {
	site := *sd.site
	site.Tiles = append([]TileKind(nil), sd.site.Tiles...)
	site.Elevation = append([]uint8(nil), sd.site.Elevation...)
	next := *sd
	next.site, next.frozen, next.out = &site, append([]bool(nil), sd.frozen...), append([]Spot(nil), sd.out...)
	return &next
}

func (sd *SpotSeeder) plant(isle int, plan spotPlan) bool {
	is := sd.site.Islands[isle]
	fit, ok := sd.pick(uint8(isle+1), plan)
	if !ok {
		return false
	}
	sd.paint(fit)
	sd.out = append(sd.out, Spot{ID: fmt.Sprintf("spot-%s-%d", is.ID, len(sd.out)+1), Isle: is.ID, X: fit.c.X - 1, Y: fit.c.Y - 1, Class: plan.class})
	return true
}

func (sd *SpotSeeder) pick(region uint8, plan spotPlan) (spotFit, bool) {
	best, bestScore, found := spotFit{}, 0.0, false
	for i, reg := range sd.site.Region {
		if reg != region {
			continue
		}
		fit, ok := sd.fits(Point{X: i % sd.site.Width, Y: i / sd.site.Width}, region, plan)
		if !ok {
			continue
		}
		if score := sd.rng.Float64()*spotLean - float64(fit.Cost); !found || score > bestScore {
			best, bestScore, found = fit, score, true
		}
	}
	return best, found
}

func (sd *SpotSeeder) fits(c Point, region uint8, plan spotPlan) (spotFit, bool) {
	if !sd.clearOfOthers(c, region) || !sd.footing(c, region) {
		return spotFit{}, false
	}
	land, blocked := sd.survey(c)
	if blocked {
		return spotFit{}, false
	}
	fit, ok := sd.Shape(land, plan)
	fit.c, fit.Cost = c, fit.Cost+land.overlap
	return fit, ok
}

func (sd *SpotSeeder) survey(c Point) (SpotLand, bool) {
	land, blocked := SpotLand{}, false
	sd.site.EachInReach(c, sd.stone.Territory, func(i, _ int) {
		x, y, t := i%sd.site.Width, i/sd.site.Width, sd.site.Tiles[i]
		blocked = blocked || sd.owner[i] != 0 || sd.underBuilding(x, y)
		ring := max(Abs(x-c.X), Abs(y-c.Y))
		switch {
		case sd.frozen[i]:
			land.overlap++
			land.FrozenWood, land.frozenStone = land.FrozenWood+btoi(t == TileForest), land.frozenStone+btoi(t == TileRock)
		case ring <= 1:
		case t == TileForest || t == TileRock:
			land.Wood, land.Stone = land.Wood+btoi(t == TileForest), land.Stone+btoi(t == TileRock)
		case ring > spotYard && paintableTile(t):
			land.Ground++
		case ring > spotYard && t == TileSand:
			land.beach++
		}
	})
	return land, blocked
}

func (sd *SpotSeeder) Shape(land SpotLand, plan spotPlan) (spotFit, bool) {
	best, found := spotFit{}, false
	wood, stone := land.Wood+land.FrozenWood, land.Stone+land.frozenStone
	for c, hs := range sd.hands {
		for g, ls := range sd.lore {
			if hs.Lo < 0 || ls.Lo < 0 || sd.classes[c][g] != plan.class || !plan.admits(c, g) {
				continue
			}
			w, s := Within(wood, hs), Within(stone, ls)
			if w < land.FrozenWood || s < land.frozenStone {
				continue
			}
			fit := spotFit{AddWood: max(0, w-wood), AddStone: max(0, s-stone), DropWood: max(0, wood-w), dropStone: max(0, stone-s)}
			fit.Cost = fit.AddWood + fit.AddStone + fit.DropWood + fit.dropStone
			roomy := fit.AddWood <= land.Ground && fit.AddWood+fit.AddStone <= land.Ground+land.beach
			if roomy && (!found || fit.Cost < best.Cost) {
				best, found = fit, true
			}
		}
	}
	return best, found
}

func (p spotPlan) admits(cpu, memory int) bool {
	if p.exact {
		return cpu == p.cpu && memory == p.memory
	}
	return cpu >= p.cpu && memory >= p.memory
}

func Within(v int, s Span) int {
	if v < s.Lo {
		return s.Lo
	}
	if s.Hi >= 0 && v > s.Hi {
		return s.Hi
	}
	return v
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (sd *SpotSeeder) clearOfOthers(c Point, region uint8) bool {
	ci, _ := sd.site.Index(c.X, c.Y)
	if region == 1 && (sd.sea[ci] < spotShore+sd.stone.Territory || Distance2(c, sd.site.TowerSpot) < sd.townGap()*sd.townGap()) {
		return false
	}
	for _, s := range sd.out {
		if max(Abs(c.X-s.X-1), Abs(c.Y-s.Y-1)) < spotApart {
			return false
		}
	}
	return true
}

func (sd *SpotSeeder) townGap() int {
	return TowerTerritory + sd.stone.Territory + spotTownGap
}

func (sd *SpotSeeder) footing(c Point, region uint8) bool {
	low, high := sd.footprintLevels(c)
	for y := c.Y - spotYard; y <= c.Y+spotYard; y++ {
		for x := c.X - spotYard; x <= c.X+spotYard; x++ {
			i, ok := sd.site.Index(x, y)
			if !ok || sd.site.Region[i] != region || !yardTile(sd.site.Tiles[i]) {
				return false
			}
			inside := Abs(x-c.X) <= 1 && Abs(y-c.Y) <= 1
			if inside && sd.frozen[i] && (!SiteBuildable(sd.site.Tiles[i], "") || sd.site.Elevation[i] != low) {
				return false
			}
		}
	}
	return high-low <= 1
}

func (sd *SpotSeeder) underBuilding(x, y int) bool {
	for _, s := range sd.site.Structures {
		if Footprint(s).Contains(x, y) {
			return true
		}
	}
	_, ok := sd.site.HavenOver(Rect{X0: x, Y0: y, X1: x, Y1: y})
	return ok
}

func yardTile(t TileKind) bool {
	return t == TileGrass || t == TileMeadow || t == TileForest || t == TileRock || t == TileSand
}

func paintableTile(t TileKind) bool {
	return t == TileGrass || t == TileMeadow || t == TileForest || t == TileRock
}

func Distance2(a, b Point) int {
	return (a.X-b.X)*(a.X-b.X) + (a.Y-b.Y)*(a.Y-b.Y)
}

func (s Site) WithSpots(seed uint64) Site {
	if len(s.Spots) > 0 || len(s.Tiles) == 0 || len(s.Region) != len(s.Tiles) || len(s.Elevation) != len(s.Tiles) {
		return s
	}
	next := s
	next.Tiles = append([]TileKind(nil), s.Tiles...)
	next.Elevation = append([]uint8(nil), s.Elevation...)
	next.Spots = seedSpots(&next, rand.New(rand.NewPCG(seed^spotSalt, seed*worldStream+3)))
	next.Roads, next.Land = nil, nil
	return next
}
