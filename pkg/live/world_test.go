package live

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func kindSnapshot(t testing.TB) Snapshot {
	t.Helper()
	f, err := os.Open("testdata/kind-snapshot.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	s, err := ReadSnapshot(f)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func worldDigest(t testing.TB, w *World) string {
	t.Helper()
	data, err := json.Marshal(struct {
		Seed      uint64
		Site      any
		Isles     []Isle
		Keeps     []KeepPlot
		Districts []District
		Crews     []Crew
		Huts      []HutPlot
	}{w.Seed, w.Site, w.Isles, w.Keeps, w.Districts, w.Crews, w.Huts})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func shuffled(s Snapshot, seed uint64) Snapshot {
	rng := rand.New(rand.NewPCG(seed, seed^0x5eed))
	out := s
	out.Nodes = append([]corev1.Node(nil), s.Nodes...)
	out.Pods = append([]corev1.Pod(nil), s.Pods...)
	out.Namespaces = append([]corev1.Namespace(nil), s.Namespaces...)
	rng.Shuffle(len(out.Nodes), func(i, j int) { out.Nodes[i], out.Nodes[j] = out.Nodes[j], out.Nodes[i] })
	rng.Shuffle(len(out.Pods), func(i, j int) { out.Pods[i], out.Pods[j] = out.Pods[j], out.Pods[i] })
	rng.Shuffle(len(out.Namespaces), func(i, j int) { out.Namespaces[i], out.Namespaces[j] = out.Namespaces[j], out.Namespaces[i] })
	rng.Shuffle(len(out.Deployments), func(i, j int) { out.Deployments[i], out.Deployments[j] = out.Deployments[j], out.Deployments[i] })
	rng.Shuffle(len(out.ReplicaSets), func(i, j int) { out.ReplicaSets[i], out.ReplicaSets[j] = out.ReplicaSets[j], out.ReplicaSets[i] })
	return out
}

func TestTheSameClusterGivesTheSameWorld(t *testing.T) {
	s := kindSnapshot(t)
	first, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	want := worldDigest(t, first)
	for seed := uint64(1); seed <= 5; seed++ {
		w, err := Build(shuffled(s, seed))
		if err != nil {
			t.Fatal(err)
		}
		if got := worldDigest(t, w); got != want {
			t.Fatalf("shuffle %d gave another world: %s, want %s", seed, got, want)
		}
	}
}

func TestTheWorldFollowsTheClustersShape(t *testing.T) {
	s := kindSnapshot(t)
	w, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Isles) != 3 || w.Isles[0].Zone != "" || w.Isles[1].Zone == w.Isles[2].Zone {
		t.Fatalf("isles %+v", w.Isles)
	}
	for _, is := range w.Isles[1:] {
		if is.By != "zone" || (is.Zone != "kind-east" && is.Zone != "kind-west") {
			t.Errorf("isle %+v is not a zone of the kind cluster", is)
		}
	}
	seen := map[string]bool{}
	for _, k := range w.Keeps {
		isle, ok := w.Site.IslandAt(k.X+1, k.Y+1)
		if !ok || isle.ID != k.Isle {
			t.Errorf("keep %s stands at %d,%d off its isle %s", k.Node, k.X, k.Y, k.Isle)
		}
		seen[k.Node] = true
	}
	if len(seen) != 3 {
		t.Errorf("keeps %+v", w.Keeps)
	}
	placed, unhoused := 0, 0
	taken := map[[2]int]bool{}
	for _, h := range w.Huts {
		if !h.Placed {
			unhoused++
			if h.Node != "" {
				t.Errorf("hut %s on %s found no plot", h.Name, h.Node)
			}
			continue
		}
		placed++
		if taken[[2]int{h.X, h.Y}] {
			t.Errorf("two huts share plot %d,%d", h.X, h.Y)
		}
		taken[[2]int{h.X, h.Y}] = true
		if ground := w.Site.Tile(h.X, h.Y); h.Shrine && !buildable(ground) || !h.Shrine && !hutGround(ground) {
			t.Errorf("hut %s stands on %v", h.Name, ground)
		}
	}
	if unhoused != 1 || placed != len(w.Huts)-1 {
		t.Errorf("placed %d, unhoused %d of %d (the giant waits Unhoused)", placed, unhoused, len(w.Huts))
	}
	districts := map[string]District{}
	for _, d := range w.Districts {
		districts[d.Namespace] = d
	}
	if !districts["kube-system"].Citadel || districts["shop"].Pods == 0 || len(districts) != len(s.Namespaces)+1 {
		t.Errorf("districts %+v", w.Districts)
	}
}

func syntheticPod(i int, node string, born time.Time) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("swarm-%05d", i), Namespace: "shop", UID: types.UID(fmt.Sprintf("00000000-0000-4000-8000-%012d", i)),
			CreationTimestamp: metav1.NewTime(born.Add(time.Duration(i) * time.Second))},
		Spec: corev1.PodSpec{NodeName: node},
	}
}

func TestAddingAndRemovingHutsMovesNoStandingHut(t *testing.T) {
	s := kindSnapshot(t)
	w, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	born := s.TakenAt.Add(time.Hour)
	workers := []string{w.Keeps[1].Node, w.Keeps[2].Node}
	pods := append([]corev1.Pod(nil), s.Pods...)
	for i := 0; i < 100; i++ {
		pods = append(pods, syntheticPod(i, workers[i%2], born))
	}
	w.Settle(pods)
	before := map[types.UID]HutPlot{}
	for _, h := range w.Huts {
		before[h.UID] = h
	}
	next := []corev1.Pod{}
	for i, p := range pods {
		if i%3 != 1 {
			next = append(next, p)
		}
	}
	for i := 100; i < 200; i++ {
		next = append(next, syntheticPod(i, workers[i%2], born))
	}
	w.Settle(next)
	moved, stayed := 0, 0
	for _, h := range w.Huts {
		if was, ok := before[h.UID]; ok {
			stayed++
			if was.X != h.X || was.Y != h.Y || was.Placed != h.Placed {
				moved++
				t.Errorf("hut %s moved from %d,%d to %d,%d", h.Name, was.X, was.Y, h.X, h.Y)
			}
		}
	}
	if stayed < 80 || moved != 0 {
		t.Errorf("%d huts stayed, %d moved", stayed, moved)
	}
}

func TestFiveThousandHutsAreAllSeated(t *testing.T) {
	s := kindSnapshot(t)
	born := s.TakenAt.Add(time.Hour)
	for i := 0; i < 5000; i++ {
		s.Pods = append(s.Pods, syntheticPod(i, s.Nodes[1+i%2].Name, born))
	}
	start := time.Now()
	w, err := Build(s)
	if err != nil {
		t.Fatal(err)
	}
	placed := 0
	for _, h := range w.Huts {
		if h.Placed {
			placed++
		}
	}
	t.Logf("5,000 synthetic huts: %d placed, %d Unhoused, built in %v, world %dx%d", placed, len(w.Huts)-placed, time.Since(start), w.Site.Width, w.Site.Height)
	if len(w.Huts) != len(s.Pods) || placed != len(s.Pods)-1 {
		t.Errorf("huts %d of %d pods, %d placed", len(w.Huts), len(s.Pods), placed)
	}
}
