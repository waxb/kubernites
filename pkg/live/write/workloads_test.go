package write

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
)

type workloadCase struct {
	what               string
	method, path, body string
	header             []string
	connect            bool
	declared           bool
}

var (
	wsExec   = []string{"Sec-Websocket-Protocol", "v5.channel.k8s.io"}
	spdyUp   = []string{"Connection", "Upgrade", "Upgrade", "SPDY/3.1"}
	wsTunnel = []string{"Sec-Websocket-Protocol", "SPDY/3.1+portforward.k8s.io"}
)

var workloadRequests = []workloadCase{
	{"exec over SPDY", http.MethodPost, "/api/v1/namespaces/shop/pods/web-1/exec?command=sh&container=app&stdin=true&stdout=true&tty=true", "", spdyUp, true, false},
	{"exec over a websocket", http.MethodGet, "/api/v1/namespaces/shop/pods/web-1/exec?command=sh&container=app&stdin=true&stdout=true&tty=true", "", wsExec, true, false},
	{"exec with no upgrade header", http.MethodPost, "/api/v1/namespaces/shop/pods/web-1/exec?command=id&container=app&stdout=true", "", nil, true, false},
	{"attach over SPDY", http.MethodPost, "/api/v1/namespaces/shop/pods/web-1/attach?container=app&stdin=true&stdout=true&tty=true", "", spdyUp, true, false},
	{"attach over a websocket", http.MethodGet, "/api/v1/namespaces/shop/pods/web-1/attach?container=app&stdin=true&stdout=true&tty=true", "", wsExec, true, false},
	{"port-forward over SPDY", http.MethodPost, "/api/v1/namespaces/shop/pods/web-1/portforward", "", spdyUp, true, false},
	{"port-forward tunnelled over a websocket", http.MethodGet, "/api/v1/namespaces/shop/pods/web-1/portforward", "", wsTunnel, true, false},
	{"a Pod's proxy", http.MethodGet, "/api/v1/namespaces/shop/pods/web-1/proxy/", "", nil, true, false},
	{"a Pod's proxy on a port", http.MethodGet, "/api/v1/namespaces/shop/pods/web-1:8080/proxy/healthz", "", nil, true, false},
	{"a Service's proxy", http.MethodGet, "/api/v1/namespaces/shop/services/web:80/proxy/admin", "", nil, true, false},
	{"a Service's proxy, posted", http.MethodPost, "/api/v1/namespaces/shop/services/https:web:443/proxy/login", `{"user":"a"}`, nil, true, false},
	{"a Node's proxy", http.MethodGet, "/api/v1/nodes/kind-worker/proxy/configz", "", nil, true, false},
	{"the kubelet's run through a Node's proxy", http.MethodPost, "/api/v1/nodes/kind-worker/proxy/run/shop/web-1/app?cmd=id", `{}`, nil, true, false},
	{"an ephemeral container", http.MethodPatch, "/api/v1/namespaces/shop/pods/web-1/ephemeralcontainers", `{"spec":{"ephemeralContainers":[{"name":"debugger","image":"busybox"}]}}`, nil, false, false},
	{"an ephemeral container, replaced", http.MethodPut, "/api/v1/namespaces/shop/pods/web-1/ephemeralcontainers", `{"kind":"Pod"}`, nil, false, false},
	{"a debug copy of a Pod", http.MethodPost, "/api/v1/namespaces/shop/pods", `{"kind":"Pod","metadata":{"name":"web-1-debug"}}`, nil, false, true},
	{"a Node's debug Pod", http.MethodPost, "/api/v1/namespaces/default/pods", `{"kind":"Pod","metadata":{"name":"node-debugger-kind-worker-x"},"spec":{"hostPID":true}}`, nil, false, true},
}

func sendIn(ctx context.Context, g *Gate, rec *recorder, method, target, body string, header ...string) error {
	tr := gated{g: g, next: rec, base: "", known: true}
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, "https://127.0.0.1:6443"+target, rd)
	if err != nil {
		return err
	}
	for i := 0; i+1 < len(header); i += 2 {
		req.Header.Set(header[i], header[i+1])
	}
	resp, err := tr.RoundTrip(req)
	if err == nil {
		resp.Body.Close()
	}
	return err
}

