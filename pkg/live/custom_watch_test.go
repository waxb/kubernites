package live_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/types"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

const metadataWatch = "application/json;as=PartialObjectMetadata;g=meta.k8s.io;v=v1"

func customOf(r *live.Realm, kind string) []string {
	var out []string
	for _, c := range r.Snapshot().Custom {
		if c.Kind == kind {
			out = append(out, c.Namespace+"/"+c.Name)
		}
	}
	slices.Sort(out)
	return out
}

func bounty(s live.Snapshot) live.CRD {
	for _, c := range s.CRDs {
		if c.Kind == "Bounty" {
			return c
		}
	}
	panic("the kind fixture has no Bounty CRD")
}

func TestCustomResourcesAreWatchedFromTheirListLikeTheOtherKinds(t *testing.T) {
	s := kindFixture(t)
	w := watchRealm(t, s, livetest.Options{})
	c := bounty(s)
	path := livetest.CustomPath(c, "")
	eventually(t, "the bounties watched", func() bool { return len(w.feed.WatchesOf(path)) == 1 })
	wt := w.feed.WatchesOf(path)[0]
	if wt.Version != livetest.ListVersion || wt.Accept != metadataWatch || wt.Query["allowWatchBookmarks"] != "true" {
		t.Fatalf("the bounties watched with %+v", wt)
	}
	before := customOf(w.r, "Bounty")
	fresh := live.CustomResource{Group: c.Group, Kind: c.Kind, Namespace: "shop", Name: "dragon", UID: "uid-dragon"}
	w.feed.Send(path, "ADDED", livetest.CustomItem(fresh, "1100"))
	eventually(t, "the new bounty in the realm", func() bool { return slices.Contains(customOf(w.r, "Bounty"), "shop/dragon") })
	w.feed.Send(path, "DELETED", livetest.CustomItem(fresh, "1101"))
	eventually(t, "the bounty gone", func() bool { return slices.Equal(customOf(w.r, "Bounty"), before) })
	w.feed.Close(path)
	eventually(t, "the bounties watched again", func() bool { return len(w.feed.WatchesOf(path)) == 2 })
	if v := w.feed.WatchesOf(path)[1].Version; v != "1101" {
		t.Errorf("watched again from %q, not from the last event", v)
	}
	if writes := w.rec.Writes(); len(writes) != 0 {
		t.Fatalf("the server saw writes: %v", writes)
	}
}

func TestACRDThatComesWhileWatchingIsListedThenWatchedAndGoesWithItsCRD(t *testing.T) {
	s := kindFixture(t)
	w := watchRealm(t, s, livetest.Options{})
	crds := "/apis/apiextensions.k8s.io/v1/customresourcedefinitions"
	eventually(t, "the CRD watch", func() bool { return len(w.feed.WatchesOf(crds)) == 1 })
	c := live.CRD{Name: "dragons.lair.example", UID: "uid-dragons", Group: "lair.example", Kind: "Dragon", Plural: "dragons", Scope: "Namespaced", Version: "v1"}
	path := livetest.CustomPath(c, "")
	first := live.CustomResource{Group: c.Group, Kind: c.Kind, Namespace: "shop", Name: "dragon-a", UID: "uid-dragon-a"}
	w.feed.SetList(path, []map[string]any{livetest.CustomItem(first, "1200")})
	w.feed.Send(crds, "ADDED", livetest.CRDObject(c, "1201"))
	eventually(t, "the dragons listed and watched", func() bool { return len(w.feed.WatchesOf(path)) == 1 })
	if got := customOf(w.r, "Dragon"); !slices.Equal(got, []string{"shop/dragon-a"}) {
		t.Fatalf("the dragons in the realm: %v", got)
	}
	if lists := w.rec.ListsOf(path); len(lists) != 1 || lists[0].Query["resourceVersionMatch"] != "" {
		t.Fatalf("the new kind was listed with %+v", lists)
	}
	second := live.CustomResource{Group: c.Group, Kind: c.Kind, Namespace: "shop", Name: "dragon-b", UID: "uid-dragon-b"}
	w.feed.Send(path, "ADDED", livetest.CustomItem(second, "1202"))
	eventually(t, "the second dragon", func() bool { return len(customOf(w.r, "Dragon")) == 2 })
	w.feed.Send(path, "DELETED", livetest.CustomItem(first, "1203"))
	eventually(t, "the first dragon gone", func() bool { return slices.Equal(customOf(w.r, "Dragon"), []string{"shop/dragon-b"}) })
	w.feed.Send(crds, "DELETED", livetest.CRDObject(c, "1204"))
	eventually(t, "the dragons gone with their CRD", func() bool { return len(customOf(w.r, "Dragon")) == 0 })
	eventually(t, "the dragons' watch closed", func() bool { return len(w.feed.Streaming(path)) == 0 })
	if _, ok := w.r.Snapshot().Lists[c.List()]; ok {
		t.Error("a kind whose CRD went is still read")
	}
	if f := w.r.Freshness(); !f.StaleSince.IsZero() {
		t.Errorf("the realm is stale after a CRD went: %+v", f)
	}
}

