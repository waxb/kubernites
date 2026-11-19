package write

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"k8s.io/client-go/rest"
)

type recorder struct {
	mu   sync.Mutex
	seen []string
}

func (r *recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	body := ""
	if req.Body != nil {
		data, _ := io.ReadAll(req.Body)
		body = string(data)
	}
	line := req.Method + " " + req.URL.EscapedPath()
	if req.URL.RawQuery != "" {
		line += "?" + req.URL.RawQuery
	}
	if body != "" {
		line += " " + body
	}
	r.seen = append(r.seen, line)
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("{}")), Request: req}, nil
}

func choice(l Level, classes ...Class) Choice {
	return Choice{Context: Context{Name: "kind-kubernites", Server: "https://127.0.0.1:6443", User: "kind-kubernites"}, Level: l, Classes: classes}
}

func gate(t *testing.T, l Level, classes ...Class) *Gate {
	t.Helper()
	g, err := New(choice(l, classes...))
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func send(g *Gate, rec *recorder, method, target, body string, header ...string) error {
	tr := gated{g: g, next: rec, base: "", known: true}
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, "https://127.0.0.1:6443"+target, rd)
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

func refusedFor(t *testing.T, err error, why Why) *Refusal {
	t.Helper()
	var r *Refusal
	if !errors.As(err, &r) || !errors.Is(err, ErrRefused) {
		t.Fatalf("want a refusal (%v), got %v", why, err)
	}
	if r.Why != why {
		t.Fatalf("refused for %v, want %v: %v", r.Why, why, err)
	}
	return r
}

func TestTheGateIsNeverBuiltAtReadOnly(t *testing.T) {
	if _, err := New(choice(ReadOnly)); err == nil {
		t.Fatal("a READ ONLY choice built a write gate")
	}
	if _, err := New(Choice{Level: Writes}); err == nil {
		t.Fatal("a choice naming no context built a write gate")
	}
	if _, err := New(Choice{Level: Level(7), Context: Context{Name: "x", Server: "https://x"}}); err == nil {
		t.Fatal("an unknown level built a write gate")
	}
}

const scalePath = "/apis/apps/v1/namespaces/shop/deployments/web/scale"

func TestAtWritesAChangeNeedsItsTicketAndTheTicketIsUsedOnce(t *testing.T) {
	g, rec := gate(t, Writes), &recorder{}
	body := `{"spec":{"replicas":3}}`
	refusedFor(t, send(g, rec, http.MethodPatch, scalePath+"?fieldManager=kubernites", body), NeedsConfirm)
	if err := send(g, rec, http.MethodPatch, scalePath+"?dryRun=All&fieldManager=kubernites", body); err != nil {
		t.Fatalf("the dry run is the preview and needs no ticket: %v", err)
	}
	g.Issue(Request{Method: http.MethodPatch, Path: scalePath, Query: url.Values{"fieldManager": {"kubernites"}}, Body: []byte(body)})
	refusedFor(t, send(g, rec, http.MethodPatch, scalePath+"?fieldManager=kubernites", `{"spec":{"replicas":30}}`), NeedsConfirm)
	refusedFor(t, send(g, rec, http.MethodPatch, scalePath+"?fieldManager=other", body), NeedsConfirm)
	refusedFor(t, send(g, rec, http.MethodPut, scalePath+"?fieldManager=kubernites", body), NeedsConfirm)
	if err := send(g, rec, http.MethodPatch, scalePath+"?fieldManager=kubernites", body); err != nil {
		t.Fatalf("the confirmed change: %v", err)
	}
	refusedFor(t, send(g, rec, http.MethodPatch, scalePath+"?fieldManager=kubernites", body), NeedsConfirm)
	want := []string{
		"PATCH " + scalePath + "?dryRun=All&fieldManager=kubernites " + body,
		"PATCH " + scalePath + "?fieldManager=kubernites " + body,
	}
	if strings.Join(rec.seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("the server saw\n%s\nwant\n%s", strings.Join(rec.seen, "\n"), strings.Join(want, "\n"))
	}
}

func TestARevokedTicketPassesNothing(t *testing.T) {
	g, rec := gate(t, Writes), &recorder{}
	tk := g.Issue(Request{Method: http.MethodDelete, Path: "/api/v1/namespaces/shop/pods/web-1"})
	g.Revoke(tk)
	refusedFor(t, send(g, rec, http.MethodDelete, "/api/v1/namespaces/shop/pods/web-1", ""), NeedsConfirm)
	if len(rec.seen) != 0 {
		t.Fatalf("the server saw %v", rec.seen)
	}
}

func TestAtSeamlessAChangeGoesAtOnce(t *testing.T) {
	g, rec := gate(t, Seamless), &recorder{}
	if err := send(g, rec, http.MethodPatch, scalePath, `{"spec":{"replicas":3}}`); err != nil {
		t.Fatal(err)
	}
	if err := send(g, rec, http.MethodDelete, "/api/v1/namespaces/shop/pods/web-1", ""); err != nil {
		t.Fatal(err)
	}
	if len(rec.seen) != 2 {
		t.Fatalf("the server saw %v", rec.seen)
	}
}

func TestReadsAndReviewsPassAtEveryWritableLevel(t *testing.T) {
	for _, l := range []Level{Writes, Seamless} {
		g, rec := gate(t, l), &recorder{}
		g.StaleWhen(func() string { return "the pods watch broke" })
		for _, p := range []string{"/api/v1/pods", "/api/v1/watch/namespaces/shop/pods?watch=true", "/apis/apps/v1/namespaces/shop/deployments/web/scale", "/version", "/openapi/v3", "/api/v1/namespaces/shop/pods/web-1/log"} {
			if err := send(g, rec, http.MethodGet, p, ""); err != nil {
				t.Errorf("%v: GET %s: %v", l, p, err)
			}
		}
		for _, p := range []string{accessReview, rulesReview, selfReview} {
			if err := send(g, rec, http.MethodPost, p, `{}`); err != nil {
				t.Errorf("%v: the review %s while stale: %v", l, p, err)
			}
		}
	}
}

func TestAStaleRealmSendsNoChangeAndNoPreview(t *testing.T) {
	for _, l := range []Level{Writes, Seamless} {
		g, rec := gate(t, l, AllClasses...), &recorder{}
		stale := "the pods watch broke"
		g.StaleWhen(func() string { return stale })
		body := `{"spec":{"replicas":3}}`
		g.Issue(Request{Method: http.MethodPatch, Path: scalePath, Body: []byte(body)})
		r := refusedFor(t, send(g, rec, http.MethodPatch, scalePath, body), Stale)
		if !strings.Contains(r.Error(), stale) {
			t.Errorf("the refusal does not say why: %v", r)
		}
		refusedFor(t, send(g, rec, http.MethodPatch, scalePath+"?dryRun=All", body), Stale)
		refusedFor(t, send(g, rec, http.MethodPost, "/api/v1/namespaces/shop/pods/web-1/eviction", `{}`), Stale)
		if len(rec.seen) != 0 {
			t.Fatalf("%v: the server saw %v", l, rec.seen)
		}
		stale = ""
		if l == Writes {
			refusedFor(t, send(g, rec, http.MethodPatch, scalePath, body), NeedsConfirm)
			g.Issue(Request{Method: http.MethodPatch, Path: scalePath, Body: []byte(body)})
		}
		if err := send(g, rec, http.MethodPatch, scalePath, body); err != nil {
			t.Fatalf("%v: fresh again, the change confirmed again: %v", l, err)
		}
	}
}

func TestEachClassNeedsItsWarningEvenForAPreview(t *testing.T) {
	cases := []struct {
		class  Class
		method string
		path   string
		body   string
		header []string
	}{
		{Secrets, http.MethodGet, "/api/v1/namespaces/shop/secrets", "", nil},
		{Secrets, http.MethodGet, "/api/v1/watch/secrets", "", nil},
		{Secrets, http.MethodPost, "/api/v1/namespaces/shop/secrets", `{"kind":"Secret"}`, nil},
		{Secrets, http.MethodPatch, "/api/v1/namespaces/shop/secrets/key", `{}`, nil},
		{Secrets, http.MethodDelete, "/api/v1/namespaces/shop/secrets/key", "", nil},
		{Destroy, http.MethodDelete, "/api/v1/namespaces/shop", "", nil},
		{Destroy, http.MethodDelete, "/api/v1/nodes/kind-worker", "", nil},
		{Destroy, http.MethodDelete, "/api/v1/persistentvolumes/pv-1", "", nil},
		{Destroy, http.MethodDelete, "/apis/apiextensions.k8s.io/v1/customresourcedefinitions/widgets.example.com", "", nil},
		{Destroy, http.MethodDelete, "/apis/rbac.authorization.k8s.io/v1/namespaces/shop/roles/reader", "", nil},
		{Destroy, http.MethodDelete, "/apis/rbac.authorization.k8s.io/v1/clusterrolebindings/admins", "", nil},
		{Destroy, http.MethodDelete, "/apis/admissionregistration.k8s.io/v1/validatingwebhookconfigurations/guard", "", nil},
		{Destroy, http.MethodDelete, "/api/v1/namespaces/shop/pods", "", nil},
		{Destroy, http.MethodDelete, "/api/v1/namespaces/shop/pods/web-1?labelSelector=app%3Dweb", "", nil},
		{Destroy, http.MethodDelete, "/api/v1/namespaces/shop/pods/web-1?gracePeriodSeconds=0", "", nil},
		{Destroy, http.MethodDelete, "/api/v1/namespaces/shop/pods/web-1", `{"kind":"DeleteOptions","apiVersion":"v1","gracePeriodSeconds":0}`, nil},
		{Destroy, http.MethodDelete, "/api/v1/namespaces/shop/pods/web-1", `not json`, nil},
		{Destroy, http.MethodDelete, "/api/v1/namespaces/shop/pods/web-1", `{"ignoreStoreReadErrorWithClusterBreakingPotential":true}`, nil},
		{Destroy, http.MethodPost, "/api/v1/namespaces/shop/pods/web-1/eviction", `{"deleteOptions":{"gracePeriodSeconds":0}}`, nil},
		{Destroy, http.MethodPut, "/api/v1/namespaces/shop/finalize", `{}`, nil},
	}
	for _, c := range cases {
		for _, l := range []Level{Writes, Seamless} {
			g, rec := gate(t, l), &recorder{}
			query, body := "", c.body
			if c.method != http.MethodGet {
				query = "?dryRun=All"
				if strings.Contains(c.path, "?") {
					query = "&dryRun=All"
				}
			}
			if c.method == http.MethodDelete && strings.HasPrefix(body, "{") {
				body = `{"dryRun":["All"],` + strings.TrimPrefix(body, "{")
			}
			dry := c.method != http.MethodGet && !(c.method == http.MethodDelete && body != "" && !strings.HasPrefix(body, "{"))
			c.body = body
			r := refusedFor(t, send(g, rec, c.method, c.path+query, c.body, c.header...), NeedsClass)
			if r.Class != c.class {
				t.Errorf("%v %s %s: needs %v, want %v", l, c.method, c.path, r.Class, c.class)
			}
			if len(rec.seen) != 0 {
				t.Errorf("%v %s %s: the server saw %v", l, c.method, c.path, rec.seen)
			}
			g.Pass(c.class)
			err := send(g, rec, c.method, c.path+query, c.body, c.header...)
			switch {
			case c.method == http.MethodGet && c.class == Secrets:
				if err != nil {
					t.Errorf("%v: reading Secrets once warned: %v", l, err)
				}
			case l == Writes && !dry:
				refusedFor(t, err, NeedsConfirm)
			case err != nil:
				t.Errorf("%v %s %s once warned: %v", l, c.method, c.path, err)
			}
		}
	}
}

func TestADeleteWithABodyIsADryRunOnlyIfItsBodySaysSo(t *testing.T) {
	g, rec := gate(t, Writes, AllClasses...), &recorder{}
	path := "/api/v1/namespaces/shop/pods/web-1"
	for _, body := range []string{
		`{"preconditions":{"uid":"u","resourceVersion":"7"}}`,
		`{"DryRun":["All"]}`,
		`{"dryrun":["All"]}`,
		`{"dryRun":["All"],"dryRun":[]}`,
		`{"dryRun":["All","None"]}`,
		`{"dryRun":"All"}`,
		`{"dryRun":["All"]} {}`,
		` `,
		`not json`,
	} {
		refusedFor(t, send(g, rec, http.MethodDelete, path+"?dryRun=All", body), NeedsConfirm)
	}
	for _, body := range []string{`{"dryRun":["All"]}`, `{"preconditions":{"uid":"u"},"dryRun":["All"]}`, `{"dry\u0052un":["All"]}`} {
		if err := send(g, rec, http.MethodDelete, path, body); err != nil {
			t.Errorf("the dry run %s: %v", body, err)
		}
	}
	if err := send(g, rec, http.MethodDelete, path+"?dryRun=All", ""); err != nil {
		t.Errorf("a DELETE without a body takes its dry run from the query: %v", err)
	}
	if len(rec.seen) != 4 {
		t.Fatalf("the server saw %v", rec.seen)
	}
}

func TestAPlainDeleteOfOneThingNeedsNoClass(t *testing.T) {
	g, rec := gate(t, Seamless), &recorder{}
	for _, body := range []string{"", `{"propagationPolicy":"Background"}`, `{"gracePeriodSeconds":1}`, `{"preconditions":{"uid":"u","resourceVersion":"7"}}`} {
		if err := send(g, rec, http.MethodDelete, "/api/v1/namespaces/shop/pods/web-1", body); err != nil {
			t.Errorf("delete with %q: %v", body, err)
		}
	}
	if err := send(g, rec, http.MethodPost, "/api/v1/namespaces/shop/pods/web-1/eviction", `{"apiVersion":"policy/v1","kind":"Eviction"}`); err != nil {
		t.Errorf("an eviction: %v", err)
	}
	if err := send(g, rec, http.MethodGet, "/api/v1/namespaces/shop/pods/web-1/ephemeralcontainers", ""); err != nil {
		t.Errorf("reading ephemeral containers: %v", err)
	}
}

func TestADeclaredClassIsRequiredToo(t *testing.T) {
	g := gate(t, Seamless)
	ctx := WithClasses(context.Background(), Destroy)
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, "https://127.0.0.1:6443/api/v1/namespaces/shop/pods/web-1", nil)
	_, err := gated{g: g, next: &recorder{}, known: true}.RoundTrip(req)
	if r := refusedFor(t, err, NeedsClass); r.Class != Destroy {
		t.Fatalf("declared Destroy, refused for %v", r.Class)
	}
}

