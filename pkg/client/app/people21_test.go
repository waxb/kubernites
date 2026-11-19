package app

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

func TestRefreshKeepsTheSelectedPerson(t *testing.T) {
	p := model.PodView{Namespace: "default", Name: "porters-7d9f-x2kq", Image: "porter:1.0", People: []model.Kubernite{
		{Name: "porter", Image: "porter:1.0"}, {Name: "loader", Image: "hauler:1.0"},
	}}
	sel := SelectionOf(world.Hit{Kind: world.HitWalker, ID: "default/" + p.Name, Pod: p.Focused("loader")}, nil)
	next := p
	next.Restarts = 2
	next.People = []model.Kubernite{{Name: "porter", Image: "porter:1.0"}, {Name: "loader", Image: "hauler:1.1", Restarts: 3}}
	got := RefreshSelection(sel, view.Site{}, []model.PodView{next})
	if got.Pod.Focus != "loader" || got.Pod.Image != "hauler:1.1" || got.Pod.Restarts != 3 {
		t.Errorf("refreshed person = %+v", got.Pod)
	}
	hut := RefreshSelection(hud.Selection{Kind: world.HitHearth, ID: "default/" + p.Name, Pod: p}, view.Site{}, []model.PodView{next})
	if hut.Pod.Focus != "" || hut.Pod.Restarts != 2 {
		t.Errorf("refreshed hut = %+v", hut.Pod)
	}
}
