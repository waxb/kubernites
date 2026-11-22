package kubectltree_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
)

func TestADotDotAfterALinkNeverReadsDotKube(t *testing.T) {
	f := newFake(t)
	home := t.TempDir()
	kube := filepath.Join(home, ".kube")
	for _, d := range []string{filepath.Join(kube, "cache"), filepath.Join(home, "Documents")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	token := "DOTDOT-CONNECTED-TOKEN"
	config := filepath.Join(kube, "config")
	cfg := fmt.Sprintf("apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: %s\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\n    namespace: shop\nusers:\n- name: fake-user\n  user:\n    token: %s\n", f.URL, token)
	if err := os.WriteFile(config, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	links := t.TempDir()
	intoCache := filepath.Join(links, "c")
	if err := os.Symlink(filepath.Join(kube, "cache"), intoCache); err != nil {
		t.Fatal(err)
	}
	intoDocuments := filepath.Join(links, "d")
	if err := os.Symlink(filepath.Join(home, "Documents"), intoDocuments); err != nil {
		t.Fatal(err)
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{config}, Context: "stone", Wrap: live.ReadOnlyConfig, Local: home})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	for _, p := range []string{intoCache + "/../config", intoDocuments + "/../.kube/config"} {
		res := run(t, tree, "create configmap c -n shop --from-file=k="+p+" --dry-run=client -o yaml")
		t.Logf("%s: code=%d refused=%q", p, res.Code, res.Refused)
		if strings.Contains(res.Output, token) {
			t.Errorf("--from-file=%s printed the connected kubeconfig's token", p)
		}
	}
}

func TestEveryLocalFileFlagChecksThePathAsTheReaderOpensIt(t *testing.T) {
	f := newFake(t)
	home := t.TempDir()
	kube := filepath.Join(home, ".kube")
	if err := os.MkdirAll(filepath.Join(kube, "cache", "deep"), 0o700); err != nil {
		t.Fatal(err)
	}
	token := "READER-PATH-TOKEN"
	config := filepath.Join(kube, "config")
	cfg := fmt.Sprintf("apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: %s\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\n    namespace: shop\nusers:\n- name: fake-user\n  user:\n    token: %s\n", f.URL, token)
	if err := os.WriteFile(config, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	links := t.TempDir()
	deep := filepath.Join(links, "deep")
	if err := os.Symlink(filepath.Join(kube, "cache", "deep"), deep); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(links, "dir")
	if err := os.Symlink(filepath.Join(kube, "cache"), dir); err != nil {
		t.Fatal(err)
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{config}, Context: "stone", Wrap: live.ReadOnlyConfig, Local: home})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	through := deep + "/../../config"
	for _, line := range []string{
		"create configmap c -n shop --from-file=k=" + through + " --dry-run=client -o yaml",
		"create configmap c -n shop --from-file=" + dir + "/.. --dry-run=client -o yaml",
		"create configmap c -n shop --from-env-file=" + through + " --dry-run=client -o yaml",
		"create secret generic s -n shop --from-file=k=" + through + " --dry-run=client -o yaml",
		"label --local -f " + through + " a=b -o yaml",
		"label --local -f " + dir + "/.. a=b -o yaml",
		"patch deployment web -n shop --patch-file=" + through + " --dry-run=client -o yaml",
		"get pods -n shop -o go-template-file=" + through,
		"get pods -n shop -o jsonpath-file=" + through,
	} {
		res := run(t, tree, line)
		if strings.Contains(res.Output, token) {
			t.Errorf("%s printed the connected kubeconfig's token", line)
		}
		if res.Refused != kubectltree.ReasonKubeReads && !strings.Contains(res.Output, kubectltree.ReasonKubeReads) {
			t.Errorf("%s was not refused as a read of ~/.kube: code=%d refused=%q\n%s", line, res.Code, res.Refused, res.Output)
		}
	}
	if !tree.Protected(through) || !tree.Protected(dir+"/..") {
		t.Error("a path whose links lead into ~/.kube before a .. is not protected")
	}
	if tree.Protected(filepath.Join(links, "elsewhere", "..", "ok.yaml")) {
		t.Error("a path that stays outside ~/.kube is protected")
	}
}
