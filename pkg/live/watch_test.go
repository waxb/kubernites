package live_test

import (
	"context"
	"net/http"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/sim/store"
)

const podsPath = "/api/v1/pods"

type watchClock struct {
	mu      sync.Mutex
	at      time.Time
	metrics chan struct{}
	sleeps  int
}

func (c *watchClock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.at
}

func (c *watchClock) move(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = c.at.Add(d)
}

func (c *watchClock) sleep(ctx context.Context, d time.Duration) bool {
	if d == live.MetricsEvery {
		select {
		case <-ctx.Done():
			return false
		case <-c.metrics:
			return true
		}
	}
	c.mu.Lock()
	c.sleeps++
	c.mu.Unlock()
	select {
	case <-ctx.Done():
		return false
	case <-time.After(2 * time.Millisecond):
		return true
	}
}

func (c *watchClock) slept() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sleeps
}

type watched struct {
	r     *live.Realm
	rec   *livetest.Record
	feed  *livetest.Feed
	clock *watchClock
	mu    sync.Mutex
	seen  []store.Event
}

func (w *watched) events() []store.Event {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]store.Event(nil), w.seen...)
}

func watchRealm(t *testing.T, s live.Snapshot, o livetest.Options) *watched {
	t.Helper()
	feed := livetest.NewFeed()
	o.Feed = feed
	if o.User == "" {
		o.User = "kind-admin@example.com"
	}
	srv, rec := livetest.ServeAs(t, s, o)
	path := livetest.Kubeconfig(t, srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r, err := live.Connect(ctx, []string{path}, livetest.Context, time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatal(err)
	}
	w := &watched{r: r, rec: rec, feed: feed, clock: &watchClock{at: time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC), metrics: make(chan struct{})}}
	live.SetWatchTiming(r, w.clock.now, w.clock.sleep)
	r.Watch(func(e store.Event) {
		w.mu.Lock()
		w.seen = append(w.seen, e)
		w.mu.Unlock()
	})
	r.StartWatching()
	t.Cleanup(r.Stop)
	return w
}

func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("never: %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func kindFixture(t *testing.T) live.Snapshot {
	return livetest.Snapshot(t, "testdata/kind-snapshot.json")
}

