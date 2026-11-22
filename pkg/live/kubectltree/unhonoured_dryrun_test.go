package kubectltree_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestADryRunTheGateDoesNotHonourIsRecordedAsAChange(t *testing.T) {
	for _, line := range []string{"annotate gauges g note=hi --dry-run=server", "delete gauges g --dry-run=server --wait=false"} {
		f := &fakeServer{seen: &seen{}, aggregated: true}
		f.Server = httptest.NewServer(http.HandlerFunc(f.serve))
		path := kubeconfig(t, f.URL, "stone", "shop")
		g := gateAt(t, write.Seamless, f, path)
		if err := g.LearnLocalGroups(context.Background()); err != nil {
			t.Fatal(err)
		}
		var mu sync.Mutex
		var passages []write.Passage
		g.Observe(func(p write.Passage) {
			mu.Lock()
			passages = append(passages, p)
			mu.Unlock()
		})
		tree := newTree(t, path, g.WrapConfig)
		tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
			return live.HoldStoneWrite(ctx, g, f.URL, h, func(live.StoneAsk) bool { return true })
		})
		res := run(t, tree, line)
		var reached []string
		for _, w := range f.seen.writes() {
			if strings.Contains(w, "/apis/metrics.example.io/") {
				reached = append(reached, w)
			}
		}
		asWrite := 0
		mu.Lock()
		for _, p := range passages {
			if p.Write && strings.Contains(p.Path, "/apis/metrics.example.io/") {
				asWrite++
			}
		}
		mu.Unlock()
		f.Close()
		if len(reached) > 0 && asWrite > 0 && len(res.Writes) == 0 {
			t.Errorf("DANGEROUSLY SEAMLESS, %q: the gate let %v reach the aggregated API server as a change it cannot vouch is only a dry run (%d passages with Write true), yet the Stone records no write, so no ledger line or history:\n%s", line, reached, asWrite, res.Output)
		}
	}
}
