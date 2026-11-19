package world

import "slices"

func sortByDepth(items []Drawable) {
	copy(items, depthOrder(items, func(d Drawable) float64 { return d.Depth }))
}

func depthOrder[T any](items []T, depth func(T) float64) []T {
	order := make([]int, len(items))
	for i := range order {
		order[i] = i
	}
	slices.SortFunc(order, func(a, b int) int {
		da, db := depth(items[a]), depth(items[b])
		switch {
		case da < db:
			return -1
		case db < da:
			return 1
		}
		return a - b
	})
	out := make([]T, len(items))
	for i, k := range order {
		out[i] = items[k]
	}
	return out
}
