package world

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func archipelago(tb testing.TB) view.Site {
	tb.Helper()
	data, err := os.ReadFile("testdata/archipelago-site.json")
	if err != nil {
		tb.Fatal(err)
	}
	var site view.Site
	if err := json.Unmarshal(data, &site); err != nil {
		tb.Fatal(err)
	}
	return site
}

func overview(tb testing.TB, zoom float64) (Scene, render.Camera, *Renderer) {
	site := archipelago(tb)
	cam := render.NewCamera(480, 60)
	cam.Zoom = zoom
	cam = cam.CenterOn(float64(site.Width)/2, float64(site.Height)/2, 960, 540)
	return Scene{Site: site, DayPhase: .5, Time: 3}, cam, NewRenderer(gfx.Load(art.Build()))
}

func warm(r *Renderer, dst *ebiten.Image, cam render.Camera, sc Scene) {
	for i := 0; i < 2000; i++ {
		r.Draw(dst, cam, sc)
		if len(r.Land.Queue) == 0 && len(r.Land.Flights) == 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func benchmarkOverview(b *testing.B, zoom float64) {
	sc, cam, r := overview(b, zoom)
	dst := ebiten.NewImage(960, 540)
	warm(r, dst, cam, sc)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.Time += 1.0 / 60
		r.Draw(dst, cam, sc)
	}
}

func BenchmarkOverviewFar(b *testing.B)   { benchmarkOverview(b, .5) }
func BenchmarkOverviewNear(b *testing.B)  { benchmarkOverview(b, 1) }
func BenchmarkOverviewClose(b *testing.B) { benchmarkOverview(b, 2) }

func BenchmarkOverviewPan(b *testing.B) {
	sc, cam, r := overview(b, .5)
	dst := ebiten.NewImage(960, 540)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.Time += 1.0 / 60
		r.Draw(dst, cam.Pan(float64(i%240)*8-960, float64(i%120)*4-240), sc)
	}
}

func BenchmarkOverviewFirstFrame(b *testing.B) {
	sc, cam, _ := overview(b, .5)
	lib := gfx.Load(art.Build())
	dst := ebiten.NewImage(960, 540)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewRenderer(lib).Draw(dst, cam, sc)
	}
}

func BenchmarkPaintChunk(b *testing.B) {
	land := LandOf(archipelago(b))
	nx, ny := art.Chunks(land.W, land.H)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		land.Chunk(nx/2, ny/2)
	}
}

func BenchmarkLandOf(b *testing.B) {
	site := archipelago(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LandOf(site)
	}
}

func overviewCam(zoom float64, n int) render.Camera {
	cam := render.NewCamera(480, 60)
	cam.Zoom = zoom
	return cam.CenterOn(float64(n)/2, float64(n)/2, 960, 540)
}
