package kubectltree_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestThePinnedStoneSendsJSONBodies(t *testing.T) {
	var mu sync.Mutex
	var sent []string
	f := &fakeServer{seen: &seen{}}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && !strings.Contains(r.URL.Path, "selfsubject") {
			mu.Lock()
			sent = append(sent, r.Method+" "+r.URL.Path+" Content-Type="+r.Header.Get("Content-Type"))
			mu.Unlock()
		}
		f.serve(w, r)
	}))
	t.Cleanup(f.Close)
	path := kubeconfig(t, f.URL, "stone", "shop")
	connected, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		t.Fatal(err)
	}
	g := gateAt(t, write.Seamless, f, path)
	g.Pass(write.Secrets)
	tree := newTree(t, path, live.PinnedTo(connected, g.WrapConfig))
	for _, line := range []string{
		"create configmap c -n shop --from-literal=a=b",
		"create secret generic s2 -n shop --from-literal=password=hunter2",
	} {
		run(t, tree, line)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(sent) == 0 {
		t.Fatal("no write reached the server")
	}
	for _, s := range sent {
		if !strings.Contains(s, "Content-Type=application/json") {
			t.Errorf("the Stone's kubectl, built as the app builds it (PinnedTo), sent a body that is not JSON: %s", s)
		}
	}
}
