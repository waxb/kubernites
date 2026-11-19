package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func benchmarkLife(b *testing.B, zoom, phase float64) {
	sc, cam, r := busyTown(b, 300, zoom)
	sc.DayPhase = phase
	dst := ebiten.NewImage(960, 540)
	warm(r, dst, cam, sc)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.Time += 1.0 / 60
		sc.Served += .05
		r.drawLifeGround(dst, cam, sc)
		for _, it := range r.lifeItems(cam, sc) {
			it.Draw(dst)
		}
		r.drawLifeOverlay(dst, cam, sc)
		r.drawLifeGlow(dst, cam, sc)
	}
}

func BenchmarkLife300Far(b *testing.B)   { benchmarkLife(b, .5, .5) }
func BenchmarkLife300Near(b *testing.B)  { benchmarkLife(b, 1, .5) }
func BenchmarkLife300Close(b *testing.B) { benchmarkLife(b, 2, .5) }
func BenchmarkLife300Dawn(b *testing.B)  { benchmarkLife(b, .5, fogPeak) }
func BenchmarkLife300Night(b *testing.B) { benchmarkLife(b, .5, .95) }
