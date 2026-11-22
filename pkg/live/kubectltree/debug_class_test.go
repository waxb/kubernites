package kubectltree_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestANodeDebugPodAndADebugCopyAskTheWorkloadsWarningFirst(t *testing.T) {
	var mu sync.Mutex
	created := map[string]int{}
	f := &fakeServer{seen: &seen{}}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/nodes/keep-1":
			f.seen.add(mark(r))
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"apiVersion":"v1","kind":"Node","metadata":{"name":"keep-1","uid":"u-node","resourceVersion":"5"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/namespaces/shop/pods/web-1":
			f.seen.add(mark(r))
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, strings.Replace(pod, `"spec":{`, `"spec":{"securityContext":{},`, 1))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/namespaces/shop/pods":
			f.seen.add(mark(r))
			body, _ := io.ReadAll(r.Body)
			mu.Lock()
			created[r.URL.Query().Get("dryRun")+" "+string(body)]++
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, pod)
		default:
			f.serve(w, r)
		}
	}))
	t.Cleanup(f.Close)
	path := kubeconfig(t, f.URL, "stone", "shop")
	connected, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		t.Fatal(err)
	}
	g := gateAt(t, write.Seamless, f, path)
	tree := newTree(t, path, live.PinnedTo(connected, g.WrapConfig))
	var asks []live.StoneAsk
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, g, f.URL, h, func(a live.StoneAsk) bool {
			asks = append(asks, a)
			return false
		})
	})
	for _, line := range []string{
		"debug node/keep-1 -n shop --image=busybox:1.36 -- chroot /host true",
		"debug web-1 -n shop --copy-to=web-1-debug --image=busybox:1.36 -- true",
	} {
		asks = nil
		mu.Lock()
		clear(created)
		mu.Unlock()
		res := run(t, tree, line)
		warned := false
		for _, a := range asks {
			if a.Why == write.NeedsClass && a.Class == write.Workloads {
				warned = true
			}
		}
		mu.Lock()
		n := len(created)
		mu.Unlock()
		if n > 0 || !warned {
			t.Errorf("%s at DANGEROUSLY SEAMLESS with the reaching-into-workloads warning not passed and declined: Pods created=%d, workloads warning asked=%v, passed=%v\n%s", line, n, warned, g.Passed(write.Workloads), strings.TrimSpace(res.Output))
		}
	}
}

func TestEveryDebugAndAnAttachingRunAskTheWorkloadsWarningBeforeAnyPodIsCreatedAtEveryWriteLevel(t *testing.T) {
	for _, level := range []write.Level{write.Writes, write.Seamless} {
		for _, line := range []string{
			"debug node/keep-1 -n shop --image=busybox:1.36 -- chroot /host true",
			"debug web-1 -n shop --copy-to=web-1-debug --image=busybox:1.36 -- true",
			"debug web-1 -n shop --image=busybox:1.36 --target=web -- true",
			"run sleeper -n shop --image=busybox:1.36 --restart=Never --attach -- true",
		} {
			var mu sync.Mutex
			var order []string
			f := &fakeServer{seen: &seen{}}
			f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/nodes/keep-1":
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"apiVersion":"v1","kind":"Node","metadata":{"name":"keep-1","uid":"u-node","resourceVersion":"5"}}`)
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/namespaces/shop/pods/web-1":
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, strings.Replace(pod, `"spec":{`, `"spec":{"securityContext":{},`, 1))
				case r.Method != http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/namespaces/shop/pods"):
					mu.Lock()
					order = append(order, r.Method+" "+r.URL.Path+" dryRun="+r.URL.Query().Get("dryRun"))
					mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					fmt.Fprint(w, pod)
				default:
					f.serve(w, r)
				}
			}))
			path := kubeconfig(t, f.URL, "stone", "shop")
			connected, err := clientcmd.BuildConfigFromFlags("", path)
			if err != nil {
				t.Fatal(err)
			}
			g := gateAt(t, level, f, path)
			tree := newTree(t, path, live.PinnedTo(connected, g.WrapConfig))
			tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
				return live.HoldStoneWrite(ctx, g, f.URL, h, func(a live.StoneAsk) bool {
					mu.Lock()
					order = append(order, fmt.Sprintf("ask %s %s", a.Why, a.Class))
					mu.Unlock()
					return false
				})
			})
			res := run(t, tree, line)
			f.Close()
			mu.Lock()
			got := slices.Clone(order)
			mu.Unlock()
			want := fmt.Sprintf("ask %s %s", write.NeedsClass, write.Workloads)
			if len(got) != 1 || got[0] != want {
				t.Errorf("%s at %s, the workloads warning declined: want only %q, got %q\n%s", line, level, want, got, strings.TrimSpace(res.Output))
			}
		}
	}
}
