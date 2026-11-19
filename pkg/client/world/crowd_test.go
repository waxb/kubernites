package world

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func moodScene(t *testing.T) (Scene, render.Camera, *Renderer) {
	sc, cam, _ := errandScene(t)
	r := NewRenderer(gfxLibrary())
	sc.Metrics = view.MetricsSnapshot{Pods: map[string]view.PodMetrics{}}
	pods := sc.Pods[:0:0]
	for i, p := range sc.Pods {
		if p.System || p.State != model.HearthReady || p.Terminating {
			continue
		}
		switch i % 4 {
		case 0:
			sc.Metrics.Pods[PodKey(p)] = view.PodMetrics{ServedRate: .5, Throttled: true}
			p.Familiars = []string{model.FamiliarRaven, model.FamiliarHound, "scout"}
		case 1:
			sc.Metrics.Pods[PodKey(p)] = view.PodMetrics{ServedRate: 0}
			p.Familiars = []string{model.FamiliarHound}
		case 2:
			p.State, p.Reason = model.HearthGuttering, "OOMKilled"
		default:
			sc.Metrics.Pods[PodKey(p)] = view.PodMetrics{ServedRate: 2}
		}
		pods = append(pods, p)
	}
	sc.Pods = append(pods, model.PodView{Namespace: "default", Name: "crashy", Node: "lone-tower", Sprite: "porter", State: model.HearthGuttering, Reason: "CrashLoopBackOff"})
	dst := ebiten.NewImage(960, 540)
	t.Cleanup(dst.Deallocate)
	r.Draw(dst, cam, sc)
	return sc, cam, r
}

func TestCrowdShowsEveryMood(t *testing.T) {
	sc, cam, r := moodScene(t)
	moods := map[string]int{}
	figs := r.Crowd(sc, RouteOf(sc.Site))
	for _, f := range figs {
		moods[f.Mood]++
	}
	for _, m := range []string{MoodRest, MoodStrained, MoodSick, MoodBrisk} {
		if moods[m] == 0 {
			t.Errorf("no %s worker in %v", m, moods)
		}
	}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	for _, reduce := range []bool{false, true} {
		sc.Reduce = reduce
		for _, lib := range []*Renderer{r, {Homes: r.Homes}} {
			for _, f := range figs {
				for _, it := range lib.figureItems(cam, sc, f, RouteOf(sc.Site)) {
					it.Draw(dst)
				}
			}
		}
	}
	for _, z := range []float64{.5, 2} {
		zc := cam
		zc.Zoom = z
		r.Draw(dst, zc, sc)
	}
}

func TestSickAndSystemFigures(t *testing.T) {
	sc, _, r := moodScene(t)
	for _, p := range sc.Pods {
		if p.Name == "crashy" {
			f, ok := r.FigureFor(sc, p, RouteOf(sc.Site))
			if !ok || f.Mood != MoodSick {
				t.Fatalf("crash-looping workers are sick: %+v", f)
			}
		}
	}
	gone := model.PodView{Name: "gone", Terminating: true, State: model.HearthReady}
	if _, ok := r.FigureFor(sc, gone, RouteOf(sc.Site)); ok {
		t.Error("terminating workers leave the crowd")
	}
	if _, ok := r.FigureFor(sc, model.PodView{Name: "nowhere", State: model.HearthGuttering}, RouteOf(sc.Site)); ok {
		t.Error("a homeless sick worker is not drawn")
	}
	if _, ok := r.FigureFor(sc, model.PodView{Name: "nowhere", State: model.HearthReady}, RouteOf(sc.Site)); ok {
		t.Error("a homeless worker is not drawn")
	}
	if _, ok := r.FigureFor(sc, model.PodView{Name: "pend", State: model.HearthPending}, RouteOf(sc.Site)); ok {
		t.Error("pending pods wait at their banner, not on the road")
	}
	sys := model.PodView{Namespace: "kube-system", Name: "etcd-lone-tower", System: true, State: model.HearthReady, Sprite: "codex-keeper"}
	if _, ok := r.FigureFor(sc, sys, RouteOf(sc.Site)); !ok {
		t.Error("control-plane keepers stand at their buildings")
	}
	sys.State = model.HearthGuttering
	if _, ok := r.FigureFor(sc, sys, RouteOf(sc.Site)); ok {
		t.Error("a broken control-plane keeper stays inside")
	}
	if _, ok := systemFigure(Scene{}, model.PodView{System: true, State: model.HearthReady, Name: "kube-proxy-x"}); ok {
		t.Error("no building, no keeper")
	}
}

