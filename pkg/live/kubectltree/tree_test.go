package kubectltree_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
)

var home string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "kubectltree-home-")
	if err != nil {
		panic(err)
	}
	home = dir
	os.Setenv("HOME", home)
	os.Setenv("KUBECONFIG", "")
	os.Setenv("KUBECACHEDIR", "")
	clientcmd.RecommendedHomeFile = filepath.Join(home, ".kube", "config")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func recordingGate(log *seen) func(*rest.Config) *rest.Config {
	return func(c *rest.Config) *rest.Config {
		c.Wrap(func(rt http.RoundTripper) http.RoundTripper {
			return rtFunc(func(r *http.Request) (*http.Response, error) {
				log.add(mark(r))
				return rt.RoundTrip(r)
			})
		})
		return c
	}
}

func newTree(t *testing.T, path string, wrap func(*rest.Config) *rest.Config) *kubectltree.Tree {
	t.Helper()
	tree, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{path}, Context: "stone", Wrap: wrap})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tree.Close() })
	return tree
}

func run(t *testing.T, tree *kubectltree.Tree, line string) kubectltree.Result {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return tree.Run(ctx, line)
}

var everyCommand = []string{
	"get pods -n shop -o name", "get deployment web -n shop -o json", "describe deployment web -n shop", "logs web-1 -n shop",
	"api-resources", "api-versions", "version", "auth can-i get pods", "auth can-i --list", "explain pods",
	"scale deployment web -n shop --replicas=3", "label deployment web -n shop tier=front", "annotate deployment web -n shop note=hi",
	"patch deployment web -n shop -p {\"spec\":{\"replicas\":4}}", "delete pod web-1 -n shop --wait=false", "rollout restart deployment web -n shop",
	"cordon keep-1", "create configmap c -n shop --from-literal=a=b", "edit deployment web -n shop",
	"get secrets -n shop", "top pods -n shop", "events -n shop", "cluster-info",
}

func TestNoTreeWithoutAGate(t *testing.T) {
	if _, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{"x"}, Context: "c"}); !errors.Is(err, kubectltree.ErrNoGate) {
		t.Fatalf("a tree without a gate: %v", err)
	}
	if _, err := kubectltree.New(kubectltree.Config{Kubeconfig: []string{"x"}, Wrap: live.ReadOnlyConfig}); !errors.Is(err, kubectltree.ErrNoContext) {
		t.Fatalf("a tree without a context: %v", err)
	}
}

func TestEveryRequestTheTreeMakesPassesTheGate(t *testing.T) {
	f := newFake(t)
	gate := &seen{}
	tree := newTree(t, kubeconfig(t, f.URL, "stone", "shop"), recordingGate(gate))
	for _, line := range everyCommand {
		run(t, tree, line)
	}
	tree.Apply(context.Background(), deploymentYAML(5), false)
	tree.Apply(context.Background(), deploymentYAML(5), true)
	server, passed := f.seen.all(), gate.all()
	slices.Sort(server)
	slices.Sort(passed)
	if len(server) == 0 {
		t.Fatal("the server saw nothing")
	}
	if !slices.Equal(server, passed) {
		t.Fatalf("the server saw %d requests and the gate passed %d:\nserver %v\ngate   %v", len(server), len(passed), server, passed)
	}
	if len(f.seen.writes()) == 0 {
		t.Fatal("with a gate that lets everything through, no write reached the server: the test proves nothing")
	}
}

func TestAtReadOnlyNoWriteLeavesTheGame(t *testing.T) {
	f := newFake(t)
	tree := newTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig)
	for _, line := range everyCommand {
		res := run(t, tree, line)
		if strings.Contains(res.Output, "refused before it left Kubernites") && res.Code == 0 {
			t.Errorf("%s was refused and still answered 0", line)
		}
	}
	a := tree.Apply(context.Background(), deploymentYAML(5), false)
	if a.Code == 0 || !strings.Contains(a.Output, "refused before it left Kubernites") {
		t.Errorf("the Scriptorium's apply at READ ONLY answered %d:\n%s", a.Code, a.Output)
	}
	if w := f.seen.writes(); len(w) != 0 {
		t.Fatalf("writes reached the server at READ ONLY: %v", w)
	}
	for _, r := range f.seen.all() {
		if strings.Contains(r, "/secrets") {
			t.Fatalf("a Secret was asked for at READ ONLY: %s", r)
		}
	}
	res := run(t, tree, "scale deployment web -n shop --replicas=3")
	if !strings.Contains(res.Output, "refused before it left Kubernites") {
		t.Fatalf("scale at READ ONLY:\n%s", res.Output)
	}
}

