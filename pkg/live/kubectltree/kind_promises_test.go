package kubectltree_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func (k reachKind) realm(level write.Level, passed ...write.Class) (*kubectltree.Tree, *write.Gate) {
	k.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	listed, err := live.ListContexts([]string{k.config})
	if err != nil {
		k.t.Fatal(err)
	}
	var choice write.Context
	for _, c := range listed {
		if c.Name == k.context {
			choice = c.Remembered()
		}
	}
	if level == write.ReadOnly {
		r, err := live.Connect(ctx, []string{k.config}, k.context, time.Now(), nil)
		if err != nil {
			k.t.Fatal(err)
		}
		k.t.Cleanup(r.Stop)
		return r.Tree(), nil
	}
	g, err := write.New(write.Choice{Context: choice, Level: level, Classes: passed})
	if err != nil {
		k.t.Fatal(err)
	}
	r, err := live.ConnectGated(ctx, []string{k.config}, k.context, g, time.Now(), nil)
	if err != nil {
		k.t.Fatal(err)
	}
	k.t.Cleanup(r.Stop)
	return r.Tree(), g
}

func (k reachKind) promises() {
	k.t.Helper()
	if _, errOut, code := k.binary("apply", "-f", filepath.Join("testdata", "reach", "promises.yaml")); code != 0 {
		k.t.Fatalf("apply: %s", errOut)
	}
	if _, errOut, code := k.binary("wait", "-n", "reach", "--for=condition=Ready", "pod/deaf-1", "--timeout=120s"); code != 0 {
		k.t.Fatalf("wait: %s", errOut)
	}
}

func (k reachKind) podNames() string {
	out, _, _ := k.binary("get", "pods", "-n", "reach", "-o", "name")
	return out
}

func (k reachKind) worker() string {
	out, _, _ := k.binary("get", "nodes", "-o", "name")
	for _, n := range strings.Fields(out) {
		if strings.HasSuffix(n, "-worker") {
			return strings.TrimPrefix(n, "node/")
		}
	}
	k.t.Fatal("no worker node")
	return ""
}

func TestOnKindEveryDebugAndAnAttachingRunAskTheWorkloadsWarningBeforeAnyPodAppears(t *testing.T) {
	k := newReachKind(t)
	pod := k.pod("app=shell")
	worker := k.worker()
	for _, level := range []write.Level{write.Writes, write.Seamless} {
		for _, line := range []string{
			"debug node/" + worker + " -it -n reach --image=busybox:1.36 -- chroot /host true",
			"debug " + pod + " -it -n reach --copy-to=shell-copy --image=busybox:1.36 -- true",
			"run tmp-1 -it -n reach --image=busybox:1.36 --image-pull-policy=Never --restart=Never -- true",
		} {
			tree, g := k.realm(level)
			before := k.podNames()
			var mu sync.Mutex
			var asks []string
			tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
				return live.HoldStoneWrite(ctx, g, g.Context().Server, h, func(a live.StoneAsk) bool {
					mu.Lock()
					defer mu.Unlock()
					asks = append(asks, fmt.Sprintf("%s %s (pods now: %d)", a.Why, a.Class, len(strings.Fields(k.podNames()))))
					return false
				})
			})
			term := kubectltree.NewTerminal(100, 30, &screen{})
			job := tree.StartIn(line, nil, term)
			for deadline := time.Now().Add(60 * time.Second); !job.Done(); time.Sleep(20 * time.Millisecond) {
				if time.Now().After(deadline) {
					job.Stop()
					t.Fatalf("%s at %s never ended", line, level)
				}
			}
			after := k.podNames()
			mu.Lock()
			first := ""
			if len(asks) > 0 {
				first = asks[0]
			}
			mu.Unlock()
			want := fmt.Sprintf("%s %s (pods now: %d)", write.NeedsClass, write.Workloads, len(strings.Fields(before)))
			if first != want || after != before {
				t.Errorf("%s at %s, the warning declined: first ask %q, want %q; pods before %q after %q", line, level, first, want, before, after)
			}
		}
	}
}

