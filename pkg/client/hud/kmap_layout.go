package hud

import (
	"slices"
	"sort"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	KMapTile      = gfx.KMapCell
	PlaqueTiles   = 9
	plaqueInset   = 4
	colTiles      = 12
	eraPadTiles   = 2
	EmptyEraTiles = 16
	SlotTiles     = 3
	RowPadTiles   = 1
	minRowTiles   = 4
	KmapRowsTiles = 28
	PlaqueW       = PlaqueTiles * KMapTile
	PlaqueH       = gfx.PlaqueHeight
)

var stackCaps = []int{2, 1}

type KMapPlaque struct {
	Page   view.GrimoirePage
	Branch int
	Era    int
	Col    int
	Row    int
	Fog    bool
}

func (p KMapPlaque) Rect() ui.Rect {
	return ui.Rect{X: p.Col * KMapTile, Y: p.Row*KMapTile + plaqueInset, W: PlaqueW, H: PlaqueH}
}

func (p KMapPlaque) RoadRow() int { return p.Row + 1 }

type KMapSpan struct{ Start, Size int }

type KMapLayout struct {
	Plaques []KMapPlaque
	Eras    []KMapSpan
	Rows    []KMapSpan
	Lanes   []int
	Width   int
	Height  int
}

func (l KMapLayout) Find(id string) (KMapPlaque, bool) {
	for _, p := range l.Plaques {
		if p.Page.ID == id {
			return p, true
		}
	}
	return KMapPlaque{}, false
}

func (l KMapLayout) Empty(era int) bool {
	for _, p := range l.Plaques {
		if p.Era == era {
			return false
		}
	}
	return true
}

type kmapNode struct {
	page   view.GrimoirePage
	fog    bool
	branch int
	era    int
	depth  int
	col    int
	slot   int
}

func LayoutKnowledge(pages []view.GrimoirePage, heightTiles int) KMapLayout {
	nodes := visibleNodes(pages)
	index := make(map[string]int, len(nodes))
	for i, n := range nodes {
		index[n.page.ID] = i
	}
	placeBranchesAndEras(nodes, index)
	measureDepths(nodes, index)
	var cols []int
	var rows []int
	for _, limit := range stackCaps {
		cols = assignColumns(nodes, index, limit)
		rows = rowSlots(nodes)
		if tilesOf(rows) <= heightTiles {
			break
		}
	}
	return position(nodes, cols, rows, heightTiles)
}

func visibleNodes(pages []view.GrimoirePage) []kmapNode {
	needed := map[string]bool{}
	for _, p := range pages {
		if p.State != "hidden" {
			for _, req := range p.Requires {
				needed[req] = true
			}
		}
	}
	out := []kmapNode{}
	for _, p := range pages {
		switch {
		case p.State != "hidden":
			out = append(out, kmapNode{page: p})
		case needed[p.ID] || p.Opens != "":
			out = append(out, kmapNode{page: p, fog: true})
		}
	}
	return out
}

func placeBranchesAndEras(nodes []kmapNode, index map[string]int) {
	const unset = -2
	for i := range nodes {
		nodes[i].branch, nodes[i].era = unset, unset
	}
	var resolve func(i int, seen map[int]bool)
	resolve = func(i int, seen map[int]bool) {
		n := &nodes[i]
		if n.branch != unset || seen[i] {
			return
		}
		seen[i] = true
		branch, era := slices.Index(view.KnowledgeBranches, n.page.Branch), slices.Index(view.KnowledgeEras, n.page.Era)
		for _, req := range n.page.Requires {
			j, ok := index[req]
			if !ok || j == i {
				continue
			}
			resolve(j, seen)
			if nodes[j].branch >= 0 && branch < 0 {
				branch = nodes[j].branch
			}
			era = max(era, nodes[j].era)
		}
		if branch < 0 {
			branch = len(view.KnowledgeBranches) - 1
		}
		n.branch, n.era = branch, max(era, 0)
	}
	for i := range nodes {
		resolve(i, map[int]bool{})
	}
}

