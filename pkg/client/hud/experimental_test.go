package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestAnExperimentalStepsCardSaysSo(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Objectives = []view.ObjectiveView{{ID: "great-keep", Text: "Raise a Great Keep (a larger instance type).", Progress: "Learn Masonry I", PastDemo: true}}
	if title, _ := h.CardTitle(f); title != experimentalStepTitle {
		t.Fatalf("an experimental step's card is titled %q", title)
	}
	text := []string{}
	for _, l := range h.ObjectiveLines(f) {
		text = append(text, l.Text)
	}
	if !strings.Contains(strings.Join(text, " "), "Past the demo's end") {
		t.Fatalf("an experimental step's card never says it is past the demo's end: %q", text)
	}
	f.State.Objectives[0].PastDemo = false
	if title, _ := h.CardTitle(f); title != NextStepTitle {
		t.Fatalf("a demo step's card is titled %q", title)
	}
}
