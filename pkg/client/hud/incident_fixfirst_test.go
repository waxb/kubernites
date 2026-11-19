package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestATroubleCardNamesItsFixBeforeTheCluesAreFound(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	inc := view.IncidentReport{Guided: true, Step: "hut", StepText: "Find the hut: select it in the world", FixText: "add or grow a Keep, pledge less, or scale the crew down",
		Clues: []view.IncidentClueView{{ID: "hut", Text: "Find the hut: select it in the world"}, {ID: "why", Text: "Read why the Quartermaster refused it"}}}
	got := strings.Join(h.stepLines(inc), " ")
	if !strings.Contains(got, "Fix it") || !strings.Contains(got, "add or grow a Keep") {
		t.Fatalf("the card's first form must name the fix, not only the clues: %q", got)
	}
}