func measureDepths(nodes []kmapNode, index map[string]int) {
	done := make([]bool, len(nodes))
	var visit func(i int, seen map[int]bool) int
	visit = func(i int, seen map[int]bool) int {
		if done[i] || seen[i] {
			return nodes[i].depth
		}
		seen[i] = true
		d := 0
		for _, req := range nodes[i].page.Requires {
			if j, ok := index[req]; ok && j != i {
				d = max(d, visit(j, seen)+1)
			}
		}
		nodes[i].depth, done[i] = d, true
		return d
	}
	for i := range nodes {
		visit(i, map[int]bool{})
	}
}

func placementOrder(nodes []kmapNode) []int {
	order := make([]int, len(nodes))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		na, nb := nodes[order[a]], nodes[order[b]]
		if na.era != nb.era {
			return na.era < nb.era
		}
		return na.depth < nb.depth
	})
	return order
}

func assignColumns(nodes []kmapNode, index map[string]int, limit int) []int {
	cols := make([]int, len(view.KnowledgeEras))
	counts := map[[3]int]int{}
	placed := make([]bool, len(nodes))
	for _, i := range placementOrder(nodes) {
		n := &nodes[i]
		c := 0
		for _, req := range n.page.Requires {
			if j, ok := index[req]; ok && placed[j] && nodes[j].era == n.era {
				c = max(c, nodes[j].col+1)
			}
		}
		for counts[[3]int{n.era, n.branch, c}] >= limit {
			c++
		}
		key := [3]int{n.era, n.branch, c}
		n.col, n.slot = c, counts[key]
		counts[key]++
		cols[n.era] = max(cols[n.era], c+1)
		placed[i] = true
	}
	return cols
}

func rowSlots(nodes []kmapNode) []int {
	rows := make([]int, len(view.KnowledgeBranches))
	for i := range rows {
		rows[i] = 1
	}
	for _, n := range nodes {
		rows[n.branch] = max(rows[n.branch], n.slot+1)
	}
	return rows
}

func rowTiles(slots int) int {
	return max(slots*SlotTiles+RowPadTiles, minRowTiles)
}

func tilesOf(rows []int) int {
	total := 0
	for _, s := range rows {
		total += rowTiles(s)
	}
	return total
}

func position(nodes []kmapNode, cols, rows []int, heightTiles int) KMapLayout {
	l := KMapLayout{Height: heightTiles}
	for _, c := range cols {
		size := EmptyEraTiles
		if c > 0 {
			size = eraPadTiles + c*colTiles
		}
		l.Eras = append(l.Eras, KMapSpan{Start: l.Width, Size: size})
		l.Width += size
	}
	l.Rows, l.Lanes = spreadRows(rows, heightTiles)
	stacks := map[[3]int]int{}
	for _, n := range nodes {
		stacks[[3]int{n.era, n.branch, n.col}]++
	}
	for _, n := range nodes {
		row := l.Rows[n.branch]
		offset := (row.Size - stacks[[3]int{n.era, n.branch, n.col}]*SlotTiles) / 2
		l.Plaques = append(l.Plaques, KMapPlaque{
			Page: n.page, Branch: n.branch, Era: n.era, Fog: n.fog,
			Col: l.Eras[n.era].Start + eraPadTiles + n.col*colTiles,
			Row: row.Start + offset + n.slot*SlotTiles,
		})
	}
	return l
}

func spreadRows(rows []int, heightTiles int) ([]KMapSpan, []int) {
	spare := max(heightTiles-tilesOf(rows), 0)
	spans, lanes := []KMapSpan{}, []int{}
	y := 0
	for i, s := range rows {
		size := rowTiles(s) + spare/len(rows)
		if i < spare%len(rows) {
			size++
		}
		spans = append(spans, KMapSpan{Start: y, Size: size})
		lanes = append(lanes, y+size-1)
		y += size
	}
	return spans, lanes
}
