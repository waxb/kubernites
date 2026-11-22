package kubectltree_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
)

func TestAGlobBehindADanglingLinkNeverReadsDotKube(t *testing.T) {
	f := newFake(t)
	local := t.TempDir()
	if err := os.MkdirAll(filepath.Join(local, ".kube"), 0o700); err != nil {
		t.Fatal(err)
	}
	token := "OTHER-CLUSTERS-TOKEN"
	other := filepath.Join(local, ".kube", "prod.yaml")
	if err := os.WriteFile(other, []byte("apiVersion: v1\nkind: Config\ncurrent-context: prod\nusers:\n- name: prod\n  user:\n    token: "+token+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Symlink(other, filepath.Join(dir, "a.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "gone"), filepath.Join(dir, "[ab].yaml")); err != nil {
		t.Fatal(err)
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{kubeconfig(t, f.URL, "stone", "shop")}, Context: "stone", Wrap: live.ReadOnlyConfig, Local: local})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	res := run(t, tree, "label --local -f "+filepath.Join(dir, "[ab].yaml")+" x=y -o yaml")
	t.Logf("code=%d refused=%q\n%s", res.Code, res.Refused, res.Output)
	if strings.Contains(res.Output, token) {
		t.Fatalf("the Stone read a file inside ~/.kube through a glob the checker never expanded")
	}
}
