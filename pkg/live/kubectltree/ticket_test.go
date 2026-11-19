package kubectltree_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestARefusedSendLeavesNoTicketBehind(t *testing.T) {
	f := newFake(t)
	path := kubeconfig(t, f.URL, "stone", "shop")
	g := gateAt(t, write.Writes, f, path)
	var stale atomic.Bool
	g.StaleWhen(func() string {
		if stale.Load() {
			return "watch lost"
		}
		return ""
	})
	tree := newTree(t, path, g.WrapConfig)
	asks := 0
	answer := true
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, g, f.URL, h, func(a live.StoneAsk) bool {
			asks++
			if a.Confirm() && answer {
				stale.Store(true)
			}
			return answer
		})
	})
	first := run(t, tree, "delete pod web-1 --wait=false")
	stale.Store(false)
	asksBefore := asks
	answer = false
	second := run(t, tree, "delete pod web-1 --wait=false")
	if first.Code == 0 {
		t.Fatalf("the send refused as stale answered 0:\n%s", first.Output)
	}
	if asks == asksBefore || second.Code == 0 {
		t.Fatalf("the same command typed again went with no CONFIRM (asks %d then %d, code %d):\n%s", asksBefore, asks, second.Code, second.Output)
	}
	if n := count(f.seen.writes(), "DELETE /api/v1/namespaces/shop/pods/web-1"); n != 0 {
		t.Fatalf("the server saw %d deletes: %v", n, f.seen.writes())
	}
}
