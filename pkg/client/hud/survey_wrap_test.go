package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestASurveyRowsWrappedTextStaysInsideItsRow(t *testing.T) {
	defer ui.SetNamer(nil)
	ui.SetNamer(names.Kubernetes(64))
	h := New(nil, Backend{})
	p := SurveyPanel{Keep: true, Name: "Stone Keep", Survey: view.Survey{OK: true, CPU: 8, Wood: 40, MemoryGiB: 4, Stone: 6, Class: "compute", Instance: "compute-xlarge"}}
	blocks, _, _ := h.SurveyBlocks(p)
	wrapped := false
	for _, b := range blocks {
		wrapped = wrapped || len(b.Lines) > 1
		if bottom := surveyTextTop(b) + len(b.Lines)*ui.LineHeight; bottom > b.h {
			t.Errorf("%q runs %d px into the row below", b.Row.Text, bottom-b.h)
		}
	}
	if !wrapped {
		t.Fatal("no row wraps, so this does not test a wrapped row")
	}
}