func TestACRDWhoseInstancesTheCredentialsMayNotListIsNotSeenAndNeverStale(t *testing.T) {
	c := live.CRD{Name: "secrets.vault.example", UID: "uid-vault", Group: "vault.example", Kind: "Hoard", Plural: "hoards", Scope: "Namespaced", Version: "v1"}
	w := watchRealm(t, kindFixture(t), livetest.Options{Forbid: func(path string) bool { return strings.Contains(path, "/vault.example/") }})
	crds := "/apis/apiextensions.k8s.io/v1/customresourcedefinitions"
	eventually(t, "the CRD watch", func() bool { return len(w.feed.WatchesOf(crds)) == 1 })
	w.feed.Send(crds, "ADDED", livetest.CRDObject(c, "1300"))
	eventually(t, "the hoards refused", func() bool { return slices.Contains(w.r.Snapshot().Denied, c.List()) })
	if lists := w.rec.ListsOf(livetest.CustomPath(c, "default")); len(lists) != 1 {
		t.Errorf("the hoards were not tried in the context's namespace: %v", lists)
	}
	w.clock.move(2 * live.StaleAfter)
	if f := w.r.Freshness(); !f.StaleSince.IsZero() {
		t.Errorf("a kind the credentials may not list made the realm stale: %+v", f)
	}
	if n := len(w.feed.WatchesOf(livetest.CustomPath(c, ""))); n != 0 {
		t.Errorf("a refused kind was watched %d times", n)
	}
}

func capSnapshot(t *testing.T, extra int) (live.Snapshot, []live.CRD) {
	s := kindFixture(t)
	var crds []live.CRD
	for i := 0; i < live.CustomWatchCap+extra; i++ {
		c := live.CRD{Name: fmt.Sprintf("kind%03d.cap.example", i), UID: types.UID(fmt.Sprintf("uid-cap-%03d", i)), Group: "cap.example",
			Kind: fmt.Sprintf("Kind%03d", i), Plural: fmt.Sprintf("kind%03d", i), Scope: "Namespaced", Version: "v1"}
		crds = append(crds, c)
		s.CRDs = append(s.CRDs, c)
		if i >= live.CustomWatchCap {
			s.Custom = append(s.Custom, live.CustomResource{Group: c.Group, Kind: c.Kind, Namespace: "shop", Name: "held", UID: types.UID("uid-held-" + c.Kind)})
		}
	}
	slices.SortFunc(s.CRDs, func(a, b live.CRD) int { return strings.Compare(a.Name, b.Name) })
	return s, crds
}

