package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestAPodCardNamesItsImageInNamesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	c := PodCard(model.PodView{Name: "porter-lone-tower", Profession: "porter", Image: "porter:1.0", Static: true, Node: "lone-tower"}, view.PodMetrics{})
	for _, l := range c.Lines {
		if shown := ui.Name(l.Text); strings.Contains(shown, "porter:1.0") && shown != "IMAGE porter:1.0" {
			t.Errorf("the Pod card's image line reads %q", shown)
		}
	}
}
