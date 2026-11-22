package kubectltree_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestApplyOfAnExistingSecretNeverEchoesItsValues(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "secret.yaml")
	plain := "PLAIN-VALUE-OF-THE-MANIFEST"
	encoded := base64.StdEncoding.EncodeToString([]byte("DATA-VALUE-OF-THE-MANIFEST"))
	body := "apiVersion: v1\nkind: Secret\nmetadata:\n  name: s\n  namespace: shop\nstringData:\n  password: " + plain + "\ndata:\n  token: " + encoded + "\n"
	if err := os.WriteFile(manifest, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{
		"apply -f " + manifest + " --validate=false --dry-run=client",
		"apply -f " + manifest + " --validate=false --dry-run=server",
		"apply -f " + manifest + " --validate=false",
	} {
		tree, _, _ := gatedTree(t, write.Writes, true)
		res := run(t, tree, line)
		t.Logf("%s: code=%d refused=%q\n%s", line, res.Code, res.Refused, res.Output)
		for _, v := range []string{plain, encoded} {
			if strings.Contains(res.Output, v) {
				t.Errorf("%s printed the Secret value %s", line, v)
			}
		}
	}
}

func TestAnApplyDeclinedAtConfirmNeverEchoesTheSecretsValues(t *testing.T) {
	const bare = `{"apiVersion":"v1","kind":"Secret","metadata":{"name":"plain","namespace":"shop","resourceVersion":"9"},"type":"Opaque","data":{"password":"b2xk"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/secrets/plain") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, bare)
			return
		}
		newFakeHandler(w, r)
	}))
	t.Cleanup(srv.Close)
	f := &fakeServer{Server: srv, seen: &seen{}}
	path := kubeconfig(t, srv.URL, "stone", "shop")
	g := gateAt(t, write.Writes, f, path)
	tree := newTree(t, path, g.WrapConfig)
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, g, srv.URL, h, func(a live.StoneAsk) bool { return a.Why == write.NeedsClass })
	})
	manifest := filepath.Join(t.TempDir(), "secret.yaml")
	plain := "VALUE-OF-A-DECLINED-APPLY"
	if err := os.WriteFile(manifest, []byte("apiVersion: v1\nkind: Secret\nmetadata:\n  name: plain\n  namespace: shop\nstringData:\n  password: "+plain+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res := run(t, tree, "apply -f "+manifest+" --validate=false")
	t.Logf("code=%d refused=%q\n%s", res.Code, res.Refused, res.Output)
	if strings.Contains(res.Output, plain) {
		t.Fatalf("an apply whose CONFIRM was declined printed the Secret value %s", plain)
	}
}

func TestClientSideApplyOfAnExistingSecretSendsExactlyTheManifestsValues(t *testing.T) {
	enc := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	original := `{"apiVersion":"v1","data":{"gone":"` + enc("GONE-VALUE") + `","password":"` + enc("OLD-VALUE") + `"},"kind":"Secret","metadata":{"name":"s","namespace":"shop"},"type":"Opaque"}`
	live := map[string]any{
		"apiVersion": "v1", "kind": "Secret", "type": "Opaque",
		"metadata": map[string]any{"name": "s", "namespace": "shop", "resourceVersion": "9", "annotations": map[string]any{"kubectl.kubernetes.io/last-applied-configuration": original}},
		"data":     map[string]any{"password": enc("OLD-VALUE"), "gone": enc("GONE-VALUE"), "extra": enc("EXTRA-VALUE")},
	}
	body, _ := json.Marshal(live)
	var patches []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/secrets/s") {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodPatch {
				b, _ := io.ReadAll(r.Body)
				patches = append(patches, string(b))
			}
			w.Write(body)
			return
		}
		newFakeHandler(w, r)
	}))
	t.Cleanup(srv.Close)
	f := &fakeServer{Server: srv, seen: &seen{}}
	path := kubeconfig(t, srv.URL, "stone", "shop")
	g := gateAt(t, write.Seamless, f, path)
	g.Pass(write.Secrets)
	tree := newTree(t, path, g.WrapConfig)
	manifest := filepath.Join(t.TempDir(), "secret.yaml")
	if err := os.WriteFile(manifest, []byte("apiVersion: v1\nkind: Secret\nmetadata:\n  name: s\n  namespace: shop\ntype: Opaque\ndata:\n  password: "+enc("NEW-VALUE")+"\n  other: "+enc("OTHER-VALUE")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	res := run(t, tree, "apply -f "+manifest+" --validate=false")
	if res.Code != 0 || len(patches) != 1 {
		t.Fatalf("client-side apply of an existing Secret did not send one patch (code %d, %d patches):\n%s", res.Code, len(patches), res.Output)
	}
	var patch struct {
		Data     map[string]any `json:"data"`
		Metadata struct {
			Annotations map[string]string `json:"annotations"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(patches[0]), &patch); err != nil {
		t.Fatalf("the patch is not JSON: %v\n%s", err, patches[0])
	}
	want := map[string]any{"password": enc("NEW-VALUE"), "other": enc("OTHER-VALUE"), "gone": nil}
	if fmt.Sprint(patch.Data) != fmt.Sprint(want) {
		t.Errorf("the three-way patch sent data %v, want exactly the manifest's values and the dropped key: %v", patch.Data, want)
	}
	for _, v := range []string{"KioqKg==", "****", enc("OLD-VALUE"), enc("EXTRA-VALUE")} {
		if strings.Contains(patches[0], v) {
			t.Errorf("the patch carries %q: %s", v, patches[0])
		}
	}
	if applied := patch.Metadata.Annotations["kubectl.kubernetes.io/last-applied-configuration"]; !strings.Contains(applied, enc("NEW-VALUE")) {
		t.Errorf("the patch does not record the manifest as last applied: %q", applied)
	}
	if strings.Contains(res.Output, "NEW-VALUE") || strings.Contains(res.Output, enc("NEW-VALUE")) {
		t.Errorf("the apply printed a value:\n%s", res.Output)
	}
}
