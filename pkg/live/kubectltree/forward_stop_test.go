package kubectltree_test

import (
	"context"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestStoppingAForwardWhileItsConfirmWaitsNeverBlocks(t *testing.T) {
	tree, _, _ := shellTree(t, write.Writes, write.Workloads)
	asked := make(chan struct{}, 1)
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	hold := func(ctx context.Context, h kubectltree.Held) bool {
		select {
		case asked <- struct{}{}:
		default:
		}
		select {
		case <-release:
		case <-ctx.Done():
		}
		return false
	}
	job := tree.Start("port-forward web-1 -n shop :80", hold)
	select {
	case <-asked:
	case <-time.After(15 * time.Second):
		t.Fatal("no CONFIRM was asked")
	}
	fws := tree.Forwards()
	for _, fw := range fws {
		stopped := make(chan struct{})
		go func() {
			tree.StopForward(fw.ID)
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-time.After(2 * time.Second):
			t.Fatalf("STOP on forward %+v waited for its CONFIRM", fw)
		}
	}
	if len(fws) != 0 {
		t.Errorf("a forward was listed with STOP before it forwarded anything: %+v", fws)
	}
	job.Stop()
	until(t, "the job ended", job.Done)
}