func watchPaths(s live.Snapshot) []string {
	out := []string{}
	for path := range livetest.Lists(s) {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func newPod(name, uid, rv string) *corev1.Pod {
	return &corev1.Pod{TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "shop", UID: types.UID("watch-" + metav1Name(uid)), ResourceVersion: rv, CreationTimestamp: metav1.NewTime(time.Date(2026, 11, 15, 12, 1, 0, 0, time.UTC))},
		Status:     corev1.PodStatus{Phase: corev1.PodPending}}
}

func metav1Name(s string) string { return strings.ReplaceAll(s, " ", "-") }

func TestEveryListedKindIsWatchedFromItsListsVersionThroughGETsAlone(t *testing.T) {
	s := kindFixture(t)
	w := watchRealm(t, s, livetest.Options{})
	marks := w.r.Snapshot().Lists
	if _, ok := marks["poddisruptionbudgets"]; !ok {
		t.Fatalf("PodDisruptionBudgets were not read: %v", marks)
	}
	eventually(t, "a watch on every listed kind", func() bool { return len(w.feed.Watches()) >= len(marks) })
	got := map[string]livetest.Watch{}
	for _, wt := range w.feed.Watches() {
		got[wt.Path] = wt
	}
	for _, path := range watchPaths(s) {
		wt, ok := got[path]
		if !ok {
			t.Errorf("%s is never watched", path)
			continue
		}
		if wt.Version != livetest.ListVersion || wt.Query["allowWatchBookmarks"] != "true" || wt.Query["timeoutSeconds"] == "" {
			t.Errorf("%s watched with %+v", path, wt.Query)
		}
	}
	if writes := w.rec.Writes(); len(writes) != 0 {
		t.Fatalf("the server saw writes: %v", writes)
	}
	for _, seen := range w.rec.Seen() {
		if !strings.HasPrefix(seen, http.MethodGet+" ") && seen != livetest.SelfReview {
			t.Errorf("a request other than a read left Kubernites: %s", seen)
		}
	}
}

func TestWatchEventsReachTheMirrorAndTheSession(t *testing.T) {
	w := watchRealm(t, kindFixture(t), livetest.Options{})
	eventually(t, "the pods watch", func() bool { return len(w.feed.WatchesOf(podsPath)) == 1 })
	p := newPod("newcomer", "1", "1001")
	w.feed.Send(podsPath, "ADDED", p)
	eventually(t, "the new hut in the mirror", func() bool {
		return live.MirrorHas(w.r, store.Key{Kind: store.KindPod, Namespace: "shop", Name: "newcomer"})
	})
	w.r.Pump(live.PumpEvery)
	ev := w.events()
	if len(ev) != 1 || ev[0].Type != store.Added || ev[0].Object.GetName() != "newcomer" {
		t.Fatalf("the session heard %+v", ev)
	}
	p.ResourceVersion, p.Status.Phase = "1002", corev1.PodRunning
	w.feed.Send(podsPath, "MODIFIED", p)
	p.ResourceVersion = "1003"
	w.feed.Send(podsPath, "DELETED", p)
	w.feed.Send(podsPath, "BOOKMARK", map[string]any{"kind": "Pod", "apiVersion": "v1", "metadata": map[string]any{"resourceVersion": "2000"}})
	w.feed.Close(podsPath)
	eventually(t, "a second watch after the stream closed", func() bool { return len(w.feed.WatchesOf(podsPath)) == 2 })
	if v := w.feed.WatchesOf(podsPath)[1].Version; v != "2000" {
		t.Errorf("watched again from %q, not from the bookmark's version", v)
	}
	w.r.Pump(live.PumpEvery)
	ev = w.events()
	if len(ev) != 3 || ev[1].Type != store.Modified || ev[2].Type != store.Deleted {
		t.Fatalf("the session heard %+v", ev)
	}
	for _, pod := range w.r.Snapshot().Pods {
		if pod.Name == "newcomer" {
			t.Error("the deleted hut is still in the snapshot")
		}
	}
}

func listsOf(rec *livetest.Record, path string) int {
	n := 0
	for _, s := range rec.Reads() {
		if s == http.MethodGet+" "+path {
			n++
		}
	}
	return n
}

func readsOf(rec *livetest.Record, path string) []string {
	var out []string
	for _, s := range rec.Reads() {
		if s == http.MethodGet+" "+path || s == http.MethodGet+" "+path+livetest.WatchMark {
			out = append(out, s)
		}
	}
	return out
}

func TestAGoneWatchClearsItsKindAndListsItAgain(t *testing.T) {
	s := kindFixture(t)
	w := watchRealm(t, s, livetest.Options{})
	eventually(t, "the pods watch", func() bool { return len(w.feed.WatchesOf(podsPath)) == 1 })
	before := len(readsOf(w.rec, podsPath))
	w.feed.Gone(podsPath, 1)
	w.feed.Send(podsPath, "ERROR", map[string]any{"kind": "Status", "apiVersion": "v1", "status": "Failure", "reason": "Expired", "code": 410, "message": "too old resource version: 1000 (2000)"})
	eventually(t, "the pods watched from the new list", func() bool { return len(w.feed.WatchesOf(podsPath)) >= 3 })
	list, watch := http.MethodGet+" "+podsPath, http.MethodGet+" "+podsPath+livetest.WatchMark
	if got, want := readsOf(w.rec, podsPath)[before:], []string{list, watch, list, watch}; !slices.Equal(got, want) {
		t.Errorf("after the 410 the pods were read as %q, want %q: listed again, a watch answered 410, listed again, watched", got, want)
	}
	last := w.feed.WatchesOf(podsPath)
	if v := last[len(last)-1].Version; v != livetest.ListVersion {
		t.Errorf("watched from %q after the relist", v)
	}
	if got := len(w.r.Snapshot().Pods); got != len(s.Pods) {
		t.Errorf("after the relist the realm holds %d pods of %d", got, len(s.Pods))
	}
}

func TestADroppedStreamIsWatchedAgainFromItsLastVersionAfterABackoff(t *testing.T) {
	w := watchRealm(t, kindFixture(t), livetest.Options{})
	eventually(t, "the pods watch", func() bool { return len(w.feed.WatchesOf(podsPath)) == 1 })
	w.feed.Break(podsPath, 1)
	w.feed.Send(podsPath, "ADDED", newPod("first", "first", "1500"))
	eventually(t, "the hut", func() bool { return len(w.r.Snapshot().Pods) == len(kindFixture(t).Pods)+1 })
	w.feed.Close(podsPath)
	eventually(t, "the broken watch and the one after it", func() bool { return len(w.feed.WatchesOf(podsPath)) >= 3 })
	for _, wt := range w.feed.WatchesOf(podsPath)[1:] {
		if wt.Version != "1500" {
			t.Errorf("watched again from %q, not from the last version", wt.Version)
		}
	}
	if w.clock.slept() == 0 {
		t.Error("the broken stream was watched again with no backoff")
	}
	if listsOf(w.rec, podsPath) != 1 {
		t.Errorf("a dropped stream listed again: %d lists", listsOf(w.rec, podsPath))
	}
}

func TestAWatchThatCannotRecoverSaysSinceWhenTheRealmIsStale(t *testing.T) {
	w := watchRealm(t, kindFixture(t), livetest.Options{})
	eventually(t, "the pods watch", func() bool { return len(w.feed.WatchesOf(podsPath)) == 1 })
	if f := w.r.Freshness(); !f.Watching || !f.StaleSince.IsZero() {
		t.Fatalf("freshness before trouble %+v", f)
	}
	broke := w.clock.now()
	w.feed.Fail(podsPath, true)
	w.feed.Close(podsPath)
	eventually(t, "failed watches", func() bool { return len(w.feed.WatchesOf(podsPath)) >= 3 })
	if f := w.r.Freshness(); !f.StaleSince.IsZero() {
		t.Errorf("stale at once: %+v", f)
	}
	w.clock.move(live.StaleAfter + time.Second)
	f := w.r.Freshness()
	if !f.StaleSince.Equal(broke) || !strings.Contains(f.Why, "pods") || !strings.Contains(f.Why, "watch cache") {
		t.Fatalf("freshness while the watch fails %+v, broke at %v", f, broke)
	}
	w.feed.Fail(podsPath, false)
	eventually(t, "the realm fresh again", func() bool { return w.r.Freshness().StaleSince.IsZero() })
}

func TestMetricsAreReadAgainOnAFixedPeriod(t *testing.T) {
	w := watchRealm(t, kindFixture(t), livetest.Options{Metrics: true})
	reads := func() int { return listsOf(w.rec, "/apis/metrics.k8s.io/v1beta1/pods") }
	at := reads()
	w.clock.metrics <- struct{}{}
	eventually(t, "a second metrics read", func() bool { return reads() == at+1 })
	if !w.r.Metrics().Augury {
		t.Error("metrics read again are not available")
	}
}

func TestAConnectedRealmTakesNoReplayedEvents(t *testing.T) {
	w := watchRealm(t, kindFixture(t), livetest.Options{})
	if err := w.r.Replay("pods", strings.NewReader(string(live.WatchEvent("ADDED", newPod("fake", "fake", "1"))))); err == nil {
		t.Fatal("a connected realm took a replayed event")
	}
	for _, p := range w.r.Snapshot().Pods {
		if p.Name == "fake" {
			t.Fatal("the replayed hut is in the mirror")
		}
	}
}