func TestCustomWatchesStopAtTheCapAndTheKindsPastItAreListedOnAPeriod(t *testing.T) {
	const extra = 3
	s, crds := capSnapshot(t, extra)
	w := watchRealm(t, s, livetest.Options{})
	prefix := "/apis/cap.example/"
	own := len(kindFixture(t).CRDs)
	eventually(t, "the custom watches up to the cap", func() bool { return len(w.feed.Streaming(prefix)) == live.CustomWatchCap-own })
	listed := w.r.Listing()
	if len(listed) != extra+own {
		t.Fatalf("listed past the cap: %+v", listed)
	}
	for _, l := range listed {
		if !l.PastCap || l.Every != live.ListEvery {
			t.Errorf("listed %+v", l)
		}
	}
	streaming := w.feed.Streaming(prefix)
	for _, c := range kindFixture(t).CRDs {
		if !slices.Contains(w.feed.Streaming("/apis/"+c.Group+"/"), livetest.CustomPath(c, "")) {
			t.Errorf("%s holds instances but is not watched", c.Name)
		}
	}
	for _, c := range crds[live.CustomWatchCap:] {
		if !slices.Contains(streaming, livetest.CustomPath(c, "")) {
			t.Errorf("%s holds instances but is not watched", c.Name)
		}
	}
	for _, l := range listed {
		path := prefix + "v1/" + strings.TrimSuffix(l.List, ".cap.example")
		eventually(t, "the periodic lists of "+l.List, func() bool { return len(w.rec.ListsOf(path)) >= 3 })
		lists := w.rec.ListsOf(path)
		for _, q := range lists[1:] {
			if q.Query["resourceVersionMatch"] != "NotOlderThan" || q.Version != livetest.ListVersion {
				t.Errorf("%s listed again with %+v", l.List, q.Query)
			}
		}
		if len(w.feed.WatchesOf(path)) != 0 {
			t.Errorf("%s past the cap is watched", l.List)
		}
	}
	if total := len(w.feed.Streaming("/")); total > len(livetest.Lists(s))+live.CustomWatchCap+2 {
		t.Errorf("%d streams open", total)
	}
	w.clock.move(2 * live.StaleAfter)
	if f := w.r.Freshness(); !f.StaleSince.IsZero() {
		t.Errorf("kinds read by listing made the realm stale: %+v", f)
	}
}

func TestACRDServedAtANewVersionIsReadAgainAtThatVersion(t *testing.T) {
	s := kindFixture(t)
	w := watchRealm(t, s, livetest.Options{})
	c := bounty(s)
	old := livetest.CustomPath(c, "")
	eventually(t, "the bounties watched", func() bool { return len(w.feed.Streaming(old)) == 1 })
	moved := c
	moved.Version = "v2"
	path := livetest.CustomPath(moved, "")
	w.feed.SetList(path, []map[string]any{livetest.CustomItem(live.CustomResource{Group: c.Group, Kind: c.Kind, Namespace: "shop", Name: "only", UID: "uid-only"}, "1400")})
	w.feed.Send("/apis/apiextensions.k8s.io/v1/customresourcedefinitions", "MODIFIED", livetest.CRDObject(moved, "1401"))
	eventually(t, "the bounties watched at v2", func() bool { return len(w.feed.Streaming(path)) == 1 })
	eventually(t, "the v1 watch closed", func() bool { return len(w.feed.Streaming(old)) == 0 })
	if got := customOf(w.r, "Bounty"); !slices.Equal(got, []string{"shop/only"}) {
		t.Fatalf("the bounties after the new version: %v", got)
	}
}

func TestAGatewayIsDrawnOnceAsTheRealmReadsItAndNotAgainAsAForeignKind(t *testing.T) {
	s := kindFixture(t)
	gw := live.CRD{Name: "gateways.gateway.networking.k8s.io", UID: "uid-gw-crd", Group: "gateway.networking.k8s.io", Kind: "Gateway", Plural: "gateways", Scope: "Namespaced", Version: "v1"}
	s.CRDs = append(s.CRDs, gw)
	s.Custom = append(s.Custom, live.CustomResource{Group: gw.Group, Kind: gw.Kind, Namespace: "shop", Name: "front", UID: "uid-front"})
	w := watchRealm(t, s, livetest.Options{})
	eventually(t, "the custom watches", func() bool { return len(w.feed.Streaming("/apis/guild.kubernites.example/")) > 0 })
	if got := customOf(w.r, "Gateway"); len(got) != 0 {
		t.Errorf("Gateways read again as custom resources: %v", got)
	}
	if _, ok := w.r.Snapshot().Lists[gw.List()]; ok {
		t.Error("the Gateway CRD's instances are read as a custom kind")
	}
	for _, st := range w.r.World().Site.Structures {
		if st.Object.Kind == "Gateway" && st.Kind != live.KindHarborGate {
			t.Errorf("a Gateway stands as %s", st.Kind)
		}
	}
}
