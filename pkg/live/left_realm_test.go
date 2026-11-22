package live_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestARealmLeftSendsNoChange(t *testing.T) {
	feed := livetest.NewFeed()
	srv, rec := livetest.ServeAs(t, kindFixture(t), livetest.Options{User: "kind-admin@example.com", Feed: feed})
	path := livetest.Kubeconfig(t, srv.URL)
	listed, err := live.ListContexts([]string{path})
	if err != nil || len(listed) != 1 {
		t.Fatalf("contexts %v %v", listed, err)
	}
	g, err := write.New(write.Choice{Context: listed[0].Remembered(), Level: write.Seamless})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r, err := live.ConnectGated(ctx, []string{path}, livetest.Context, g, time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatal(err)
	}
	r.StartWatching()
	eventually(t, "the pods watch", func() bool { return len(feed.WatchesOf(podsPath)) >= 1 })
	r.Stop()
	evict := write.Order{Request: write.Request{Method: http.MethodPost, Path: "/api/v1/namespaces/default/pods/web-1/eviction", Body: []byte(`{"apiVersion":"policy/v1","kind":"Eviction","metadata":{"name":"web-1","namespace":"default"}}`)}, ContentType: "application/json"}
	g.Send(context.Background(), evict)
	for _, s := range rec.Seen() {
		if strings.HasPrefix(s, "POST ") && strings.HasSuffix(s, "/eviction") {
			t.Errorf("the realm was left (Stop: no watch keeps its picture, the Stone is closed) and the gate still let a change go: %s; freshness %+v", s, r.Freshness())
		}
	}
}
