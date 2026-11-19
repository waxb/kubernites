package world

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func TestAgeClockPrimesWithoutAnUpgrade(t *testing.T) {
	var a ageClock
	a.track(2, 5)
	if age, busy := a.phase("codex-4"); age != 2 || busy {
		t.Fatalf("first sight draws the current age at rest, got %d %v", age, busy)
	}
}

func TestAgeClockStaggersTheUpgrade(t *testing.T) {
	var a ageClock
	a.track(1, 0)
	a.track(2, 10)
	id := "chancery-5"
	d := AgeDelay(id)
	if d < 0 || d >= AgeStagger {
		t.Fatalf("delay %v outside [0,%v)", d, AgeStagger)
	}
	steps := []struct {
		at   float64
		age  int
		busy bool
	}{{10 + d - .01, 1, false}, {10 + d + .1, 2, true}, {10 + d + AgeWork + .1, 2, false}}
	for _, st := range steps {
		a.track(2, st.at)
		if age, busy := a.phase(id); age != st.age || busy != st.busy {
			t.Errorf("at %.2f got %d %v, want %d %v", st.at, age, busy, st.age, st.busy)
		}
	}
	a.track(2, 10+AgeStagger+AgeWork+1)
	if a.from != a.to {
		t.Fatal("the upgrade settles once every building is done")
	}
}

func TestAgeClockDropsBackAtOnce(t *testing.T) {
	var a ageClock
	a.track(3, 0)
	a.track(0, 1)
	if age, busy := a.phase("x"); age != 0 || busy {
		t.Fatalf("dropping back to tier 0 is immediate, got %d %v", age, busy)
	}
}

func TestStructureLookResolvesToAgedSprites(t *testing.T) {
	r := NewRenderer(testLib())
	s := view.Structure{ID: "codex-4", Kind: view.StructCodex, State: view.StateReady}
	sc := Scene{Age: 2, Time: 1}
	names, busy := r.structureLook(s, sc)
	if busy || len(names) == 0 || !r.Lib.Has(names[0]) {
		t.Fatalf("names %v busy %v", names, busy)
	}
	if !strings.HasPrefix(names[0], "building/codex/ready/2/") {
		t.Fatalf("the codex should draw its tier 2 sprite, got %s", names[0])
	}
	sc.Age, sc.Time = 3, 2
	for sc.Time = 2; sc.Time < 2+AgeStagger+AgeWork; sc.Time += .05 {
		if names, busy = r.structureLook(s, sc); busy {
			break
		}
	}
	if !busy || !strings.Contains(names[0], "construction-2") {
		t.Fatalf("mid-upgrade the codex stands in scaffold, got %v", names)
	}
	keep := view.Structure{ID: "keep-3", Kind: view.StructKeep, Variant: "tower", State: view.StateReady}
	if kn, _ := r.structureLook(keep, sc); !strings.HasPrefix(kn[0], "building/keep-tower") || kn[len(kn)-1] != "building/keep-tower/ready" {
		t.Fatalf("keeps resolve through their own names, got %v", kn)
	}
}

func TestHutNamesFollowTheAge(t *testing.T) {
	r := NewRenderer(testLib())
	r.agesOf().track(2, 0)
	p := model.PodView{Name: "porters-77fd67cd44-21tn2", Tier: 1, State: model.HearthReady, QoS: "Burstable"}
	name := r.hutName(p)
	want := gfx.HearthLookNames(1, model.HearthReady, "Burstable", 2, gfx.LookVariant(p.Name))
	found := false
	for _, n := range want {
		if n == name {
			found = true
		}
	}
	if !found || !r.Lib.Has(name) {
		t.Fatalf("hut %s not in %v", name, want)
	}
}

func TestUpgradesRaiseDustAcrossTheTown(t *testing.T) {
	site := archipelago(t)
	if len(site.Structures) == 0 {
		t.Skip("the founded realm has no structures")
	}
	r := NewRenderer(testLib())
	cam := render.NewCamera(480, 60).CenterOn(float64(site.Structures[0].X), float64(site.Structures[0].Y), 960, 540)
	sc := Scene{Site: site, Age: 1, Time: 0}
	base := len(r.structureItems(cam, sc))
	sc.Age = 2
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	most := 0
	for sc.Time = .05; sc.Time < AgeStagger+AgeWork; sc.Time += .05 {
		items := r.structureItems(cam, sc)
		most = max(most, len(items)-base)
		if _, busy := r.AgePhase(site.Structures[0].ID); busy {
			for _, it := range items {
				it.Draw(dst)
			}
		}
	}
	if most == 0 {
		t.Fatal("no dust rose while the town upgraded")
	}
	sc.Time += 1
	r.structureItems(cam, sc)
	if len(r.structureItems(cam, sc)) != base {
		t.Fatal("the dust settles once the upgrade is done")
	}
}

func TestBurstPuffsRiseInTurnAndFade(t *testing.T) {
	if _, live := BurstPuff(0, 0); !live {
		t.Fatal("the first puff starts at once")
	}
	if _, live := BurstPuff(.05, 1); live {
		t.Fatal("later puffs wait their turn")
	}
	if tt, live := BurstPuff(.3, 2); !live || tt <= 0 {
		t.Fatalf("puff 2 at .3 s: %v %v", tt, live)
	}
	if _, live := BurstPuff(burstLife+float64(burstPuffs)*burstStep, burstPuffs-1); live {
		t.Fatal("every puff fades within the burst")
	}
	if burstLife+float64(burstPuffs-1)*burstStep > AgeWork {
		t.Fatal("the burst must end inside the upgrade window")
	}
	c := FootprintCentre(view.Structure{X: 4, Y: 6, W: 3, H: 2})
	if c.X != 5.5 || c.Y != 7 {
		t.Fatalf("centre %v", c)
	}
}
