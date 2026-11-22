package live

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/waxb/kubernites/pkg/live/write"
)

func identityKubeconfig(t *testing.T, path, cluster, user string) {
	t.Helper()
	text := "apiVersion: v1\nkind: Config\nclusters:\n- name: prod\n  cluster:\n" + cluster +
		"users:\n- name: prod\n  user:\n" + user +
		"contexts:\n- name: prod\n  context:\n    cluster: prod\n    user: prod\ncurrent-context: prod\n"
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func identityListed(t *testing.T, path string) ContextInfo {
	t.Helper()
	list, err := ListContexts([]string{path})
	if err != nil || len(list) != 1 {
		t.Fatalf("%s: %v %v", path, list, err)
	}
	return list[0]
}

func TestTheIdentityCoversImpersonationTokenFileContentsAndTLSServerName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	tokenFile := filepath.Join(dir, "token")
	if err := os.WriteFile(tokenFile, []byte("reader-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	cluster := "    server: https://127.0.0.1:6443\n    certificate-authority-data: Y2E=\n"
	user := "    tokenFile: " + tokenFile + "\n"
	identityKubeconfig(t, path, cluster, user)
	base := identityListed(t, path).Identity

	identityKubeconfig(t, path, cluster, user+"    as: system:admin\n    as-groups:\n    - system:masters\n")
	if identityListed(t, path).Identity == base {
		t.Error("impersonating system:admin in system:masters kept the identity: a remembered WRITES or DANGEROUSLY SEAMLESS carries over with no warning")
	}

	identityKubeconfig(t, path, cluster+"    tls-server-name: other-cluster.internal\n", user)
	if identityListed(t, path).Identity == base {
		t.Error("another tls-server-name kept the identity")
	}

	identityKubeconfig(t, path, cluster, user)
	if err := os.WriteFile(tokenFile, []byte("admin-token"), 0o600); err != nil {
		t.Fatal(err)
	}
	if identityListed(t, path).Identity == base {
		t.Error("the token file now holds another token and the identity is unchanged: only the file's name is hashed")
	}
}

func TestConnectRefusesCredentialsChangedSinceTheLevelWasChosen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	cluster := "    server: https://127.0.0.1:6443\n    insecure-skip-tls-verify: true\n"
	identityKubeconfig(t, path, cluster, "    token: reader-token\n")
	picked := identityListed(t, path)
	gate, err := write.New(write.Choice{Context: picked.Remembered(), Level: write.Seamless})
	if err != nil {
		t.Fatal(err)
	}
	identityKubeconfig(t, path, cluster, "    token: cluster-admin-token\n")
	if now := identityListed(t, path); now.Identity == picked.Identity {
		t.Fatalf("precondition: the rewritten token should change the identity")
	}
	conn, err := DialGated([]string{path}, picked.Name, gate)
	if err == nil {
		t.Errorf("the kubeconfig's token changed between the picker's listing and CONNECT, and DialGated connected at %s with the new credentials (connected identity %q, chosen for %q)", gate.Level(), conn.Context.Identity, gate.Context().Identity)
	}
}
