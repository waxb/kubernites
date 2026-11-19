package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestEveryResourceEffectReadsWholeInBothModes(t *testing.T) {
	w := crewW - 2*crewPad
	for _, mode := range []string{"story", "names"} {
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		}
		for _, rf := range resFields {
			text := rf.label + ": " + ui.Bridge(rf.story, rf.k8s)
			rows := ui.Font{}.WrapLines(text, w, 1, resEffectRows)
			if last := rows[len(rows)-1]; strings.HasSuffix(last, "...") {
				t.Errorf("%s: %q is cut to %q", mode, ui.Name(text), rows)
			}
		}
		ui.SetNamer(nil)
	}
}
