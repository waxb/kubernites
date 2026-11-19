package kubectltree_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/kubectltree"
	"github.com/waxb/kubernites/pkg/live/write"
)

func gateAt(t *testing.T, level write.Level, f *fakeServer, path string) *write.Gate {
	t.Helper()
	g, err := write.New(write.Choice{Context: write.Context{Name: "stone", Server: f.URL, User: "fake-user"}, Level: level})
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
	return g
}

func gatedTree(t *testing.T, level write.Level, answer bool) (*kubectltree.Tree, *fakeServer, *[]live.StoneAsk) {
	t.Helper()
	f := newFake(t)
	path := kubeconfig(t, f.URL, "stone", "shop")
	g := gateAt(t, level, f, path)
	tree := newTree(t, path, g.WrapConfig)
	asks := &[]live.StoneAsk{}
	tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
		return live.HoldStoneWrite(ctx, g, f.URL, h, func(a live.StoneAsk) bool {
			*asks = append(*asks, a)
			return answer
		})
	})
	return tree, f, asks
}

func count(list []string, want string) int {
	n := 0
	for _, s := range list {
		if s == want {
			n++
		}
	}
	return n
}

func TestAtWritesATypedWriteShowsItsDryRunAndWaitsForConfirm(t *testing.T) {
	tree, f, asks := gatedTree(t, write.Writes, true)
	res := run(t, tree, "scale deployment web --replicas=3")
	if res.Code != 0 || !strings.Contains(res.Output, "deployment.apps/web scaled") {
		t.Fatalf("the confirmed scale answered %d:\n%s", res.Code, res.Output)
	}
	if len(*asks) != 1 || !(*asks)[0].Confirm() || !(*asks)[0].Preview.OK() || (*asks)[0].Order.Request.Method != "PATCH" || (*asks)[0].Order.Request.Path != "/apis/apps/v1/namespaces/shop/deployments/web/scale" {
		t.Fatalf("the Stone asked %+v", *asks)
	}
	if !strings.Contains(string((*asks)[0].Order.Request.Body), `"replicas":3`) || (*asks)[0].Command != "kubectl scale deployment web --replicas=3" {
		t.Fatalf("the ask carries %s for %q", (*asks)[0].Order.Request.Body, (*asks)[0].Command)
	}
	seen := f.seen.all()
	dry, real := count(seen, "PATCH /apis/apps/v1/namespaces/shop/deployments/web/scale?dryRun"), count(seen, "PATCH /apis/apps/v1/namespaces/shop/deployments/web/scale")
	if dry != 1 || real != 1 || slices.Index(seen, "PATCH /apis/apps/v1/namespaces/shop/deployments/web/scale?dryRun") > slices.Index(seen, "PATCH /apis/apps/v1/namespaces/shop/deployments/web/scale") {
		t.Fatalf("the server saw %d dry runs and %d writes, in the order %v", dry, real, seen)
	}
	if len(res.Writes) != 1 || res.Writes[0] != (kubectltree.Write{Method: "PATCH", Path: "/apis/apps/v1/namespaces/shop/deployments/web/scale", Code: 200, Status: "200 OK"}) {
		t.Fatalf("the tree reported %+v", res.Writes)
	}
}

func TestAtWritesCancelSendsNothingButTheDryRun(t *testing.T) {
	tree, f, asks := gatedTree(t, write.Writes, false)
	res := run(t, tree, "label deployment web tier=front")
	if res.Code == 0 || !strings.Contains(res.Output, "CONFIRM") {
		t.Fatalf("the cancelled label answered %d:\n%s", res.Code, res.Output)
	}
	if len(*asks) != 1 || len(res.Writes) != 0 {
		t.Fatalf("the Stone asked %d times and reported %+v", len(*asks), res.Writes)
	}
	for _, s := range f.seen.writes() {
		if !strings.HasSuffix(s, "?dryRun") {
			t.Fatalf("a write reached the server after CANCEL: %s", s)
		}
	}
}

func TestAtWritesAClassWarningComesFirst(t *testing.T) {
	tree, f, asks := gatedTree(t, write.Writes, true)
	run(t, tree, "get secrets -o name")
	if len(*asks) != 1 || (*asks)[0].Why != write.NeedsClass || (*asks)[0].Class != write.Secrets {
		t.Fatalf("the Stone asked %+v", *asks)
	}
	if count(f.seen.all(), "GET /api/v1/namespaces/shop/secrets") != 1 {
		t.Fatalf("the Secrets read after the warning: %v", f.seen.all())
	}
	run(t, tree, "get secrets -o name")
	if len(*asks) != 1 {
		t.Fatal("a passed class warning was asked again")
	}
}

