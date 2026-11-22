package kubectltree_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
)

func TestABigClientSideSecretShowsNoValueHoweverLargeItsOutput(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newFake(t)
	tree := stoneTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	dir := t.TempDir()
	big := filepath.Join(dir, "big.yaml")
	manifest := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: bundle\n  namespace: shop\nstringData:\n  bundle: " + strings.Repeat("x", 5<<20) + "\n  password: hunter2\n"
	if err := os.WriteFile(big, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, format := range []string{"json", "yaml"} {
		res := run(t, tree, "label --local -f "+big+" tier=db -o "+format)
		t.Logf("-o %s: code %d, %d bytes shown, refused %q", format, res.Code, len(res.Output), res.Refused)
		if strings.Contains(res.Output, "hunter2") {
			i := strings.Index(res.Output, "hunter2")
			t.Errorf("a client-side Secret whose output passes the Stone's bound printed a value with -o %s: ...%q", format, res.Output[max(0, i-40):min(len(res.Output), i+20)])
		}
	}
}
