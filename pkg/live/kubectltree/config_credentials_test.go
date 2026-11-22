package kubectltree_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
)

func TestTheStonePrintsNoCredentialTheConnectedKubeconfigHolds(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newFake(t)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := "apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: " + f.URL + "\n- name: corp\n  cluster:\n    server: https://corp.invalid\n    proxy-url: http://alice:KEPT-PROXY-PASSWORD@proxy.invalid:3128\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\n    namespace: shop\nusers:\n- name: fake-user\n  user:\n    token: KEPT-PLAIN-TOKEN\n"
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := stoneTree(t, path, live.ReadOnlyConfig)
	res := run(t, tree, "config view")
	t.Logf("config view: code=%d refused=%q\n%s", res.Code, res.Refused, res.Output)
	raw := run(t, tree, "create configmap k -n shop --from-file=config="+path+" --dry-run=client -o yaml")
	t.Logf("create configmap --from-file=<connected kubeconfig>: code=%d refused=%q\n%s", raw.Code, raw.Refused, raw.Output)
	if strings.Contains(res.Output, "KEPT-PROXY-PASSWORD") || strings.Contains(raw.Output, "KEPT-PLAIN-TOKEN") {
		t.Fatalf("the Stone printed a credential held in the connected kubeconfig")
	}
}

func TestConfigViewRedactsAProxysCredentialsAndKeepsItsAddress(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newFake(t)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := "apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: " + f.URL + "\n    proxy-url: http://alice:KEPT-PROXY-PASSWORD@proxy.invalid:3128\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\nusers:\n- name: fake-user\n  user:\n    token: KEPT-PLAIN-TOKEN\n"
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := stoneTree(t, path, live.ReadOnlyConfig)
	for _, line := range []string{"config view", "config view -o json", "config view -o jsonpath={.clusters[0].cluster.proxy-url}"} {
		res := run(t, tree, line)
		if strings.Contains(res.Output, "KEPT-PROXY-PASSWORD") || strings.Contains(res.Output, "alice") {
			t.Errorf("%s printed the proxy's credentials:\n%s", line, res.Output)
		}
		if !strings.Contains(res.Output, "proxy.invalid:3128") {
			t.Errorf("%s dropped the proxy's address:\n%s", line, res.Output)
		}
	}
}
