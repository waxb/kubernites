package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestThePayoffIsLabelledAsWhatBuyingBrings(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, k8s := range []bool{false, true} {
		ui.SetNamer(nil)
		want := "ENDOWING BRINGS: "
		if k8s {
			ui.SetNamer(names.Kubernetes(64))
			want = "BUYING BRINGS: "
		}
		if got := ClassPayoff(porterState(1.2, 2, 8), "porter:2.0"); !strings.HasPrefix(got, want) {
			t.Errorf("k8s=%v result = %q, want prefix %q", k8s, got, want)
		}
	}
}

func TestTheRollOutTipLabelsItsPayoffAsTheRollOut(t *testing.T) {
	line := rollOutPayoff(porterState(1.2, 2, 8), "porter:2.0")
	if !strings.HasPrefix(line, "ROLLING OUT BRINGS: ") {
		t.Fatalf("roll out result = %q", line)
	}
}
