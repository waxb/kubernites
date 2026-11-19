package kubectltree_test

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/write"
)

const proxyLine = "get --raw /api/v1/namespaces/shop/services/web:80/proxy/"

func TestAtWritesAProxyAsksTheWarningThenShowsWhatItOpensWithNoDryRun(t *testing.T) {
	tree, f, asks := gatedTree(t, write.Writes, true)
	res := run(t, tree, proxyLine)
	if res.Code != 0 || !strings.Contains(res.Output, "proxied to /api/v1/namespaces/shop/services/web:80/proxy/") {
		t.Fatalf("the confirmed proxy answered %d:\n%s", res.Code, res.Output)
	}
	if len(*asks) != 2 || (*asks)[0].Why != write.NeedsClass || (*asks)[0].Class != write.Workloads {
		t.Fatalf("the Stone asked %+v", *asks)
	}
	a := (*asks)[1]
	if !a.Confirm() || !a.Connect || a.Opens.Channel != "proxy" || a.Opens.Kind != "Service" || a.Opens.Namespace != "shop" || a.Opens.Name != "web:80" || a.Opens.Path != "/" {
		t.Fatalf("the CONFIRM shows %+v", a)
	}
	if a.Preview.Code != 0 || a.Preview.Err != nil {
		t.Fatalf("a connect request was previewed: %+v", a.Preview)
	}
	seen := f.seen.all()
	if count(seen, "GET /api/v1/namespaces/shop/services/web:80/proxy/") != 1 {
		t.Fatalf("the server saw %v", seen)
	}
	for _, s := range seen {
		if strings.Contains(s, "dryRun") {
			t.Fatalf("a dry run was sent for a connect request: %v", seen)
		}
	}
	again := run(t, tree, proxyLine)
	if again.Code != 0 || len(*asks) != 3 || (*asks)[2].Why != write.NeedsConfirm {
		t.Fatalf("the second proxy asked %+v and answered %d", (*asks)[2:], again.Code)
	}
}

func TestAtWritesACancelledWarningOrConfirmSendsNothingIntoTheWorkload(t *testing.T) {
	tree, f, asks := gatedTree(t, write.Writes, false)
	res := run(t, tree, proxyLine)
	if res.Code == 0 || len(*asks) != 1 || (*asks)[0].Class != write.Workloads {
		t.Fatalf("answered %d after %+v", res.Code, *asks)
	}
	if n := count(f.seen.all(), "GET /api/v1/namespaces/shop/services/web:80/proxy/"); n != 0 {
		t.Fatalf("the proxy reached the server %d times", n)
	}
}

func TestAtSeamlessAProxyGoesOnceTheWarningIsPassed(t *testing.T) {
	tree, f, asks := gatedTree(t, write.Seamless, true)
	for i := 0; i < 2; i++ {
		if res := run(t, tree, proxyLine); res.Code != 0 {
			t.Fatalf("the proxy answered %d:\n%s", res.Code, res.Output)
		}
	}
	if len(*asks) != 1 || (*asks)[0].Class != write.Workloads {
		t.Fatalf("the Stone asked %+v", *asks)
	}
	if n := count(f.seen.all(), "GET /api/v1/namespaces/shop/services/web:80/proxy/"); n != 2 {
		t.Fatalf("the server saw %d proxies", n)
	}
}

func TestAtReadOnlyTheStoneReachesIntoNoWorkload(t *testing.T) {
	f := newFake(t)
	path := kubeconfig(t, f.URL, "stone", "shop")
	tree := newTree(t, path, live.ReadOnlyConfig)
	for _, line := range []string{proxyLine, "get --raw /api/v1/nodes/keep-1/proxy/configz", "get --raw /api/v1/namespaces/shop/pods/web-1/proxy/", "create --raw /api/v1/nodes/keep-1/proxy/run/shop/web-1/web -f /dev/null"} {
		if res := run(t, tree, line); res.Code == 0 {
			t.Errorf("%s answered %d at READ ONLY:\n%s", line, res.Code, res.Output)
		}
	}
	for _, s := range f.seen.all() {
		if strings.Contains(s, "/proxy") {
			t.Fatalf("the server saw %s at READ ONLY", s)
		}
	}
}