func TestNothingOutsideTheAPIOrOddPassesAtAnyLevel(t *testing.T) {
	odd := []struct{ method, path string }{
		{http.MethodGet, "/metrics"}, {http.MethodGet, "/logs/kube-apiserver.log"}, {http.MethodGet, "/debug/pprof/"}, {http.MethodGet, "/"},
		{http.MethodPost, "/version"}, {http.MethodHead, "/api/v1/pods"}, {http.MethodOptions, "/api/v1/pods"}, {http.MethodConnect, "/api/v1/pods"},
		{http.MethodTrace, "/api/v1/pods"}, {"get", "/api/v1/pods"}, {"LIST", "/api/v1/pods"}, {http.MethodGet, "/apis/../api/v1/secrets"},
		{http.MethodDelete, "/api/v1//namespaces/shop/pods/web-1"}, {http.MethodGet, "/api/v1/namespaces/shop/pods/web%2D1"},
	}
	for _, l := range []Level{Writes, Seamless} {
		g, rec := gate(t, l, AllClasses...), &recorder{}
		for _, o := range odd {
			refusedFor(t, send(g, rec, o.method, o.path, ""), NotAtLevel)
		}
		if len(rec.seen) != 0 {
			t.Fatalf("%v: the server saw %v", l, rec.seen)
		}
	}
}

