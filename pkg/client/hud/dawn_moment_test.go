package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func TestTheDawnPanelSaysWhatEachKeepNowCosts(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Age = view.AgeFeudal
	f.State.AgeMoment = &view.AgeMomentView{Age: view.AgeFeudal, Opened: []string{"7 manual pages: Pledges", "More Stone Keeps: 37 times the first size"},
		Changed: []string{"Each Keep now carries overhead: 11 requests a second"}}
	h.StartDawn(view.AgeFeudal, []string{"Archive"}, []string{"Pledges"})
	opened, changed := h.dawnRows(f)
	rows := strings.Join(append(opened, changed...), "\n")
	for _, want := range []string{"More Stone Keeps: 37 times the first size", "Each Keep now carries overhead"} {
		if !strings.Contains(rows, want) {
			t.Errorf("the dawn panel reads %q, missing %q", rows, want)
		}
	}
}
