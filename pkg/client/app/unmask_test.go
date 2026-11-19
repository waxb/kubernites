package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

type bareTree struct {
	tree *kubectltree.Tree
}

func (b bareTree) Tree() *kubectltree.Tree { return b.tree }
func (b bareTree) Gate() *write.Gate       { return nil }
func (b bareTree) Snapshot() live.Snapshot { return live.Snapshot{} }

func TestOnlyACoreSecretsHiddenValuesAreReadBackOnSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api":
			fmt.Fprint(w, `{"kind":"APIVersions","versions":["v1"],"serverAddressByClientCIDRs":[]}`)
		case "/apis":
			fmt.Fprint(w, `{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`)
		case "/api/v1":
			fmt.Fprint(w, `{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"v1","resources":[{"name":"secrets","singularName":"secret","namespaced":true,"kind":"Secret","verbs":["get","list"]}]}`)
		case "/api/v1/namespaces/shop/secrets/db":
			fmt.Fprint(w, `{"apiVersion":"v1","kind":"Secret","metadata":{"name":"db","namespace":"shop"},"data":{"password":"Q09SRS1TRUNSRVQ="}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"NotFound","code":404}`)
		}
	}))
	t.Cleanup(srv.Close)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	os.WriteFile(path, []byte("apiVersion: v1\nkind: Config\ncurrent-context: c\nclusters:\n- name: c\n  cluster:\n    server: "+srv.URL+"\ncontexts:\n- name: c\n  context:\n    cluster: c\n    user: u\nusers:\n- name: u\n  user: {}\n"), 0o600)
	g, err := write.New(write.Choice{Context: write.Context{Name: "c", Server: srv.URL, User: "u"}, Level: write.Seamless, Classes: []write.Class{write.Secrets}})
	if err != nil {
		t.Fatal(err)
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{path}, Context: "c", Wrap: g.WrapConfig, Local: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	custom := "apiVersion: evil.example/v1\nkind: Secret\nmetadata:\n  name: db\n  namespace: shop\ndata:\n  password: " + hud.MaskedValue + "\n"
	e := hud.LiveEdit{Ref: "secret.evil.example/db", Kind: "Secret", Namespace: "shop", Name: "db", Format: "yaml", Text: custom}
	if keys := maskedKeys(e.Kind, custom); len(keys) != 0 {
		t.Errorf("a custom resource of kind Secret shows hidden keys %v", keys)
	}
	out, err := unmask(bareTree{tree}, e, custom)
	if strings.Contains(out, "Q09SRS1TRUNSRVQ=") {
		t.Fatalf("SEND would write the core Secret's value into a custom resource:\n%s", out)
	}
	core := strings.Replace(custom, "evil.example/v1", "v1", 1)
	if keys := maskedKeys("Secret", core); len(keys) != 1 {
		t.Fatalf("a core Secret's hidden keys: %v", keys)
	}
	if back, err := unmask(bareTree{tree}, e, core); err != nil || !strings.Contains(back, "Q09SRS1TRUNSRVQ=") {
		t.Fatalf("a core Secret's hidden value was not read back: %v\n%s", err, back)
	}
}
