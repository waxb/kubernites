package write

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAGroupAggregatedAfterConnectGetsNoDryRunAtWrites(t *testing.T) {
	var aggregated atomic.Bool
	f := &fakeServer{answer: func(r *http.Request, body string) (int, string) {
		if r.URL.Path == "/apis/apiregistration.k8s.io/v1/apiservices" {
			service := "null"
			if aggregated.Load() {
				service = `{"namespace":"kyverno","name":"reports-server","port":443}`
			}
			return http.StatusOK, `{"items":[{"spec":{"group":"wgpolicyk8s.io","version":"v1alpha2","service":` + service + `}}]}`
		}
		return http.StatusOK, body
	}}
	g := bound(t, Writes, f)
	if err := g.LearnLocalGroups(context.Background()); err != nil {
		t.Fatal(err)
	}
	aggregated.Store(true)
	before := len(f.lines())
	o := Order{
		Request:     Request{Method: http.MethodPatch, Path: "/apis/wgpolicyk8s.io/v1alpha2/namespaces/shop/policyreports/r1", Body: []byte(`{"metadata":{"labels":{"reviewed":"yes"}}}`)},
		ContentType: "application/merge-patch+json",
	}
	g.Preview(context.Background(), o)
	for _, line := range f.lines()[before:] {
		if strings.Contains(line, "/apis/wgpolicyk8s.io/") {
			t.Errorf("at WRITES the preview went to a group whose APIService now names a service (an aggregated server that may carry out a dry run): %s", line)
		}
	}
}

func TestAtWritesAGroupOutsideTheBuiltInsIsPreviewedOnlyWhileItsServingIsConfirmed(t *testing.T) {
	var state atomic.Int32
	f := &fakeServer{answer: func(r *http.Request, body string) (int, string) {
		if r.URL.Path == "/apis/apiregistration.k8s.io/v1/apiservices" {
			switch state.Load() {
			case 1:
				return http.StatusOK, `{"items":[{"spec":{"group":"lair.example","version":"v1","service":{"namespace":"x","name":"agg","port":443}}}]}`
			case 2:
				return http.StatusForbidden, `{"kind":"Status","code":403,"message":"apiservices is forbidden"}`
			case 3:
				return http.StatusOK, `{"items":[]}`
			}
			return http.StatusOK, `{"items":[{"spec":{"group":"lair.example","version":"v1","service":null}}]}`
		}
		return http.StatusOK, `{"metadata":{"labels":{"a":"b"}}}`
	}}
	g := bound(t, Writes, f)
	if err := g.LearnLocalGroups(context.Background()); err != nil {
		t.Fatal(err)
	}
	o := Order{Request: Request{Method: http.MethodPatch, Path: "/apis/lair.example/v1/namespaces/shop/dragons/d1", Body: []byte(`{"metadata":{"labels":{"a":"b"}}}`)}, ContentType: "application/merge-patch+json"}
	sentTo := func(from int) []string {
		var out []string
		for _, l := range f.lines()[from:] {
			if strings.Contains(l, "/apis/lair.example/") {
				out = append(out, l)
			}
		}
		return out
	}
	before := len(f.lines())
	if ans, _ := g.Preview(context.Background(), o); !ans.OK() || len(sentTo(before)) != 1 {
		t.Fatalf("a group the API server serves itself lost its preview: %+v %v", ans, sentTo(before))
	}
	for s, why := range map[int32]string{1: "now names a service", 2: "can no longer be listed", 3: "is gone"} {
		state.Store(s)
		before := len(f.lines())
		ans, _ := g.Preview(context.Background(), o)
		send := g.SendConfirmed(context.Background(), o)
		if sent := sentTo(before); len(sent) != 0 || ans.OK() || send.OK() {
			t.Errorf("at WRITES a group whose APIService %s still reached its server: %v (preview %+v, send %+v)", why, sent, ans, send)
		}
		var refusal *Refusal
		if !errors.As(send.Err, &refusal) || refusal.Why != NotAtLevel {
			t.Errorf("a group whose APIService %s is not refused with its reason: %+v", why, send)
		}
	}
	state.Store(0)
	before = len(f.lines())
	if ans, _ := g.Preview(context.Background(), o); !ans.OK() || len(sentTo(before)) != 1 {
		t.Errorf("a group served by the API server itself again is not previewed: %+v", ans)
	}
}
