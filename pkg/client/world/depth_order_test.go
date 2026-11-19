package world

import (
	"math/rand/v2"
	"sort"
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func depthCrowd(seed uint64, walkers, blocks int) []Drawable {
	rng := rand.New(rand.NewPCG(seed, 1))
	out := make([]Drawable, 0, walkers+blocks)
	for i := 0; i < walkers+blocks; i++ {
		d := Drawable{Depth: float64(rng.IntN(400)) / 4}
		x, y := rng.Float64()*40, rng.Float64()*40
		if i < walkers {
			d.stand = stand(Vec{x, y}, float64(rng.IntN(960)), float64(rng.IntN(540)))
		} else {
			lo := Point{int(x), int(y)}
			d.box = footprint{on: true, lo: lo, hi: Point{lo.X + 2, lo.Y + 2}, rect: ui.Rect{X: rng.IntN(900), Y: rng.IntN(500), W: 60, H: 60}}
		}
		out = append(out, d)
	}
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func liftWalkersByScan(items []Drawable, zoom float64) {
	blocks := []int{}
	for i, it := range items {
		if it.box.on {
			blocks = append(blocks, i)
		}
	}
	sort.SliceStable(blocks, func(a, b int) bool { return items[blocks[a]].Depth < items[blocks[b]].Depth })
	for i := range items {
		w := &items[i]
		for _, j := range blocks {
			b := items[j]
			if w.stand.on && b.Depth > w.Depth && InFront(w.stand.foot, b.box.lo, b.box.hi) && overlapsFigure(b.box.rect, w.stand, zoom) {
				w.Depth = b.Depth + liftStep
			}
		}
	}
}

func TestDepthOrderMatchesTheFullScanAndStableSort(t *testing.T) {
	for seed := uint64(0); seed < 20; seed++ {
		got, want := depthCrowd(seed, 300, 80), depthCrowd(seed, 300, 80)
		for i := range got {
			got[i].SeeThrough, want[i].SeeThrough = i%2 == 0, i%2 == 0
			got[i].Hit, want[i].Hit = &Hit{ID: string(rune('a' + i%26))}, &Hit{ID: string(rune('a' + i%26))}
		}
		liftWalkers(got, 1)
		sortByDepth(got)
		liftWalkersByScan(want, 1)
		sort.SliceStable(want, func(i, j int) bool { return want[i].Depth < want[j].Depth })
		for i := range got {
			if got[i].Depth != want[i].Depth || got[i].stand != want[i].stand || got[i].box != want[i].box || got[i].Hit.ID != want[i].Hit.ID || got[i].SeeThrough != want[i].SeeThrough {
				t.Fatalf("seed %d item %d differs", seed, i)
			}
		}
	}
}

func BenchmarkDepthOrder(b *testing.B) {
	base := depthCrowd(1, 10000, 2000)
	items := make([]Drawable, len(base))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		copy(items, base)
		liftWalkers(items, 1)
		sortByDepth(items)
	}
}
