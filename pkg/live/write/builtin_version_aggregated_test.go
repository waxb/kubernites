package write

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func TestAVersionOfABuiltInGroupAggregatedAfterConnectGetsNoDryRun(t *testing.T) {
	var aggregated atomic.Bool
	f := &fakeServer{answer: func(r *http.Request, body string) (int, string) {
		if r.URL.Path == "/apis/apiregistration.k8s.io/v1/apiservices" {
			items := `{"spec":{"group":"batch","version":"v1","service":null}}`
			if aggregated.Load() {
				items += `,{"spec":{"group":"batch","version":"v2","service":{"namespace":"ops","name":"batch-v2","port":443}}}`
			}
			return http.StatusOK, `{"items":[` + items + `]}`
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
		Request:     Request{Method: http.MethodPatch, Path: "/apis/batch/v2/namespaces/shop/runs/r1", Body: []byte(`{"metadata":{"labels":{"reviewed":"yes"}}}`)},
		ContentType: "application/merge-patch+json",
	}
	g.Preview(context.Background(), o)
	for _, line := range f.lines()[before:] {
		if strings.Contains(line, "/apis/batch/v2/") {
			t.Errorf("at WRITES a dry run went to batch/v2, whose APIService names a service since connect (an aggregated server that may carry out a dry run), because the gate lists APIServices again only for groups outside the built-in names: %s", line)
		}
	}
}
