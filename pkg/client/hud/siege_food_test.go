package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestKeepFoodIsTheSameWhicheverOrderTheNodesComeIn(t *testing.T) {
	for i := 0; i < 200; i++ {
		nodes := map[string]view.NodeMetrics{"a": {CPUAllocatableMilli: 100}, "b": {CPUAllocatableMilli: 200}, "c": {CPUAllocatableMilli: 300}}
		if got := keepFood(nodes); got != float64(600)/1000 {
			t.Fatalf("pass %d: the Keeps feed %v, want 0.6 whatever the order", i, got)
		}
	}
}
