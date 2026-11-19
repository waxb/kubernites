package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheNextLineIsNotRepeatedByTheChecklistBelowIt(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	inc := view.IncidentReport{Guided: true, Step: "hut", StepText: "Find the hut: select it in the world",
		Clues: []view.IncidentClueView{{ID: "hut", Text: "Find the hut: select it in the world", K8s: "Select the Pod"}, {ID: "why", Text: "Read why the Quartermaster refused it"}}}
	if got := h.stepLines(inc); len(got) != 0 {
		t.Fatalf("the NEXT line repeats the first clue: %v", got)
	}
	if rows := clueRows(inc); !rows[0].step || rows[1].step {
		t.Fatalf("the clue that is the next step stands out: %+v", rows)
	}
	inc.Step, inc.StepText = "fix", "Fix it: scale the porters"
	if got := h.stepLines(inc); len(got) == 0 {
		t.Fatal("a step that is no clue keeps its NEXT line")
	}
}
