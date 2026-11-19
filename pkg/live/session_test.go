package live_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/kubectl"
	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/sim/store"
	"github.com/waxb/kubernites/pkg/view"
)

var everyAction = []string{
	"serve", "claimGolden", "build", "doorHearth", "summon", "raiseMuster", "scaleMuster", "bindFamiliar", "setRank", "raiseCrew",
	"openDepot", "rebindCodex", "buyKeep", "unlock", "buySoulstone", "ageUp", "chronicle", "speed",
	view.ActScaleCrew, view.ActCrewImage, view.ActCrewResources, view.ActRolloutCrew, view.ActAutoscaleCrew, view.ActSilenceHorn,
	view.ActRetuneHorn, view.ActMoveWorker, view.ActCrewProbes, view.ActDeployCrew, view.ActPlaceWarden, view.ActRaiseDaemonSet,
	view.ActEnroll, view.ActStrike, view.ActTearUpRoll, view.ActLookFor, view.ActCloseBerth,
}

func TestAFullLiveSessionSendsNoWrite(t *testing.T) {
	s := livetest.Snapshot(t, "testdata/kind-snapshot.json")
	srv, rec := livetest.Serve(t, s)
	path := livetest.Kubeconfig(t, srv.URL)
	ctxs, err := live.ListContexts([]string{path})
	if err != nil || len(ctxs) != 1 {
		t.Fatalf("contexts %v, %v", ctxs, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r, err := live.Connect(ctx, []string{path}, livetest.Context, time.Now(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range everyAction {
		if _, err := r.Perform(view.Action{Action: a, Name: "web", Replicas: 9}); !errors.Is(err, live.ErrReadOnly) {
			t.Errorf("Perform %s answered %v", a, err)
		}
		if _, err := r.Action(view.Action{Action: a}); !errors.Is(err, live.ErrReadOnly) {
			t.Errorf("Action %s answered %v", a, err)
		}
	}
	if out, ok := r.Apply("apiVersion: v1\nkind: Pod\nmetadata:\n  name: x\n"); ok || !strings.Contains(out, "read-only") {
		t.Errorf("Apply answered %q %v", out, ok)
	}
	if err := r.PlacementError(view.StructKeep, 10, 10); !errors.Is(err, live.ErrReadOnly) {
		t.Errorf("PlacementError answered %v", err)
	}
	if _, ok := reflect.TypeOf(r).MethodByName("Reforge"); ok {
		t.Error("a live realm offers the rebuild method")
	}
	if _, err := r.Marshal(time.Now()); err == nil {
		t.Error("a live realm was saved")
	}
	r.Bootstrap()
	r.Advance(5)
	crasher := ""
	for _, p := range s.Pods {
		if strings.HasPrefix(p.Name, "crasher-") {
			crasher = p.Name
		}
	}
	for _, verb := range append(kubectl.Verbs(), kubectl.WriteVerbs()...) {
		for _, cmd := range []string{
			"kubectl " + verb, "kubectl " + verb + " pod " + crasher + " -n shop", "kubectl " + verb + " deployment web -n shop --replicas=0",
			"kubectl " + verb + " node kubernites-worker", "kubectl -n shop " + verb + " -f web.yaml",
		} {
			out := r.Kubectl(cmd).Output
			if !live.ReadOnlyCommand(cmd) && strings.TrimSpace(out) != live.ReadOnlyAnswer {
				t.Errorf("%s answered %q", cmd, out)
			}
			r.Inspect(cmd)
		}
	}
	for _, cmd := range []string{"kubectl rollout restart deployment web -n shop", "kubectl rollout undo deployment web -n shop", "etcdctl get / --prefix", "kubectl auth can-i create pods"} {
		if out := r.Kubectl(cmd).Output; strings.TrimSpace(out) != live.ReadOnlyAnswer {
			t.Errorf("%s answered %q", cmd, out)
		}
	}
	if out := r.Kubectl("kubectl cluster-info").Output; !strings.Contains(out, "control plane is running at "+srv.URL) {
		t.Errorf("cluster-info names another server:\n%s", out)
	}
	if out := r.Kubectl("kubectl logs -n shop " + crasher).Output; !strings.Contains(out, "opening the ledger") {
		t.Errorf("logs did not read through the API: %q", out)
	}
	r.GameState()
	r.Metrics()
	r.Site()
	r.WithSnapshot(func([]store.Object) {})
	if w := rec.Writes(); len(w) != 0 {
		t.Fatalf("the fake API server saw writes: %v", w)
	}
	if len(rec.Seen()) < len(livetest.Lists(s)) {
		t.Fatalf("the session read only %v", rec.Seen())
	}
}
