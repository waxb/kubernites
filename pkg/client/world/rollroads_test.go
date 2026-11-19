package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func rollScene(selected string) Scene {
	stall := view.Structure{ID: "stall-1", Kind: "depot-parcel", X: 10, Y: 10, W: 2, H: 1, State: view.StateReady, Object: view.ObjectRef{Kind: "Service", Name: "parcels"}}
	return Scene{
		Site:     view.Site{Width: 40, Height: 40, Structures: []view.Structure{stall}},
		Selected: selected,
		Crews:    []model.CrewChain{{Namespace: "default", Name: "porters", Depot: "parcels"}},
		Pods: []model.PodView{
			{Namespace: "default", Name: "porters-a", Crew: "porters", Ready: true},
			{Namespace: "default", Name: "porters-b", Crew: "porters", Ready: true},
			{Namespace: "default", Name: "clerks-c", Crew: "clerks", Ready: true},
		},
		Watch: &view.WatchView{Huts: []view.HutWatchView{{Pod: "porters-b", Crew: "porters", Shut: true}}},
	}
}

func TestTheGuildRollLightsReadyHutsAndDimsShutOnes(t *testing.T) {
	homes := map[string]Vec{"porters-a": {X: 14, Y: 12}, "porters-b": {X: 16, Y: 12}, "clerks-c": {X: 18, Y: 12}}
	for _, sel := range []string{"stall-1", "default/porters"} {
		roads := rollRoads(rollScene(sel), homes)
		got := map[string]bool{}
		for _, r := range roads {
			got[r.pod] = r.ready
		}
		if len(roads) != 2 || !got["porters-a"] || got["porters-b"] {
			t.Fatalf("selecting %s draws %+v", sel, roads)
		}
	}
	if roads := rollRoads(rollScene(""), homes); len(roads) != 0 {
		t.Fatalf("nothing selected, yet roads %+v", roads)
	}
	r := NewRenderer(testLib())
	r.Homes = homes
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	r.drawRollRoads(dst, render.NewCamera(480, 60).CenterOn(14, 12, 960, 540), rollScene("stall-1"))
}
