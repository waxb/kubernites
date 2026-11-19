package kubectltree_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func drainServer(t *testing.T, s *seen) *httptest.Server {
	podOf := func(name string) string {
		return fmt.Sprintf(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":%q,"namespace":"shop","uid":"u-%s","resourceVersion":"5","ownerReferences":[{"apiVersion":"apps/v1","kind":"ReplicaSet","name":"web-rs","uid":"u-rs","controller":true}]},"spec":{"nodeName":"node-1","containers":[{"name":"c","image":"nginx"}]},"status":{"phase":"Running"}}`, name, name)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.ReadAll(r.Body)
		s.add(mark(r))
		w.Header().Set("Content-Type", "application/json")
		p := r.URL.Path
		switch {
		case p == "/api":
			fmt.Fprint(w, `{"kind":"APIVersions","versions":["v1"],"serverAddressByClientCIDRs":[]}`)
		case p == "/apis":
			fmt.Fprintf(w, `{"kind":"APIGroupList","apiVersion":"v1","groups":[%s]}`, group("policy"))
		case p == "/apis/policy/v1":
			fmt.Fprint(w, `{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"policy/v1","resources":[]}`)
		case p == "/api/v1":
			fmt.Fprint(w, `{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"v1","resources":[{"name":"nodes","singularName":"node","namespaced":false,"kind":"Node","verbs":["get","list","patch"]},{"name":"pods","singularName":"pod","namespaced":true,"kind":"Pod","verbs":["get","list","delete"]},{"name":"pods/eviction","singularName":"","namespaced":true,"group":"policy","version":"v1","kind":"Eviction","verbs":["create"]}]}`)
		case p == "/api/v1/nodes/node-1":
			fmt.Fprint(w, `{"apiVersion":"v1","kind":"Node","metadata":{"name":"node-1","uid":"u-n","resourceVersion":"3"},"spec":{"unschedulable":true}}`)
		case p == "/api/v1/pods":
			fmt.Fprintf(w, `{"apiVersion":"v1","kind":"PodList","metadata":{"resourceVersion":"9"},"items":[%s,%s]}`, podOf("p1"), podOf("p2"))
		case r.Method == http.MethodPost && strings.HasSuffix(p, "/eviction"):
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","status":"Success","code":201}`)
		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"NotFound","code":404,"message":"not found"}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestADrainAsksOneConfirmAtATimeAndEachReleasesTheEvictionItShows(t *testing.T) {
	s := &seen{}
	srv := drainServer(t, s)
	path := kubeconfig(t, srv.URL, "stone", "shop")
	g := gateAt(t, write.Writes, &fakeServer{Server: srv, seen: s}, path)
	tree := newTree(t, path, g.WrapConfig)
	var mu sync.Mutex
	inflight, most := 0, 0
	var shown []string
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, g, srv.URL, h, func(a live.StoneAsk) bool {
			mu.Lock()
			inflight++
			most = max(most, inflight)
			shown = append(shown, a.Order.Request.Path)
			first := len(shown) == 1
			mu.Unlock()
			for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
				mu.Lock()
				n := inflight
				mu.Unlock()
				if n >= 2 {
					break
				}
			}
			mu.Lock()
			inflight--
			mu.Unlock()
			return first
		})
	})
	res := run(t, tree, "drain node-1 --ignore-daemonsets --timeout=20s")
	mu.Lock()
	defer mu.Unlock()
	if most != 1 {
		t.Fatalf("%d CONFIRMs waited at once (asked %v)\n%s", most, shown, res.Output)
	}
	if len(shown) == 0 {
		t.Fatalf("no CONFIRM was asked\n%s", res.Output)
	}
	var evicted []string
	for _, r := range s.all() {
		if strings.HasPrefix(r, "POST ") && strings.HasSuffix(r, "/eviction") {
			evicted = append(evicted, strings.TrimPrefix(r, "POST "))
		}
	}
	if len(evicted) != 1 || evicted[0] != shown[0] {
		t.Fatalf("the one CONFIRM showed %s, the server saw evictions %v", shown[0], evicted)
	}
}
