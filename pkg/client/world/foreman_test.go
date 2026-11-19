package world

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func porterChain(ready, desired int, pods ...string) model.CrewChain {
	fm := model.Foreman{Namespace: "default", Name: "porters-7d9f", Revision: 1, Desired: desired, Ready: ready, Newest: true, Pods: pods}
	return model.CrewChain{Namespace: "default", Name: "porters", Desired: desired, Ready: ready, Depot: "parcels", Foremen: []model.Foreman{fm}}
}

func TestOwnerRSNamesTheForeman(t *testing.T) {
	p := model.PodView{Namespace: "default", Owner: "ReplicaSet/porters-7d9f"}
	if OwnerRS(p) != "default/porters-7d9f" || OwnerRS(model.PodView{Owner: "Node/lone-tower"}) != "" {
		t.Fatal("owner lookup")
	}
}

func TestPostsStandAtTheDepotOrTheKeepOfTheirHuts(t *testing.T) {
	depot := view.Structure{ID: "d", Kind: "guildhall", X: 10, Y: 20, W: 3, H: 3, State: view.StateReady, Object: view.ObjectRef{Kind: "Service", Name: "parcels"}}
	keep := view.Structure{ID: "k", Kind: view.StructKeep, X: 40, Y: 40, W: 3, H: 3, Object: view.ObjectRef{Kind: "Node", Name: "lone-tower"}}
	site := view.Site{Structures: []view.Structure{depot, keep}}
	chain := porterChain(2, 2)
	posts := PostsOf(site, []model.CrewChain{chain}, nil, nil)
	if len(posts) != 1 || posts[0].Banner != (Vec{10.5, 23.5}) || posts[0].Foremen[0] != (Vec{11.5, 23.5}) {
		t.Fatalf("depot posts = %+v", posts)
	}
	east := func(string) (Vec, bool) { return Vec{30, 10}, true }
	pod := model.PodView{Namespace: "default", Name: "porters-7d9f-a", Node: "lone-tower", Owner: "ReplicaSet/porters-7d9f"}
	posts = PostsOf(site, []model.CrewChain{chain}, []model.PodView{pod}, east)
	if posts[0].Banner != (Vec{13.5, 22.5}) || posts[0].Foremen[0] != (Vec{13.5, 21.5}) {
		t.Fatalf("huts to the east turn the post to the east face: %+v", posts)
	}
	chain.Depot = ""
	pod = model.PodView{Namespace: "default", Name: "porters-7d9f-a", Node: "lone-tower", Owner: "ReplicaSet/porters-7d9f"}
	posts = PostsOf(site, []model.CrewChain{chain}, []model.PodView{pod}, nil)
	if len(posts) != 1 || posts[0].Banner != (Vec{40.5, 43.5}) {
		t.Fatalf("keep posts = %+v", posts)
	}
	if got := PostsOf(site, []model.CrewChain{chain}, nil, nil); len(got) != 0 {
		t.Fatalf("a crew with no depot and no huts stands nowhere: %+v", got)
	}
}

func hutPods(names ...string) []model.PodView {
	out := []model.PodView{}
	for _, n := range names {
		out = append(out, model.PodView{Namespace: "default", Name: n, Owner: "ReplicaSet/porters-7d9f", State: model.HearthReady})
	}
	return out
}

func TestTheForemanWhistlesWhenAHutFallsAndCallsOutTheNewOne(t *testing.T) {
	post := []Post{{Crew: porterChain(2, 2), Banner: Vec{1, 1}, Foremen: []Vec{{2, 1}}}}
	born := map[string]float64{}
	birth := func(pod string) (float64, Vec, bool) { t, ok := born[pod]; return t, Vec{5, 5}, ok }
	f := NewForemen()
	f.Step(post, hutPods("a", "b"), birth, 1)
	if f.Whistling("default/porters-7d9f", 1) {
		t.Fatal("whistled on the first look")
	}
	f.Step(post, hutPods("b"), birth, 2)
	if !f.Whistling("default/porters-7d9f", 2.5) || f.Whistling("default/porters-7d9f", 2+WhistleLife+.01) {
		t.Fatal("no whistle for a fallen hut, or one that never stops")
	}
	born["c"] = 2.4
	f.Step(post, hutPods("b", "c"), birth, 2.5)
	says := f.Says()
	if len(says) != 1 || says[0].Count != 1 || !says[0].Replaced || says[0].At != (Vec{2, 1}) {
		t.Fatalf("says = %+v", says)
	}
	head, tail := SayText(says[0])
	if head != "the shift of porters-7d9f (ReplicaSet)" || tail != "raised a new hut to keep 2" {
		t.Fatalf("callout = %q / %q", head, tail)
	}
	if kh, kt := SayTextK8s(says[0]); kh != "ReplicaSet porters-7d9f: SuccessfulCreate" || kt != "Created pod: c" {
		t.Fatalf("k8s callout = %q / %q", kh, kt)
	}
	f.Raised(says[0].Foreman, "d", 2, Vec{2, 1}, 2.6)
	if s := f.Says(); s[0].Count != 2 {
		t.Fatalf("a second hut within the burst is merged: %+v", s)
	}
	if _, tail := SayText(f.Says()[0]); tail != "raised 2 new huts to keep 2" {
		t.Fatalf("merged tail = %q", tail)
	}
	if _, kt := SayTextK8s(f.Says()[0]); kt != "Created pod: d, and 1 more" {
		t.Fatalf("merged k8s tail = %q", kt)
	}
	f.Step(post, hutPods("b", "c"), birth, 2.6+CalloutLife+.01)
	if len(f.Says()) != 0 {
		t.Fatal("callouts never fade")
	}
}

