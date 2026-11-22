package kubectltree_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
)

func TestAClientSideRunPrintsNoSecretValueInKyaml(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newFake(t)
	tree := stoneTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	manifest := filepath.Join(t.TempDir(), "secret.yaml")
	if err := os.WriteFile(manifest, []byte("apiVersion: v1\nkind: Secret\nmetadata:\n  name: s5\n  namespace: shop\ndata:\n  token: aHVudGVyMg==\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var leaked []string
	for _, line := range []string{
		"create secret generic s4 -n shop --from-literal=password=hunter2 --dry-run=client -o kyaml",
		"create secret generic s4 -n shop --from-literal=password=hunter2 --dry-run=client -o KYAML",
		"label --local -f " + manifest + " a=b -o kyaml",
	} {
		res := run(t, tree, line)
		t.Logf("%s: code=%d refused=%q\n%s", line, res.Code, res.Refused, res.Output)
		if strings.Contains(res.Output, "aHVudGVyMg==") || strings.Contains(res.Output, "hunter2") {
			leaked = append(leaked, line)
		}
	}
	if len(leaked) > 0 {
		t.Fatalf("a client-side run printed a Secret's value unmasked:\n%s", strings.Join(leaked, "\n"))
	}
}
