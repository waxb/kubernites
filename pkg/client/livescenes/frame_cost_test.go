package livescenes

import (
	"fmt"
	"slices"
	"syscall"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"k8s.io/apimachinery/pkg/types"

	app "github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live"
)

const (
	scaledTimes = 10
	frameWarm   = 30
)

func ScaledSnapshot(snap live.Snapshot, times int) live.Snapshot {
	nodes := slices.Clone(snap.Nodes)
	pods := slices.Clone(snap.Pods)
	for k := 1; k < times; k++ {
		suffix := fmt.Sprintf("-x%02d", k)
		for _, n := range snap.Nodes {
			c := *n.DeepCopy()
			c.Name += suffix
			c.UID += types.UID(suffix)
			if _, ok := c.Labels["kubernetes.io/hostname"]; ok {
				c.Labels["kubernetes.io/hostname"] = c.Name
			}
			nodes = append(nodes, c)
		}
		for _, p := range snap.Pods {
			c := *p.DeepCopy()
			c.Name += suffix
			c.UID += types.UID(suffix)
			if c.Spec.NodeName != "" {
				c.Spec.NodeName += suffix
			}
			pods = append(pods, c)
		}
	}
	snap.Nodes, snap.Pods = nodes, pods
	return snap
}

func TestScaledSnapshotHasTenTimesTheNodesAndPodsWithNewNames(t *testing.T) {
	base := ManagedSnapshot()
	snap := ScaledSnapshot(base, scaledTimes)
	if len(snap.Nodes) != scaledTimes*len(base.Nodes) || len(snap.Pods) != scaledTimes*len(base.Pods) {
		t.Fatalf("%d nodes and %d Pods, want %d and %d", len(snap.Nodes), len(snap.Pods), scaledTimes*len(base.Nodes), scaledTimes*len(base.Pods))
	}
	names, uids := map[string]bool{}, map[types.UID]bool{}
	for _, p := range snap.Pods {
		key := p.Namespace + "/" + p.Name
		if names[key] || uids[p.UID] {
			t.Fatalf("Pod %s or its UID %s is not new", key, p.UID)
		}
		names[key], uids[p.UID] = true, true
	}
	nodes := map[string]bool{}
	for _, n := range snap.Nodes {
		if nodes[n.Name] {
			t.Fatalf("node %s is not new", n.Name)
		}
		nodes[n.Name] = true
	}
	for _, p := range snap.Pods {
		if p.Spec.NodeName != "" && !nodes[p.Spec.NodeName] {
			t.Fatalf("Pod %s/%s runs on %s, which is not a node", p.Namespace, p.Name, p.Spec.NodeName)
		}
	}
}

func frameGame(tb testing.TB, snap live.Snapshot, far bool) *app.Game {
	tb.Helper()
	r, err := live.OpenRecorded(snap)
	if err != nil {
		tb.Fatal(err)
	}
	g := app.NewGame(nil, app.Options{LiveOnly: true}, nil, nil)
	g.Settings.LiveNames = storyNames()
	g.EnterLive(r)
	DismissIntros(g.Sess)
	if far {
		zoomedOut(g.Sess)
	}
	return g
}

func frame(g *app.Game, dst *ebiten.Image, p *app.Pointer) {
	g.Sess.Update(idleInput(), app.TickSeconds, p, true)
	g.Draw(dst)
}

func cpuNanos() int64 {
	var u syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &u); err != nil {
		return 0
	}
	return u.Utime.Nano() + u.Stime.Nano()
}

func BenchmarkLiveFrame(b *testing.B) {
	for _, c := range []struct {
		name  string
		times int
	}{{"fixture", 1}, {"scaled", scaledTimes}} {
		for _, z := range []struct {
			name string
			far  bool
		}{{"close", false}, {"far", true}} {
			b.Run(c.name+"/"+z.name, func(b *testing.B) { benchmarkLiveFrame(b, c.times, z.far) })
		}
	}
}

func benchmarkLiveFrame(b *testing.B, times int, far bool) {
	g := frameGame(b, ScaledSnapshot(ManagedSnapshot(), times), far)
	dst := ebiten.NewImage(hud.Width, hud.Height)
	defer dst.Deallocate()
	p := &app.Pointer{}
	for range frameWarm {
		frame(g, dst, p)
	}
	b.ReportAllocs()
	b.ResetTimer()
	start := cpuNanos()
	for range b.N {
		frame(g, dst, p)
	}
	b.ReportMetric(float64(cpuNanos()-start)/float64(b.N), "cpu-ns/op")
}
