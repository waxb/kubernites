package world

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
)

func plainLibrary() *gfx.Library {
	set := art.Build()
	walkers := map[string]art.Region{}
	for name, reg := range set.Walkers.Regions {
		if !strings.HasPrefix(name, "walker/raven/") && !strings.HasPrefix(name, "walker/hound/") {
			walkers[name] = reg
		}
	}
	set.Walkers.Regions = walkers
	return gfx.Load(set)
}

func TestRavenCirclesItsOwnerAndPerchesWhenItRests(t *testing.T) {
	owner := Pose{Pos: Vec{10, 10}, Action: "walk", Dir: "se", Frame: 1}
	angles := map[bool]bool{}
	for tt := 0.0; tt < 4; tt += .1 {
		p := RavenPose(tt, 3, owner, MoodSteady)
		if d := Dist(p.Pos, owner.Pos); d < RavenRadius-.01 || d > RavenRadius+.01 {
			t.Fatalf("raven strays: %v", d)
		}
		if p.Action != "walk" {
			t.Fatalf("raven flies: %+v", p)
		}
		angles[p.Pos.X > owner.Pos.X] = true
	}
	if len(angles) != 2 {
		t.Error("the raven circles around")
	}
	owner.Carry = true
	if RavenPose(1, 3, owner, MoodBrisk).Action != "carry" {
		t.Error("the raven carries a scroll when its owner carries a load")
	}
	actions := map[string]bool{}
	for tt := 0.0; tt < 12; tt += .5 {
		p := RavenPose(tt, 3, owner, MoodRest)
		actions[p.Action] = true
		if Dist(p.Pos, owner.Pos) > 1 {
			t.Fatal("a perched raven stays near")
		}
	}
	if !actions["idle"] || !actions["work"] {
		t.Errorf("perched raven actions = %v", actions)
	}
}

func TestHoundHeelsBehindItsOwner(t *testing.T) {
	se := HoundPose(1, 2, Pose{Pos: Vec{5, 5}, Action: "walk", Dir: "se", Frame: 2}, MoodSteady)
	sw := HoundPose(1, 2, Pose{Pos: Vec{5, 5}, Action: "walk", Dir: "sw", Frame: 2}, MoodSteady)
	if se.Pos.X >= 5 || sw.Pos.Y >= 5 || se.Frame != 2 || se.Action != "walk" {
		t.Fatalf("hound se=%+v sw=%+v", se, sw)
	}
	cases := map[string]string{MoodBrisk: "carry", MoodRest: "idle", MoodSick: "idle"}
	for mood, want := range cases {
		if got := HoundPose(1, 2, Pose{Pos: Vec{5, 5}, Action: "walk", Dir: "se"}, mood); got.Action != want {
			t.Errorf("%s hound = %s, want %s", mood, got.Action, want)
		}
	}
	if HoundPose(1, 2, Pose{Action: "work", Dir: "se"}, MoodSteady).Action != "work" {
		t.Error("the hound sniffs about while its owner works")
	}
	if HoundPose(1, 2, Pose{Action: "idle", Dir: "se"}, MoodSteady).Action != "idle" {
		t.Error("the hound sits while its owner stands")
	}
}

func TestFamiliarItemsFollowZoomAndArt(t *testing.T) {
	sc, cam, r := errandScene(t)
	f := Figure{Pod: model.PodView{Name: "p", Familiars: []string{model.FamiliarRaven, model.FamiliarHound, "scout"}}, Pose: Pose{Pos: Vec{24, 24}, Action: "walk", Dir: "se"}, Mood: MoodSteady}
	items := r.FamiliarItems(cam, sc, f)
	if len(items) != maxFamiliars {
		t.Fatalf("familiars = %d", len(items))
	}
	far := cam
	far.Zoom = .5
	if r.FamiliarItems(far, sc, f) != nil {
		t.Error("familiars are hidden when zoomed far out")
	}
	if r.FamiliarItems(cam, sc, Figure{}) != nil {
		t.Error("no sidecars, no familiars")
	}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	bare := &Renderer{Lib: plainLibrary()}
	for _, rr := range []*Renderer{r, bare} {
		for _, mood := range []string{MoodSteady, MoodRest} {
			f.Mood = mood
			for _, it := range rr.FamiliarItems(cam, sc, f) {
				it.Draw(dst)
			}
		}
	}
}
