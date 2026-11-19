package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

func TestEveryPodOfAFullLiveNodeGetsAHut(t *testing.T) {
	s := livetest.Crowded(2, 6, 110)
	w, err := live.Build(s)
	if err != nil {
		t.Fatal(err)
	}
	pods := make([]model.PodView, 0, len(s.Pods))
	for _, p := range s.Pods {
		pods = append(pods, model.PodView{Namespace: p.Namespace, Name: p.Name, UID: string(p.UID), Node: p.Spec.NodeName, State: "Running", Ready: true})
	}
	h := NewHamlet()
	h.Place(w.Site, pods, 0)
	if len(h.Homeless) != 0 || len(h.Spots()) != len(pods) {
		t.Fatalf("housed %d of %d pods, %d Unhoused at the gates", len(h.Spots()), len(pods), len(h.Homeless))
	}
}
