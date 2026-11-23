package world

import "slices"

type depthSort struct {
	keys   []float64
	order  []int
	items  []Drawable
	blocks []liftBlock
	lifted []liftBlock
}

func sortByDepth(items []Drawable) {
	new(depthSort).sort(items)
}

func (s *depthSort) orderBy(n int, key func(i int) float64) []int {
	s.keys, s.order = s.keys[:0], s.order[:0]
	for i := range n {
		s.keys = append(s.keys, key(i))
		s.order = append(s.order, i)
	}
	keys := s.keys
	slices.SortFunc(s.order, func(a, b int) int {
		switch {
		case keys[a] < keys[b]:
			return -1
		case keys[b] < keys[a]:
			return 1
		}
		return a - b
	})
	return s.order
}

func (s *depthSort) sort(items []Drawable) {
	s.items = s.items[:0]
	for _, k := range s.orderBy(len(items), func(i int) float64 { return items[i].Depth }) {
		s.items = append(s.items, items[k])
	}
	copy(items, s.items)
}
