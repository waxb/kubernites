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
	return new(depthSort).blocksOf(items)
}

func (s *depthSort) blocksOf(items []Drawable) []liftBlock {
	s.blocks = s.blocks[:0]
	for _, it := range items {
		if it.box.on {
			s.blocks = append(s.blocks, liftBlock{depth: it.Depth, lo: it.box.lo, hi: it.box.hi, rect: it.box.rect})
		}
	}
	s.lifted = s.lifted[:0]
	for _, k := range s.orderBy(len(s.blocks), func(i int) float64 { return s.blocks[i].depth }) {
		s.lifted = append(s.lifted, s.blocks[k])
	}
	return s.lifted
}

func liftWalkers(items []Drawable, zoom float64) {
	new(depthSort).lift(items, zoom)
}

func (s *depthSort) lift(items []Drawable, zoom float64) {
	blocks := s.blocksOf(items)
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
