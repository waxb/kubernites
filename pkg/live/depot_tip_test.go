package live_test

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live/livetest"
)

func TestADepotTipCountsOneHutAsOne(t *testing.T) {
	w := build(t, livetest.Managed())
	seen := 0
	for id, tip := range w.Tips {
		if !strings.HasPrefix(tip.Story, "Depot ") {
			continue
		}
		for _, wrong := range []string{" 1 huts", " 1 endpoints"} {
			if strings.Contains(tip.Story, wrong) || strings.Contains(tip.Names, wrong) {
				t.Errorf("depot %s: %+v", id, tip)
			}
		}
		if strings.HasSuffix(tip.Story, " 1 hut on its roll") && strings.HasSuffix(tip.Names, " 1 endpoint") {
			seen++
		}
	}
	if seen == 0 {
		t.Fatal("no depot with one hut, so this does not test the singular")
	}
}
