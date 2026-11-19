package world

import (
	"sort"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	liftStep   = .001
	walkerTall = 20.0
)

type standing struct {
	on     bool
	foot   Vec
	sx, sy float64
}

type footprint struct {
	on     bool
	lo, hi Point
	rect   ui.Rect
}

func stand(foot Vec, sx, sy float64) standing { return standing{on: true, foot: foot, sx: sx, sy: sy} }

func InFront(foot Vec, lo, hi Point) bool {
	return foot.X > float64(hi.X)+.5 || foot.Y > float64(hi.Y)+.5
}

func overlapsFigure(r ui.Rect, s standing, zoom float64) bool {
	top := s.sy - walkerTall*zoom
	return s.sx >= float64(r.X) && s.sx <= float64(r.X+r.W) && s.sy >= float64(r.Y) && top <= float64(r.Y+r.H)
}

type liftBlock struct {
	depth  float64
	lo, hi Point
	rect   ui.Rect
}

func liftBlocks(items []Drawable) []liftBlock {
	blocks := []liftBlock{}
	for _, it := range items {
		if it.box.on {
			blocks = append(blocks, liftBlock{depth: it.Depth, lo: it.box.lo, hi: it.box.hi, rect: it.box.rect})
		}
	}
	return depthOrder(blocks, func(b liftBlock) float64 { return b.depth })
}

func liftWalkers(items []Drawable, zoom float64) {
	blocks := liftBlocks(items)
	for i := range items {
		w := &items[i]
		if !w.stand.on {
			continue
		}
		deeper := sort.Search(len(blocks), func(k int) bool { return blocks[k].depth > w.Depth })
		for _, b := range blocks[deeper:] {
			if overlapsFigure(b.rect, w.stand, zoom) && b.depth > w.Depth && InFront(w.stand.foot, b.lo, b.hi) {
				w.Depth = b.depth + liftStep
			}
		}
	}
}
