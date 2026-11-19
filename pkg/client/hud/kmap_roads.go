package hud

import (
	"sort"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

var roadStates = []string{"locked", "open", "learned"}

type KMapRoadTile struct {
	X, Y       int
	State      string
	N, E, S, W bool
}

func (t KMapRoadTile) Mask() string { return art.KMapRoadMask(t.N, t.E, t.S, t.W) }

type roadCell struct{ x, y int }

func satisfied(p view.GrimoirePage) bool {
	return p.State == "unlocked" || p.Rank > 0
}

func RoadState(q, p view.GrimoirePage) string {
	switch {
	case satisfied(q) && satisfied(p):
		return "learned"
	case satisfied(q):
		return "open"
	}
	return "locked"
}

func KnowledgeRoads(l KMapLayout) []KMapRoadTile {
	grid := map[string]map[roadCell]*KMapRoadTile{}
	for _, st := range roadStates {
		grid[st] = map[roadCell]*KMapRoadTile{}
	}
	for _, p := range l.Plaques {
		for _, req := range p.Page.Requires {
			q, ok := l.Find(req)
			if !ok || p.Col < q.Col+colTiles {
				continue
			}
			pave(grid[RoadState(q.Page, p.Page)], roadPath(l, q, p))
		}
	}
	return flatten(grid)
}

func roadPath(l KMapLayout, q, p KMapPlaque) []roadCell {
	sx, sy := q.Col+PlaqueTiles, q.RoadRow()
	jx, ey := p.Col-2, p.RoadRow()
	if rowClear(l, sy, sx, jx) {
		return walk(roadCell{sx, sy}, roadCell{jx, sy}, roadCell{jx, ey}, roadCell{p.Col - 1, ey})
	}
	lane := l.Lanes[q.Branch]
	if p.Branch < q.Branch {
		lane = l.Lanes[p.Branch]
	}
	return walk(roadCell{sx, sy}, roadCell{sx, lane}, roadCell{jx, lane}, roadCell{jx, ey}, roadCell{p.Col - 1, ey})
}

func rowClear(l KMapLayout, y, x0, x1 int) bool {
	for _, p := range l.Plaques {
		if y >= p.Row && y < p.Row+SlotTiles && p.Col <= x1 && p.Col+PlaqueTiles > x0 {
			return false
		}
	}
	return true
}

func walk(points ...roadCell) []roadCell {
	out := []roadCell{points[0]}
	for _, to := range points[1:] {
		at := out[len(out)-1]
		for at != to {
			at = roadCell{at.x + sign0(to.x-at.x), at.y + sign0(to.y-at.y)}
			out = append(out, at)
		}
	}
	return out
}

func sign0(v int) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return 0
}

func pave(layer map[roadCell]*KMapRoadTile, path []roadCell) {
	tile := func(c roadCell) *KMapRoadTile {
		if t, ok := layer[c]; ok {
			return t
		}
		t := &KMapRoadTile{X: c.x, Y: c.y}
		layer[c] = t
		return t
	}
	tile(path[0]).W = true
	tile(path[len(path)-1]).E = true
	for i := 1; i < len(path); i++ {
		a, b := tile(path[i-1]), tile(path[i])
		switch {
		case b.X > a.X:
			a.E, b.W = true, true
		case b.X < a.X:
			a.W, b.E = true, true
		case b.Y > a.Y:
			a.S, b.N = true, true
		case b.Y < a.Y:
			a.N, b.S = true, true
		}
	}
}

func flatten(grid map[string]map[roadCell]*KMapRoadTile) []KMapRoadTile {
	out := []KMapRoadTile{}
	for _, st := range roadStates {
		layer := make([]KMapRoadTile, 0, len(grid[st]))
		for _, t := range grid[st] {
			tile := *t
			tile.State = st
			layer = append(layer, tile)
		}
		sort.Slice(layer, func(i, j int) bool {
			if layer[i].Y != layer[j].Y {
				return layer[i].Y < layer[j].Y
			}
			return layer[i].X < layer[j].X
		})
		out = append(out, layer...)
	}
	return out
}
