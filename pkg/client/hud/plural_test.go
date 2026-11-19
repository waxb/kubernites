package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestOneWorkerNeverReadsAsWorkersInEitherMode(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, mode := range namesModes() {
		mode.apply()
		texts := []string{
			RaiseRowText(RaiseRow{Crew: "porters", Name: "parcels", Standing: true, Replicas: 1, Images: []string{"porter:1.0"}}),
			foremanChooserLine(RaiseRow{Crew: "porters"}, 1),
		}
		title, cause := RolloutBanner(model.RolloutDone{Crew: "porters", Image: "porter:2.0", Huts: 1})
		texts = append(texts, title, cause)
		for _, text := range texts {
			for _, bad := range []string{"1 workers", "1 Pods", "1 huts", "1 porters", "stands, "} {
				if strings.Contains(text, bad) {
					t.Errorf("[%s] %q reads %q", mode.name, text, bad)
				}
			}
		}
	}
}
