package live

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/waxb/kubernites/pkg/live/write"
)

func tokenKubeconfig(t *testing.T, path, token string) {
	t.Helper()
	text := "apiVersion: v1\nkind: Config\nclusters:\n- name: default\n  cluster:\n    server: https://127.0.0.1:6443\n    insecure-skip-tls-verify: true\n" +
		"users:\n- name: default\n  user:\n    token: " + token + "\n" +
		"contexts:\n- name: default\n  context:\n    cluster: default\n    user: default\ncurrent-context: default\n"
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestAnIdentityChangedBetweenTheListAndConnectIsRefused(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "config.yaml")
	tokenKubeconfig(t, path, "laptop-token")
	list, err := ListContexts([]string{path})
	if err != nil || len(list) != 1 {
		t.Fatalf("contexts %v %v", list, err)
	}
	picked := list[0].Remembered()
	same, err := write.New(write.Choice{Context: picked, Level: write.Seamless, Classes: write.AllClasses})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DialGated([]string{path}, picked.Name, same); err != nil {
		t.Fatalf("the unchanged kubeconfig does not dial: %v", err)
	}
	tokenKubeconfig(t, path, "prod-admin-token")
	again, err := ListContexts([]string{path})
	if err != nil || len(again) != 1 || again[0].Identity == picked.Identity {
		t.Fatalf("the rewritten kubeconfig keeps the identity: %v %v", again, err)
	}
	gate, err := write.New(write.Choice{Context: picked, Level: write.Seamless, Classes: write.AllClasses})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := DialGated([]string{path}, picked.Name, gate)
	if err == nil {
		t.Errorf("DANGEROUSLY SEAMLESS, chosen for identity %s, connects with credentials whose identity is now %s; the connection's identity is %q", picked.Identity[:12], again[0].Identity[:12], conn.Context.Identity)
	}
}