func (c workloadCase) ctx() context.Context {
	if c.declared {
		return WithClasses(context.Background(), Workloads)
	}
	return context.Background()
}

func (c workloadCase) request() Request {
	path, query, _ := strings.Cut(c.path, "?")
	q, _ := url.ParseQuery(query)
	var body []byte
	if c.body != "" {
		body = []byte(c.body)
	}
	return Request{Method: c.method, Path: path, Query: q, Body: body}
}

func withDryRun(target string) string {
	if strings.Contains(target, "?") {
		return target + "&dryRun=All"
	}
	return target + "?dryRun=All"
}

func TestReachingIntoAWorkloadClimbsTheLadder(t *testing.T) {
	for _, l := range []Level{Writes, Seamless} {
		for _, c := range workloadRequests {
			g, rec := gate(t, l), &recorder{}
			r := refusedFor(t, sendIn(c.ctx(), g, rec, c.method, c.path, c.body, c.header...), NeedsClass)
			if r.Class != Workloads {
				t.Errorf("%v %s: needs %v, want the workloads warning", l, c.what, r.Class)
			}
			if c.method != http.MethodGet {
				refusedFor(t, sendIn(c.ctx(), g, rec, c.method, withDryRun(c.path), c.body, c.header...), NeedsClass)
			}
			if len(rec.seen) != 0 {
				t.Fatalf("%v %s: the server saw %v before the warning", l, c.what, rec.seen)
			}
			g.Pass(Workloads)
			if l == Seamless {
				if err := sendIn(c.ctx(), g, rec, c.method, c.path, c.body, c.header...); err != nil {
					t.Errorf("%v %s once warned: %v", l, c.what, err)
				}
				if len(rec.seen) != 1 {
					t.Errorf("%v %s: the server saw %v, want the one request", l, c.what, rec.seen)
				}
				continue
			}
			if c.connect {
				refusedFor(t, sendIn(c.ctx(), g, rec, c.method, withDryRun(c.path), c.body, c.header...), NeedsConfirm)
			} else if err := sendIn(c.ctx(), g, rec, c.method, withDryRun(c.path), c.body, c.header...); err != nil {
				t.Errorf("%v %s: its dry run is its preview: %v", l, c.what, err)
			}
			refusedFor(t, sendIn(c.ctx(), g, rec, c.method, c.path, c.body, c.header...), NeedsConfirm)
			g.Issue(c.request())
			if err := sendIn(c.ctx(), g, rec, c.method, c.path, c.body, c.header...); err != nil {
				t.Errorf("%v %s with its ticket: %v", l, c.what, err)
			}
			refusedFor(t, sendIn(c.ctx(), g, rec, c.method, c.path, c.body, c.header...), NeedsConfirm)
			want := 1
			if !c.connect {
				want = 2
			}
			if len(rec.seen) != want {
				t.Errorf("%v %s: the server saw %v", l, c.what, rec.seen)
			}
		}
	}
}

