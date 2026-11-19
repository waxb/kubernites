package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func incidentScene() (Scene, render.Camera) {
	sc := Scene{Site: view.Site{Width: 40, Height: 40}, Time: 1.3,
		Pods:      []model.PodView{{Namespace: "default", Name: "porters-a1", Restarts: 4}, {Namespace: "default", Name: "clerks-b2"}},
		Incidents: []IncidentMark{{Pod: "default/porters-a1", Crash: true}, {Pod: "default/clerks-b2"}, {Pod: "default/nowhere"}}}
	return sc, render.NewCamera(480, 60).CenterOn(10, 10, 960, 540)
}

func TestAnIncidentPinsItsHutAndACrashLoopShowsItsWait(t *testing.T) {
	r := NewRenderer(testLib())
	r.Homes = map[string]Vec{"porters-a1": {X: 10, Y: 10}, "clerks-b2": {X: 12, Y: 10}}
	sc, cam := incidentScene()
	names := map[string]int{}
	for _, s := range r.incidentSprites(cam, sc) {
		names[s.name]++
		if !r.Lib.Has(s.name) {
			t.Errorf("%s is not in the atlas", s.name)
		}
	}
	frame := IncidentFrame(sc.Time, false)
	want := map[string]int{art.IncidentMarkerName(frame): 2, art.BackoffRingName(4): 1, art.RestartBadgeName(4): 1}
	for name, n := range want {
		if names[name] != n {
			t.Errorf("%s drawn %d times, want %d (all: %v)", name, names[name], n, names)
		}
	}
	if len(names) != len(want) {
		t.Errorf("sprites = %v", names)
	}
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.drawIncidents(dst, cam, sc)
}

func TestTheIncidentMarkerPulsesUnlessEffectsAreReduced(t *testing.T) {
	seen := map[int]bool{}
	for step := range 40 {
		seen[IncidentFrame(float64(step)*0.05, false)] = true
	}
	if len(seen) != art.IncidentFrames {
		t.Fatalf("the marker shows %d of %d frames", len(seen), art.IncidentFrames)
	}
	if IncidentFrame(1.37, true) != 0 || IncidentFrame(2.91, true) != 0 {
		t.Fatal("reduced effects still pulse the marker")
	}
}

func TestTheBackoffRingFollowsTheKubeletsDoubling(t *testing.T) {
	for restarts, want := range map[int32]int{0: 0, 1: 1, 3: 3, 6: 6, 40: 6} {
		if got := BackoffStep(restarts); got != want {
			t.Errorf("BackoffStep(%d) = %d, want %d", restarts, got, want)
		}
	}
}
