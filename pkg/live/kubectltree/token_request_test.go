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

func tokenPosts(f *fakeServer) []string {
	var out []string
	for _, w := range f.seen.writes() {
		if strings.HasPrefix(w, "POST ") && strings.Contains(w, "/serviceaccounts/deployer/token") {
			out = append(out, w)
		}
	}
	return out
}

func TestATokenRequestAsksTheNoPreviewWarningAndAtWritesLeavesOnlyAtItsConfirm(t *testing.T) {
	request := filepath.Join(t.TempDir(), "tr.json")
	if err := os.WriteFile(request, []byte(`{"apiVersion":"authentication.k8s.io/v1","kind":"TokenRequest","spec":{"expirationSeconds":3600}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, level := range []write.Level{write.Writes, write.Seamless} {
		for _, line := range []string{"create token deployer -n shop"} {
			for _, yes := range []bool{false, true} {
				f := newFake(t)
				path := kubeconfig(t, f.URL, "stone", "shop")
				g := gateAt(t, level, f, path)
				tree := newTree(t, path, g.WrapConfig)
				var asks []live.StoneAsk
				posted := 0
				tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
					return live.HoldStoneWrite(ctx, g, f.URL, h, func(a live.StoneAsk) bool {
						asks = append(asks, a)
						posted = len(tokenPosts(f))
						if a.Why == write.NeedsClass {
							return true
						}
						return yes
					})
				})
				run(t, tree, line)
				var class, confirm *live.StoneAsk
				for i := range asks {
					switch asks[i].Why {
					case write.NeedsClass:
						class = &asks[i]
					case write.NeedsConfirm:
						confirm = &asks[i]
					}
				}
				if class == nil || class.Class != write.Tokens || !strings.Contains(class.Detail, "mints a real token") {
					t.Errorf("%s at %s: the token warning was not asked first, saying it mints a real token: %+v", line, level, asks)
				}
				if level == write.Seamless {
					if confirm != nil {
						t.Errorf("%s at DANGEROUSLY SEAMLESS asked a CONFIRM", line)
					}
					continue
				}
				if confirm == nil || !confirm.Unpreviewed || !strings.Contains(confirm.Detail, "valid until it expires") {
					t.Errorf("%s at WRITES: its CONFIRM does not say it has no preview and mints a real token: %+v", line, confirm)
				}
				if posted != 0 {
					t.Errorf("%s at WRITES: %d token requests reached the server before CONFIRM was answered", line, posted)
				}
				switch got := tokenPosts(f); {
				case !yes && len(got) != 0:
					t.Errorf("%s at WRITES with CONFIRM declined sent %v", line, got)
				case yes && len(got) != 1:
					t.Errorf("%s at WRITES with CONFIRM sent %v, not one token request", line, got)
				}
			}
		}
	}
	f := newFake(t)
	path := kubeconfig(t, f.URL, "stone", "shop")
	g := gateAt(t, write.Writes, f, path)
	g.Pass(write.Tokens)
	tree := newTree(t, path, g.WrapConfig)
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, g, f.URL, h, func(live.StoneAsk) bool { return true })
	})
	res := run(t, tree, "create --raw /api/v1/namespaces/shop/serviceaccounts/deployer/token?dryRun=All -f "+request)
	if got := tokenPosts(f); len(got) != 0 || res.Code == 0 || !strings.Contains(res.Output, write.TokenDryRun) {
		t.Errorf("at WRITES a TokenRequest marked dryRun=All was not refused with its reason: %v (code %d)\n%s", got, res.Code, res.Output)
	}
}
