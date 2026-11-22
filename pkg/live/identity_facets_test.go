package live

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTurningOffServerVerificationIsNamedAsSuchAndNotAsAnotherCertificateAuthority(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	write := func(name, insecure string) []ContextInfo {
		path := filepath.Join(dir, name)
		cfg := "apiVersion: v1\nkind: Config\nclusters:\n- name: prod\n  cluster:\n    server: https://prod.example:6443\n" + insecure +
			"users:\n- name: admin\n  user:\n    token: abc\ncontexts:\n- name: prod\n  context:\n    cluster: prod\n    user: admin\ncurrent-context: prod\n"
		if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
			t.Fatal(err)
		}
		list, err := ListContexts([]string{path})
		if err != nil || len(list) != 1 {
			t.Fatalf("list %v %v", list, err)
		}
		return list
	}
	before := write("before.yaml", "")
	after := write("after.yaml", "    insecure-skip-tls-verify: true\n")
	if before[0].Identity == after[0].Identity {
		t.Fatal("turning off server verification kept the identity")
	}
	facets := ChangedFacets(before[0].Facets, after[0].Facets)
	named := strings.Join(facets, ", ")
	if !strings.Contains(strings.ToLower(named), "verif") {
		t.Errorf("only insecure-skip-tls-verify changed (the server's certificate is no longer checked at all), but the picker is told the kubeconfig now holds %q, read as \"another %s\"", named, named)
	}
}
