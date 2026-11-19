package live_test

import (
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

func TestAClusterWhoseFiefsCrowdTheMarketOpensOnAnIsleItsSize(t *testing.T) {
	for name, s := range map[string]live.Snapshot{"staging": livetest.Staging(), "crammed": livetest.Crammed()} {
		w, err := live.Build(s)
		if err != nil {
			t.Errorf("%s (%d nodes, %d pods, %d namespaces) did not open: %v", name, len(s.Nodes), len(s.Pods), len(s.Namespaces), err)
			continue
		}
		if w.Site.Width > 160 || w.Site.Height > 160 {
			t.Errorf("%s: %d nodes and %d pods drew a %dx%d realm: its fiefs were boxed in by their neighbours, not short of land", name, len(s.Nodes), len(s.Pods), w.Site.Width, w.Site.Height)
		}
		lots := map[string]bool{}
		for _, l := range w.Lots {
			if l.Node == "" {
				lots[l.Namespace] = true
			}
		}
		for _, ns := range []string{"harbourside", "ledgerhall", "strongroom", "steward", "market", "watchtower"} {
			if hasNamespace(s, ns) && !lots[ns] {
				t.Errorf("%s: the fief %s has no lot in its town", name, ns)
			}
		}
	}
}

func hasNamespace(s live.Snapshot, name string) bool {
	for _, ns := range s.Namespaces {
		if ns.Name == name {
			return true
		}
	}
	return false
}
