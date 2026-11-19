package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func TestKeepSmokeFollowsNodeCPU(t *testing.T) {
	cases := []struct {
		used, alloc float64
		want        int
	}{
		{0, 3900, 0}, {100, 3900, 0}, {400, 3900, 1}, {1800, 3900, 2}, {3500, 3900, 3}, {5000, 3900, 3}, {800, 0, 0}, {-5, 3900, 0},
	}
	for _, c := range cases {
		got := KeepSmoke(view.NodeMetrics{CPUMilli: c.used, CPUAllocatableMilli: int64(c.alloc)})
		if got != c.want {
			t.Errorf("KeepSmoke(%v of %v) = %d, want %d", c.used, c.alloc, got, c.want)
		}
	}
}

func TestPodSmokeFollowsPodCPUAndThrottling(t *testing.T) {
	cases := []struct {
		m    view.PodMetrics
		want int
	}{
		{view.PodMetrics{}, 0}, {view.PodMetrics{CPUMilli: 20}, 0}, {view.PodMetrics{CPUMilli: 120}, 1},
		{view.PodMetrics{CPUMilli: 400}, 2}, {view.PodMetrics{CPUMilli: 900}, 3}, {view.PodMetrics{CPUMilli: 10, Throttled: true}, SmokeHeavy},
	}
	for _, c := range cases {
		if got := PodSmoke(c.m); got != c.want {
			t.Errorf("PodSmoke(%+v) = %d, want %d", c.m, got, c.want)
		}
	}
}

func TestHutVentsSitOnTheirChimneys(t *testing.T) {
	plain := HutVent("hearth/1/ready")
	if plain.dy > -30 || plain.dy < -40 || plain.stack {
		t.Errorf("tier 1 vent = %+v", plain)
	}
	if HutVent("hearth/3/ready").dy >= plain.dy {
		t.Error("a taller house vents higher")
	}
	shack := HutVent("hearth/2/ready/vagrant")
	if shack.dy <= HutVent("hearth/2/ready/freeman").dy || shack.dy > -20 {
		t.Errorf("a shack vents from its stovepipe, above its roof and below the cottage chimney: %+v", shack)
	}
	a := art.Build().Buildings
	r := a.Regions["hearth/2/ready/vagrant"]
	x, y := r.X+r.AnchorX+int(shack.dx), r.Y+r.AnchorY+int(shack.dy)
	if a.Image.NRGBAAt(x, y+1).A == 0 && a.Image.NRGBAAt(x, y+2).A == 0 {
		t.Errorf("the shack's smoke starts in thin air at %d,%d", x, y)
	}
	if manor := HutVent("hearth/2/ready/landed"); manor.dy > HutVent("hearth/2/ready/freeman").dy {
		t.Error("the manor chimney stands above the cottage chimney")
	}
	if HutVent("nonsense") != HutVent("hearth/1/ready") {
		t.Error("unknown huts fall back to the tier 1 chimney")
	}
}

func TestKeepVentsCarryAStack(t *testing.T) {
	for _, variant := range []string{"tower", "stone", "great", "citadel", "odd"} {
		v := KeepVent(view.Structure{Kind: view.StructKeep, Variant: variant})
		if !v.stack || v.dy > -60 {
			t.Errorf("%s vent = %+v", variant, v)
		}
	}
}

func TestSmokePuffsScaleWithLevel(t *testing.T) {
	if SmokePuffs(0, false) != 0 || SmokePuffs(1, false) != 1 || SmokePuffs(3, false) != 3 || SmokePuffs(9, false) != 3 {
		t.Error("one puff per smoke level, at most three")
	}
	if SmokePuffs(3, true) != 1 {
		t.Error("reduced effects keep a single calm puff")
	}
}

func TestSmokeRisesFromBusyKeepsAndHuts(t *testing.T) {
	site := lifeSite()
	r := NewRenderer(testLib())
	r.Homes = map[string]Vec{"hot": {12, 12}, "cold": {13, 12}}
	sc := Scene{Site: site, Time: 3, Pods: []model.PodView{
		{Namespace: "default", Name: "hot", State: model.HearthReady, Tier: 1},
		{Namespace: "default", Name: "cold", State: model.HearthReady, Tier: 1},
	}, Metrics: view.MetricsSnapshot{
		Nodes: map[string]view.NodeMetrics{"lone-tower": {CPUMilli: 1900, CPUAllocatableMilli: 1900}},
		Pods:  map[string]view.PodMetrics{"default/hot": {CPUMilli: 150, Throttled: true}},
	}}
	k, _ := FirstOfKind(site, view.StructKeep, false)
	cam := render.NewCamera(480, 270).CenterOn(float64(k.X), float64(k.Y), 960, 540)
	items := r.smokeItems(nil, cam, sc)
	if len(items) != 2 {
		t.Fatalf("plumes = %d, want the keep and the hot hut", len(items))
	}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	for _, it := range items {
		it.Draw(dst)
	}
	sc.Reduce = true
	if len(r.smokeItems(nil, cam, sc)) != 2 {
		t.Error("reduced effects still show the smoke, calmly")
	}
}
