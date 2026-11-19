package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func householdView(name, node string, people []model.Kubernite, pets ...string) model.PodView {
	return model.PodView{Namespace: "default", Name: name, Node: node, Image: people[0].Image, Sprite: people[0].Sprite,
		Profession: people[0].Profession, Tier: 1, State: model.HearthReady, Ready: true, Owner: "ReplicaSet/porters-7d9f",
		Crew: "porters", People: people, Familiars: pets}
}

func porterFolk(names ...string) []model.Kubernite {
	out := make([]model.Kubernite, 0, len(names))
	for _, n := range names {
		out = append(out, model.Kubernite{Name: n, Image: "porter:1.0", Profession: "Porter", Sprite: "porter", Tier: 1, Ready: true})
	}
	return out
}

func peopleScene(t *testing.T, rate float64, pods ...model.PodView) (Scene, render.Camera, *Renderer) {
	t.Helper()
	sc, cam, r := errandScene(t)
	sc.Pods = append(sc.Pods, pods...)
	for _, p := range pods {
		sc.Metrics.Pods[PodKey(p)] = view.PodMetrics{ServedRate: rate}
	}
	dst := ebiten.NewImage(960, 540)
	t.Cleanup(dst.Deallocate)
	r.Draw(dst, cam, sc)
	if home, ok := r.Homes[pods[0].Name]; ok {
		cam = cam.CenterOn(home.X, home.Y, 960, 540)
		r.Draw(dst, cam, sc)
	}
	return sc, cam, r
}

func figuresOf(figs []Figure, pod string) []Figure {
	out := []Figure{}
	for _, f := range figs {
		if f.Pod.Name == pod {
			out = append(out, f)
		}
	}
	return out
}

func TestEachContainerIsOnePersonOfTheHut(t *testing.T) {
	pair := householdView("porters-7d9f-pair", "lone-tower", porterFolk("porter", "loader"))
	trio := householdView("porters-7d9f-trio", "lone-tower", porterFolk("porter", "loader", "packer"))
	sc, _, r := peopleScene(t, .5, pair, trio)
	figs := r.Crowd(sc, RouteOf(sc.Site))
	for _, tc := range []struct {
		pod  model.PodView
		want []string
	}{{pair, []string{"porter", "loader"}}, {trio, []string{"porter", "loader", "packer"}}} {
		got := figuresOf(figs, tc.pod.Name)
		if len(got) != len(tc.want) {
			t.Fatalf("%s: %d people, want %d", tc.pod.Name, len(got), len(tc.want))
		}
		seeds := map[uint32]bool{}
		for i, f := range got {
			if f.Pod.Focus != tc.want[i] || f.Person != i {
				t.Errorf("%s person %d = %q", tc.pod.Name, i, f.Pod.Focus)
			}
			seeds[f.Seed] = true
		}
		if len(seeds) != len(tc.want) {
			t.Errorf("%s: people share a stride seed", tc.pod.Name)
		}
	}
}

func TestPetsFollowTheirPeople(t *testing.T) {
	cases := []struct {
		people int
		pets   []string
		want   [][]string
	}{
		{1, []string{model.FamiliarRaven, model.FamiliarHound}, [][]string{{model.FamiliarRaven, model.FamiliarHound}}},
		{2, []string{model.FamiliarRaven, model.FamiliarHound}, [][]string{{model.FamiliarRaven}, {model.FamiliarHound}}},
		{2, []string{model.FamiliarHound}, [][]string{{model.FamiliarHound}, nil}},
		{3, []string{model.FamiliarRaven, model.FamiliarHound, model.FamiliarRaven, model.FamiliarHound}, [][]string{{model.FamiliarRaven, model.FamiliarHound}, {model.FamiliarHound}, {model.FamiliarRaven}}},
	}
	for _, tc := range cases {
		names := []string{"a", "b", "c"}[:tc.people]
		p := householdView("h", "lone-tower", porterFolk(names...), tc.pets...)
		for i := range tc.people {
			got := PetsOf(p, i)
			if len(got) != len(tc.want[i]) {
				t.Fatalf("%d people, pets %v: person %d has %v, want %v", tc.people, tc.pets, i, got, tc.want[i])
			}
			for j := range got {
				if got[j] != tc.want[i][j] {
					t.Errorf("%d people: person %d pets %v, want %v", tc.people, i, got, tc.want[i])
				}
			}
		}
	}
}

func TestAPersonsPetsWalkWithThatPerson(t *testing.T) {
	pair := householdView("porters-7d9f-pets", "lone-tower", porterFolk("porter", "loader"), model.FamiliarRaven, model.FamiliarHound)
	sc, cam, r := peopleScene(t, .5, pair)
	for _, f := range figuresOf(r.Crowd(sc, RouteOf(sc.Site)), pair.Name) {
		if got := len(r.FamiliarItems(cam, sc, f)); got != 1 {
			t.Errorf("%s walks with %d pets, want 1", f.Pod.Focus, got)
		}
	}
}

