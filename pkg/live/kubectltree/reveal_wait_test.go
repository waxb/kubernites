package kubectltree_test

import (
	"context"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestARevealNeverWaitsBehindASendHeldAtItsConfirm(t *testing.T) {
	f := newFake(t)
	path := kubeconfig(t, f.URL, "stone", "shop")
	g := gateAt(t, write.Writes, f, path)
	tree := newTree(t, path, g.WrapConfig)
	held := make(chan struct{})
	job := tree.StartApply("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: c\n  namespace: shop\ndata:\n  a: b\n", false, func(ctx context.Context, h kubectltree.Held) bool {
		select {
		case <-held:
		default:
			close(held)
		}
		<-ctx.Done()
		return false
	})
	select {
	case <-held:
	case <-time.After(20 * time.Second):
		job.Stop()
		t.Fatal("the send never reached its CONFIRM")
	}
	revealed := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := tree.Reveal(ctx, "shop", "s", "password")
		revealed <- err
	}()
	select {
	case err := <-revealed:
		t.Logf("reveal returned: %v", err)
		job.Stop()
	case <-time.After(4 * time.Second):
		job.Stop()
		<-revealed
		t.Fatal("a reveal with a one-second deadline waited more than four seconds behind a send held at its CONFIRM; on the game thread that send's CONFIRM can never be answered")
	}
}
