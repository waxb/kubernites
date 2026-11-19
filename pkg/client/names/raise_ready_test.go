package names

import (
	"strings"
	"testing"
)

func TestAReadyLineNamesTheCreateButtonInNamesMode(t *testing.T) {
	named := Kubernetes(64)
	for _, story := range []string{"ready: open CREWS, then RAISE", "ready: press X on the STONE KEEP 1 card, then open CREWS, then RAISE"} {
		if got := named(story); !strings.HasSuffix(got, "then CREATE") {
			t.Errorf("%q reads %q in NAMES mode, but the button there says CREATE", story, got)
		}
	}
	if got := named("RAISE LIMIT"); got != "RAISE LIMIT" {
		t.Errorf("RAISE LIMIT reads %q", got)
	}
}