func TestWorkersRestThenWakeWhenPilgrimsCome(t *testing.T) {
	sc, _, r := moodScene(t)
	var p model.PodView
	for _, q := range sc.Pods {
		if q.State == model.HearthReady && sc.Metrics.Pods[PodKey(q)].ServedRate == 0 {
			p = q
		}
	}
	rt := RouteOf(sc.Site)
	f, _ := r.FigureFor(sc, p, rt)
	if f.Mood != MoodRest || f.deliver {
		t.Fatalf("idle worker = %+v", f)
	}
	sc.Metrics.Pods[PodKey(p)] = view.PodMetrics{ServedRate: .6}
	sc.Time += .25
	f, _ = r.FigureFor(sc, p, rt)
	if f.Mood == MoodRest || f.Pose.Action == "" {
		t.Fatalf("woken worker = %+v", f)
	}
	if Dist(f.Pose.Pos, Vec{r.Homes[p.Name].X + .35, r.Homes[p.Name].Y + .35}) > .6 {
		t.Fatalf("wakes at home, not mid-route: %+v", f.Pose)
	}
}

func TestVisibleCullsAndBudgets(t *testing.T) {
	sc, cam := sceneFor(t, .5)
	cam = cam.CenterOn(24, 24, 960, 540)
	figs := []Figure{}
	for i := 0; i < 400; i++ {
		figs = append(figs, Figure{Pod: model.PodView{Namespace: "default", Name: fmt.Sprintf("p-%03d", i)}, Pose: Pose{Pos: Vec{24 + float64(i%9)*.3, 24 + float64(i/40)*.3}}})
	}
	figs = append(figs, Figure{Pod: model.PodView{Name: "far"}, Pose: Pose{Pos: Vec{-300, 300}}})
	got := Visible(cam, sc.Site, figs, "default/p-399")
	if len(got) != WalkerBudget(cam.Scale()) {
		t.Fatalf("visible = %d, budget %d", len(got), WalkerBudget(cam.Scale()))
	}
	if got[0].Pod.Name != "p-399" {
		t.Error("the selected worker is always drawn")
	}
	for _, f := range got {
		if f.Pod.Name == "far" {
			t.Error("off-screen workers are culled")
		}
	}
	if !(WalkerBudget(.5) < WalkerBudget(1) && WalkerBudget(1) < WalkerBudget(2)) {
		t.Error("zooming in affords more walkers")
	}
	if few := Visible(cam, sc.Site, figs[:5], ""); len(few) != 5 {
		t.Errorf("small crowds are untouched: %d", len(few))
	}
}

func TestPosturesDrawWithAndWithoutArt(t *testing.T) {
	dst := ebiten.NewImage(64, 64)
	defer dst.Deallocate()
	lib := gfxLibrary()
	names := figureNames(NameKey{Role: "clerk", fallback: "codex-keeper", Action: "idle", Dir: "se", Tier: 1})
	if len(names) < 4 {
		t.Fatalf("names = %v", names)
	}
	if got := figureNames(NameKey{Role: "wanderer", Action: "walk", Dir: "sw", frame: 1, Tier: 1}); len(got) < 4 {
		t.Errorf("self fallback = %v", got)
	}
	for _, r := range []*Renderer{NewRenderer(lib), NewRenderer(nil)} {
		r.drawSeated(dst, names, 32, 40, 1, logSeat, "se", look{})
		r.drawSeated(dst, names, 32, 40, 2, groundSeat, "sw", look{tint: sickTint, alpha: .5})
		drawLying(dst, r.Lib, names, 32, 40, 1, -1.2, look{tint: fallTint, alpha: .5})
		drawLying(dst, r.Lib, names, 32, 40, 1, -.4, look{})
		r.drawSweat(dst, 32, 40, 1, 1.3, false)
		r.drawSweat(dst, 32, 40, 2, 1.3, true)
		r.drawDizzy(dst, 32, 40, 1, .7)
		r.drawSnore(dst, 32, 40, 1, .5)
		r.drawSnore(dst, 32, 40, 1, 2)
		r.drawBanner(dst, 32, 40, 1, BannerHue("x"), .3)
		r.drawBanner(dst, 32, 40, 1, BannerHue("x"), .1)
	}
	if len(NewRenderer(nil).namesFor("clerk", "codex-keeper", "work", "se", 1)) != len(names) {
		t.Error("cached names match")
	}
	r := NewRenderer(nil)
	if a, b := r.namesFor("reeve", "", "walk", "se", 2), r.namesFor("reeve", "", "walk", "se", 2); &a[0] != &b[0] {
		t.Error("names are cached")
	}
	r.props().draw(dst, "missing", 1, 1, 1, render.SpriteOptions{})
	if c := (look{alpha: .5}).scale(inkColor); c.A != 127 {
		t.Errorf("faded ink = %v", c)
	}
	if c := (look{}).scale(inkColor); c != inkColor {
		t.Errorf("opaque ink = %v", c)
	}
}

