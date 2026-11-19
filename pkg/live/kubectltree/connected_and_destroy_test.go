package kubectltree_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func confirmingTree(t *testing.T, level write.Level, f *fakeServer, path string) (*kubectltree.Tree, *[]live.StoneAsk) {
	t.Helper()
	g := gateAt(t, level, f, path)
	tree := newTree(t, path, g.WrapConfig)
	asks := &[]live.StoneAsk{}
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, g, f.URL, h, func(a live.StoneAsk) bool {
			*asks = append(*asks, a)
			return true
		})
	})
	return tree, asks
}

func TestTheStoneNeverWritesToAServerAKubeconfigRewriteNames(t *testing.T) {
	a, b := newFake(t), newFake(t)
	path := kubeconfig(t, a.URL, "stone", "shop")
	tree, asks := confirmingTree(t, write.Writes, a, path)
	other, err := os.ReadFile(kubeconfig(t, b.URL, "stone", "shop"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, other, 0o600); err != nil {
		t.Fatal(err)
	}
	res := run(t, tree, "scale deployment web --replicas=3")
	for _, x := range *asks {
		t.Logf("CONFIRM asked: why=%v connect=%v request=%s %s preview=%d", x.Why, x.Connect, x.Order.Request.Method, x.Order.Request.Path, x.Preview.Code)
	}
	if len(b.seen.writes()) > 0 {
		t.Fatalf("a write reached server B %s, which no warning named; the dry run shown came from server A %s: %v", b.URL, a.URL, b.seen.writes())
	}
	if got := b.seen.all(); len(got) != 0 {
		t.Fatalf("server B %s, which no warning named, saw %v\n%s", b.URL, got, res.Output)
	}
}

func TestDeletingBySelectorAsksTheDestroyingWarning(t *testing.T) {
	f := newFake(t)
	path := kubeconfig(t, f.URL, "stone", "shop")
	tree, asks := confirmingTree(t, write.Seamless, f, path)
	res := run(t, tree, "delete pods -l app=web -n shop --wait=false")
	destroyAsked := false
	for _, x := range *asks {
		if x.Class == write.Destroy {
			destroyAsked = true
		}
	}
	if len(f.seen.writes()) > 0 && !destroyAsked {
		t.Fatalf("kubectl delete -l deleted %v at DANGEROUSLY SEAMLESS without the destroying-more-than-one-thing warning\n%s", f.seen.writes(), strings.TrimSpace(res.Output))
	}
}

func TestEveryDeleteOfMoreThanOneThingAsksTheDestroyingWarning(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "web.yaml")
	if err := os.WriteFile(manifest, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: web\n  namespace: shop\n  labels:\n    app: web\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for line, want := range map[string]bool{
		"delete pods -l app=web -n shop --wait=false":                            true,
		"delete pods --selector=app=web -n shop --wait=false":                    true,
		"delete pods --all -n shop --wait=false":                                 true,
		"delete pods --field-selector=status.phase=Running -n shop --wait=false": true,
		"delete pod web-1 web-2 -n shop --wait=false":                            true,
		"delete pod/web-1 pod/web-2 -n shop --wait=false":                        true,
		"delete pods,services -l app=web -n shop --wait=false":                   true,
		"apply --prune -l app=web -f " + manifest:                                true,
		"delete pod web-1 -n shop --wait=false":                                  false,
		"delete pod/web-1 -n shop --wait=false":                                  false,
	} {
		f := newFake(t)
		tree, asks := confirmingTree(t, write.Seamless, f, kubeconfig(t, f.URL, "stone", "shop"))
		res := run(t, tree, line)
		asked := false
		for _, x := range *asks {
			if x.Class == write.Destroy {
				asked = true
			}
		}
		if asked != want {
			t.Errorf("%s: the destroying warning asked=%v, want %v (writes %v)\n%s", line, asked, want, f.seen.writes(), strings.TrimSpace(res.Output))
		}
	}
}