func TestTheTreesConfigCarriesTheGate(t *testing.T) {
	f := newFake(t)
	for _, line := range []string{"get pods -n shop -o name", "scale deployment web -n shop --replicas=3", "version"} {
		cf := kubectltree.ConfigFor(t, newTree(t, kubeconfig(t, f.URL, "stone", "shop"), live.ReadOnlyConfig), line)
		c, err := cf.ToRESTConfig()
		if err != nil {
			t.Fatal(err)
		}
		if c.WrapTransport == nil {
			t.Fatalf("%s: the config handed to kubectl has no wrapped transport", line)
		}
		rt := c.WrapTransport(rtFunc(func(r *http.Request) (*http.Response, error) {
			t.Fatalf("%s: %s %s passed the wrapped transport", line, r.Method, r.URL.Path)
			return nil, nil
		}))
		for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
			req, _ := http.NewRequest(m, f.URL+"/apis/apps/v1/namespaces/shop/deployments/web", strings.NewReader("{}"))
			if _, err := rt.RoundTrip(req); !errors.Is(err, live.ErrReadOnly) {
				t.Errorf("%s: %s through the tree's config answered %v", line, m, err)
			}
		}
	}
}

func TestTheTreeUsesOnlyTheChosenKubeconfigAndContext(t *testing.T) {
	chosen, other := newFake(t), newFake(t)
	path := kubeconfig(t, chosen.URL, "stone", "shop")
	otherPath := kubeconfig(t, other.URL, "elsewhere", "default")
	t.Setenv("KUBECONFIG", otherPath)
	before, _ := os.ReadFile(path)
	info, _ := os.Stat(path)
	tree := newTree(t, path, live.ReadOnlyConfig)
	for _, line := range []string{
		"get pods -o name", "--kubeconfig=" + otherPath + " get pods -o name", "get pods --context=elsewhere -o name",
		"get pods -s " + other.URL + " -o name", "get pods --server=" + other.URL, "get pods --token=x", "get pods --as=admin",
		"get pods --user=x", "get pods --cluster=x", "get pods --insecure-skip-tls-verify", "get pods --certificate-authority=/x",
		"get pods --cache-dir=" + t.TempDir(), "get pods --proxy-url=http://127.0.0.1:1", "get pods --as-group=system:masters",
	} {
		res := run(t, tree, line)
		if line != "get pods -o name" && !strings.Contains(res.Output, kubectltree.ReasonConnection) {
			t.Errorf("%s answered:\n%s", line, res.Output)
		}
	}
	if len(chosen.seen.all()) == 0 {
		t.Fatal("the chosen cluster saw nothing")
	}
	if s := other.seen.all(); len(s) != 0 {
		t.Fatalf("another cluster was reached: %v", s)
	}
	for _, line := range []string{"config view", "config get-contexts", "config current-context"} {
		res := run(t, tree, line)
		if strings.Contains(res.Output, "elsewhere") || strings.Contains(res.Output, other.URL) {
			t.Errorf("%s read another kubeconfig:\n%s", line, res.Output)
		}
	}
	after, _ := os.ReadFile(path)
	info2, _ := os.Stat(path)
	if string(before) != string(after) || !info.ModTime().Equal(info2.ModTime()) {
		t.Fatal("the kubeconfig changed")
	}
}

