package app

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

type ledgerRig struct {
	tree    *kubectltree.Tree
	session *Session
	changes *atomic.Int32
	dir     string
}

func newPrefixedLedgerRig(t *testing.T, prefix string) ledgerRig {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	changes := &atomic.Int32{}
	const secret = `{"apiVersion":"v1","kind":"Secret","metadata":{"name":"db","namespace":"shop"},"data":{"password":"aHVudGVyMg=="}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := strings.CutPrefix(r.URL.Path, prefix)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case !ok:
			w.WriteHeader(http.StatusNotFound)
		case p == "/api":
			fmt.Fprint(w, `{"kind":"APIVersions","versions":["v1"],"serverAddressByClientCIDRs":[{"clientCIDR":"0.0.0.0/0","serverAddress":"127.0.0.1"}]}`)
		case p == "/apis":
			fmt.Fprint(w, `{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`)
		case p == "/api/v1":
			fmt.Fprint(w, `{"kind":"APIResourceList","groupVersion":"v1","resources":[{"name":"pods","singularName":"pod","namespaced":true,"kind":"Pod","verbs":["create","delete","get","list","patch","update","watch"]},{"name":"secrets","singularName":"secret","namespaced":true,"kind":"Secret","verbs":["create","delete","get","list","patch","update","watch"]}]}`)
		case r.Method == http.MethodPatch && p == "/api/v1/namespaces/shop/secrets/db":
			changes.Add(1)
			fmt.Fprint(w, secret)
		case r.Method == http.MethodPost && p == "/api/v1/namespaces/shop/secrets":
			changes.Add(1)
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, secret)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"NotFound","code":404}`)
		}
	}))
	t.Cleanup(srv.Close)
	server := srv.URL + prefix
	dir := t.TempDir()
	path := filepath.Join(dir, "kubeconfig")
	cfg := fmt.Sprintf("apiVersion: v1\nkind: Config\ncurrent-context: work\nclusters:\n- name: work\n  cluster:\n    server: %s\ncontexts:\n- name: work\n  context:\n    cluster: work\n    user: u\n    namespace: shop\nusers:\n- name: u\n  user:\n    token: not-a-real-token\n", server)
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	manifests := filepath.Join(dir, "manifests")
	if err := os.Mkdir(manifests, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(manifests, "db.yaml"), []byte("apiVersion: v1\nkind: Secret\nmetadata:\n  name: db\n  namespace: shop\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	g, err := write.New(write.Choice{Context: write.Context{Name: "work", Server: server, User: "u"}, Level: write.Seamless})
	if err != nil {
		t.Fatal(err)
	}
	rc, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Client(rc); err != nil {
		t.Fatal(err)
	}
	for _, c := range write.AllClasses {
		g.Pass(c)
	}
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{path}, Context: "work", Wrap: live.PinnedTo(rc, g.WrapConfig), Local: t.TempDir(), Server: server})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, g, server, h, func(live.StoneAsk) bool { return true })
	})
	return ledgerRig{tree: tree, session: &Session{Writer: newLiveWriter(g, t.TempDir(), time.Now, nil)}, changes: changes, dir: manifests}
}

func (r ledgerRig) stone(t *testing.T, cmd string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	res := r.tree.Run(ctx, cmd)
	t.Logf("%s: code %d, refused %q, writes %+v\n%s", cmd, res.Code, res.Refused, res.Writes, res.Output)
	for _, w := range res.Writes {
		r.session.recordStoneWrite("kubectl "+cmd, w)
	}
}

func (r ledgerRig) ledger(t *testing.T) string {
	t.Helper()
	if r.changes.Load() == 0 {
		t.Fatalf("no change reached the server")
	}
	data, err := os.ReadFile(r.session.Writer.ledger.Path())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("ledger:\n%s", data)
	return string(data)
}

func TestTheLedgerKeepsNoPatchedSecretValueOnAServerWithAPathPrefix(t *testing.T) {
	r := newPrefixedLedgerRig(t, "/k8s/clusters/c-m-1")
	db := filepath.Join(r.dir, "db.yaml")
	for _, patch := range []string{
		`'{stringData: {password: hunter2}}'`,
		`'{"string` + string(rune(92)) + `u0044ata":{"password":"hunter2"}}'`,
	} {
		r.stone(t, "patch -f "+db+" --type merge -p "+patch)
	}
	if data := r.ledger(t); strings.Contains(data, "hunter2") {
		t.Errorf("a Secret value typed in a patch reached the ledger on a server whose address has a path:\n%s", data)
	}
}