func insideAny(site view.Site, v Vec) (string, bool) {
	for _, s := range site.Structures {
		lo, hi := Footprint(s)
		if v.X > float64(lo.X)+.15 && v.X < float64(hi.X)+.85 && v.Y > float64(lo.Y)+.15 && v.Y < float64(hi.Y)+.85 {
			return s.Kind, true
		}
	}
	return "", false
}

func TestNoWalkerStandsInsideABuilding(t *testing.T) {
	sc, _, r := moodScene(t)
	rt := RouteOf(sc.Site)
	for tt := 0.0; tt < 30; tt += .5 {
		sc.Time += .5
		for _, f := range r.Crowd(sc, rt) {
			if kind, in := insideAny(sc.Site, f.Pose.Pos); in && f.Pod.System {
				t.Fatalf("%s stands inside the %s at %+v", f.Pod.Name, kind, f.Pose.Pos)
			}
		}
	}
	for _, s := range sc.Site.Structures {
		if kind, in := insideAny(sc.Site, Porch(s)); in {
			t.Errorf("the porch of %s lies inside the %s", s.Kind, kind)
		}
		if s.Kind == view.StructCodex {
			if kind, in := insideAny(sc.Site, ClerkSpot(s)); in {
				t.Errorf("the clerk stands inside the %s", kind)
			}
		}
	}
}

func TestReevesScribesAndBareWalkers(t *testing.T) {
	sc, cam, r := moodScene(t)
	for i := range sc.Site.Structures {
		if sc.Site.Structures[i].Kind == view.StructKeep {
			sc.Site.Structures[i].State = view.StateConstruction
		}
	}
	if items := r.reeveItems(cam, sc); len(items) != 0 {
		t.Errorf("unfinished Keeps have no reeve: %d", len(items))
	}
	bare := NewRenderer(nil)
	dst := ebiten.NewImage(64, 64)
	defer dst.Deallocate()
	pod := sc.Pods[0]
	bare.walkerDrawable(cam, sc, "porter", Pose{Action: "carry", Dir: "se", Carry: true}, &pod, true).Draw(dst)
	scribe := sc.Pods[0]
	scribe.Sprite = ScribeRole
	fresh := NewRenderer(nil)
	fresh.Homes = r.Homes
	if f, ok := fresh.scribeFigure(sc, scribe); !ok || f.Mood != MoodRest {
		t.Errorf("without an Archive the scribes rest: %+v", f)
	}
}

func TestPodKeysMatchWithoutBuildingStrings(t *testing.T) {
	p := model.PodView{Namespace: "default", Name: "porters-a"}
	for key, want := range map[string]bool{"default/porters-a": true, "default/porters-b": false, "defaultXporters-a": false, "default/porters-ab": false, "": false} {
		if got := isPodKey(p, key); got != want {
			t.Errorf("isPodKey(%q) = %v", key, got)
		}
	}
}

func TestAHiredHutOnNoRollRestsWhateverTheMetricsSay(t *testing.T) {
	sc, _, r := moodScene(t)
	var p model.PodView
	for _, q := range sc.Pods {
		if q.State == model.HearthReady && !q.Static && !q.System {
			p = q
		}
	}
	rt := RouteOf(sc.Site)
	sc.Metrics.Pods[PodKey(p)] = view.PodMetrics{ServedRate: .6}
	if f, _ := r.FigureFor(sc, p, rt); f.Mood == MoodRest {
		t.Fatalf("the control hut already rests: %+v", f)
	}
	sc.Idle = []string{p.Name}
	if f, ok := r.FigureFor(sc, p, rt); !ok || f.Mood != MoodRest || f.deliver {
		t.Fatalf("a hired hut on no roll looks busy: %+v", f)
	}
	delete(sc.Metrics.Pods, PodKey(p))
	if f, ok := r.FigureFor(sc, p, rt); !ok || f.Mood != MoodRest {
		t.Fatalf("an unmeasured idle hut looks busy: %+v", f)
	}
}