func TestACompressedBodyIsNeverSent(t *testing.T) {
	g, rec := gate(t, Seamless, AllClasses...), &recorder{}
	refusedFor(t, send(g, rec, http.MethodPatch, scalePath, "x", "Content-Encoding", "gzip"), NotAtLevel)
}

func TestAPathOutsideTheServersBaseIsRefused(t *testing.T) {
	g, rec := gate(t, Seamless, AllClasses...), &recorder{}
	tr := gated{g: g, next: rec, base: "/k8s/clusters/c-1", known: true}
	req, _ := http.NewRequest(http.MethodDelete, "https://rancher.example/api/v1/namespaces/shop/pods/web-1", nil)
	if _, err := tr.RoundTrip(req); !errors.Is(err, ErrRefused) {
		t.Fatalf("outside the base: %v", err)
	}
	req, _ = http.NewRequest(http.MethodDelete, "https://rancher.example/k8s/clusters/c-1/api/v1/namespaces/shop/pods/web-1", nil)
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatalf("inside the base: %v", err)
	}
	g2 := gate(t, Writes)
	g2.Issue(Request{Method: http.MethodDelete, Path: "/api/v1/namespaces/shop/pods/web-1"})
	tr2 := gated{g: g2, next: rec, base: "/k8s/clusters/c-1", known: true}
	req, _ = http.NewRequest(http.MethodDelete, "https://rancher.example/k8s/clusters/c-1/api/v1/namespaces/shop/pods/web-1", nil)
	if _, err := tr2.RoundTrip(req); err != nil {
		t.Fatalf("a ticket names the API path without the server's base: %v", err)
	}
}

