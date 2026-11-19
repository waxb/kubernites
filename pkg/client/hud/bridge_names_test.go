package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheSeeItInKubernetesPanelSpeaksKubernetesInNamesMode(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	v := view.BridgeView{Bends: []view.Liberty{{Area: view.LibertyScheduler, Bends: "The Quartermaster runs only a few scheduler plugins: taints, nodeSelector, resource fit (LeastAllocated) and topology spread."}}}
	rows := append(bridgeBends(v, bridgeBody().W), bridgeNext())
	for _, row := range rows {
		for _, story := range []string{"Quartermaster", "Stone"} {
			if strings.Contains(row, story) {
				t.Errorf("NAMES mode panel row says %q: %q", story, row)
			}
		}
	}
}
