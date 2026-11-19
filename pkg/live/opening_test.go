package live_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

type told struct {
	mu   sync.Mutex
	seen []live.Progress
}

func (t *told) tell(p live.Progress) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.seen = append(t.seen, p)
}

func (t *told) last() live.Progress {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.seen) == 0 {
		return live.Progress{}
	}
	return t.seen[len(t.seen)-1]
}

func TestTheStagingShapeOpensStepByStepInTime(t *testing.T) {
	s := livetest.Staging()
	if len(s.Nodes) != 3 || len(s.Namespaces) != 15 || len(s.Pods) != 58 {
		t.Fatalf("the staging shape is %d nodes, %d namespaces, %d pods; the player's is 3, 15, 58", len(s.Nodes), len(s.Namespaces), len(s.Pods))
	}
	at := time.Now()
	if _, err := live.NewMirror(s); err != nil || time.Since(at) > time.Second {
		t.Errorf("NewMirror took %v: %v", time.Since(at), err)
	}
	at = time.Now()
	if _, err := live.Build(s); err != nil || time.Since(at) > 5*time.Second {
		t.Errorf("Build took %v: %v", time.Since(at), err)
	}
	srv, _ := livetest.ServeAs(t, s, livetest.Options{User: "player@example.com", Metrics: true})
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	steps := &told{}
	at = time.Now()
	r, err := live.Connect(ctx, []string{livetest.Kubeconfig(t, srv.URL)}, livetest.Context, s.TakenAt, steps.tell)
	if err != nil || time.Since(at) > 10*time.Second {
		t.Fatalf("the staging cluster opened in %v: %v", time.Since(at), err)
	}
	if !r.Metrics().Augury {
		t.Error("metrics-server answered, yet the realm has no Augur")
	}
	order := []string{}
	for _, p := range steps.seen {
		if p.Step != "" && (len(order) == 0 || order[len(order)-1] != p.Step) {
			order = append(order, p.Step)
		}
	}
	if strings.Join(order, ",") != live.StepMetrics+","+live.StepDrawing {
		t.Errorf("after the lists the opening said %v; it must say when it reads the metrics and when it draws", order)
	}
}

func TestTheReadingNamesTheCustomListItWaitsOn(t *testing.T) {
	s := livetest.Staging()
	crd := s.CRDs[0]
	srv, _ := livetest.ServeAs(t, s, livetest.Options{User: "player@example.com", Stall: func(path string) bool { return strings.HasSuffix(path, "/"+crd.Plural) }})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	steps := &told{}
	done := make(chan error, 1)
	go func() {
		_, err := live.Connect(ctx, []string{livetest.Kubeconfig(t, srv.URL)}, livetest.Context, s.TakenAt, steps.tell)
		done <- err
	}()
	for deadline := time.Now().Add(10 * time.Second); steps.last().List != crd.Plural+"."+crd.Group && time.Now().Before(deadline); {
		time.Sleep(20 * time.Millisecond)
	}
	if p := steps.last(); p.List != crd.Plural+"."+crd.Group || p.Done >= p.Total {
		t.Errorf("while %s.%s does not answer, the opening says %+v (the player saw \"Every list is read: drawing the realm\")", crd.Plural, crd.Group, p)
	}
	cancel()
	<-done
}

func TestAStalledMetricsAPICostsOnlyTheAugur(t *testing.T) {
	defer live.SetUsageLimit(300 * time.Millisecond)()
	s := livetest.Staging()
	srv, _ := livetest.ServeAs(t, s, livetest.Options{User: "player@example.com", Metrics: true, Stall: func(path string) bool { return strings.HasSuffix(path, "/metrics.k8s.io/v1beta1/pods") }})
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	at := time.Now()
	r, err := live.Connect(ctx, []string{livetest.Kubeconfig(t, srv.URL)}, livetest.Context, s.TakenAt, nil)
	if err != nil || time.Since(at) > 10*time.Second {
		t.Fatalf("with metrics-server stalled the realm opened in %v: %v", time.Since(at), err)
	}
	if r.Metrics().Augury {
		t.Error("a stalled metrics API still gave the realm an Augur")
	}
}