func TestScalingDownIsNoWhistle(t *testing.T) {
	if LostHut(map[string]bool{"a": true, "b": true, "c": true}, map[string]bool{"a": true, "b": true}, 2) {
		t.Fatal("a scale-down whistled")
	}
	if !LostHut(map[string]bool{"a": true, "b": true}, map[string]bool{"b": true, "c": true}, 2) {
		t.Fatal("a replaced hut did not whistle")
	}
}

func TestAnOldForemanWalksOffWhenHisReplicaSetReachesZero(t *testing.T) {
	old := model.Foreman{Namespace: "default", Name: "porters-old", Revision: 1, Desired: 1, Current: 1}
	fresh := model.Foreman{Namespace: "default", Name: "porters-new", Revision: 2, Desired: 2, Newest: true}
	rolling := model.CrewChain{Namespace: "default", Name: "porters", Foremen: []model.Foreman{fresh, old}}
	f := NewForemen()
	f.Step([]Post{{Crew: rolling, Foremen: []Vec{{2, 1}, {3, 1}}}}, nil, nil, 1)
	old.Desired, old.Current = 0, 0
	done := model.CrewChain{Namespace: "default", Name: "porters", Foremen: []model.Foreman{fresh, old}}
	f.Step([]Post{{Crew: done, Foremen: []Vec{{2, 1}}}}, nil, nil, 2)
	gone := f.Departures()
	if len(gone) != 1 || gone[0].Foreman.Name != "porters-old" || gone[0].From != (Vec{3, 1}) {
		t.Fatalf("departures = %+v", gone)
	}
	f.Step([]Post{{Crew: done, Foremen: []Vec{{2, 1}}}}, nil, nil, 2+WalkOffLife)
	if len(f.Departures()) != 0 {
		t.Fatal("the old shift never leaves")
	}
	if at, ok := f.Standing("default/porters-new"); !ok || at != (Vec{2, 1}) {
		t.Fatal("the new shift is not at its post")
	}
}

func TestTallyAndThreadFade(t *testing.T) {
	if TallyText(model.Foreman{Ready: 2, Desired: 3}) != "2/3" {
		t.Fatal("tally")
	}
	if ThreadAlpha(-1) != 0 || ThreadAlpha(0) != 1 || ThreadAlpha(ThreadLife) != 0 {
		t.Fatal("thread fade")
	}
	colors := map[string]bool{}
	for _, fm := range []model.Foreman{{Desired: 0}, {Desired: 2, Ready: 2}, {Desired: 3, Ready: 1}} {
		colors[string([]byte{TallyColor(fm).R, TallyColor(fm).G})] = true
	}
	if len(colors) != 3 {
		t.Fatal("the tally does not tell idle, whole and short apart")
	}
}

func foremanScene(t *testing.T) (Scene, render.Camera, *Renderer, *ebiten.Image) {
	t.Helper()
	a := householdView("porters-7d9f-first", "lone-tower", porterFolk("porter"))
	sc, cam, r := peopleScene(t, 0, a)
	sc.Crews = []model.CrewChain{porterChain(1, 1, a.Name)}
	sc.Crews[0].Depot = ""
	dst := ebiten.NewImage(960, 540)
	t.Cleanup(dst.Deallocate)
	return sc, cam, r, dst
}

