package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func TestAPersonsCardNamesTheContainerAndItsImage(t *testing.T) {
	p := model.PodView{Namespace: "default", Name: "porters-7d9f-x2kq", Image: "porter:1.0", Profession: "Porter", Sprite: "porter",
		State: model.HearthReady, People: []model.Kubernite{
			{Name: "porter", Image: "porter:1.0", Profession: "Porter", Sprite: "porter"},
			{Name: "loader", Image: "hauler:1.0", Profession: "Hauler", Sprite: "hauler"},
		}}
	lines := func(c Card) string {
		out := []string{}
		for _, l := range c.Lines {
			out = append(out, l.Text)
		}
		return strings.Join(out, "|")
	}
	whole := PodCard(p, view.PodMetrics{})
	if strings.Contains(lines(whole), "WORKER ") {
		t.Errorf("the hut's card names no single container: %s", lines(whole))
	}
	person := PodCard(p.Focused("loader"), view.PodMetrics{})
	if !strings.Contains(lines(person), "WORKER loader (2 OF 2)") || !strings.Contains(lines(person), "TRADE hauler:1.0") {
		t.Errorf("the person's card = %s", lines(person))
	}
	if person.Title != "HAULER" || person.K8s != "Pod/"+p.Name {
		t.Errorf("title %q k8s %q", person.Title, person.K8s)
	}
	if got := KuberniteLine(model.PodView{Focus: "solo"}); got != "WORKER solo" {
		t.Errorf("line without people = %q", got)
	}
}
