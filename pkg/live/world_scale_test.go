package live_test

import (
	"testing"
	"time"

	"github.com/waxb/kubernites/pkg/live"
	"github.com/waxb/kubernites/pkg/live/livetest"
)

func TestAFullClusterDrawsEveryKeepApartAndEveryHut(t *testing.T) {
	s := livetest.Crowded(3, 10, 110)
	start := time.Now()
	w, err := live.Build(s)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d nodes, %d pods: world %dx%d, built in %v", len(s.Nodes), len(s.Pods), w.Site.Width, w.Site.Height, time.Since(start))
	if len(w.Keeps) != len(s.Nodes) {
		t.Fatalf("%d keeps for %d nodes", len(w.Keeps), len(s.Nodes))
	}
	for i, a := range w.Keeps {
		isle, ok := w.Site.IslandAt(a.X+1, a.Y+1)
		if !ok || isle.ID != a.Isle {
			t.Errorf("keep %s stands at %d,%d off its isle %s", a.Node, a.X, a.Y, a.Isle)
		}
		if a.Reach == 0 {
			t.Errorf("keep %s with %d huts kept the small territory", a.Node, a.Pods)
		}
		for _, b := range w.Keeps[i+1:] {
			if abs(a.X-b.X) < 3 && abs(a.Y-b.Y) < 3 {
				t.Errorf("keeps %s and %s overlap at %d,%d and %d,%d", a.Node, b.Node, a.X, a.Y, b.X, b.Y)
			}
		}
	}
	for _, h := range w.Huts {
		if !h.Placed {
			t.Fatalf("hut %s on %s found no plot", h.Name, h.Node)
		}
	}
	for _, st := range w.Site.Structures {
		if st.Object.Kind == "Node" && st.Territory == 0 {
			t.Errorf("structure %s has no territory for its huts", st.ID)
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