func TestTheForemanStandsDrawsAndIsPickable(t *testing.T) {
	sc, cam, r, dst := foremanScene(t)
	r.Draw(dst, cam, sc)
	at, ok := r.ForemanAt("default/porters-7d9f")
	if !ok {
		t.Fatal("the shift has no post")
	}
	cam = cam.CenterOn(at.X, at.Y, 960, 540)
	sc.Selected = "default/porters-7d9f"
	r.Draw(dst, cam, sc)
	kinds := map[string]bool{}
	for _, h := range r.HitList {
		kinds[h.Kind] = true
	}
	if !kinds[HitForeman] || !kinds[HitCrew] {
		t.Fatalf("hits = %v", kinds)
	}
	fresh := householdView("porters-7d9f-fresh", "lone-tower", porterFolk("porter"))
	sc.Pods = append(sc.Pods, fresh)
	sc.Time += .2
	r.Draw(dst, cam, sc)
	home := r.Homes[fresh.Name]
	found := false
	for _, th := range r.Threads(sc) {
		found = found || th.To == (Vec{home.X + .5, home.Y + .5}) && th.From == at && th.Age == 0
	}
	if !found {
		t.Fatalf("no thread from the shift to the new hut: %+v", r.Threads(sc))
	}
	sc.Time += .3
	r.Draw(dst, cam, sc)
	r.DrawForemanCallouts(dst, cam, sc)
	if len(r.ForemenOf().Says()) != 1 {
		t.Fatal("the new hut raised no callout")
	}
	sc.Time += RiseSeconds
	if len(r.PennantItems(cam, sc)) == 0 {
		t.Fatal("huts carry no revision pennant")
	}
}

func TestSayRectFitsBothLines(t *testing.T) {
	s := ForemanSay{Foreman: model.Foreman{Name: "porters-7d9f5c8b6d"}, Desired: 3, Start: 1, Now: 1.5}
	head, tail := SayLines(s)
	r := SayRect(s, 400, 300, 1, false)
	if r.W < len(head)*5 || r.W < len(tail)*5 || r.H < 2*9 || !strings.Contains(head, "porters-7d9f5c8b6d") {
		t.Fatalf("rect %+v for %q / %q", r, head, tail)
	}
}

func TestClearAboveSlidesThenLiftsThenFallsBack(t *testing.T) {
	r := ui.Rect{X: 100, Y: 200, W: 100, H: 20}
	if got, ok := ClearAbove(r, nil); !ok || got != r {
		t.Fatalf("a clear rect moved: %+v", got)
	}
	right := ui.Rect{X: 190, Y: 190, W: 40, H: 40}
	if got, ok := ClearAbove(r, []ui.Rect{right}); !ok || got.Y != r.Y || Overlaps(got, right) {
		t.Fatalf("slide = %+v", got)
	}
	wide := ui.Rect{X: 0, Y: 190, W: ViewWidth, H: 40}
	if got, ok := ClearAbove(r, []ui.Rect{wide}); !ok || got.Y != wide.Y-r.H-SayGap {
		t.Fatalf("lift = %+v", got)
	}
	tall := ui.Rect{X: 0, Y: 0, W: ViewWidth, H: 230}
	if got, ok := ClearAbove(r, []ui.Rect{tall}); !ok || got.Y != tall.Y+tall.H+4 {
		t.Fatalf("fallback = %+v", got)
	}
}

func TestForemanMarksPaint(t *testing.T) {
	sc, cam, r, dst := foremanScene(t)
	old := model.Foreman{Namespace: "default", Name: "porters-54cd", Revision: 1}
	sc.Crews[0].Paused = true
	sc.Crews[0].Foremen = append(sc.Crews[0].Foremen, old)
	sc.Crews[0].Foremen[0].Revision = 2
	r.Draw(dst, cam, sc)
	at, _ := r.ForemanAt("default/porters-7d9f")
	cam = cam.CenterOn(at.X, at.Y, 960, 540)
	f := r.ForemenOf()
	f.Whistles = WithTime(f.Whistles, "default/porters-7d9f", sc.Time)
	r.Draw(dst, cam, sc)
	whistled := false
	for _, m := range f.Marks {
		whistled = whistled || m.Whistle
	}
	if !whistled {
		t.Fatal("a whistling shift shows no mark")
	}
	sc.Time += RiseSeconds
	for _, it := range r.PennantItems(cam, sc) {
		it.Draw(dst)
	}
	d := Departure{Foreman: old, From: at, Since: sc.Time}
	it, ok := r.DepartureItem(cam, sc, d)
	if !ok {
		t.Fatal("a departing shift is not drawn")
	}
	it.Draw(dst)
	if _, gone := r.DepartureItem(cam, Scene{Site: sc.Site, Time: sc.Time + WalkOffLife}, d); gone {
		t.Fatal("a shift who left is still drawn")
	}
	sc.Callouts = []Callout{{Portrait: "steward", Text: "The Stewards raised a hut", At: at, Age: 1}}
	if obstacles := r.SayObstacles(cam, sc); len(obstacles) < 2 {
		t.Fatalf("obstacles = %+v", obstacles)
	}
}
