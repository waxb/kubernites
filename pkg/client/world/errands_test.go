package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func TestErrandWalksOutActsAndReturns(t *testing.T) {
	from, to := Vec{0, 0}, Vec{6, 0}
	e := Errand{Kind: ErrandSchedule, From: from, To: to, Start: 10, Life: ErrandLife(ErrandSchedule, from, to)}
	walk := Dist(from, to) / ErrandSpeed
	phases := map[string]bool{}
	for age := 0.0; age <= e.Life; age += .05 {
		step, ok := ErrandAt(e, to, 10+age)
		if !ok {
			t.Fatalf("errand vanished at %v", age)
		}
		phases[step.Phase] = true
		if step.Alpha < 0 || step.Alpha > 1 {
			t.Fatalf("alpha %v", step.Alpha)
		}
	}
	if !phases[PhaseOut] || !phases[PhaseAct] || !phases[PhaseBack] {
		t.Fatalf("phases = %v", phases)
	}
	mid, _ := ErrandAt(e, to, 10+walk/2)
	if Dist(mid.Pose.Pos, Vec{3, 0}) > .01 || mid.Pose.Action != "walk" || mid.Pose.Dir != "se" {
		t.Fatalf("halfway out = %+v", mid.Pose)
	}
	at, _ := ErrandAt(e, to, 10+walk+.2)
	if at.Pose.Pos != to || at.Pose.Action != "work" {
		t.Fatalf("acts at the target: %+v", at.Pose)
	}
	if _, ok := ErrandAt(e, to, 9.9); ok {
		t.Error("errands wait for their start")
	}
	if _, ok := ErrandAt(e, to, 10+e.Life+.1); ok {
		t.Error("errands end")
	}
	if _, ok := ErrandAt(Errand{}, to, 0); ok {
		t.Error("a lifeless errand never shows")
	}
}

func TestStationaryErrands(t *testing.T) {
	if far := ErrandLife(ErrandSchedule, Vec{}, Vec{40, 40}); far > 2*MaxLeg+ErrandActs[ErrandSchedule]+1e-9 {
		t.Fatalf("far errands hurry: %v", far)
	}
	if ErrandLife(ErrandScribble, Vec{}, Vec{9, 9}) != ScribbleLife || ErrandLife(ErrandFallen, Vec{}, Vec{}) != FallLife {
		t.Fatal("stationary errands have fixed lives")
	}
	e := Errand{Kind: ErrandScribble, From: Vec{2, 2}, To: Vec{2, 2}, Start: 0, Life: ScribbleLife}
	step, ok := ErrandAt(e, e.To, .4)
	if !ok || step.Phase != PhaseStay || step.Pose.Action != "work" {
		t.Fatalf("scribble = %+v", step)
	}
	sc := Scene{Time: .4, Errands: []Errand{e, {Kind: ErrandScribble, Start: 5, Life: 1}, {Kind: ErrandLight, Start: 0, Life: 3}}}
	if got := Scribbles(sc); len(got) != 1 {
		t.Fatalf("scribbles = %v", got)
	}
}

func errandScene(t *testing.T) (Scene, render.Camera, *Renderer) {
	sc, cam := sceneFor(t, .5)
	for i := range sc.Pods {
		sc.Pods[i].State = "ready"
	}
	if k, ok := FirstOfKind(sc.Site, view.StructKeep, false); ok {
		cam = cam.CenterOn(float64(k.X), float64(k.Y), 960, 540)
	}
	r := NewRenderer(gfxLibrary())
	dst := ebiten.NewImage(960, 540)
	t.Cleanup(dst.Deallocate)
	r.Draw(dst, cam, sc)
	return sc, cam, r
}

