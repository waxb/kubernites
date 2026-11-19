package kubectl

import (
	"fmt"
	"slices"
	"sort"
	"testing"
)

func TestTheOpenStoneServesNoWriteVerb(t *testing.T) {
	if !sort.StringsAreSorted(WriteVerbs()) {
		t.Errorf("the write verbs are not sorted: %v", WriteVerbs())
	}
	for _, verb := range WriteVerbs() {
		if slices.Contains(Verbs(), verb) {
			t.Errorf("the open Stone serves the write %s", verb)
		}
		f := newCluster(t)
		want := fmt.Sprintf("error: unknown command %q for \"kubectl\"\n", verb)
		if res := run(f, "kubectl "+verb+" pod lantern"); res.OK || res.Output != want {
			t.Errorf("kubectl %s answered %q", verb, res.Output)
		}
	}
}
