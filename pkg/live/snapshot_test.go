package live_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

func connect(t *testing.T, kubeconfig string, progress func(live.Progress)) *live.Realm {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r, err := live.Connect(ctx, []string{kubeconfig}, livetest.Context, time.Now(), progress)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestTheSnapshotTellsItsRealProgress(t *testing.T) {
	s := livetest.Crowded(1, 3, 400)
	srv, _ := livetest.Serve(t, s)
	var steps []live.Progress
	connect(t, livetest.Kubeconfig(t, srv.URL), func(p live.Progress) { steps = append(steps, p) })
	if len(steps) == 0 || steps[0].List != "version" || steps[0].Done != 1 {
		t.Fatalf("first step %+v", steps)
	}
	pods := []int{}
	lists := []string{}
	for i, p := range steps {
		if i > 0 && (p.Done < steps[i-1].Done || p.Pods < steps[i-1].Pods || p.Nodes < steps[i-1].Nodes) {
			t.Errorf("progress went back: %+v after %+v", p, steps[i-1])
		}
		if p.List == "pods" && (len(pods) == 0 || pods[len(pods)-1] != p.Pods) {
			pods = append(pods, p.Pods)
		}
		if len(lists) == 0 || lists[len(lists)-1] != p.List {
			lists = append(lists, p.List)
		}
	}
	if !slices.Equal(pods, []int{500, 1000, 1200}) {
		t.Errorf("the huts arrived as %v, want page by page 500, 1000, 1200", pods)
	}
	last := steps[len(steps)-1]
	if last.Done != last.Total || last.Nodes != 3 || last.Pods != 1200 {
		t.Errorf("last step %+v", last)
	}
	if !slices.Equal(lists[:4], []string{"version", "nodes", "namespaces", "pods"}) {
		t.Errorf("lists read in order %v", lists)
	}
}

func TestTheClusterSaysWhoYouAre(t *testing.T) {
	srv, rec := livetest.Serve(t, livetest.Snapshot(t, "testdata/kind-snapshot.json"))
	path := livetest.Kubeconfig(t, srv.URL)
	who, err := live.WhoAmI(context.Background(), []string{path}, livetest.Context)
	if err != nil || who.Username != "kind-admin@example.com" {
		t.Fatalf("who %+v, %v", who, err)
	}
	if seen := rec.Seen(); len(seen) != 1 || seen[0] != livetest.SelfReview {
		t.Errorf("asking who you are sent %v", seen)
	}
	r := connect(t, path, nil)
	if r.Snapshot().Identity.Username != "kind-admin@example.com" {
		t.Errorf("the snapshot holds %+v", r.Snapshot().Identity)
	}
	if w := rec.Writes(); len(w) != 0 {
		t.Errorf("writes %v", w)
	}
}

func TestARestrictedUserReadsItsNamespaceAndNamesWhatItCannotSee(t *testing.T) {
	s := livetest.Snapshot(t, "testdata/kind-snapshot.json")
	srv, rec := livetest.ServeAs(t, s, livetest.Options{User: "dev@example.com", Forbid: func(path string) bool {
		return path != "/version" && !strings.Contains(path, "/namespaces/shop/")
	}})
	r := connect(t, livetest.KubeconfigIn(t, srv.URL, "shop"), nil)
	snap := r.Snapshot()
	if snap.Identity.Username != "dev@example.com" || len(snap.Pods) == 0 {
		t.Fatalf("identity %+v, %d pods", snap.Identity, len(snap.Pods))
	}
	for _, p := range snap.Pods {
		if p.Namespace != "shop" {
			t.Errorf("pod %s/%s read outside the namespace", p.Namespace, p.Name)
		}
	}
	narrowed := map[string]string{}
	for _, f := range snap.Refused {
		narrowed[f.List] = f.Namespace
		if !strings.Contains(f.Message, "dev@example.com") {
			t.Errorf("refusal %+v does not say who was refused", f)
		}
	}
	for _, list := range []string{"pods", "services", "deployments", "events"} {
		if narrowed[list] != "shop" {
			t.Errorf("%s was not read in shop: %v", list, narrowed)
		}
	}
	for _, list := range []string{"nodes", "namespaces", "persistentvolumes", "storageclasses"} {
		if !slices.Contains(snap.Denied, list) {
			t.Errorf("%s is not named as unseen: %v", list, snap.Denied)
		}
	}
	if len(snap.Namespaces) != 1 || snap.Namespaces[0].Name != "shop" {
		t.Errorf("namespaces %v", snap.Namespaces)
	}
	w := r.World()
	if len(w.Keeps) == 0 {
		t.Fatal("no Keep stands for the Pods' nodes")
	}
	for _, h := range w.Huts {
		if h.Node != "" && !h.Placed {
			t.Errorf("hut %s on %s found no plot", h.Name, h.Node)
		}
	}
	if len(rec.Writes()) != 0 {
		t.Errorf("writes %v", rec.Writes())
	}
}
