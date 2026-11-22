package app

import (
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestTheClustersOwnWordsAreExactUnderKubernetesNamesAndNeverRenamedInStory(t *testing.T) {
	defer ui.SetNamer(nil)
	raw := `deployments.apps "web" is forbidden: User "viewer@example.com" cannot patch resource "deployments/scale" in API group "apps" in the namespace "market"`
	at := time.Date(2026, 11, 21, 9, 30, 0, 0, time.UTC)
	line := ledgerLine{at: at, verb: write.OrderScale, target: write.Ref{GVR: write.Deployments, Kind: "Deployment", Namespace: "market", Name: "web"}, code: 403, answer: raw}
	failed := drainFailure{hut: "kv-0", why: raw}
	shown := func() []string {
		return []string{line.text(), failed.shown(), clusterSaid("SCALE: ", raw), clusterSaid("", raw), refusedTip("scale it", raw)}
	}

	ui.SetStoryNamer(newStoryNamer())
	for _, text := range shown() {
		drawn := ui.Name(text)
		for _, word := range []string{"forbidden", "API group", "/scale", "viewer@example.com", "deployments"} {
			if strings.Contains(drawn, word) {
				t.Errorf("story mode shows the cluster's %q: %q", word, drawn)
			}
		}
		if !strings.Contains(strings.ToLower(drawn), "switch names to read") {
			t.Errorf("story mode does not say where the cluster's words are: %q", drawn)
		}
	}

	ui.SetNamer(names.Kubernetes(64))
	var f ui.Font
	for _, text := range shown() {
		if drawn := ui.Name(text); !strings.Contains(drawn, raw) {
			t.Errorf("kubernetes names change the cluster's words: %q", drawn)
		}
		if lines := f.Wrap(text, 200*ui.Advance, 1); !strings.Contains(strings.Join(lines, " "), raw) {
			t.Errorf("a wrapped line changes the cluster's words: %q", lines)
		}
	}
}
