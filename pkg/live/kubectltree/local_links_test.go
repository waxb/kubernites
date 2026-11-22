package kubectltree_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

func TestALinkIntoDotKubeIsNeverRead(t *testing.T) {
	f := newFake(t)
	local := t.TempDir()
	if err := os.MkdirAll(filepath.Join(local, ".kube"), 0o700); err != nil {
		t.Fatal(err)
	}
	token := "OTHER-CLUSTERS-TOKEN"
	other := filepath.Join(local, ".kube", "prod.yaml")
	if err := os.WriteFile(other, []byte("apiVersion: v1\nkind: Config\nusers:\n- name: prod\n  user:\n    token: "+token+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.Symlink(other, link); err != nil {
		t.Fatal(err)
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{kubeconfig(t, f.URL, "stone", "shop")}, Context: "stone", Wrap: live.ReadOnlyConfig, Local: local})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	direct := run(t, tree, "create configmap c -n shop --from-file=k="+other+" --dry-run=client -o yaml")
	t.Logf("direct: code=%d refused=%q", direct.Code, direct.Refused)
	linked := run(t, tree, "create configmap c -n shop --from-file=k="+link+" --dry-run=client -o yaml")
	t.Logf("through a link: code=%d refused=%q\n%s", linked.Code, linked.Refused, linked.Output)
	if strings.Contains(direct.Output, token) || strings.Contains(linked.Output, token) {
		t.Fatalf("the Stone read a file inside ~/.kube")
	}
}

func TestTheConnectedUsersCredentialFilesAreNeverRead(t *testing.T) {
	f := newFake(t)
	creds := t.TempDir()
	token := "TOKEN-FILE-CREDENTIAL"
	tokenFile := filepath.Join(creds, "token")
	if err := os.WriteFile(tokenFile, []byte(token), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := "apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: " + f.URL + "\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\n    namespace: shop\nusers:\n- name: fake-user\n  user:\n    tokenFile: " + tokenFile + "\n"
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := stoneTree(t, path, live.ReadOnlyConfig)
	res := run(t, tree, "create configmap c -n shop --from-file=k="+tokenFile+" --dry-run=client -o yaml")
	t.Logf("code=%d refused=%q\n%s", res.Code, res.Refused, res.Output)
	if strings.Contains(res.Output, token) {
		t.Fatalf("the Stone printed the connected user's credential file")
	}
}

func TestNoLinkAndNoCredentialFileOfAnyUserEntryIsReadOrSent(t *testing.T) {
	f := newFake(t)
	f.shell = &livetest.Shell{}
	local := t.TempDir()
	kube := filepath.Join(local, ".kube")
	if err := os.MkdirAll(kube, 0o700); err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(kube, "prod.yaml")
	if err := os.WriteFile(inside, []byte("KUBE-DIR-SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	creds := t.TempDir()
	otherToken := filepath.Join(creds, "other-token")
	otherKey := filepath.Join(creds, "other.key")
	for p, body := range map[string]string{otherToken: "OTHER-USERS-TOKEN", otherKey: "OTHER-USERS-KEY"} {
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := "apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: " + f.URL + "\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\n    namespace: shop\n- name: other\n  context:\n    cluster: fake\n    user: other-user\nusers:\n- name: fake-user\n  user:\n    token: not-a-real-token\n- name: other-user\n  user:\n    tokenFile: " + otherToken + "\n    client-key: " + otherKey + "\n    client-certificate-data: Y2VydA==\n"
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	links := t.TempDir()
	hard := filepath.Join(links, "hard.txt")
	if err := os.Link(inside, hard); err != nil {
		t.Fatal(err)
	}
	dirLink := filepath.Join(links, "kube-dir")
	if err := os.Symlink(kube, dirLink); err != nil {
		t.Fatal(err)
	}
	chain := filepath.Join(links, "chain.txt")
	if err := os.Symlink(filepath.Join(dirLink, "prod.yaml"), chain); err != nil {
		t.Fatal(err)
	}
	tokenLink := filepath.Join(links, "token-link")
	if err := os.Symlink(otherToken, tokenLink); err != nil {
		t.Fatal(err)
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{path}, Context: "stone", Wrap: live.ReadOnlyConfig, Local: local})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	secrets := []string{"KUBE-DIR-SECRET", "OTHER-USERS-TOKEN", "OTHER-USERS-KEY"}
	for _, p := range []string{hard, dirLink, chain, otherToken, otherKey, tokenLink} {
		res := run(t, tree, "create configmap c -n shop --from-file="+p+" --dry-run=client -o yaml")
		for _, s := range secrets {
			if strings.Contains(res.Output, s) {
				t.Errorf("--from-file=%s printed %s", p, s)
			}
		}
		if res.Refused == "" {
			t.Errorf("--from-file=%s was not refused: code %d\n%s", p, res.Code, res.Output)
		}
	}
	for _, p := range []string{hard, chain, otherToken} {
		if !tree.Protected(p) {
			t.Errorf("cp would send %s into a container", p)
		}
	}
}

func TestAHardLinkToTheKubeconfigIsCaughtHoweverManyFilesKubectlsCacheHolds(t *testing.T) {
	defer kubectltree.SetProtectedWalk(100)()
	f := newFake(t)
	local := t.TempDir()
	kube := filepath.Join(local, ".kube")
	cache := filepath.Join(kube, "cache", "http")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 300; i++ {
		if err := os.WriteFile(filepath.Join(cache, fmt.Sprintf("%04d", i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	config := filepath.Join(kube, "config")
	if err := os.WriteFile(config, []byte("KUBE-CONFIG-SECRET"), 0o600); err != nil {
		t.Fatal(err)
	}
	hard := filepath.Join(t.TempDir(), "hard.txt")
	if err := os.Link(config, hard); err != nil {
		t.Fatal(err)
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{kubeconfig(t, f.URL, "stone", "shop")}, Context: "stone", Wrap: live.ReadOnlyConfig, Local: local})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	res := run(t, tree, "create configmap c -n shop --from-file="+hard+" --dry-run=client -o yaml")
	if strings.Contains(res.Output, "KUBE-CONFIG-SECRET") || res.Refused == "" {
		t.Errorf("a hard link to ~/.kube/config behind a large cache was read: code %d\n%s", res.Code, res.Output)
	}
	if !tree.Protected(hard) {
		t.Error("cp would send a hard link to ~/.kube/config into a container")
	}
}
