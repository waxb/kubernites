package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func TestAPodServingAtItsNodesPortIsNotCalledIdle(t *testing.T) {
	for _, p := range []model.PodView{
		{Name: "porter-lone-tower", State: model.HearthReady, NodePort: 8080, HostNetwork: true},
		{Name: "porter-lone-tower", State: model.HearthReady, NodePort: 80},
	} {
		tip := HutTip(p, view.PodMetrics{}, "", "")
		if strings.Contains(tip, "idle") || !strings.Contains(tip, "door") && !strings.Contains(tip, "port") {
			t.Fatalf("a door Pod between requests reads:\n%s", tip)
		}
	}
	if tip := HutTip(model.PodView{Name: "web-1", State: model.HearthReady}, view.PodMetrics{}, "", ""); !strings.Contains(tip, "idle") {
		t.Fatalf("a Pod no Service selects and no port serves is idle:\n%s", tip)
	}
}
