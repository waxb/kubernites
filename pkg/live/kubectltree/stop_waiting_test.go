package kubectltree_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestStopEndsCommandsThatWait(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var stuck []string
	for _, c := range []struct {
		line  string
		level write.Level
	}{
		{"rollout status deployment/web -n shop", write.ReadOnly},
		{"delete pod web-1 -n shop", write.Seamless},
	} {
		f := newFake(t)
		path := kubeconfig(t, f.URL, "stone", "shop")
		var tree *kubectltree.Tree
		if c.level == write.ReadOnly {
			tree = stoneTree(t, path, live.ReadOnlyConfig)
		} else {
			tree = stoneTree(t, path, gateAt(t, c.level, f, path).WrapConfig)
		}
		job := tree.Start(c.line, nil)
		time.Sleep(2 * time.Second)
		job.Stop()
		deadline := time.Now().Add(6 * time.Second)
		for time.Now().Before(deadline) && !job.Done() {
			time.Sleep(50 * time.Millisecond)
		}
		chunks, _ := job.Take()
		var out []string
		for _, ch := range chunks {
			out = append(out, ch.Text)
		}
		next := make(chan struct{})
		go func() {
			tree.Run(context.Background(), "version --client")
			close(next)
		}()
		nextWaits := false
		select {
		case <-next:
		case <-time.After(3 * time.Second):
			nextWaits = true
		}
		t.Logf("%s: done 6s after Stop=%v, next command still waiting 3s later=%v, server saw %d requests, output:\n%s", c.line, job.Done(), nextWaits, len(f.seen.all()), strings.Join(out, ""))
		if !job.Done() {
			stuck = append(stuck, c.line)
		}
	}
	if len(stuck) > 0 {
		t.Fatalf("STOP did not end these Stone commands; each keeps the Stone's lock, so every later command waits behind it:\n%s", strings.Join(stuck, "\n"))
	}
}
