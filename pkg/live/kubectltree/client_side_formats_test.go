package kubectltree_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
)

func TestAClientSideSecretShowsNoValueInAnyOutputFormatKubectlAccepts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	f := newFake(t)
	tree := stoneTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	dir := t.TempDir()
	two := filepath.Join(dir, "two.yaml")
	if err := os.WriteFile(two, []byte("apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: plain\n  namespace: shop\ndata:\n  colour: blue\n---\napiVersion: v1\nkind: Secret\nmetadata:\n  name: s5\n  namespace: shop\ndata:\n  token: aHVudGVyMg==\n---\napiVersion: v1\nkind: Secret\nmetadata:\n  name: s6\n  namespace: shop\nstringData:\n  password: hunter2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	list := filepath.Join(dir, "list.json")
	if err := os.WriteFile(list, []byte(`{"apiVersion":"v1","kind":"List","items":[{"apiVersion":"v1","kind":"Secret","metadata":{"name":"s7","namespace":"shop"},"data":{"token":"aHVudGVyMg=="}}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	sources := []string{
		"create secret generic s4 -n shop --from-literal=password=hunter2 --dry-run=client",
		"create -f " + two + " --dry-run=client --validate=false",
		"create -f " + list + " --dry-run=client --validate=false",
		"label --local -f " + two + " a=b",
		"annotate --local -f " + list + " a=b",
	}
	formats := []string{"", "json", "JSON", "Json", "yaml", "YAML", "Yaml", "kyaml", "KYAML", "KYaml", "name", "NAME",
		"jsonpath={.data}", "jsonpath-as-json={.data}", "go-template={{.data}}", "template={{.data}}", "wide", "custom-columns=D:.data"}
	for _, source := range sources {
		for _, format := range formats {
			line := source
			if format != "" {
				line += " -o " + format
			}
			res := run(t, tree, line)
			if strings.Contains(res.Output, "aHVudGVyMg==") || strings.Contains(res.Output, "hunter2") {
				t.Errorf("%s printed a Secret's value:\n%s", line, res.Output)
			}
		}
	}
	res := run(t, tree, "create -f "+two+" --dry-run=client --validate=false -o yaml")
	if !strings.Contains(res.Output, "colour: blue") || strings.Count(res.Output, "name: s") < 2 {
		t.Errorf("masking dropped the objects around the Secrets:\n%s\n%s", res.Output, res.Stderr)
	}
}
