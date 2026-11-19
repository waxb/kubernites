package world

import (
	"fmt"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
)

var coverOffsets = []Point{{1, 1}, {1, 0}, {0, 1}, {2, 1}, {1, 2}}

func covering(spots map[string]Point) []string {
	var out []string
	for a, pa := range spots {
		for b, pb := range spots {
			for _, d := range coverOffsets {
				if pa.X-pb.X == d.X && pa.Y-pb.Y == d.Y {
					out = append(out, fmt.Sprintf("%s at %v covers %s at %v", a, pa, b, pb))
				}
			}
		}
	}
	return out
}

func TestHutsWithRoomNeverCoverEachOther(t *testing.T) {
	site := hamletSite()
	cases := map[string][]model.PodView{
		"one crew":       crewPods("porters", "parcels", "lone-tower", 12),
		"three crews":    append(append(crewPods("porters", "parcels", "lone-tower", 8), crewPods("clerks", "ledgers", "lone-tower", 6)...), crewPods("smiths", "", "lone-tower", 6)...),
		"crews and bare": append(crewPods("porters", "", "lone-tower", 10), bareHuts("lone-tower", 4)...),
		"the small Keep": crewPods("clerks", "ledgers", "stone-keep-2", 9),
	}
	for name, pods := range cases {
		spots := hutsOf(t, site, pods)
		if len(spots) != len(pods) {
			t.Errorf("%s: housed %d of %d", name, len(spots), len(pods))
		}
		for _, c := range covering(spots) {
			t.Errorf("%s: %s", name, c)
		}
	}
}

func bareHuts(node string, n int) []model.PodView {
	out := []model.PodView{}
	for i := range n {
		out = append(out, model.PodView{Namespace: "default", Name: fmt.Sprintf("bare-%d", i), Node: node, Sprite: "porter", State: model.HearthReady})
	}
	return out
}

func TestACrowdedKeepPacksEveryPlotBeforeAnyoneWaits(t *testing.T) {
	site := hamletSite()
	site.Structures[1].Territory = 3
	pods := crewPods("porters", "", "stone-keep-2", 40)
	h := NewHamlet()
	h.Place(site, pods, 0)
	land := len(h.list(site, h.areas["stone"]))
	if housed := len(h.Spots()); housed != land || len(h.Homeless) != len(pods)-land {
		t.Errorf("housed %d on %d plots, %d wait", housed, land, len(h.Homeless))
	}
}
