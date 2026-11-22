package kubectltree_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestAServerPasswordWithAQuoteNeverShows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/namespaces/kube-system/services") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"apiVersion":"v1","kind":"ServiceList","metadata":{"resourceVersion":"1"},"items":[]}`)
			return
		}
		newFakeHandler(w, r)
	}))
	t.Cleanup(srv.Close)
	withUser := strings.Replace(srv.URL, "http://", "http://alice:SERVER'PW@", 1)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := "apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: " + withUser + "\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\n    namespace: shop\nusers:\n- name: fake-user\n  user:\n    token: not-a-real-token\n"
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := stoneTree(t, path, live.ReadOnlyConfig)
	for _, line := range []string{"cluster-info", "config view"} {
		res := run(t, tree, line)
		t.Logf("%s:\n%s", line, res.Output)
		if strings.Contains(res.Output, "PW") {
			t.Errorf("%s showed the server's password:\n%s", line, res.Output)
		}
	}
	contexts, err := live.ListContexts([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range contexts {
		if strings.Contains(c.Server, "PW") {
			t.Errorf("the picker lists context %s at %s", c.Name, c.Server)
		}
	}
	if got := write.Redact("at https://alice:SERVER'PW@prod.example:6443/api"); strings.Contains(got, "PW") {
		t.Errorf("Redact kept the password: %s", got)
	}
}