func TestOnlyTheFailingContainerSitsSick(t *testing.T) {
	folk := porterFolk("porter", "loader")
	folk[1].Ready, folk[1].Reason = false, "CrashLoopBackOff"
	p := householdView("porters-7d9f-sick", "lone-tower", folk)
	p.State, p.Reason, p.Ready = model.HearthGuttering, "CrashLoopBackOff", false
	sc, _, r := peopleScene(t, .5, p)
	got := figuresOf(r.Crowd(sc, RouteOf(sc.Site)), p.Name)
	if len(got) != 2 {
		t.Fatalf("people = %d", len(got))
	}
	if got[0].Mood == MoodSick || got[1].Mood != MoodSick {
		t.Errorf("moods = %s and %s, want the loader alone sick", got[0].Mood, got[1].Mood)
	}
	home := r.Homes[p.Name]
	for _, f := range got {
		if Dist(f.Pose.Pos, home) > 1.6 {
			t.Errorf("%s waits %v away from the hut", f.Pod.Focus, Dist(f.Pose.Pos, home))
		}
	}
}

func TestPeopleKeepTheirOwnLookAndTheFirstKeepsThePods(t *testing.T) {
	p := householdView("porters-7d9f-look", "lone-tower", porterFolk("porter", "loader"))
	if PersonSeed(p.Person(0)) != model.Hash(p.Name) || PersonSeed(p) != model.Hash(p.Name) {
		t.Error("the first person keeps the Pod's own seed")
	}
	if PersonSeed(p.Person(1)) == PersonSeed(p.Person(0)) {
		t.Error("the second person has a seed of their own")
	}
	if v, _ := LookOf(p.Person(0)); v != FaceOf("porter", p.Name) {
		t.Error("the first person keeps the Pod's face")
	}
	if PersonKey(p.Person(1)) == PersonKey(p.Person(0)) || PersonKey(p) != PodKey(p) {
		t.Error("people stride apart")
	}
	if PersonAside(0) != (Vec{}) || PersonAside(1) == PersonAside(2) {
		t.Error("people stand beside each other")
	}
}

func TestClickingAPersonSelectsThatContainer(t *testing.T) {
	pair := householdView("porters-7d9f-click", "lone-tower", porterFolk("porter", "loader"))
	pair.People[1].Image = "porter:1.1"
	_, _, r := peopleScene(t, 0, pair)
	found := map[string]string{}
	for _, h := range r.HitList {
		if h.Kind == HitWalker && h.Pod.Name == pair.Name {
			found[h.Pod.Focus] = h.Pod.Image
			if h.ID != PodKey(pair) {
				t.Errorf("a person's hit id %q, want the Pod's %q", h.ID, PodKey(pair))
			}
		}
		if h.Kind == HitHearth && h.Pod.Name == pair.Name && h.Pod.Focus != "" {
			t.Error("the hut selects the whole Pod")
		}
	}
	if found["loader"] != "porter:1.1" || found["porter"] != "porter:1.0" {
		t.Errorf("person hits = %v", found)
	}
}

func TestAnOOMKilledPersonDropsTheBoxAndFaints(t *testing.T) {
	lib := gfxLibrary()
	for _, role := range []string{"porter", "hauler", "clerk"} {
		if name, ok := DroppedBox(role); !ok || !lib.Has(name) {
			t.Errorf("a fainted %s drops %q (%v)", role, name, ok)
		}
	}
	if _, ok := DroppedBox("lamplighter"); ok {
		t.Error("only a trade carries boxes")
	}
	folk := porterFolk("porter")
	folk[0].Ready, folk[0].Reason, folk[0].LastExit = false, "CrashLoopBackOff", "OOMKilled"
	p := householdView("porters-7d9f-oom", "lone-tower", folk)
	p.State, p.Reason, p.LastExit = model.HearthGuttering, "CrashLoopBackOff", "OOMKilled"
	sc, cam, r := peopleScene(t, .5, p)
	got := figuresOf(r.Crowd(sc, RouteOf(sc.Site)), p.Name)
	if len(got) != 1 || got[0].Mood != MoodSick || !got[0].Pod.OOM() {
		t.Fatalf("oom figure = %+v", got)
	}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	for _, rr := range []*Renderer{r, NewRenderer(nil)} {
		for _, reduce := range []bool{false, true} {
			sc.Reduce = reduce
			rr.FigureDrawable(cam, sc, got[0]).Draw(dst)
		}
	}
}

func TestNewcomersWalkOutFromTheirOwnHut(t *testing.T) {
	sc, cam, r := peopleScene(t, .5, householdView("porters-7d9f-old", "lone-tower", porterFolk("porter")))
	fresh := householdView("porters-7d9f-new", "lone-tower", porterFolk("porter", "loader"))
	sc.Pods = append(sc.Pods, fresh)
	sc.Metrics.Pods[PodKey(fresh)] = view.PodMetrics{ServedRate: .5}
	sc.Time += .02
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
	home, ok := r.Homes[fresh.Name]
	if !ok || !r.Newcomer(fresh.Name, sc.Time) {
		t.Fatalf("the new household has no hut (%v)", ok)
	}
	for _, f := range figuresOf(r.Crowd(sc, RouteOf(sc.Site)), fresh.Name) {
		if d := Dist(f.Pose.Pos, Vec{home.X + .35, home.Y + .35}); d > 1 {
			t.Errorf("%s starts %.1f tiles from the hut", f.Pod.Focus, d)
		}
	}
}
