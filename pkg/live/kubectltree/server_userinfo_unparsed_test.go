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

func TestAServerAddressGoCannotParseShowsNoUserinfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/namespaces/kube-system/services") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"apiVersion":"v1","kind":"ServiceList","metadata":{"resourceVersion":"1"},"items":[]}`)
			return
		}
		newFakeHandler(w, r)
	}))
	t.Cleanup(srv.Close)
	host := strings.TrimPrefix(srv.URL, "http://")
	for _, pw := range []string{`Q"PW`, `S PW`, `B\PW`, `L<PW`, `C^PW`, `P|PW`, `K{PW`, "T`PW", `UéPW`} {
		broken := "http://alice:" + pw + "@" + host
		for _, server := range []string{broken, "  " + broken, "alice:" + pw + "@" + host, "//alice:" + pw + "@" + host} {
			path := filepath.Join(t.TempDir(), "kubeconfig")
			cfg := "apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: " + srv.URL + "\n- name: broken\n  cluster:\n    server: '" + server + "'\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\n    namespace: shop\n- name: other\n  context:\n    cluster: broken\n    user: fake-user\nusers:\n- name: fake-user\n  user:\n    token: not-a-real-token\n"
			if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
				t.Fatal(err)
			}
			contexts, err := live.ListContexts([]string{path})
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range contexts {
				if strings.Contains(c.Server, "PW") || strings.Contains(c.Server, "alice") {
					t.Errorf("server %q: the picker lists context %s at %s", server, c.Name, c.Server)
				}
			}
			if _, err := live.Dial([]string{path}, "other"); err == nil || strings.Contains(err.Error(), "PW") || strings.Contains(err.Error(), "alice") {
				t.Errorf("server %q: connecting says %v", server, err)
			}
			tree := stoneTree(t, path, live.ReadOnlyConfig)
			for _, line := range []string{"config view", "config view -o json", "config view -o yaml", "config view --minify --context other -o json", "config view -o jsonpath={.clusters[*].cluster.server}"} {
				if res := run(t, tree, line); strings.Contains(res.Output, "PW") || strings.Contains(res.Output, "alice") {
					t.Errorf("server %q: %s showed:\n%s", server, line, res.Output)
				}
			}
			if got := write.RedactServer(server); strings.Contains(got, "PW") || strings.Contains(got, "alice") || !strings.HasSuffix(got, host) {
				t.Errorf("RedactServer(%q) = %q", server, got)
			}
		}
	}
}