func TestErrandItemsDrawEveryKind(t *testing.T) {
	sc, cam, r := errandScene(t)
	pod := sc.Pods[0].Name
	home := r.Homes[pod]
	keep, _ := FirstOfKind(sc.Site, view.StructKeep, false)
	door := Door(keep)
	now := sc.Time
	mk := func(kind, role string, age float64) Errand {
		life := ErrandLife(kind, door, home)
		return Errand{Kind: kind, Role: role, From: door, To: home, Pod: pod, Start: now - age*life, Life: life}
	}
	sc.Errands = []Errand{
		mk(ErrandSchedule, "quartermaster", .2), mk(ErrandSchedule, "quartermaster", .5),
		mk(ErrandRaise, "steward", .5), mk(ErrandLight, "reeve", .5), mk(ErrandDouse, "reeve", .5),
		mk(ErrandLight, "clerk", .9), {Kind: ErrandScribble, Start: now - .2, Life: ScribbleLife},
		{Kind: ErrandFallen, Role: "porter", To: home, Start: now - .1, Life: FallLife},
		{Kind: ErrandFallen, Role: "porter", To: home, Start: now - .4, Life: FallLife},
		{Kind: ErrandFallen, Start: now + 5, Life: FallLife},
		mk(ErrandDouse, "reeve", 3),
	}
	items := r.ErrandItems(cam, sc)
	if len(items) < 10 {
		t.Fatalf("errand items = %d", len(items))
	}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	for _, it := range items {
		it.Draw(dst)
	}
	if got := r.ErrandTarget(Errand{Kind: ErrandLight, Pod: pod, To: Vec{1, 1}}); got == (Vec{1, 1}) {
		t.Error("pod errands walk to the hut")
	}
	if got := r.ErrandTarget(Errand{Kind: ErrandSchedule, Pod: pod, To: Vec{1, 1}}); got != (Vec{1, 1}) {
		t.Error("the Quartermaster walks to the Keep")
	}
	sc.Reduce = true
	bare := NewRenderer(nil)
	bare.Homes = r.Homes
	for _, it := range append(bare.ErrandItems(cam, sc), bare.ClerkItems(cam, sc)...) {
		it.Draw(dst)
	}
}

func TestErrandsAreCappedAndCulled(t *testing.T) {
	sc, cam, r := errandScene(t)
	keep, _ := FirstOfKind(sc.Site, view.StructKeep, false)
	door := Door(keep)
	for i := 0; i < MaxErrands+10; i++ {
		sc.Errands = append(sc.Errands, Errand{Kind: ErrandSchedule, Role: "quartermaster", From: door, To: Vec{door.X + 3, door.Y}, Start: sc.Time - .5, Life: 6})
	}
	sc.Errands = append(sc.Errands, Errand{Kind: ErrandSchedule, From: Vec{-400, -400}, To: Vec{-400, -401}, Start: sc.Time - .5, Life: 6})
	people := 0
	for _, it := range r.ErrandItems(cam, sc) {
		if it.Hit == nil {
			people++
		}
	}
	if people > 2*MaxErrands {
		t.Fatalf("errand drawables = %d", people)
	}
	far := Errand{Kind: ErrandSchedule, From: Vec{-400, -400}, To: Vec{-400, -401}, Start: sc.Time - .5, Life: 6}
	if items := r.WalkingErrand(cam, sc, far); items != nil {
		t.Error("off-screen errands are culled")
	}
}

func TestClerkScribblesAtTheCodex(t *testing.T) {
	sc, cam, r := errandScene(t)
	codex, ok := FirstOfKind(sc.Site, view.StructCodex, true)
	if !ok {
		t.Skip("no Codex in the lively realm")
	}
	spot := ClerkSpot(codex)
	if Covers(codex, int(spot.X), int(spot.Y)) {
		t.Fatal("the clerk stands beside the Codex, not inside it")
	}
	if items := r.ClerkItems(cam, sc); len(items) != 1 {
		t.Fatalf("clerk items = %d", len(items))
	}
	sc.Errands = []Errand{{Kind: ErrandScribble, Start: sc.Time - .1, Life: ScribbleLife}, {Kind: ErrandScribble, Start: sc.Time - .5, Life: ScribbleLife}}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	for _, it := range r.ClerkItems(cam, sc) {
		it.Draw(dst)
	}
	r.DrawFlecks(dst, 10, 10, 1, []float64{.1, .2, .3, .4, .5})
	hues := map[[4]uint8]bool{}
	for _, pod := range []string{"a", "b", "c", "d", "e", "f"} {
		c := BannerHue(pod)
		hues[[4]uint8{c.R, c.G, c.B, c.A}] = true
	}
	if len(hues) < 2 || BannerHue("a") != BannerHue("a") {
		t.Error("banners are coloured per hut")
	}
	if items := r.ClerkItems(cam, Scene{}); items != nil {
		t.Error("no Codex, no clerk")
	}
}
