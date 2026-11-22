package kubectltree_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/waxb/kubernites/pkg/live/write"
)

func TestADeleteOfSeveralObjectsFromFilesAsksTheDestroyingWarning(t *testing.T) {
	manifest := filepath.Join(t.TempDir(), "pods.yaml")
	if err := os.WriteFile(manifest, []byte("apiVersion: v1\nkind: Pod\nmetadata:\n  name: web-1\n  namespace: shop\n---\napiVersion: v1\nkind: Pod\nmetadata:\n  name: shell-1\n  namespace: shop\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	f := newFake(t)
	tree, asks := confirmingTree(t, write.Seamless, f, kubeconfig(t, f.URL, "stone", "shop"))
	res := run(t, tree, "delete -f "+manifest+" --wait=false")
	asked := false
	for _, x := range *asks {
		if x.Class == write.Destroy {
			asked = true
		}
	}
	t.Logf("writes %v\n%s", f.seen.writes(), res.Output)
	if !asked {
		t.Fatalf("a delete of two objects named in a file sent %v with no destroying warning", f.seen.writes())
	}
}
