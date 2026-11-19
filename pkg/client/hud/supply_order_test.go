package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheSupplyIsTheSameWhicheverOrderTheNodesComeIn(t *testing.T) {
	for i := 0; i < 200; i++ {
		nodes := map[string]view.NodeMetrics{
			"a": {CPURequestedMilli: 100, CPUAllocatableMilli: 100, MemoryRequestedBytes: 100 << 20, MemoryAllocatableBytes: 300 << 20},
			"b": {CPURequestedMilli: 200, CPUAllocatableMilli: 200, MemoryRequestedBytes: 200 << 20, MemoryAllocatableBytes: 300 << 20},
			"c": {CPURequestedMilli: 300, CPUAllocatableMilli: 300, MemoryRequestedBytes: 300 << 20, MemoryAllocatableBytes: 300 << 20},
		}
		hands, lore := SupplyOf(nodes)
		if hands.Used != float64(600)/1000 || hands.Total != float64(600)/1000 {
			t.Fatalf("pass %d: hands %v/%v, want 0.6/0.6 whatever the order", i, hands.Used, hands.Total)
		}
		if lore.Used != float64(600<<20)/gib || lore.Total != float64(900<<20)/gib {
			t.Fatalf("pass %d: lore %v/%v", i, lore.Used, lore.Total)
		}
	}
}
