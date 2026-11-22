package kubectltree_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
)

func protectedTree(t *testing.T, local string) *kubectltree.Tree {
	t.Helper()
	f := newFake(t)
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{kubeconfig(t, f.URL, "stone", "shop")}, Context: "stone", Wrap: live.ReadOnlyConfig, Local: local})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	return tree
}

func TestALargeDirectoryInDotKubeNeverHidesADotKubeFile(t *testing.T) {
	defer kubectltree.SetProtectedWalk(100)()
	local := t.TempDir()
	kube := filepath.Join(local, ".kube")
	legacy := filepath.Join(kube, "http-cache")
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 300; i++ {
		if err := os.WriteFile(filepath.Join(legacy, fmt.Sprintf("%04d", i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	prod := filepath.Join(kube, "prod.yaml")
	if err := os.WriteFile(prod, []byte("PROD-KUBECONFIG"), 0o600); err != nil {
		t.Fatal(err)
	}
	hard := filepath.Join(t.TempDir(), "hard.txt")
	if err := os.Link(prod, hard); err != nil {
		t.Fatal(err)
	}
	if !protectedTree(t, local).Protected(hard) {
		t.Error("a hard link to ~/.kube/prod.yaml is not protected once ~/.kube/http-cache holds more entries than the walk bound")
	}
}

func TestASymlinkedDotKubeIsWalkedForHardLinks(t *testing.T) {
	local := t.TempDir()
	dotfiles := filepath.Join(t.TempDir(), "kube")
	if err := os.MkdirAll(dotfiles, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dotfiles, filepath.Join(local, ".kube")); err != nil {
		t.Fatal(err)
	}
	prod := filepath.Join(dotfiles, "prod.yaml")
	if err := os.WriteFile(prod, []byte("PROD-KUBECONFIG"), 0o600); err != nil {
		t.Fatal(err)
	}
	hard := filepath.Join(t.TempDir(), "hard.txt")
	if err := os.Link(prod, hard); err != nil {
		t.Fatal(err)
	}
	if !protectedTree(t, local).Protected(hard) {
		t.Error("a hard link to a file in ~/.kube is not protected when ~/.kube is a symbolic link to a directory")
	}
}
