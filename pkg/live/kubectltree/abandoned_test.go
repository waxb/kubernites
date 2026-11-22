package kubectltree_test

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live"
)

func reflectors() int {
	buf := make([]byte, 8<<20)
	n := runtime.Stack(buf, true)
	return strings.Count(string(buf[:n]), "k8s.io/client-go/tools/cache.(*Reflector)")
}

func TestAnAbandonedCommandStopsRetryingAtItsNextRequest(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newFake(t)
	path := kubeconfig(t, f.URL, "stone", "shop")
	tree := stoneTree(t, path, live.ReadOnlyConfig)
	job := tree.Start("rollout status deployment/web -n shop", nil)
	for deadline := time.Now().Add(10 * time.Second); reflectors() == 0; time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("rollout status never started watching")
		}
	}
	job.Stop()
	for deadline := time.Now().Add(10 * time.Second); !job.Done(); time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("STOP did not end the command")
		}
	}
	for deadline := time.Now().Add(20 * time.Second); reflectors() > 0; time.Sleep(100 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("20 s after the Stone left rollout status, %d of its watch loops still retry against the Stone's refusals", reflectors())
		}
	}
}
