package world

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func BenchmarkDraw150Pods(b *testing.B) {
	sc, cam := sceneFor(b, .9)
	pods := make([]model.PodView, 0, 150)
	for i := 0; i < 150; i++ {
		pods = append(pods, model.PodView{Namespace: "default", Name: fmt.Sprintf("porter-%03d", i), Node: "lone-tower", Image: "porter:1.0", Sprite: "porter", Tier: 1 + i%3, State: model.HearthReady})
	}
	sc.Pods = pods
	if k, ok := FirstOfKind(sc.Site, "keep", false); ok {
		cam = cam.CenterOn(float64(k.X), float64(k.Y), 960, 540)
	}
	r := NewRenderer(gfx.Load(art.Build()))
	dst := ebiten.NewImage(960, 540)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.Time += 1.0 / 60
		r.Draw(dst, cam, sc)
		r.DrawLamps(dst, cam, sc)
	}
}

func busyTown(b *testing.B, n int, zoom float64) (Scene, render.Camera, *Renderer) {
	sc, cam := sceneFor(b, .5)
	pods := make([]model.PodView, 0, n)
	sc.Metrics = view.MetricsSnapshot{Pods: map[string]view.PodMetrics{}}
	keeps := Keeps(sc.Site)
	for i := 0; i < n; i++ {
		p := model.PodView{Namespace: "default", Name: fmt.Sprintf("porter-%03d", i), Node: keeps[i%len(keeps)].Object.Name, Image: "porter:1.0", Sprite: "porter", Tier: 1 + i%3, State: model.HearthReady,
			QoS: model.QoSBurstable, Requests: []model.ContainerRequest{{Name: "porter", CPUMilli: int64(50 + 150*(i%4)), MemoryBytes: int64(32+64*(i%4)) * Mebi}}}
		switch i % 5 {
		case 0:
			p.Familiars = []string{model.FamiliarRaven}
		case 1:
			p.Familiars = []string{model.FamiliarHound}
		case 2:
			p.State = model.HearthGuttering
		}
		sc.Metrics.Pods[PodKey(p)] = view.PodMetrics{ServedRate: float64(i%4) * .3, Throttled: i%3 == 0}
		pods = append(pods, p)
	}
	sc.Pods = pods
	if k, ok := FirstOfKind(sc.Site, "keep", false); ok {
		door := Door(k)
		for i := 0; i < MaxErrands; i++ {
			sc.Errands = append(sc.Errands, Errand{Kind: ErrandSchedule, Role: "quartermaster", From: door, To: Vec{door.X + float64(i%5), door.Y + 2}, Start: sc.Time - float64(i%4), Life: 8})
		}
		cam = cam.CenterOn(float64(k.X), float64(k.Y), 960, 540)
	}
	cam.Zoom = zoom
	return sc, cam, NewRenderer(gfx.Load(art.Build()))
}

func benchmarkTown(b *testing.B, n int, zoom float64) {
	sc, cam, r := busyTown(b, n, zoom)
	dst := ebiten.NewImage(960, 540)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.Time += 1.0 / 60
		r.Draw(dst, cam, sc)
		r.DrawLamps(dst, cam, sc)
	}
}

func BenchmarkDrawTown300Far(b *testing.B)   { benchmarkTown(b, 300, .5) }
func BenchmarkDrawTown300Near(b *testing.B)  { benchmarkTown(b, 300, 1) }
func BenchmarkDrawTown300Close(b *testing.B) { benchmarkTown(b, 300, 2) }

func benchmarkWalkers(b *testing.B, n int, zoom float64) {
	sc, cam, r := busyTown(b, n, zoom)
	dst := ebiten.NewImage(960, 540)
	r.HearthItems(cam, sc)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.Time += 1.0 / 60
		for _, it := range r.WalkerItems(cam, sc) {
			it.Draw(dst)
		}
	}
}

func BenchmarkWalkers300Far(b *testing.B)   { benchmarkWalkers(b, 300, .5) }
func BenchmarkWalkers300Near(b *testing.B)  { benchmarkWalkers(b, 300, 1) }
func BenchmarkWalkers300Close(b *testing.B) { benchmarkWalkers(b, 300, 2) }

func busyHarbor(b *testing.B, zoom float64) (Scene, render.Camera, *Renderer) {
	sc, _ := sceneFor(b, .5)
	sc.Pods, sc.Ghost = nil, nil
	havens := append([]view.Haven(nil), sc.Site.Havens...)
	for i := range havens {
		havens[i].Open, havens[i].Tier, havens[i].Share = true, 3, 1/float64(len(havens))
	}
	sc.Site.Havens = havens
	sc.ArrivalRate, sc.Queue, sc.Surge, sc.GateOpen, sc.Hint, sc.Beckon = 24, 120, true, true, true, true
	sc.RealmTime = sc.Time
	sc.Freight = synthFreight(sc.RealmTime, 3, 20)
	r := NewRenderer(gfx.Load(art.Build()))
	r.HarborOf().Replan(sc.Site)
	q := r.HarborOf().Plans[0].Quay
	cam := render.NewCamera(480, 60)
	cam.Zoom = zoom
	return sc, cam.CenterOn(q.X, q.Y, 960, 540), r
}

func benchmarkHarbor(b *testing.B, zoom float64, draw bool) {
	sc, cam, r := busyHarbor(b, zoom)
	dst := ebiten.NewImage(960, 540)
	warm(r, dst, cam, sc)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sc.Time += 1.0 / 60
		sc.RealmTime += 1.0 / 60
		if i%15 == 0 {
			sc.Freight = synthFreight(sc.RealmTime, 3, 20)
		}
		if draw {
			r.Draw(dst, cam, sc)
			continue
		}
		h := r.HarborOf()
		h.Step(sc)
		r.HarborItems(cam, sc)
	}
}

func BenchmarkHarborLogicFar(b *testing.B)  { benchmarkHarbor(b, .5, false) }
func BenchmarkHarborLogicNear(b *testing.B) { benchmarkHarbor(b, 1, false) }
func BenchmarkHarborDrawFar(b *testing.B)   { benchmarkHarbor(b, .5, true) }
func BenchmarkHarborDrawNear(b *testing.B)  { benchmarkHarbor(b, 1, true) }
