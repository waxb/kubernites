package view

import (
	"math"
)

const stepRoad = 2

const stepOpen = 3

const stepSand = 4

const stepForest = 8

const stepRock = 14

const stepSea = 120

const stepClimb = 4

const pathBuckets = 256

const NoPath = int32(-1)

var stepCosts = [...]int32{
	TileWater: stepSea, TileShallows: stepSea, TileSand: stepSand, TileGrass: stepOpen, TileMeadow: stepOpen,
	TileForest: stepForest, TileRock: stepRock, TileRoad: stepRoad, TilePlaza: stepRoad, TileForum: stepRoad,
}

type PathGrid struct {
	site    Site
	Blocked []bool `json:"-"`
}

func NewPathGrid(site Site) PathGrid {
	blocked := make([]bool, len(site.Tiles))
	for _, s := range site.Structures {
		fp := Footprint(s)
		for y := fp.Y0; y <= fp.Y1; y++ {
			for x := fp.X0; x <= fp.X1; x++ {
				if i, ok := site.Index(x, y); ok {
					blocked[i] = true
				}
			}
		}
	}
	return PathGrid{site: site, Blocked: blocked}
}

func (g PathGrid) step(from, to int) int32 {
	cost := int32(stepSea)
	if t := int(g.site.Tiles[to]); t < len(stepCosts) {
		cost = stepCosts[t]
	}
	if len(g.site.Elevation) == len(g.site.Tiles) {
		cost += stepClimb * int32(AbsDiff(g.site.Elevation[from], g.site.Elevation[to]))
	}
	return cost
}

func (g PathGrid) neighbours(i int) [4]int {
	w := g.site.Width
	x, y := i%w, i/w
	out := [4]int{-1, -1, -1, -1}
	if x+1 < w {
		out[0] = i + 1
	}
	if y+1 < g.site.Height {
		out[1] = i + w
	}
	if x > 0 {
		out[2] = i - 1
	}
	if y > 0 {
		out[3] = i - w
	}
	return out
}

func (g PathGrid) Search(from int, targets []int) []int32 {
	n := len(g.site.Tiles)
	dist, parent := make([]int32, n), make([]int32, n)
	for i := range dist {
		dist[i], parent[i] = math.MaxInt32, NoPath
	}
	want := map[int]bool{}
	for _, t := range targets {
		want[t] = true
	}
	remaining := len(want)
	buckets := make([][]int32, pathBuckets)
	dist[from], buckets[0] = 0, []int32{int32(from)}
	pending := 1
	for d := int32(0); pending > 0 && remaining > 0; d++ {
		slot := d % pathBuckets
		for k := 0; k < len(buckets[slot]); k++ {
			i := int(buckets[slot][k])
			pending--
			if dist[i] != d {
				continue
			}
			if want[i] {
				remaining--
			}
			for _, j := range g.neighbours(i) {
				if j < 0 || g.Blocked[j] && !want[j] {
					continue
				}
				if nd := d + g.step(i, j); nd < dist[j] {
					dist[j], parent[j] = nd, int32(i)
					buckets[nd%pathBuckets] = append(buckets[nd%pathBuckets], int32(j))
					pending++
				}
			}
		}
		buckets[slot] = buckets[slot][:0]
	}
	return parent
}

func TracePath(parent []int32, from, to int) []int32 {
	if to != from && parent[to] == NoPath {
		return nil
	}
	path := []int32{}
	for i := int32(to); ; i = parent[i] {
		path = append(path, i)
		if int(i) == from {
			return path
		}
	}
}

func doorCandidates(kind string, fp Rect) []Point {
	out := []Point{{X: fp.X1, Y: fp.Y1 + 1}, {X: fp.X1 + 1, Y: fp.Y1 + 1}, {X: fp.X1 + 1, Y: fp.Y1}}
	if kind == StructGate {
		out = []Point{{X: fp.X1 + 1, Y: fp.Y1}, {X: fp.X1 + 1, Y: fp.Y1 + 1}, {X: fp.X1, Y: fp.Y1 + 1}}
	}
	for x := fp.X1 - 1; x >= fp.X0; x-- {
		out = append(out, Point{X: x, Y: fp.Y1 + 1})
	}
	for y := fp.Y1 - 1; y >= fp.Y0; y-- {
		out = append(out, Point{X: fp.X1 + 1, Y: y})
	}
	for x := fp.X0; x <= fp.X1; x++ {
		out = append(out, Point{X: x, Y: fp.Y0 - 1})
	}
	for y := fp.Y0; y <= fp.Y1; y++ {
		out = append(out, Point{X: fp.X0 - 1, Y: y})
	}
	return out
}

func (g PathGrid) Door(s Structure) (int, bool) {
	candidates := doorCandidates(s.Kind, Footprint(s))
	for _, c := range candidates {
		if i, ok := g.site.Index(c.X, c.Y); ok && !g.Blocked[i] && IsLand(g.site.Tiles[i]) {
			return i, true
		}
	}
	for _, c := range candidates {
		if i, ok := g.site.Index(c.X, c.Y); ok {
			return i, true
		}
	}
	return 0, false
}

func AbsDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
