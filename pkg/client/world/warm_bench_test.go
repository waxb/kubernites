package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/view"
)

const warmFrames = 45

func benchmarkWarmTown(b *testing.B, zoom float64, busy bool) {
	sc, cam, r := busyTown(b, 300, zoom)
	if busy {
		sc.Metrics.Nodes = map[string]view.NodeMetrics{}
		for _, k := range Keeps(sc.Site) {
			sc.Metrics.Nodes[k.Object.Name] = view.NodeMetrics{CPURequestedMilli: 1900, CPUAllocatableMilli: 1900, MemoryRequestedBytes: 3 << 30, MemoryAllocatableBytes: 3 << 30}
		}
	}
	dst := ebiten.NewImage(960, 540)
	for range warmFrames {
		sc.Time += 1.0 / 60
		r.Draw(dst, cam, sc)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.Time += 1.0 / 60
		r.Draw(dst, cam, sc)
		r.DrawLamps(dst, cam, sc)
	}
}

func BenchmarkWarmTown300Far(b *testing.B)       { benchmarkWarmTown(b, .5, false) }
func BenchmarkWarmTown300Near(b *testing.B)      { benchmarkWarmTown(b, 1, false) }
func BenchmarkWarmTown300Close(b *testing.B)     { benchmarkWarmTown(b, 2, false) }
func BenchmarkWarmTown300NearBusy(b *testing.B)  { benchmarkWarmTown(b, 1, true) }
func BenchmarkWarmTown300CloseBusy(b *testing.B) { benchmarkWarmTown(b, 2, true) }
