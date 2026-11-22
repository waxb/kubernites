package kubectltree_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestAStoneChangeWhoseAnswerNeverArrivesIsRecorded(t *testing.T) {
	for _, level := range []write.Level{write.Writes, write.Seamless} {
		var applied atomic.Int32
		f := &fakeServer{seen: &seen{}}
		f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/deployments/web") && r.URL.Query().Get("dryRun") == "" {
				applied.Add(1)
				f.seen.add(mark(r))
				select {
				case <-r.Context().Done():
				case <-time.After(3 * time.Second):
				}
				return
			}
			f.serve(w, r)
		}))
		path := kubeconfig(t, f.URL, "stone", "shop")
		g := gateAt(t, level, f, path)
		tree := newTree(t, path, g.WrapConfig)
		asks := 0
		tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
			return live.HoldStoneWrite(ctx, g, f.URL, h, func(live.StoneAsk) bool {
				asks++
				return true
			})
		})
		res := run(t, tree, "annotate deployment web -n shop note=hi --request-timeout=1s")
		f.Close()
		if applied.Load() == 1 && len(res.Writes) == 0 {
			t.Errorf("%s: the gate sent a PATCH (after %d asks) that the API server carried out, its answer never came back before kubectl gave up, yet the Stone records no write, so no ledger line or history:\n%s", level, asks, res.Output)
		}
	}
}
