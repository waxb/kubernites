package live

import (
	"os"
	"path/filepath"
	"testing"
)

func identityOfSample(t *testing.T, insecure bool) ContextInfo {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	extra := "    certificate-authority-data: Y2VydA==\n"
	if insecure {
		extra = "    insecure-skip-tls-verify: true\n"
	}
	cfg := "apiVersion: v1\nkind: Config\nclusters:\n- name: prod\n  cluster:\n    server: https://prod.example:6443\n    tls-server-name: api.prod\n" + extra +
		"users:\n- name: admin\n  user:\n    token: abc\ncontexts:\n- name: prod\n  context:\n    cluster: prod\n    user: admin\ncurrent-context: prod\n"
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	list, err := ListContexts([]string{path})
	if err != nil || len(list) != 1 {
		t.Fatalf("list %v %v", list, err)
	}
	return list[0]
}

func TestTheIdentityKeyIsUnchangedByNamingServerVerificationApart(t *testing.T) {
	for insecure, want := range map[bool][2]string{false: {"680c69805b9304d6fb55352d16bf76c29b98fb42bf372cad4ccf1918d7d4954f", "e770c872265487d69c73de4d4d9f748b15aff2d0f4570714970b037941b1e894"}, true: {"d8b76b87ff9d448d71a47a135ecc7a27b577d46123225fa091b01e7050e0f1f2", "f8aef848fbbd72ca8006abc6167346a93d4a59712aa715e54211f72a5f72f5a9"}} {
		got := identityOfSample(t, insecure)
		if got.Identity != want[0] || got.ClusterID != want[1] {
			t.Errorf("insecure %v: the identity %s and cluster %s differ from those remembered before, so every remembered choice would be dropped", insecure, got.Identity, got.ClusterID)
		}
	}
}