func TestTheKubeconfigIsNeverWritten(t *testing.T) {
	f := newFake(t)
	path := kubeconfig(t, f.URL, "stone", "shop")
	tree := newTree(t, path, live.ReadOnlyConfig)
	before, _ := os.ReadFile(path)
	info, _ := os.Stat(path)
	for _, line := range []string{
		"config use-context stone", "config use stone", "config set-context x --cluster=fake", "config set-cluster fake --server=https://x",
		"config set-credentials u --token=t", "config set users.u.token t", "config unset current-context", "config delete-context stone",
		"config delete-cluster fake", "config delete-user fake-user", "config rename-context stone renamed",
		"config --kubeconfig=" + filepath.Join(t.TempDir(), "x") + " set-context y",
	} {
		res := run(t, tree, line)
		if res.Code == 0 || !(strings.Contains(res.Output, kubectltree.ReasonKubeconfig) || strings.Contains(res.Output, kubectltree.ReasonConnection)) {
			t.Errorf("%s answered %d:\n%s", line, res.Code, res.Output)
		}
	}
	for _, line := range []string{"config view --raw", "config view --flatten"} {
		res := run(t, tree, line)
		if !strings.Contains(res.Output, kubectltree.ReasonCredential) || strings.Contains(res.Output, "not-a-real-token") {
			t.Errorf("%s answered:\n%s", line, res.Output)
		}
	}
	if res := run(t, tree, "config view"); res.Code != 0 || strings.Contains(res.Output, "not-a-real-token") || !strings.Contains(res.Output, "REDACTED") {
		t.Errorf("config view:\n%s", res.Output)
	}
	after, _ := os.ReadFile(path)
	info2, _ := os.Stat(path)
	if string(before) != string(after) || !info.ModTime().Equal(info2.ModTime()) {
		t.Fatal("the kubeconfig changed")
	}
	if _, err := os.Stat(filepath.Join(home, ".kube")); !errors.Is(err, os.ErrNotExist) {
		entries, _ := os.ReadDir(filepath.Join(home, ".kube"))
		t.Fatalf("something was written under ~/.kube: %v", entries)
	}
}

func TestTheTreeRefusesWhereClientGoWouldMigrateAKubeconfig(t *testing.T) {
	f := newFake(t)
	path := kubeconfig(t, f.URL, "stone", "shop")
	tree := newTree(t, path, live.ReadOnlyConfig)
	old := filepath.Join(home, ".kube", ".kubeconfig")
	if err := os.MkdirAll(filepath.Dir(old), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(old, []byte("apiVersion: v1\nkind: Config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(filepath.Join(home, ".kube")) })
	res := run(t, tree, "get pods -o name")
	if res.Refused == "" || !strings.Contains(res.Output, "never writes a kubeconfig") {
		t.Fatalf("the tree ran where client-go would migrate a kubeconfig:\n%s", res.Output)
	}
	if _, err := os.Stat(clientcmd.RecommendedHomeFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("a kubeconfig was migrated")
	}
	if len(f.seen.all()) != 0 {
		t.Fatalf("the server was reached: %v", f.seen.all())
	}
}

func deploymentYAML(replicas int) string {
	return strings.ReplaceAll(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  namespace: shop
  resourceVersion: "41"
  uid: u-web
  creationTimestamp: "2026-11-01T10:00:00Z"
  generation: 3
spec:
  replicas: N
  selector:
    matchLabels:
      app: web
  template:
    metadata:
      labels:
        app: web
    spec:
      containers:
      - name: web
        image: nginx:1.27
status:
  replicas: 2
`, "N", strconv.Itoa(replicas))
}

func TestAStoneCommandKeepsItsOutputBounded(t *testing.T) {
	big := strings.Repeat("0123456789abcdef", 1<<16) + "\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/namespaces/shop/pods/web-1/log" {
			w.Header().Set("Content-Type", "text/plain")
			for range 10 {
				w.Write([]byte(big))
			}
			return
		}
		newFakeHandler(w, r)
	}))
	t.Cleanup(srv.Close)
	tree := newTree(t, kubeconfig(t, srv.URL, "stone", "shop"), live.ReadOnlyConfig)
	res := run(t, tree, "logs web-1 -n shop")
	if res.Code != 0 {
		t.Fatalf("logs answered %d", res.Code)
	}
	if len(res.Output) > kubectltree.MaxOutput || len(res.Stdout) > kubectltree.MaxOutput {
		t.Fatalf("the Stone kept %d and %d bytes of one command's output", len(res.Output), len(res.Stdout))
	}
}
