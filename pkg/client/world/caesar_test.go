package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func TestHeatLevelsAndTints(t *testing.T) {
	cases := map[float64]int{0: 0, .24: 0, .25: 1, .5: 2, .8: 3, .9: 4, 1.3: 4}
	for v, want := range cases {
		if got := HeatLevel(v); got != want {
			t.Errorf("HeatLevel(%v) = %d, want %d", v, got, want)
		}
	}
	if HeatTint(0) == HeatTint(1) || HeatEdge(1).A != 255 {
		t.Error("heat tints must differ, and edges are opaque")
	}
}

func TestLensAreaCoversAKeepsPlot(t *testing.T) {
	keep := view.Structure{Kind: view.StructKeep, X: 10, Y: 10, W: 3, H: 3, Plot: 5}
	lo, hi := LensArea(keep)
	if lo != (Point{5, 5}) || hi != (Point{17, 17}) {
		t.Errorf("keep area = %v..%v", lo, hi)
	}
	small := keep
	small.Plot = 0
	if lo, _ := LensArea(small); lo != (Point{7, 7}) {
		t.Errorf("a keep without a plot still shows its yard: %v", lo)
	}
	codex := view.Structure{Kind: view.StructCodex, X: 4, Y: 4, W: 2, H: 2}
	if lo, hi := LensArea(codex); lo != (Point{3, 3}) || hi != (Point{6, 6}) {
		t.Errorf("codex area = %v..%v", lo, hi)
	}
	if _, ok := clipSpan(Point{0, 0}, Point{3, 3}, 5, 5, 9, 9); ok {
		t.Error("an area out of view is skipped")
	}
}

func TestMiddenSitsOnFreeGroundBesideTheKeep(t *testing.T) {
	site := coastSite(40, 32, 8)
	keep := view.Structure{ID: "keep-1", Kind: view.StructKeep, X: 20, Y: 10, W: 3, H: 3, Object: view.ObjectRef{Name: "lone-tower"}}
	site.Structures = []view.Structure{keep}
	spot, ok := MiddenSpot(site, keep)
	if !ok || spot != (Vec{19.5, 13.5}) {
		t.Fatalf("midden spot = %v %v", spot, ok)
	}
	site.Structures = append(site.Structures, view.Structure{ID: "b", Kind: view.StructCodex, X: 18, Y: 13, W: 2, H: 2})
	if spot, ok := MiddenSpot(site, keep); !ok || spot != (Vec{23.5, 9.5}) {
		t.Errorf("blocked west corner: spot = %v", spot)
	}
	if MiddenName(7, 3) != "midden/3/1" || MiddenName(-1, -1) != "midden/0/1" || CodexBloatName(9) != "codexbloat/3" {
		t.Error("midden and bloat names clamp to the art's levels")
	}
	if AlarmFrame(1.0, true) != 0 || AlarmFrame(.5, false) != 4 {
		t.Error("the alarm pulses unless effects are reduced")
	}
}

func TestHutNamesFallBackToThePlainHut(t *testing.T) {
	got := HutNames(2, model.HearthReady, model.QoSGuaranteed)
	if len(got) != 2 || got[0] != "hearth/2/ready/landed" || got[1] != "hearth/2/ready" {
		t.Errorf("names = %v", got)
	}
	if got := HutNames(1, model.HearthPending, ""); got[0] != "hearth/1/pending" || got[len(got)-1] != "hearth/1/ready" {
		t.Errorf("unknown class = %v", got)
	}
	r := NewRenderer(nil)
	p := model.PodView{Tier: 1, State: model.HearthReady, QoS: model.QoSBestEffort}
	if r.hutName(p) != "hearth/1/ready" || r.hutName(p) != "hearth/1/ready" {
		t.Errorf("without art the plain hut is chosen, and cached: %q", r.hutName(p))
	}
	lib := gfx.Load(art.Build())
	full := NewRenderer(lib)
	want := gfx.HearthLookNames(1, model.HearthReady, model.QoSBestEffort, 0, gfx.LookVariant(p.Name))[0]
	if lib.Has(want) && full.hutName(p) != want {
		t.Errorf("with QoS art the hut is %q", full.hutName(p))
	}
}

func TestRankChevronsAndQoSShifts(t *testing.T) {
	cases := []struct {
		rank string
		n    int
		up   bool
	}{{model.RankHigh, 2, true}, {model.RankNormal, 1, true}, {model.RankLow, 1, false}, {"", 0, false}}
	for _, c := range cases {
		if n, up, _ := RankChevrons(c.rank); n != c.n || up != c.up {
			t.Errorf("rank %q = %d %v", c.rank, n, up)
		}
	}
	shifts := []struct {
		from, to, want string
		ok             bool
	}{
		{model.QoSBestEffort, model.QoSBurstable, QoSUp, true},
		{model.QoSGuaranteed, model.QoSBestEffort, QoSDown, true},
		{model.QoSBurstable, model.QoSBurstable, "", false},
		{"", model.QoSGuaranteed, "", false},
	}
	for _, c := range shifts {
		if got, ok := QoSShiftKind(c.from, c.to); got != c.want || ok != c.ok {
			t.Errorf("%s -> %s = %q %v", c.from, c.to, got, ok)
		}
	}
}

func TestCaesarWorldDrawsWithAndWithoutArt(t *testing.T) {
	sc, cam := sceneFor(t, .6)
	sc.Ghost = nil
	keeps := Keeps(sc.Site)
	if len(keeps) == 0 {
		t.Fatal("scene has no keeps")
	}
	node := keeps[0].Object.Name
	sc.Caesar = model.Caesar{
		Codex:   model.Codex{SizeBytes: 1.9 * (1 << 30), QuotaBytes: 2 << 30, Alarm: true},
		Middens: map[string]model.Midden{node: {Node: node, UsedBytes: 50 << 30, CapacityBytes: 60 << 30}},
	}
	sc.Lens = Lens{Mode: LensUnhoused, Heat: map[string]float64{keeps[0].ID: .95}, Labels: map[string]string{keeps[0].ID: "HANDS 95%"}}
	sc.Pods = []model.PodView{
		{Namespace: "default", Name: "porter-x", Tier: 1, State: model.HearthPending, QoS: model.QoSBestEffort},
		{Namespace: "default", Name: "porter-y", Node: node, Tier: 2, State: model.HearthReady, QoS: model.QoSGuaranteed, Rank: model.RankHigh},
	}
	sc.Transients = []Transient{{Kind: QoSUp, At: Vec{10, 10}, Start: sc.Time, Life: 1}, {Kind: QoSDown, At: Vec{11, 10}, Start: sc.Time, Life: 1}}
	dst := ebiten.NewImage(render.TileWidth*30, render.TileHeight*40)
	for _, lib := range []*gfx.Library{nil, gfx.Load(art.Build())} {
		r := NewRenderer(lib)
		cam = cam.CenterOn(float64(keeps[0].X), float64(keeps[0].Y), dst.Bounds().Dx(), dst.Bounds().Dy())
		r.Draw(dst, cam, sc)
		r.DrawLensMarks(dst, cam, sc)
		r.DrawCodexAlarm(dst, cam, sc)
		if len(r.middenItems(cam, sc)) != 1 {
			t.Error("a filling floor raises a heap beside its Keep")
		}
	}
	sc.Ghost = &Ghost{Kind: view.StructKeep}
	r := NewRenderer(nil)
	r.Draw(dst, cam, sc)
	r.DrawLensMarks(dst, cam, sc)
}
