package app

import (
	"fmt"
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/world"
)

func TestFallenWorkersTakeTheirSlotsInTheSameOrderEveryTime(t *testing.T) {
	homes := func(name string) (world.Vec, bool) { return world.Vec{X: float64(len(name)), Y: 2}, true }
	pods := []model.PodView{}
	for _, n := range []string{"c", "a", "d", "b"} {
		pods = append(pods, model.PodView{UID: "uid-" + n, Name: "porters-" + n, Sprite: "porter", State: model.HearthReady})
	}
	var first []string
	for pass := 0; pass < 60; pass++ {
		town := &Town{}
		town.WatchPods(pods, homes, 0, true)
		town.WatchPods(nil, homes, 1, true)
		got := []string{}
		for _, e := range town.Active {
			got = append(got, fmt.Sprintf("%s@%.2f", e.Pod, e.Start))
		}
		if len(got) == 0 {
			t.Fatal("no worker fell when every hut went out")
		}
		if first == nil {
			first = got
			continue
		}
		if !slices.Equal(got, first) {
			t.Fatalf("pass %d: the workers fell as %v, the first pass as %v", pass, got, first)
		}
	}
}
