package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestHowNamesOnlyAnActThePlayerReallyDidBefore(t *testing.T) {
	if tip := HintTip(view.ObjectiveView{Practiced: true}); strings.Contains(tip, "before") {
		t.Fatalf("with no act done before, HOW? says %q", tip)
	}
	if tip := HintTip(view.ObjectiveView{Practiced: true, DoneAct: "raised a building with BUILD"}); !strings.Contains(tip, "You have raised a building with BUILD before") {
		t.Fatalf("HOW? does not name the act done before: %q", tip)
	}
}
