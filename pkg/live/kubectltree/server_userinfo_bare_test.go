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
)

func TestAServerAddressWithoutASchemeShowsNoUserinfo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/namespaces/kube-system/services") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"apiVersion":"v1","kind":"ServiceList","metadata":{"resourceVersion":"1"},"items":[]}`)
			return
		}
		newFakeHandler(w, r)
	}))
	t.Cleanup(srv.Close)
	bare := strings.Replace(srv.URL, "http://", "", 1)
	path := filepath.Join(t.TempDir(), "kubeconfig")
	cfg := "apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: http://" + bare + "\n- name: other\n  cluster:\n    server: carol:BARE'PW@other.invalid:6443\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\n    namespace: shop\n- name: other\n  context:\n    cluster: other\n    user: fake-user\nusers:\n- name: fake-user\n  user:\n    token: not-a-real-token\n"
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	tree := stoneTree(t, path, live.ReadOnlyConfig)
	for _, line := range []string{"config view", "config view -o json", "config view -o jsonpath={.clusters[*].cluster.server}", "config view --minify --context other"} {
		res := run(t, tree, line)
		if strings.Contains(res.Output, "BARE") || strings.Contains(res.Output, "carol") {
			t.Errorf("%s showed a password held in a server address with no scheme:\n%s", line, res.Output)
		}
	}
	contexts, err := live.ListContexts([]string{path})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range contexts {
		if strings.Contains(c.Server, "BARE") || strings.Contains(c.Server, "carol") {
			t.Errorf("the picker lists context %s at %s", c.Name, c.Server)
		}
	}
}

func TestTheConnectedServersUserinfoNeverShowsInTheStoneWithOrWithoutAScheme(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/namespaces/kube-system/services") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"apiVersion":"v1","kind":"ServiceList","metadata":{"resourceVersion":"1"},"items":[]}`)
			return
		}
		newFakeHandler(w, r)
	}))
	t.Cleanup(srv.Close)
	hostPort := strings.TrimPrefix(srv.URL, "http://")
	for _, server := range []string{"dave:BARE'PW@" + hostPort, "http://dave:BARE'PW@" + hostPort} {
		path := filepath.Join(t.TempDir(), "kubeconfig")
		cfg := "apiVersion: v1\nkind: Config\ncurrent-context: stone\nclusters:\n- name: fake\n  cluster:\n    server: " + server + "\ncontexts:\n- name: stone\n  context:\n    cluster: fake\n    user: fake-user\n    namespace: shop\nusers:\n- name: fake-user\n  user:\n    token: not-a-real-token\n"
		if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
			t.Fatal(err)
		}
		tree := stoneTree(t, path, live.ReadOnlyConfig)
		for _, line := range []string{"cluster-info", "config view --minify", "get pods -n shop", "get --raw /api/v1/namespaces/shop/pods/missing"} {
			res := run(t, tree, line)
			if strings.Contains(res.Output, "BARE") || strings.Contains(res.Output, "dave") {
				t.Errorf("server %s: %s showed the server address's userinfo:\n%s", server, line, res.Output)
			}
		}
	}
}
