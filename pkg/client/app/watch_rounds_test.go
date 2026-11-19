package app

import (
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheWatchmanWalksAmongTheHutsWithABell(t *testing.T) {
	w := &view.WatchView{
		Crews: []view.CrewWatchView{{Crew: "porters", Liveness: &view.ProbeSetting{}}, {Crew: "clerks"}},
		Huts:  []view.HutWatchView{{Pod: "porters-b", Crew: "porters"}, {Pod: "clerks-a", Crew: "clerks"}, {Pod: "porters-a", Crew: "porters"}},
	}
	if got := BellHuts(w); !slices.Equal(got, []string{"porters-a", "porters-b"}) {
		t.Fatalf("bell huts = %v", got)
	}
	if got := BellHuts(nil); len(got) != 0 {
		t.Fatalf("no watch, yet huts %v", got)
	}
	r := watchRounds{}
	from, to, ok := r.next([]string{"porters-a", "porters-b"})
	if !ok || from != "porters-a" || to != "porters-b" {
		t.Fatalf("first round %s -> %s", from, to)
	}
	if from, to, _ = r.next([]string{"porters-a", "porters-b"}); from != "porters-b" || to != "porters-a" {
		t.Fatalf("second round %s -> %s", from, to)
	}
	if _, _, ok := r.next([]string{"porters-a"}); ok {
		t.Fatal("a single hut makes a round")
	}
}
