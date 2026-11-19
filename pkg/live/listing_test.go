package live_test

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/sim/store"
)

func forbidPodWatch(path string) bool { return path == podsPath }

func watchesTried(rec *livetest.Record, path string) int {
	n := 0
	for _, s := range rec.Seen() {
		if s == "GET "+path+livetest.WatchMark {
			n++
		}
	}
	return n
}

func listedNames(r *live.Realm) []string {
	var out []string
	for _, l := range r.Listing() {
		out = append(out, l.List)
	}
	return out
}

func TestAWatchTheCredentialsForbidIsReadByListingOnAPeriodAndStaysFresh(t *testing.T) {
	s := kindFixture(t)
	w := watchRealm(t, s, livetest.Options{ForbidWatch: forbidPodWatch})
	eventually(t, "the pods read by listing", func() bool { return slices.Equal(listedNames(w.r), []string{"pods"}) })
	if l := w.r.Listing()[0]; l.Every != live.ListEvery || l.PastCap {
		t.Fatalf("the pods are listed %+v", l)
	}
	eventually(t, "lists on the period", func() bool { return len(w.rec.ListsOf(podsPath)) >= 4 })
	for _, q := range w.rec.ListsOf(podsPath)[1:] {
		if q.Query["resourceVersionMatch"] != "NotOlderThan" || q.Version != livetest.ListVersion || q.Query["limit"] == "" {
			t.Errorf("listed again with %+v", q.Query)
		}
	}
	watched := watchesTried(w.rec, podsPath)
	added := newPod("listed-in", "listed-in", "1500")
	w.feed.SetList(podsPath, append(slices.Clone(s.Pods), *added))
	eventually(t, "the listed hut in the mirror", func() bool {
		return live.MirrorHas(w.r, store.Key{Kind: store.KindPod, Namespace: "shop", Name: "listed-in"})
	})
	w.r.Pump(live.PumpEvery)
	if ev := w.events(); len(ev) == 0 || ev[len(ev)-1].Object.GetName() != "listed-in" {
		t.Errorf("the session heard %+v", ev)
	}
	w.feed.SetList(podsPath, s.Pods)
	eventually(t, "the hut gone from the mirror", func() bool {
		return !live.MirrorHas(w.r, store.Key{Kind: store.KindPod, Namespace: "shop", Name: "listed-in"})
	})
	w.clock.move(3 * live.StaleAfter)
	if f := w.r.Freshness(); !f.Watching || !f.StaleSince.IsZero() {
		t.Fatalf("a kind read by listing made the realm stale: %+v", f)
	}
	if n := watchesTried(w.rec, podsPath); n != watched || n != 1 {
		t.Errorf("the forbidden watch was tried %d times, then %d: a listed kind must not flicker back to watching", watched, n)
	}
	if writes := w.rec.Writes(); len(writes) != 0 {
		t.Fatalf("the server saw writes: %v", writes)
	}
}

func TestAKindReadByListingIsStaleOnlyWhenItsListsFailForLongerThanStaleAfter(t *testing.T) {
	s := kindFixture(t)
	var failing atomic.Bool
	w := watchRealm(t, s, livetest.Options{ForbidWatch: forbidPodWatch, Forbid: func(path string) bool { return failing.Load() && path == podsPath }})
	eventually(t, "the pods read by listing", func() bool { return len(w.rec.ListsOf(podsPath)) >= 2 })
	broke := w.clock.now()
	failing.Store(true)
	before := len(w.rec.ListsOf(podsPath))
	eventually(t, "failed lists", func() bool { return len(w.rec.ListsOf(podsPath)) >= before+3 })
	if f := w.r.Freshness(); !f.StaleSince.IsZero() {
		t.Errorf("stale at once: %+v", f)
	}
	w.clock.move(live.StaleAfter + time.Second)
	f := w.r.Freshness()
	if f.StaleSince.Before(broke) || f.StaleSince.After(broke.Add(time.Second)) || !strings.HasPrefix(f.Why, "pods: ") {
		t.Fatalf("freshness while the lists fail %+v, broke at %v", f, broke)
	}
	failing.Store(false)
	eventually(t, "fresh again after a list", func() bool { return w.r.Freshness().StaleSince.IsZero() })
	if !slices.Equal(listedNames(w.r), []string{"pods"}) {
		t.Errorf("after the failed lists the kind is read by %v", listedNames(w.r))
	}
}

func mirrorPods(r *live.Realm) map[string]string {
	out := map[string]string{}
	r.WithSnapshot(func(objs []store.Object) {
		for _, o := range objs {
			if p, ok := o.(*corev1.Pod); ok {
				out[string(p.UID)] = p.ResourceVersion + "/" + string(p.Status.Phase)
			}
		}
	})
	return out
}

func listedPods(list []corev1.Pod) map[string]string {
	out := map[string]string{}
	for _, p := range list {
		out[string(p.UID)] = p.ResourceVersion + "/" + string(p.Status.Phase)
	}
	return out
}

func TestTheMirrorAfterAListingPeriodEqualsAFreshList(t *testing.T) {
	s := kindFixture(t)
	w := watchRealm(t, s, livetest.Options{ForbidWatch: forbidPodWatch})
	eventually(t, "the pods read by listing", func() bool { return len(w.rec.ListsOf(podsPath)) >= 2 })
	rng := rand.New(rand.NewPCG(23, 37))
	phases := []corev1.PodPhase{corev1.PodPending, corev1.PodRunning, corev1.PodSucceeded, corev1.PodFailed}
	list := slices.Clone(s.Pods)
	for round := 0; round < 40; round++ {
		for n := rng.IntN(4); n > 0 && len(list) > 0; n-- {
			i := rng.IntN(len(list))
			list = append(list[:i], list[i+1:]...)
		}
		for n := rng.IntN(4); n > 0; n-- {
			p := newPod(fmt.Sprintf("r%d-%d", round, n), fmt.Sprintf("r%d-%d", round, n), fmt.Sprint(2000+round))
			list = append(list, *p)
		}
		for n := rng.IntN(4); n > 0 && len(list) > 0; n-- {
			i := rng.IntN(len(list))
			list[i] = *list[i].DeepCopy()
			list[i].ResourceVersion, list[i].Status.Phase = fmt.Sprint(3000+round*10+n), phases[rng.IntN(len(phases))]
		}
		w.feed.SetList(podsPath, slices.Clone(list))
		at := len(w.rec.ListsOf(podsPath))
		eventually(t, "two lists after the change", func() bool { return len(w.rec.ListsOf(podsPath)) >= at+2 })
		want := listedPods(list)
		if got := listedPods(w.r.Snapshot().Pods); !mapsEqual(got, want) {
			t.Fatalf("round %d: the snapshot after the period differs from a fresh list:\n got %v\nwant %v", round, got, want)
		}
		if got := mirrorPods(w.r); !mapsEqual(got, want) {
			t.Fatalf("round %d: the mirror after the period differs from a fresh list:\n got %v\nwant %v", round, got, want)
		}
	}
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
