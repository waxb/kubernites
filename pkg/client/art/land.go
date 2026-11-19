package art

import "image"

type Ground uint8

const (
	GroundWater Ground = iota
	GroundShallows
	GroundSand
	GroundGrass
	GroundMeadow
	GroundForest
	GroundRock
	GroundRoad
	GroundPlaza
)

const (
	ChunkTiles = 16
	MaxLevel   = 3
	MaxLift    = MaxLevel * ElevationStep
	RoadLevels = 3
	seaReach   = 6
	openSea    = 3
)

type Land struct {
	W, H   int
	Ground []Ground
	Height []uint8
	Road   []uint8
	Wild   []bool
	Power  []uint8
	depth  []uint8
	links  [][]int32
	segs   []segment
}

func NewLand(w, h int) *Land {
	n := w * h
	return &Land{W: w, H: h, Ground: make([]Ground, n), Height: make([]uint8, n), Road: make([]uint8, n), Wild: make([]bool, n), Power: make([]uint8, n)}
}

func (l *Land) index(x, y int) (int, bool) {
	if x < 0 || y < 0 || x >= l.W || y >= l.H {
		return 0, false
	}
	return y*l.W + x, true
}

func (l *Land) groundAt(x, y int) Ground {
	if i, ok := l.index(x, y); ok {
		return l.Ground[i]
	}
	return GroundWater
}

func (l *Land) heightAt(x, y int) uint8 {
	if i, ok := l.index(x, y); ok {
		return min(l.Height[i], MaxLevel)
	}
	return 0
}

func (l *Land) roadAt(x, y int) uint8 {
	if i, ok := l.index(x, y); ok {
		return min(l.Road[i], RoadLevels)
	}
	return 0
}

func (l *Land) depthAt(x, y int) uint8 {
	if i, ok := l.index(x, y); ok && i < len(l.depth) {
		return l.depth[i]
	}
	return seaReach
}

func isWet(g Ground) bool { return g == GroundWater || g == GroundShallows }

func (l *Land) Prepare() {
	l.depth = seaDepths(l)
	l.segs, l.links = roadSegments(l)
}

func seaDepths(l *Land) []uint8 {
	out := make([]uint8, l.W*l.H)
	queue := make([]int, 0, l.W*l.H)
	for i, g := range l.Ground {
		if isWet(g) {
			out[i] = seaReach
			continue
		}
		queue = append(queue, i)
	}
	for head := 0; head < len(queue); head++ {
		i := queue[head]
		x, y := i%l.W, i/l.W
		for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			j, ok := l.index(x+d[0], y+d[1])
			if ok && out[j] > out[i]+1 {
				out[j] = out[i] + 1
				queue = append(queue, j)
			}
		}
	}
	return out
}

func (l *Land) Empty(cx, cy int) bool {
	a := ChunkArea(cx, cy)
	for y := a.Min.Y - 2; y < a.Max.Y+2; y++ {
		for x := a.Min.X - 2; x < a.Max.X+2; x++ {
			if !isWet(l.groundAt(x, y)) || l.depthAt(x, y) < openSea {
				return false
			}
		}
	}
	return true
}

func ChunkArea(cx, cy int) image.Rectangle {
	return image.Rect(cx*ChunkTiles, cy*ChunkTiles, (cx+1)*ChunkTiles, (cy+1)*ChunkTiles)
}

func ChunkBounds(cx, cy int) image.Rectangle {
	x0, y0 := cx*ChunkTiles, cy*ChunkTiles
	n := ChunkTiles
	return image.Rect(16*(x0-y0)-16*n, 8*(x0+y0)-8-MaxLift, 16*(x0-y0)+16*n, 8*(x0+y0)+16*n-8)
}

func Chunks(w, h int) (int, int) {
	return (w + ChunkTiles - 1) / ChunkTiles, (h + ChunkTiles - 1) / ChunkTiles
}