func TestTheWorkloadsWarningIsAskedOncePerContextAndAgainAfterReset(t *testing.T) {
	dir := t.TempDir()
	ctxName := choice(Writes).Context
	c := workloadRequests[1]
	g, rec := gate(t, Seamless), &recorder{}
	refusedFor(t, sendIn(c.ctx(), g, rec, c.method, c.path, c.body, c.header...), NeedsClass)
	g.Pass(Workloads)
	for i := 0; i < 3; i++ {
		if err := sendIn(c.ctx(), g, rec, c.method, c.path, c.body, c.header...); err != nil {
			t.Fatalf("passed once, asked again: %v", err)
		}
	}
	m := Memory{}
	m.Keep(Remembered{Context: ctxName, Level: Seamless, Warned: Seamless, Classes: g.Classes()})
	if err := SaveMemory(dir, m); err != nil {
		t.Fatal(err)
	}
	back, err := LoadMemory(dir)
	if err != nil {
		t.Fatal(err)
	}
	r, _, ok := back.Recall(ctxName)
	if !ok || !slices.Contains(r.Classes, Workloads) {
		t.Fatalf("remembered %+v %v", r, ok)
	}
	again, err := New(r.Choice())
	if err != nil {
		t.Fatal(err)
	}
	if err := sendIn(c.ctx(), again, rec, c.method, c.path, c.body, c.header...); err != nil {
		t.Fatalf("a remembered context asked again: %v", err)
	}
	other := r.Choice()
	other.Context.Name = "kind-other"
	other.Classes = nil
	fresh, err := New(other)
	if err != nil {
		t.Fatal(err)
	}
	refusedFor(t, sendIn(c.ctx(), fresh, rec, c.method, c.path, c.body, c.header...), NeedsClass)
	back.Forget(ctxName.Name)
	if err := SaveMemory(dir, back); err != nil {
		t.Fatal(err)
	}
	reset, err := LoadMemory(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := reset.Recall(ctxName); ok {
		t.Fatal("RESET LIVE CHOICES kept the context")
	}
	after, err := New(Choice{Context: ctxName, Level: Seamless})
	if err != nil {
		t.Fatal(err)
	}
	refusedFor(t, sendIn(c.ctx(), after, rec, c.method, c.path, c.body, c.header...), NeedsClass)
	if len(rec.seen) != 4 {
		t.Fatalf("the server saw %v", rec.seen)
	}
}

func TestOnlyAPodsChannelsAndProxiesReachIntoAWorkload(t *testing.T) {
	odd := []workloadCase{
		{"exec on a Deployment", http.MethodPost, "/apis/apps/v1/namespaces/shop/deployments/web/exec?command=sh", "", spdyUp, false, false},
		{"attach on a ConfigMap", http.MethodPost, "/api/v1/namespaces/shop/configmaps/conf/attach", "", nil, false, false},
		{"port-forward on a Service", http.MethodGet, "/api/v1/namespaces/shop/services/web/portforward", "", wsTunnel, false, false},
		{"a custom resource's proxy", http.MethodGet, "/apis/lair.example/v1/namespaces/shop/dragons/a/proxy/", "", nil, false, false},
		{"a Secret's proxy", http.MethodGet, "/api/v1/namespaces/shop/secrets/key/proxy/", "", nil, false, false},
		{"exec under a watch", http.MethodGet, "/api/v1/watch/namespaces/shop/pods/web-1/exec?command=sh", "", wsExec, false, false},
		{"the old proxy path to a Pod", http.MethodGet, "/api/v1/proxy/namespaces/shop/pods/web-1", "", nil, false, false},
		{"the old proxy path to a Node", http.MethodGet, "/api/v1/proxy/nodes/kind-worker/configz", "", nil, false, false},
		{"a namespace's proxy", http.MethodGet, "/api/v1/namespaces/shop/proxy", "", nil, false, false},
		{"an upgrade on a list", http.MethodGet, "/api/v1/pods", "", spdyUp, false, false},
		{"a websocket on a list", http.MethodGet, "/api/v1/namespaces/shop/pods", "", wsExec, false, false},
		{"an upgrade on a scale", http.MethodPatch, scalePath, `{"spec":{"replicas":3}}`, spdyUp, false, false},
		{"an upgrade on a review", http.MethodPost, accessReview, `{}`, spdyUp, false, false},
		{"exec on every Pod", http.MethodPost, "/api/v1/namespaces/shop/pods/exec", "", spdyUp, false, false},
		{"exec on a Pod in no namespace", http.MethodPost, "/api/v1/pods/web-1/exec?command=sh", "", spdyUp, false, false},
		{"exec past the subresource", http.MethodPost, "/api/v1/namespaces/shop/pods/web-1/exec/more", "", spdyUp, false, false},
	}
	for _, l := range []Level{Writes, Seamless} {
		for _, c := range odd {
			g, rec := gate(t, l, AllClasses...), &recorder{}
			g.Issue(c.request())
			r := refusedFor(t, sendIn(c.ctx(), g, rec, c.method, c.path, c.body, c.header...), NotAtLevel)
			if !strings.Contains(r.Error(), "reach") {
				t.Errorf("%v %s: the refusal does not say why: %v", l, c.what, r)
			}
			if len(rec.seen) != 0 {
				t.Errorf("%v %s: the server saw %v", l, c.what, rec.seen)
			}
		}
	}
}

func TestAConnectRequestSaysWhatItOpens(t *testing.T) {
	cases := []struct {
		path string
		want Opening
	}{
		{"/api/v1/namespaces/shop/pods/web-1/exec?command=sh&command=-c&command=top&container=app&stdin=true&stdout=true&tty=true",
			Opening{Channel: "exec", Kind: "Pod", Namespace: "shop", Name: "web-1", Container: "app", Command: []string{"sh", "-c", "top"}, Stdin: true, TTY: true}},
		{"/api/v1/namespaces/shop/pods/web-1/attach?container=app&stdout=true",
			Opening{Channel: "attach", Kind: "Pod", Namespace: "shop", Name: "web-1", Container: "app"}},
		{"/api/v1/namespaces/shop/pods/web-1/portforward", Opening{Channel: "portforward", Kind: "Pod", Namespace: "shop", Name: "web-1"}},
		{"/api/v1/namespaces/shop/services/web:80/proxy/admin/x", Opening{Channel: "proxy", Kind: "Service", Namespace: "shop", Name: "web:80", Path: "/admin/x"}},
		{"/api/v1/nodes/kind-worker/proxy", Opening{Channel: "proxy", Kind: "Node", Name: "kind-worker", Path: "/"}},
	}
	for _, c := range cases {
		r := workloadCase{method: http.MethodPost, path: c.path}.request()
		got, ok := Opens(r)
		if !ok {
			t.Errorf("%s opens nothing", c.path)
			continue
		}
		if got.Channel != c.want.Channel || got.Kind != c.want.Kind || got.Namespace != c.want.Namespace || got.Name != c.want.Name || got.Container != c.want.Container ||
			!slices.Equal(got.Command, c.want.Command) || got.Path != c.want.Path || got.Stdin != c.want.Stdin || got.TTY != c.want.TTY {
			t.Errorf("%s opens %+v, want %+v", c.path, got, c.want)
		}
	}
	for _, p := range []string{scalePath, "/api/v1/namespaces/shop/pods/web-1/ephemeralcontainers", "/api/v1/namespaces/shop/pods/web-1/log"} {
		if o, ok := Opens(Request{Method: http.MethodPost, Path: p}); ok {
			t.Errorf("%s opens %+v", p, o)
		}
	}
}

func TestAStaleRealmOpensNoWorkloadSession(t *testing.T) {
	for _, l := range []Level{Writes, Seamless} {
		for _, c := range workloadRequests {
			g, rec := gate(t, l, AllClasses...), &recorder{}
			g.StaleWhen(func() string { return "the pods watch broke" })
			g.Issue(c.request())
			refusedFor(t, sendIn(c.ctx(), g, rec, c.method, c.path, c.body, c.header...), Stale)
			if len(rec.seen) != 0 {
				t.Fatalf("%v %s: the server saw %v while stale", l, c.what, rec.seen)
			}
		}
	}
}

func TestForgetClearsEveryWarningPassedOnTheGate(t *testing.T) {
	g := gate(t, Seamless, AllClasses...)
	g.Forget()
	for _, c := range AllClasses {
		if g.Passed(c) {
			t.Errorf("%s is still passed", c)
		}
	}
	rec := &recorder{}
	refusedFor(t, send(g, rec, http.MethodDelete, "/api/v1/namespaces/shop", ""), NeedsClass)
}

func TestAnUpgradeIsSeenWhateverItsHeadersCase(t *testing.T) {
	for _, key := range []string{"Sec-WebSocket-Protocol", "sec-websocket-protocol", "Sec-WebSocket-Key", "UPGRADE", "connection"} {
		for _, l := range []Level{Writes, Seamless} {
			g, rec := gate(t, l, AllClasses...), &recorder{}
			req, _ := http.NewRequest(http.MethodGet, "https://h/apis/subresources.kubevirt.io/v1/namespaces/x/virtualmachineinstances/vm/console", nil)
			value := "plain.kubevirt.io"
			if strings.EqualFold(key, "connection") {
				value = "Upgrade"
			}
			req.Header[key] = []string{value}
			tr := gated{g: g, next: rec, base: "", known: true}
			if _, err := tr.RoundTrip(req); err == nil {
				t.Errorf("%v: a websocket marked by %q passed as a read", l, key)
			}
		}
	}
}