func TestOnKindStopOfAForwardToAPodThatReadsNothingReturnsAtOnceAndClosesThePort(t *testing.T) {
	k := newReachKind(t)
	k.promises()
	tree, _ := k.realm(write.Seamless, write.Workloads)
	res := run(t, tree, "port-forward deaf-1 -n reach :8080")
	fws := tree.Forwards()
	if res.Code != 0 || len(fws) != 1 || len(fws[0].Listening) != 1 {
		t.Fatalf("the forward did not open: %+v\n%s", fws, res.Output)
	}
	addr, _, _ := strings.Cut(fws[0].Listening[0], " -> ")
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var wrote atomic.Int64
	go func() {
		chunk := make([]byte, 64<<10)
		for {
			n, err := conn.Write(chunk)
			wrote.Add(int64(n))
			if err != nil {
				return
			}
		}
	}()
	last := int64(-1)
	for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); time.Sleep(2 * time.Second) {
		if now := wrote.Load(); now == last {
			break
		} else {
			last = now
		}
	}
	t.Logf("the pod stopped reading after %d bytes", wrote.Load())
	began := time.Now()
	tree.StopForward(fws[0].ID)
	took := time.Since(began)
	if took > time.Second {
		t.Errorf("STOP took %v on the game's thread", took)
	}
	if c, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
		c.Close()
		t.Errorf("%s still listens after STOP", addr)
	}
	if len(tree.Forwards()) != 0 {
		t.Errorf("the live card still lists %+v", tree.Forwards())
	}
	t.Logf("STOP returned in %v", took)
}

func TestOnKindEscEndsADeleteHeldByAFinalizerAndARolloutStatusAndTheStoneTakesTheNext(t *testing.T) {
	k := newReachKind(t)
	k.promises()
	if _, errOut, code := k.binary("run", "held-1", "-n", "reach", "--image=busybox:1.36", "--image-pull-policy=Never", "--restart=Never", "--overrides", `{"metadata":{"finalizers":["kubernites.example/hold"]}}`, "--", "sleep", "3600"); code != 0 {
		t.Fatalf("run held-1: %s", errOut)
	}
	t.Cleanup(func() {
		k.binary("patch", "pod", "held-1", "-n", "reach", "--type=merge", "-p", `{"metadata":{"finalizers":null}}`)
		k.binary("delete", "pod", "held-1", "-n", "reach", "--ignore-not-found", "--wait=false")
	})
	for _, c := range []struct {
		line  string
		level write.Level
	}{
		{"delete pod held-1 -n reach", write.Seamless},
		{"rollout status deployment/stuck -n reach", write.ReadOnly},
	} {
		tree, _ := k.realm(c.level)
		job := tree.Start(c.line, nil)
		time.Sleep(3 * time.Second)
		if job.Done() {
			t.Fatalf("%s ended on its own: %+v", c.line, job.Result())
		}
		job.Stop()
		for deadline := time.Now().Add(5 * time.Second); !job.Done(); time.Sleep(20 * time.Millisecond) {
			if time.Now().After(deadline) {
				t.Fatalf("Esc did not end %s within 5 s", c.line)
			}
		}
		next := make(chan kubectltree.Result, 1)
		go func() { next <- tree.Run(context.Background(), "get pods -n reach -o name") }()
		select {
		case res := <-next:
			if res.Code != 0 || !strings.Contains(res.Output, "pod/") {
				t.Errorf("after %s the next command answered %d:\n%s", c.line, res.Code, res.Output)
			}
		case <-time.After(10 * time.Second):
			t.Errorf("after Esc on %s the Stone did not take the next command", c.line)
		}
	}
}

func TestOnKindAtReadOnlyAQuotedWebManifestOrDashIsRefusedAndNothingIsFetched(t *testing.T) {
	k := newReachKind(t)
	var hits atomic.Int32
	web := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		fmt.Fprint(w, "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: from-the-web\n  namespace: reach\n")
	}))
	t.Cleanup(web.Close)
	tree, _ := k.realm(write.ReadOnly)
	for _, line := range []string{
		"get -f " + web.URL + "/cm.yaml",
		`get -f '"` + web.URL + `/cm.yaml"'`,
		`get --filename='"` + web.URL + `/cm.yaml"' -o name`,
		`apply -f '"` + web.URL + `/cm.yaml"' --dry-run=server`,
		`create -f '"-"' --dry-run=client -o name`,
	} {
		res := run(t, tree, line)
		if res.Refused == "" || res.Code == 0 {
			t.Errorf("%s at READ ONLY was not refused (code %d):\n%s", line, res.Code, res.Output)
		}
	}
	if n := hits.Load(); n != 0 {
		t.Fatalf("the Stone fetched from the web %d times at READ ONLY", n)
	}
}
