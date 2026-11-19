package world

import (
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/client/render"
)

func TestWornGroundIsDrawnBackToFrontWhateverTheMapOrder(t *testing.T) {
	worn := map[Point]float64{}
	for x := 0; x < 6; x++ {
		for y := 0; y < 6; y++ {
			worn[Point{x, y}] = 1
		}
	}
	first := wornInView(worn, 1, 1, 4, 4)
	if len(first) != 16 {
		t.Fatalf("%d worn tiles in view, want 16", len(first))
	}
	for i := 1; i < len(first); i++ {
		a, b := first[i-1], first[i]
		if render.DepthKey(float64(a.X), float64(a.Y)) >= render.DepthKey(float64(b.X), float64(b.Y)) {
			t.Fatalf("tile %v is drawn after %v, which lies in front of it", b, a)
		}
	}
	for pass := 0; pass < 50; pass++ {
		if got := wornInView(worn, 1, 1, 4, 4); !slices.Equal(got, first) {
			t.Fatalf("pass %d drew the worn ground in another order: %v, then %v", pass, first, got)
		}
	}
}
