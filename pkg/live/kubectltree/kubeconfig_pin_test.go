package kubectltree_test

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func repoint(t *testing.T, path, server string) {
	t.Helper()
	data, err := os.ReadFile(kubeconfig(t, server, "stone", "shop"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestAtSeamlessTheStoneNeverFollowsAKubeconfigRewrittenAfterConnect(t *testing.T) {
	a, b := newFake(t), newFake(t)
	path := kubeconfig(t, a.URL, "stone", "shop")
	g := gateAt(t, write.Seamless, a, path)
	tree := newTree(t, path, g.WrapConfig)
	repoint(t, path, b.URL)
	res := run(t, tree, "annotate deployment web note=hi")
	if got := b.seen.all(); len(got) != 0 {
		t.Fatalf("the server the kubeconfig was rewritten to saw %v\n%s", got, res.Output)
	}
}

func TestAtWritesTheConfirmsDryRunAndTheWriteGoToTheConnectedServerOnly(t *testing.T) {
	a, b := newFake(t), newFake(t)
	path := kubeconfig(t, a.URL, "stone", "shop")
	g := gateAt(t, write.Writes, a, path)
	tree := newTree(t, path, g.WrapConfig)
	asks := 0
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, g, a.URL, h, func(live.StoneAsk) bool { asks++; return true })
	})
	repoint(t, path, b.URL)
	res := run(t, tree, "annotate deployment web note=hi")
	if got := b.seen.all(); len(got) != 0 {
		t.Fatalf("the server the kubeconfig was rewritten to saw %v\n%s", got, res.Output)
	}
	if n := count(a.seen.writes(), "PATCH /apis/apps/v1/namespaces/shop/deployments/web"); n != 0 {
		t.Fatalf("a write went with no CONFIRM shown for it (asks=%d): %v\n%s", asks, a.seen.writes(), res.Output)
	}
}

func TestTheStoneKeepsTheServerAndCredentialsResolvedAtConnectAtEveryLevel(t *testing.T) {
	for _, level := range []write.Level{write.ReadOnly, write.Writes, write.Seamless} {
		t.Run(level.String(), func(t *testing.T) {
			a, b := newFake(t), newFake(t)
			path := kubeconfig(t, a.URL, "stone", "shop")
			connected, err := clientcmd.BuildConfigFromFlags("", path)
			if err != nil {
				t.Fatal(err)
			}
			wrap := live.ReadOnlyConfigAt(connected)
			var g *write.Gate
			if level != write.ReadOnly {
				g = gateAt(t, level, a, path)
				wrap = g.WrapConfig
			}
			tree := newTree(t, path, live.PinnedTo(connected, wrap))
			asks := 0
			tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
				return live.HoldStoneWrite(ctx, g, a.URL, h, func(live.StoneAsk) bool { asks++; return true })
			})
			repoint(t, path, b.URL)
			got := run(t, tree, "get deployment web -o name")
			if got.Code != 0 || !strings.Contains(got.Output, "deployment.apps/web") {
				t.Fatalf("a read after the rewrite:\n%s", got.Output)
			}
			res := run(t, tree, "annotate deployment web note=hi")
			if seen := b.seen.all(); len(seen) != 0 {
				t.Fatalf("the server the kubeconfig was rewritten to saw %v\n%s", seen, res.Output)
			}
			wrote := count(a.seen.writes(), "PATCH /apis/apps/v1/namespaces/shop/deployments/web")
			switch level {
			case write.ReadOnly:
				if wrote != 0 || res.Code == 0 {
					t.Fatalf("READ ONLY wrote %d times:\n%s", wrote, res.Output)
				}
			case write.Writes:
				if asks != 1 || wrote != 1 {
					t.Fatalf("asks=%d writes=%d on the connected server:\n%s", asks, wrote, res.Output)
				}
			default:
				if wrote != 1 {
					t.Fatalf("writes=%d on the connected server:\n%s", wrote, res.Output)
				}
			}
		})
	}
}

func tlsKubeconfig(t *testing.T, f *fakeServer, token string) string {
	t.Helper()
	ca := base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: f.Certificate().Raw}))
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := fmt.Sprintf("apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: %s\n    certificate-authority-data: %s\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\n    namespace: shop\nusers:\n- name: fake-user\n  user:\n    token: %s\n", f.URL, ca, token)
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTheWriteGateRefusesAConfigForAnotherServerOrUser(t *testing.T) {
	a := &fakeServer{seen: &seen{}}
	a.Server = httptest.NewTLSServer(http.HandlerFunc(a.serve))
	t.Cleanup(a.Close)
	b := newFake(t)
	path := tlsKubeconfig(t, a, "the-connected-token")
	g := gateAt(t, write.Seamless, a, path)
	if res := run(t, newTree(t, path, g.WrapConfig), "get deployment web -o name"); res.Code != 0 {
		t.Fatalf("the connected config was refused:\n%s", res.Output)
	}
	before := len(a.seen.all())
	for name, p := range map[string]string{"another server": kubeconfig(t, b.URL, "stone", "shop"), "another user": tlsKubeconfig(t, a, "another-token")} {
		res := run(t, newTree(t, p, g.WrapConfig), "get deployment web -o name")
		if res.Code == 0 || !strings.Contains(res.Output, "resolved at connect") {
			t.Errorf("%s: the gate let a request through:\n%s", name, res.Output)
		}
	}
	if seen := b.seen.all(); len(seen) != 0 {
		t.Fatalf("another server saw %v", seen)
	}
	if seen := a.seen.all()[before:]; len(seen) != 0 {
		t.Fatalf("the connected server saw requests made with another user's credentials: %v", seen)
	}
}
