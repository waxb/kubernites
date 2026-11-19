package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func gfxLibrary() *gfx.Library { return gfx.Load(art.Build()) }

func TestOmenDrawsImpAndMark(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	r := NewRenderer(nil)
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
	pod := sc.Pods[0].Name
	sc.Omen = &Omen{Pod: pod, Start: sc.Time - 1, Arrive: sc.Time + 1}
	sc.Time = 12
	items := r.OmenItems(cam, sc)
	r.DrawOmenMark(dst, cam, sc)
	r.DrawOmenXray(dst, cam, sc)
	if len(items) != 1 || !Blink(12) || Blink(12.3) {
		t.Fatalf("omen items = %d (home known: %v)", len(items), r.Homes[pod])
	}
}

func TestJuiceAndServingWalkers(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	r := NewRenderer(nil)
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	for i := range sc.Pods {
		sc.Pods[i].State = "ready"
	}
	sc.Site.Structures[len(sc.Site.Structures)-1].State = "construction"
	r.Draw(dst, cam, sc)
	pod := sc.Pods[0].Name
	sc.Transients = []Transient{
		{Kind: "horn", At: Vec{20, 20}, Start: sc.Time - .1, Life: .8},
		{Kind: "sparkle", Pod: pod, Start: sc.Time - .1, Life: .5},
		{Kind: "puff", At: Vec{21, 21}, Start: sc.Time - .1, Life: .3},
		{Kind: "reweave", Pod: "missing", Start: sc.Time, Life: 1},
		{Kind: "mystery", At: Vec{1, 1}, Start: sc.Time, Life: 1},
	}
	if n := len(r.TransientItems(cam, sc)); n != 4 {
		t.Errorf("transients drawn = %d", n)
	}
	start := sc.Time
	for tick := 0.0; tick < 12; tick += .25 {
		sc.Time += .25
		r.Draw(dst, cam, sc)
	}
	sc.Time, sc.Reduce = start, true
	if n := len(r.TransientItems(cam, sc)); n != 1 {
		t.Errorf("reduced effects keep only non-juice transients: %d", n)
	}
	if _, ok := r.DeliveryItem(cam, sc, 0, Vec{1, 1}); ok {
		t.Error("no floating mark under reduced effects")
	}
	sc.Reduce = false
	found := false
	for tt := 0.0; tt < 60 && !found; tt += .1 {
		sc.Time = tt
		_, found = r.DeliveryItem(cam, sc, DeliveryAgeAt(tt, Vec{2, 2}, Vec{8, 8}, Vec{12, 4}), Vec{12, 4})
	}
	if !found {
		t.Error("deliveries float +1 at the store")
	}
}

func TestShoreAndHills(t *testing.T) {
	sc, _ := sceneFor(t, .5)
	shore := NearestShore(sc.Site, Vec{24, 24})
	if !ShoreTile(sc.Site, int(shore.X), int(shore.Y)) {
		t.Errorf("nearest shore %v is not a shore tile", shore)
	}
	if ShoreTile(sc.Site, -1, 0) {
		t.Error("off the map is not shore")
	}
	sea := view.Site{Width: 3, Height: 3, Tiles: make([]view.TileKind, 9)}
	if got := NearestShore(sea, Vec{1, 1}); got.X != 5 {
		t.Errorf("no shore falls back beside the target: %v", got)
	}
	if HillLight(0).Tint != [3]float32{} || HillLight(2).Tint[0] < 1.11 {
		t.Error("hill lift")
	}
	a := DeliveryAge(0, 1, Vec{0, 0}, Vec{5, 0}, Vec{5, 5})
	b := DeliveryAge(.5, 1, Vec{0, 0}, Vec{5, 0}, Vec{5, 5})
	if a < 0 || b < 0 || a == b {
		t.Errorf("delivery ages %v %v", a, b)
	}
}

func TestGatewayAndHelmbearer(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	g := GatewayOf(sc.Site)
	if l := g.Inward.X*g.Inward.X + g.Inward.Y*g.Inward.Y; l < .99 || l > 1.01 {
		t.Errorf("inward is not a unit vector: %+v", g.Inward)
	}
	if Dist(HandSpot(sc.Site), g.Door) > HelmAhead+.01 {
		t.Error("the hand-serve spot stands at the Gate")
	}
	r := NewRenderer(nil)
	sc.GateOpen, sc.ServedAt = true, sc.Time-.2
	if d := r.Helmbearer(cam, sc, g); d.Draw == nil {
		t.Error("the Helmbearer is drawn at the open Gate")
	}
}

func TestScribesAndLedgerBar(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	for i := range sc.Pods {
		sc.Pods[i].State = "ready"
	}
	sc.Pods[0].Sprite = ScribeRole
	if keep, ok := FirstOfKind(sc.Site, view.StructKeep, false); ok {
		sc.Site.Structures = append(sc.Site.Structures, view.Structure{ID: "archive-x", Kind: view.StructArchive, X: keep.X - 5, Y: keep.Y, W: 3, H: 3, State: view.StateReady})
	}
	sc.Ledger, sc.Surge = Ledger{Open: true, Share: .95}, true
	r := NewRenderer(nil)
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
	r.DrawStructureBars(dst, cam, sc)
	if items := r.ScribeItems(cam, sc, sc.Pods[0]); len(items) != 1 {
		t.Errorf("scribe walker items = %d", len(items))
	}
	if _, ok := r.RecordSource(sc, 7); !ok {
		t.Error("scribes fetch records from huts that hold them")
	}
	lib := NewRenderer(gfxLibrary())
	lib.Draw(dst, cam, sc)
	lib.DrawStructureBars(dst, cam, sc)
	sc.Ledger.Open = false
	lib.DrawStructureBars(dst, cam, sc)
	if _, ok := r.RecordSource(Scene{}, 1); ok {
		t.Error("no huts, no records")
	}
	if items := r.ScribeItems(cam, sc, model.PodView{Name: "nobody"}); items != nil {
		t.Error("a homeless scribe draws nothing")
	}
}