func TestWrapConfigPutsTheGateOnAClientGoTransport(t *testing.T) {
	var seen []string
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path+" "+r.UserAgent())
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{}`)
	}))
	defer srv.Close()
	g, err := New(Choice{Context: Context{Name: "kind-kubernites", Server: srv.URL + "/base", User: "kind-kubernites"}, Level: Writes})
	if err != nil {
		t.Fatal(err)
	}
	cfg := g.WrapConfig(&rest.Config{Host: srv.URL + "/base", UserAgent: "kubectl/v1.37.0"})
	if cfg.UserAgent != "kubectl/v1.37.0" {
		t.Errorf("WrapConfig changed the user agent to %q", cfg.UserAgent)
	}
	client, err := rest.HTTPClientFor(cfg)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/base/api/v1/namespaces/shop/pods/web-1", nil)
	if _, err := client.Do(req); !errors.Is(err, ErrRefused) {
		t.Fatalf("an unconfirmed delete through the wrapped config: %v", err)
	}
	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/base/api/v1/namespaces/shop/pods", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	own, err := g.Client(&rest.Config{Host: srv.URL + "/base"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Client(&rest.Config{Host: srv.URL}); err == nil {
		t.Fatal("the gate was bound to a second connection")
	}
	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/base/api/v1/nodes", nil)
	resp, err = own.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	want := []string{"GET /base/api/v1/namespaces/shop/pods kubectl/v1.37.0", "GET /base/api/v1/nodes " + UserAgent(Writes)}
	if strings.Join(seen, "\n") != strings.Join(want, "\n") {
		t.Fatalf("the server saw %q, want %q", seen, want)
	}
}

func TestTheObserverSeesWhatPassedAndHowItWasAnswered(t *testing.T) {
	g, rec := gate(t, Seamless), &recorder{}
	var got []Passage
	g.Observe(func(p Passage) { got = append(got, p) })
	send(g, rec, http.MethodGet, "/api/v1/pods", "")
	send(g, rec, http.MethodPatch, scalePath+"?dryRun=All", `{}`)
	send(g, rec, http.MethodPatch, scalePath, `{}`)
	send(g, rec, http.MethodDelete, "/api/v1/namespaces/shop", "")
	if len(got) != 3 || got[0].Write || got[1].Write || !got[1].DryRun || !got[2].Write || got[2].Code != 200 {
		t.Fatalf("observed %+v", got)
	}
}

func TestClassesPassedAndRemembered(t *testing.T) {
	g := gate(t, Writes, Secrets, Class("nonsense"))
	if !g.Passed(Secrets) || g.Passed(Destroy) || len(g.Classes()) != 1 {
		t.Fatalf("classes %v", g.Classes())
	}
	g.Pass(Destroy)
	g.Pass(Class("nonsense"))
	if got := g.Classes(); len(got) != 2 || got[0] != Secrets || got[1] != Destroy {
		t.Fatalf("classes %v", got)
	}
	if g.Level() != Writes || g.Context().Name != "kind-kubernites" {
		t.Fatalf("level %v context %+v", g.Level(), g.Context())
	}
}
