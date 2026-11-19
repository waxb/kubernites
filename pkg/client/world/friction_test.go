package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func frictionScene() Scene {
	site := lifeSite()
	site.Structures = append(site.Structures, view.Structure{ID: "k2", Kind: view.StructKeep, Variant: "stone", X: 12, Y: 3, W: 3, H: 3, State: view.StateDark, Object: view.ObjectRef{Kind: "Node", Name: "keep-2"}})
	return Scene{
		Site: site, Queue: 6, ArrivalRate: 1,
		Pods: []model.PodView{
			{Namespace: "default", Name: "busy", State: model.HearthReady},
			{Namespace: "default", Name: "quiet", State: model.HearthReady},
			{Namespace: "default", Name: "fresh", State: model.HearthReady},
			{Namespace: "default", Name: "door", State: model.HearthReady, Static: true},
			{Namespace: "default", Name: "homeless", State: model.HearthPending},
			{Namespace: "kube-system", Name: "coredns-x", State: model.HearthPending, System: true},
			{Namespace: "default", Name: "hot", State: model.HearthReady},
		},
		Metrics: view.MetricsSnapshot{Pods: map[string]view.PodMetrics{
			"default/busy": {ServedRate: .6}, "default/quiet": {}, "default/door": {ServedRate: .2}, "default/hot": {ServedRate: .4, Throttled: true},
		}},
		Ledger: Ledger{Open: true, Share: .95},
	}
}

func derived(sc Scene) []Friction {
	out := []Friction{}
	var c census
	eachDerived(sc, c.of(sc), func(f Friction) { out = append(out, f) })
	return out
}

func kinds(fs []Friction) map[string][]string {
	out := map[string][]string{}
	for _, f := range fs {
		out[f.Kind] = append(out[f.Kind], f.Object)
	}
	return out
}

func TestDerivedFrictionsNameTheirObjects(t *testing.T) {
	got := kinds(derived(frictionScene()))
	want := map[string][]string{
		FrictionIdle: {"pod/default/quiet"}, FrictionUnhoused: {"pod/default/homeless"}, FrictionStrained: {"pod/default/hot"},
		FrictionWarded: {"node/keep-2"}, FrictionUnrecorded: {"service/default/ledgers"},
	}
	for kind, objs := range want {
		if len(got[kind]) != len(objs) || (len(objs) > 0 && got[kind][0] != objs[0]) {
			t.Errorf("%s = %v, want %v", kind, got[kind], objs)
		}
	}
}

func TestEveryDeclaredHearthIdlesWithoutAGuildhall(t *testing.T) {
	sc := frictionScene()
	sc.Site.Structures = sc.Site.Structures[:1]
	idle := kinds(derived(sc))[FrictionIdle]
	if len(idle) != 4 {
		t.Errorf("idle without a Guildhall = %v", idle)
	}
	sc.Queue = 0
	if len(kinds(derived(sc))[FrictionIdle]) != 0 {
		t.Error("with nobody waiting, resting is not idling")
	}
}

func TestMergeKeepsGameFrictionsFirstAndDropsDuplicates(t *testing.T) {
	game := []Friction{{Kind: FrictionWarded, Object: "node/keep-9", Text: "The ward"}, {Kind: FrictionIdle, Object: "pod/default/quiet", Text: "idle"}}
	derived := []Friction{{Kind: FrictionIdle, Object: "pod/default/quiet"}, {Kind: FrictionUnhoused, Object: "pod/default/x"}}
	merged := MergeFrictions(game, derived)
	if len(merged) != 3 || merged[0].Text != "The ward" || merged[1].Text != "idle" || merged[2].Object != "pod/default/x" {
		t.Errorf("merged = %+v", merged)
	}
	if !FrictionOn(merged, FrictionWarded, "node/keep-9") || FrictionOn(merged, FrictionWarded, "node/keep-1") {
		t.Error("FrictionOn")
	}
}

func TestScrollPileFollowsTheLedger(t *testing.T) {
	cases := []struct {
		l    Ledger
		want int
	}{
		{Ledger{}, -1}, {Ledger{Open: true, Share: .05}, -1}, {Ledger{Open: true, Share: .2}, 0},
		{Ledger{Open: true, Share: .5}, 1}, {Ledger{Open: true, Share: .8}, 2}, {Ledger{Open: true, Share: 1}, 3},
	}
	for _, c := range cases {
		if got := ScrollPile(c.l); got != c.want {
			t.Errorf("ScrollPile(%+v) = %d, want %d", c.l, got, c.want)
		}
	}
}
