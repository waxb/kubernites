package write

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func podOn(node, ns, name, owner string, annotations map[string]string) corev1.Pod {
	p := corev1.Pod{ObjectMeta: meta(ns, name, name, "1"+name[len(name)-1:])}
	p.Spec.NodeName = node
	p.Annotations = annotations
	if owner != "" {
		yes := true
		p.OwnerReferences = []metav1.OwnerReference{{Kind: owner, Name: "o", Controller: &yes}}
	}
	return p
}

func drainFixture() []corev1.Pod {
	return []corev1.Pod{
		podOn("kind-worker", "shop", "web-1", "ReplicaSet", nil),
		podOn("kind-worker", "kube-system", "kube-proxy-1", "DaemonSet", nil),
		podOn("kind-worker", "kube-system", "etcd-2", "Node", map[string]string{mirrorKey: "x"}),
		podOn("kind-worker", "shop", "loner-3", "", nil),
		podOn("kind-control-plane", "shop", "web-4", "ReplicaSet", nil),
	}
}

func TestADrainCordonsAndEvictsAllButMirrorAndDaemonSetHuts(t *testing.T) {
	d, err := PlanDrain(fixtureNode(false), drainFixture())
	if err != nil {
		t.Fatal(err)
	}
	if d.Cordon == nil || d.Cordon.Verb != OrderCordon {
		t.Fatalf("no cordon first: %+v", d.Cordon)
	}
	var evicted []string
	for _, o := range d.Evict {
		evicted = append(evicted, o.Target.Namespace+"/"+o.Target.Name)
	}
	if strings.Join(evicted, " ") != "shop/loner-3 shop/web-1" {
		t.Fatalf("evicts %v", evicted)
	}
	if len(d.Skipped) != 2 || len(d.Orphans) != 1 || len(d.Classes) != 1 || d.Classes[0] != Destroy {
		t.Fatalf("skipped %v orphans %v classes %v", d.Skipped, d.Orphans, d.Classes)
	}
	cordoned, _ := PlanDrain(fixtureNode(true), drainFixture()[:1])
	if cordoned.Cordon != nil || len(cordoned.Classes) != 0 {
		t.Fatalf("a cordoned Keep is cordoned again: %+v", cordoned)
	}
}

func TestADrainWaitsOnABudgetRetriesAndNeverRetriesAConflict(t *testing.T) {
	var mu sync.Mutex
	tries := map[string]int{}
	f := &fakeServer{answer: func(r *http.Request, body string) (int, string) {
		mu.Lock()
		defer mu.Unlock()
		tries[r.URL.Path]++
		switch {
		case strings.HasSuffix(r.URL.Path, "/web-1/eviction") && tries[r.URL.Path] <= 2:
			return http.StatusTooManyRequests, `{"kind":"Status","code":429,"message":"Cannot evict pod as it would violate the pod's disruption budget.","details":{"causes":[{"reason":"DisruptionBudget","message":"The disruption budget web needs 1 healthy pods and has 1 currently"}]}}`
		case strings.HasSuffix(r.URL.Path, "/loner-3/eviction"):
			return http.StatusConflict, status(409, "Conflict", "the object has been modified")
		}
		return http.StatusCreated, `{}`
	}}
	g := bound(t, Writes, f, Destroy)
	d, _ := PlanDrain(fixtureNode(false), drainFixture())
	g.IssueDrain(d)
	var reports []DrainReport
	if err := g.RunDrain(context.Background(), d, time.Millisecond, func(r DrainReport) { reports = append(reports, r) }); err != nil {
		t.Fatal(err)
	}
	if tries["/api/v1/namespaces/shop/pods/loner-3/eviction"] != 1 {
		t.Errorf("the conflict was tried %d times", tries["/api/v1/namespaces/shop/pods/loner-3/eviction"])
	}
	if tries["/api/v1/namespaces/shop/pods/web-1/eviction"] != 3 {
		t.Errorf("the blocked eviction was tried %d times", tries["/api/v1/namespaces/shop/pods/web-1/eviction"])
	}
	if len(reports) != 4 || reports[0].Failed == "" || !strings.Contains(reports[1].Blocked, "The disruption budget web needs 1") || !reports[3].Done {
		t.Fatalf("reports %+v", reports)
	}
	if lines := f.lines(); lines[0] != "PATCH /api/v1/nodes/kind-worker?fieldManager=kubernites" {
		t.Fatalf("the drain did not cordon first: %v", lines)
	}
}

func TestSTOPEndsADrain(t *testing.T) {
	f := &fakeServer{answer: func(r *http.Request, body string) (int, string) {
		if strings.HasSuffix(r.URL.Path, "/eviction") {
			return http.StatusTooManyRequests, `{"kind":"Status","code":429,"message":"blocked"}`
		}
		return http.StatusOK, `{}`
	}}
	g := bound(t, Seamless, f, Destroy)
	d, _ := PlanDrain(fixtureNode(true), drainFixture())
	ctx, stop := context.WithCancel(context.Background())
	n := 0
	err := g.RunDrain(ctx, d, time.Millisecond, func(DrainReport) {
		n++
		if n == 3 {
			stop()
		}
	})
	if err == nil || n != 3 {
		t.Fatalf("STOP: %v after %d reports", err, n)
	}
}