func TestAtSeamlessCommandsRunAsKubectlRuns(t *testing.T) {
	tree, f, asks := gatedTree(t, write.Seamless, false)
	res := run(t, tree, "annotate deployment web note=hi")
	if res.Code != 0 || len(*asks) != 0 {
		t.Fatalf("seamless annotate answered %d with %d asks:\n%s", res.Code, len(*asks), res.Output)
	}
	if count(f.seen.all(), "PATCH /apis/apps/v1/namespaces/shop/deployments/web") != 1 || slices.ContainsFunc(f.seen.all(), func(s string) bool { return strings.HasSuffix(s, "?dryRun") }) {
		t.Fatalf("seamless sent %v", f.seen.all())
	}
}

func TestAtWritesWithoutAHoldNothingIsSent(t *testing.T) {
	f := newFake(t)
	path := kubeconfig(t, f.URL, "stone", "shop")
	tree := newTree(t, path, gateAt(t, write.Writes, f, path).WrapConfig)
	res := run(t, tree, "delete pod web-1 --wait=false")
	if res.Code == 0 || !strings.Contains(res.Output, "refused by the write gate") {
		t.Fatalf("delete without a hold answered %d:\n%s", res.Code, res.Output)
	}
	if w := f.seen.writes(); len(w) != 0 {
		t.Fatalf("writes reached the server: %v", w)
	}
}

func TestTheTreeReportsOnlyTheChangesThatReachTheServer(t *testing.T) {
	tree, _, _ := gatedTree(t, write.Seamless, false)
	for _, line := range []string{"get pods", "auth can-i delete pods", "auth whoami", "delete pod web-1 --dry-run=server", "scale deployment web --replicas=3 --dry-run=server"} {
		if res := run(t, tree, line); len(res.Writes) != 0 {
			t.Fatalf("%s reported %+v:\n%s", line, res.Writes, res.Output)
		}
	}
	res := run(t, tree, "delete pod web-1 --wait=false")
	if len(res.Writes) != 1 || res.Writes[0].Method != "DELETE" || res.Writes[0].Path != "/api/v1/namespaces/shop/pods/web-1" || res.Writes[0].Code != 200 {
		t.Fatalf("the delete reported %+v:\n%s", res.Writes, res.Output)
	}
	job := tree.Start("scale deployment web --replicas=3", nil)
	for deadline := time.Now().Add(10 * time.Second); !job.Done(); time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the scale never ended")
		}
	}
	if w := job.TakeWrites(); len(w) != 1 || w[0].Method != "PATCH" {
		t.Fatalf("the job reported %+v", w)
	}
	if w := job.TakeWrites(); len(w) != 0 {
		t.Fatalf("a write was reported twice: %+v", w)
	}
}

func TestAtWritesTheStoneSendsNoChangeWithoutItsPreviewEvenToAnAggregatedAPIServer(t *testing.T) {
	for _, learnt := range []bool{false, true} {
		for _, level := range []write.Level{write.Writes, write.Seamless} {
			f := &fakeServer{seen: &seen{}, aggregated: true}
			f.Server = httptest.NewServer(http.HandlerFunc(f.serve))
			path := kubeconfig(t, f.URL, "stone", "shop")
			g := gateAt(t, level, f, path)
			if learnt {
				if err := g.LearnLocalGroups(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
			tree := newTree(t, path, g.WrapConfig)
			var asks []live.StoneAsk
			tree.SetHold(func(ctx context.Context, h kubectltree.Held) bool {
				return live.HoldStoneWrite(ctx, g, f.URL, h, func(a live.StoneAsk) bool {
					asks = append(asks, a)
					return true
				})
			})
			for _, line := range []string{`patch gauges g --type=merge -p {"spec":{"a":1}}`, "annotate gauges g note=hi", "delete gauges g --wait=false"} {
				asks = nil
				before := len(f.seen.writes())
				res := run(t, tree, line)
				sent := f.seen.writes()[before:]
				if level == write.Seamless {
					if res.Code != 0 || len(asks) != 0 || len(sent) != 1 || strings.HasSuffix(sent[0], "?dryRun") {
						t.Errorf("seamless, learnt %v: %s answered %d after %d asks, the server saw %v:\n%s", learnt, line, res.Code, len(asks), sent, res.Output)
					}
					continue
				}
				if res.Code == 0 || len(asks) != 0 || len(sent) != 0 {
					t.Errorf("writes, learnt %v: %s answered %d after asks %+v, the server saw %v:\n%s", learnt, line, res.Code, asks, sent, res.Output)
				}
				if !strings.Contains(res.Output, "refused by the write gate") || !strings.Contains(res.Output, "metrics.example.io") || !strings.Contains(res.Output, "no preview") || !strings.Contains(res.Output, "DANGEROUSLY SEAMLESS") {
					t.Errorf("writes, learnt %v: %s does not say why:\n%s", learnt, line, res.Output)
				}
			}
			f.Close()
		}
	}
}
