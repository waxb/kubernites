package kubectltree_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
)

func TestConfigViewRedactsAServersUserinfo(t *testing.T) {
	f := newFake(t)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := "apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: " + f.URL + "\n- name: corp\n  cluster:\n    server: https://alice:SERVER-ADDRESS-PASSWORD@corp.invalid:6443\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\nusers:\n- name: fake-user\n  user:\n    token: not-a-real-token\n"
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := stoneTree(t, path, live.ReadOnlyConfig)
	for _, line := range []string{"config view", "config view -o jsonpath={.clusters[*].cluster.server}"} {
		res := run(t, tree, line)
		t.Logf("%s: code=%d\n%s", line, res.Code, res.Output)
		if strings.Contains(res.Output, "SERVER-ADDRESS-PASSWORD") {
			t.Errorf("%s printed a password held in a cluster's server address", line)
		}
	}
}
