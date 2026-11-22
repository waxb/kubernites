package kubectltree_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
)

func TestNoStoneAnswerShowsTheUserinfoOfAServerAddress(t *testing.T) {
	f := newFake(t)
	withUser := strings.Replace(f.URL, "http://", "http://alice:SERVER-PW@", 1)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := "apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: " + withUser + "\n- name: gone\n  cluster:\n    server: https://bob:GONE-PW@127.0.0.1:1\n    proxy-url: http://carol:PROXY-PW@proxy.invalid:3128\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\n    namespace: shop\n- name: gone\n  context:\n    cluster: gone\n    user: fake-user\nusers:\n- name: fake-user\n  user:\n    token: not-a-real-token\n"
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := stoneTree(t, path, live.ReadOnlyConfig)
	for _, line := range []string{
		"config view", "config view -o json", "config view -o yaml", "config view -o jsonpath={.clusters[*].cluster.server}",
		"config view --minify", "cluster-info", "get pods -n shop --request-timeout=2s", "get --raw /api/v1/namespaces/shop/pods/missing",
	} {
		res := run(t, tree, line)
		for _, s := range []string{"alice", "SERVER-PW", "bob", "GONE-PW", "carol", "PROXY-PW"} {
			if strings.Contains(res.Output, s) {
				t.Errorf("%s showed %q:\n%s", line, s, res.Output)
			}
		}
	}
	gone, err := live.ListContexts([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range gone {
		if strings.Contains(c.Server, "@") {
			t.Errorf("the picker lists context %s at %s", c.Name, c.Server)
		}
	}
	if _, err := live.WhoAmI(t.Context(), []string{path}, "gone"); err == nil || strings.Contains(err.Error(), "GONE-PW") || strings.Contains(err.Error(), "bob") {
		t.Errorf("an error reaching a server with userinfo shows it: %v", err)
	}
}
