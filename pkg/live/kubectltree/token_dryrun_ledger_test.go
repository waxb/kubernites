package kubectltree_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestATokenRequestSentAsADryRunNeverMintsATokenUnrecorded(t *testing.T) {
	request := filepath.Join(t.TempDir(), "tr.json")
	if err := os.WriteFile(request, []byte(`{"apiVersion":"authentication.k8s.io/v1","kind":"TokenRequest","spec":{"expirationSeconds":3600}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	minted := 0
	f := &fakeServer{seen: &seen{}}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/serviceaccounts/deployer/token") {
			minted++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"apiVersion":"authentication.k8s.io/v1","kind":"TokenRequest","metadata":{"name":"deployer","namespace":"shop"},"spec":{"expirationSeconds":3600},"status":{"token":"MINTED","expirationTimestamp":"2026-11-20T12:00:00Z"}}`)
			return
		}
		f.serve(w, r)
	}))
	t.Cleanup(f.Close)
	path := kubeconfig(t, f.URL, "stone", "shop")
	g := gateAt(t, write.Seamless, f, path)
	g.Pass(write.Unpreviewed)
	tree := newTree(t, path, g.WrapConfig)
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, g, f.URL, h, func(live.StoneAsk) bool { return true })
	})
	res := run(t, tree, "create --raw /api/v1/namespaces/shop/serviceaccounts/deployer/token?dryRun=All -f "+request)
	t.Logf("code=%d minted=%d writes=%+v\n%s", res.Code, minted, res.Writes, res.Output)
	if minted == 1 && len(res.Writes) == 0 {
		t.Fatalf("a token request that minted a real token at DANGEROUSLY SEAMLESS is not recorded as a change")
	}
}

func TestATokenRequestCarryingDryRunIsRefusedAtEveryLevelAndAMintedTokenIsRecorded(t *testing.T) {
	request := filepath.Join(t.TempDir(), "tr.json")
	if err := os.WriteFile(request, []byte(`{"apiVersion":"authentication.k8s.io/v1","kind":"TokenRequest","spec":{"expirationSeconds":3600}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, level := range []write.Level{write.Writes, write.Seamless} {
		for _, query := range []string{"?dryRun=All", "?dryRun=", "?dryRun=None", "?dryRun", ""} {
			minted := 0
			f := &fakeServer{seen: &seen{}}
			f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/serviceaccounts/deployer/token") {
					minted++
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					fmt.Fprint(w, `{"apiVersion":"authentication.k8s.io/v1","kind":"TokenRequest","metadata":{"name":"deployer","namespace":"shop"},"status":{"token":"MINTED","expirationTimestamp":"2026-11-20T12:00:00Z"}}`)
					return
				}
				f.serve(w, r)
			}))
			path := kubeconfig(t, f.URL, "stone", "shop")
			g := gateAt(t, level, f, path)
			for _, c := range write.AllClasses {
				g.Pass(c)
			}
			tree := newTree(t, path, g.WrapConfig)
			tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
				return live.HoldStoneWrite(ctx, g, f.URL, h, func(live.StoneAsk) bool { return true })
			})
			line := "create --raw /api/v1/namespaces/shop/serviceaccounts/deployer/token" + query + " -f " + request
			if query == "" {
				line = "create token deployer -n shop"
			}
			res := run(t, tree, line)
			f.Close()
			if query != "" {
				if minted != 0 || !strings.Contains(res.Output, write.TokenDryRun) {
					t.Errorf("at %s a token request with %s minted %d tokens or gave no reason:\n%s", level, query, minted, res.Output)
				}
				continue
			}
			if minted != 1 || len(res.Writes) != 1 || res.Writes[0].Method != http.MethodPost || !strings.HasSuffix(res.Writes[0].Path, "/serviceaccounts/deployer/token") {
				t.Errorf("at %s a minted token is not recorded as the command's one change: minted %d, writes %+v", level, minted, res.Writes)
			}
		}
	}
}
