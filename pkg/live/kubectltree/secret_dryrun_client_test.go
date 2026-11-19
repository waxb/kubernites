package kubectltree_test

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
)

func TestAClientDryRunNeverPrintsASecretValue(t *testing.T) {
	f := newFake(t)
	tree := newTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	key := filepath.Join(t.TempDir(), "id_key")
	value := "-----BEGIN PRIVATE KEY-----\nTOPSECRETKEYMATERIAL\n-----END PRIVATE KEY-----\n"
	if err := os.WriteFile(key, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
	res := run(t, tree, "create secret generic s4 -n shop --from-file=key="+key+" --dry-run=client -o yaml")
	t.Logf("exit %d, server saw %v\n%s", res.Code, f.seen.all(), res.Output)
	if strings.Contains(res.Output, base64.StdEncoding.EncodeToString([]byte(value))) {
		t.Fatalf("at READ ONLY the Stone printed the Secret's value unmasked")
	}
}

func TestNoClientSideRunPrintsASecretValueInAnyForm(t *testing.T) {
	f := newFake(t)
	tree := newTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	dir := t.TempDir()
	key := filepath.Join(dir, "id_key")
	value := "TOPSECRETKEYMATERIAL"
	os.WriteFile(key, []byte(value), 0o600)
	manifest := filepath.Join(dir, "secret.yaml")
	os.WriteFile(manifest, []byte("apiVersion: v1\nkind: Secret\nmetadata:\n  name: s5\n  namespace: shop\nstringData:\n  password: hunter2\ndata:\n  token: aHVudGVyMg==\n"), 0o600)
	values := []string{value, base64.StdEncoding.EncodeToString([]byte(value)), "hunter2", "aHVudGVyMg=="}
	for _, line := range []string{
		"create secret generic s4 -n shop --from-file=key=" + key + " --dry-run=client -o json",
		"create secret generic s4 -n shop --from-literal=password=hunter2 --dry-run=client -o yaml",
		"create secret generic s4 -n shop --from-literal=password=hunter2 --dry-run client -o yaml",
		"create secret generic s4 -n shop --from-literal=password=hunter2 --dry-run=client -o jsonpath={.data.password}",
		"create secret generic s4 -n shop --from-literal=password=hunter2 --dry-run=client -o go-template={{.data.password}}",
		"label --local -f " + manifest + " a=b -o yaml",
		"annotate --local -f " + manifest + " a=b -o json",
		"label --local -f " + manifest + " a=b -o jsonpath={.stringData.password}",
		"create -f " + manifest + " --dry-run=client -o yaml",
	} {
		res := run(t, tree, line)
		for _, v := range values {
			if strings.Contains(res.Output, v) {
				t.Errorf("%s printed %s:\n%s", line, v, res.Output)
			}
		}
	}
	res := run(t, tree, "create configmap c -n shop --from-literal=a=plain --dry-run=client -o yaml")
	if res.Code != 0 || !strings.Contains(res.Output, "a: plain") {
		t.Fatalf("a client dry run of a ConfigMap no longer prints it:\n%s", res.Output)
	}
}
