package kubectltree_test

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestANewSecretsValuesNeverReachTheConfirmOrTheLedger(t *testing.T) {
	f := newFake(t)
	path := kubeconfig(t, f.URL, "stone", "shop")
	material := "-----BEGIN PRIVATE KEY-----\nTOPSECRETKEYMATERIAL\n-----END PRIVATE KEY-----\n"
	key := filepath.Join(t.TempDir(), "id_key")
	if err := os.WriteFile(key, []byte(material), 0o600); err != nil {
		t.Fatal(err)
	}
	values := []string{"hunter2", "aHVudGVyMg==", "TOPSECRETKEYMATERIAL", base64.StdEncoding.EncodeToString([]byte(material))}
	leaks := func(s string) []string {
		var out []string
		for _, v := range values {
			if strings.Contains(s, v) {
				out = append(out, v)
			}
		}
		return out
	}
	tree, asks := confirmingTree(t, write.Writes, f, path)
	for _, line := range []string{
		"create secret generic s2 -n shop --from-literal=password=hunter2",
		"create secret generic s3 -n shop --from-file=key=" + key,
		"patch secret s -n shop -p data:\\ {password:\\ aHVudGVyMg==}",
	} {
		*asks = nil
		res := run(t, tree, line)
		if len(*asks) == 0 {
			t.Errorf("%s asked no CONFIRM:\n%s", line, res.Output)
		}
		for _, a := range *asks {
			shown := write.Mask(strings.TrimSpace(string(a.Order.Request.Body)))
			if got := leaks(shown); len(got) > 0 {
				t.Errorf("%s: the CONFIRM's body line holds %v (%s)", line, got, a.Order.ContentType)
			}
		}
	}

	dir := t.TempDir()
	ctx := write.Context{Name: "stone", Server: f.URL, User: "fake-user"}
	ledger := write.OpenLedger(dir, ctx)
	object := write.Ref{GVR: write.GVR{Version: "v1", Resource: "secrets"}, Kind: "Secret", Namespace: "shop", Name: "s"}.String()
	for _, typed := range []string{
		"kubectl patch secret s -n shop -p 'data: {password: aHVudGVyMg==}'",
		"kubectl patch secret s -n shop -p '{data: {password: aHVudGVyMg==}}'",
		"kubectl patch secret s -n shop -p '{\"data\":{\"password\":\"aHVudGVyMg==\"}}'",
		"kubectl patch secret s -n shop --patch='stringData: {password: hunter2}'",
		"kubectl patch secret/s -n shop --type=json -p '[{\"op\":\"add\",\"path\":\"/stringData\",\"value\":{\"password\":\"hunter2\"}}]'",
		"kubectl create secret generic s2 -n shop --from-literal password=hunter2",
		"kubectl create secret generic s3 -n shop --from-env-file=" + key,
	} {
		if err := ledger.Append(write.Entry{At: time.Unix(0, 0), Context: ctx.Name, Level: "WRITES", Order: typed, Object: object, Code: 200, Answer: "OK"}); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(ledger.Path())
	if err != nil {
		t.Fatal(err)
	}
	if got := leaks(string(data)); len(got) > 0 {
		t.Fatalf("the ledger holds Secret values %v:\n%s", got, data)
	}
}

func TestEveryAnswerUnderSecretsReachesTheStoneMasked(t *testing.T) {
	tree, f := secretsTree(t)
	manifest := filepath.Join(t.TempDir(), "secret.yaml")
	if err := os.WriteFile(manifest, []byte("apiVersion: v1\nkind: Secret\nmetadata:\n  name: s\n  namespace: shop\ntype: Opaque\nstringData:\n  note: plain\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		`patch secret s -n shop --type=merge -p {"metadata":{"labels":{"a":"b"}}} -o json`,
		`patch secret s -n shop --type=merge -p {"metadata":{"labels":{"a":"b"}}} --dry-run=server -o yaml`,
		"label secret s -n shop a=b --overwrite -o yaml",
		"annotate secret s -n shop a=b --overwrite -o json",
		"create -f " + manifest + " -o json",
		"replace -f " + manifest + " -o yaml",
		"apply --server-side -f " + manifest + " -o yaml",
	} {
		res := run(t, tree, line)
		for _, value := range []string{"aHVudGVyMg==", "hunter2", "Y2VydA=="} {
			if strings.Contains(res.Output, value) {
				t.Errorf("%s shows %s:\n%s", line, value, res.Output)
			}
		}
	}
	for _, a := range f.accepts {
		if strings.Contains(a, "protobuf") || strings.Contains(a, "cbor") {
			t.Errorf("a Secret was asked for as %s", a)
		}
	}
}

func TestAScriptoriumSendWritesNoSecretToDisk(t *testing.T) {
	f := &fakeServer{seen: &seen{}}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openapi/v3/api/v1" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, openAPI("/api/v1/namespaces/{namespace}/secrets/{name}", "", "Secret"))
			return
		}
		f.serve(w, r)
	}))
	t.Cleanup(f.Close)
	path := kubeconfig(t, f.URL, "stone", "shop")
	g := gateAt(t, write.Writes, f, path)
	g.Pass(write.Secrets)
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	tree := newTree(t, path, g.WrapConfig)
	found := make(chan []string, 1)
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		var hits []string
		filepath.WalkDir(tmp, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				if data, _ := os.ReadFile(p); strings.Contains(string(data), "S0VQVC1PTi1ESVNL") {
					hits = append(hits, p)
				}
			}
			return nil
		})
		select {
		case found <- hits:
		default:
		}
		return false
	})
	res := tree.Apply(context.Background(), "apiVersion: v1\nkind: Secret\nmetadata:\n  name: s\n  namespace: shop\ndata:\n  password: S0VQVC1PTi1ESVNL\n", false)
	select {
	case hits := <-found:
		if len(hits) != 0 {
			t.Fatalf("while its CONFIRM waited, the Secret's value sat on disk in %v", hits)
		}
	default:
		t.Fatalf("no CONFIRM was asked (%d, %q, %+v):\n%s", res.Code, res.Refused, res.Command, res.Output)
	}
}
