package live

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/lexicon"
)

func TestALiveKeepsLabelNamesNoSizeTheClusterDoesNotHave(t *testing.T) {
	for _, variant := range []string{"tower", "stone", "great", "citadel"} {
		st := keepStructure(1, KeepPlot{Variant: variant, Node: "kubernites-worker"})
		for _, size := range []string{"Stone", "Great", "Citadel", "Lone", "Tower"} {
			if strings.Contains(st.Label, size) {
				t.Errorf("a live %s Keep is labelled %q", variant, st.Label)
			}
		}
		if got := lexicon.TrueNames(st.Label); got != "Node kubernites-worker" {
			t.Errorf("a live %s Keep labelled %q reads %q in NAMES mode", variant, st.Label, got)
		}
	}
}
