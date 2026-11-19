package write

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/rest"
)

func refusedWhy(t *testing.T, a Answer) Why {
	t.Helper()
	var ref *Refusal
	if !errors.As(a.Err, &ref) {
		return 0
	}
	return ref.Why
}

func TestARefusedSendUsesUpItsTicket(t *testing.T) {
	f := &fakeServer{}
	g := bound(t, Writes, f)
	stale := "watch lost"
	g.StaleWhen(func() string { return stale })
	o := Order{Request: Request{Method: http.MethodDelete, Path: "/api/v1/namespaces/shop/pods/web-1"}}
	g.Issue(o.Request)
	if why := refusedWhy(t, g.Send(context.Background(), o)); why != Stale {
		t.Fatalf("a send while stale was refused for %v", why)
	}
	stale = ""
	if why := refusedWhy(t, g.Send(context.Background(), o)); why != NeedsConfirm {
		t.Fatalf("the ticket of a refused send let the same delete go later (%v): %v", why, f.lines())
	}
	if len(f.lines()) != 0 {
		t.Fatalf("the server saw %v", f.lines())
	}
}

func TestTwoConfirmsOfOneRequestReleaseItOnce(t *testing.T) {
	f := &fakeServer{}
	g := bound(t, Writes, f)
	o := Order{Request: Request{Method: http.MethodDelete, Path: "/api/v1/namespaces/shop/pods/web-1"}}
	g.Issue(o.Request)
	g.Issue(o.Request)
	if a := g.Send(context.Background(), o); !a.OK() {
		t.Fatalf("the confirmed send: %v", a.Text())
	}
	if why := refusedWhy(t, g.Send(context.Background(), o)); why != NeedsConfirm {
		t.Fatalf("a second CONFIRM of the same request let it go twice (%v): %v", why, f.lines())
	}
}

func TestADrainLeavesNoTicketBehind(t *testing.T) {
	f := &fakeServer{answer: func(r *http.Request, body string) (int, string) {
		if strings.HasSuffix(r.URL.Path, "/eviction") {
			return http.StatusCreated, `{"kind":"Status","status":"Success"}`
		}
		return http.StatusOK, `{}`
	}}
	g := bound(t, Writes, f)
	d, err := PlanDrain(fixtureNode(false), drainFixture())
	if err != nil {
		t.Fatal(err)
	}
	g.IssueDrain(d)
	if err := g.RunDrain(context.Background(), d, time.Millisecond, func(DrainReport) {}); err != nil {
		t.Fatal(err)
	}
	sent := len(f.lines())
	if d.Cordon != nil {
		if why := refusedWhy(t, g.Send(context.Background(), *d.Cordon)); why != NeedsConfirm {
			t.Errorf("the cordon went again with no CONFIRM (%v)", why)
		}
	}
	for _, o := range d.Evict {
		if why := refusedWhy(t, g.Send(context.Background(), o)); why != NeedsConfirm {
			t.Errorf("the eviction of %s went again with no CONFIRM (%v)", o.Target.Name, why)
		}
	}
	if extra := f.lines()[sent:]; len(extra) != 0 {
		t.Fatalf("after the drain the server saw %v", extra)
	}
}

func TestTheWriteGateFollowsNoRedirectToAnotherHost(t *testing.T) {
	other := &fakeServer{}
	elsewhere := httptest.NewServer(other)
	t.Cleanup(elsewhere.Close)
	f := &fakeServer{answer: func(r *http.Request, body string) (int, string) {
		return http.StatusTemporaryRedirect, ""
	}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.ServeHTTP(httptest.NewRecorder(), r)
		http.Redirect(w, r, elsewhere.URL+r.URL.Path, http.StatusTemporaryRedirect)
	}))
	t.Cleanup(srv.Close)
	g, err := New(Choice{Context: Context{Name: "c", Server: srv.URL, User: "u"}, Level: Seamless})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Client(&rest.Config{Host: srv.URL, BearerToken: "not-a-real-token"}); err != nil {
		t.Fatal(err)
	}
	o := Order{Request: Request{Method: http.MethodPatch, Path: "/apis/apps/v1/namespaces/shop/deployments/web", Body: []byte(`{"spec":{"replicas":3}}`)}, ContentType: "application/merge-patch+json"}
	if a := g.Send(context.Background(), o); a.Err == nil {
		t.Errorf("the send followed a redirect: %d", a.Code)
	}
	if v := g.Review(context.Background(), Access{Verb: "get", Resource: "pods"}); v.Err == nil {
		t.Errorf("the review followed a redirect")
	}
	if seen := other.lines(); len(seen) != 0 {
		t.Fatalf("another host saw %v", seen)
	}
}

func TestADryRunGoesWithoutATicketOnlyToAnAPIServerThatHonoursIt(t *testing.T) {
	f := &fakeServer{answer: func(r *http.Request, body string) (int, string) {
		if strings.HasSuffix(r.URL.Path, "/apiservices") {
			return http.StatusOK, `{"kind":"APIServiceList","items":[{"spec":{"group":"metrics.example.io","version":"v1","service":{"name":"m","namespace":"kube-system"}}},{"spec":{"group":"widgets.example.io","version":"v1"}}]}`
		}
		return http.StatusOK, `{}`
	}}
	g := bound(t, Writes, f)
	aggregated := Request{Method: http.MethodPatch, Path: "/apis/metrics.example.io/v1/namespaces/shop/gauges/g", Query: map[string][]string{"dryRun": {"All"}}, Body: []byte(`{"spec":{"a":1}}`)}
	real := Request{Method: http.MethodPatch, Path: aggregated.Path, Body: aggregated.Body}
	nothing := func(when string) {
		t.Helper()
		for _, r := range []Request{aggregated, real} {
			g.Issue(r)
			if why := refusedWhy(t, g.Send(context.Background(), Order{Request: r, ContentType: "application/merge-patch+json"})); why != NotAtLevel {
				t.Fatalf("%s, at WRITES a change to an aggregated API server was not refused outright even with a ticket (%v)", when, why)
			}
		}
	}
	nothing("before learning the groups")
	g.LearnLocalGroups(context.Background())
	for _, path := range []string{"/apis/apps/v1/namespaces/shop/deployments/web", "/api/v1/namespaces/shop/configmaps/c", "/apis/widgets.example.io/v1/namespaces/shop/widgets/w"} {
		dry := Request{Method: http.MethodPatch, Path: path, Query: map[string][]string{"dryRun": {"All"}}, Body: []byte(`{"metadata":{"labels":{"a":"b"}}}`)}
		if a := g.Send(context.Background(), Order{Request: dry, ContentType: "application/merge-patch+json"}); !a.OK() {
			t.Errorf("the dry run of %s needs a ticket: %v", path, a.Text())
		}
	}
	nothing("after learning the groups")
	for _, l := range f.lines() {
		if strings.Contains(l, "metrics.example.io") {
			t.Fatalf("the aggregated API server saw %s", l)
		}
	}
	s := bound(t, Seamless, f)
	s.LearnLocalGroups(context.Background())
	if a := s.Send(context.Background(), Order{Request: real, ContentType: "application/merge-patch+json"}); !a.OK() {
		t.Fatalf("at DANGEROUSLY SEAMLESS the change to an aggregated API server was not sent as kubectl sends it: %v", a.Text())
	}
}
