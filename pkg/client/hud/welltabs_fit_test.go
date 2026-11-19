package hud

import (
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestEveryWellTabStaysClearOfTheNextInBothModes(t *testing.T) {
	f := navFrame(t)
	f.State.Reveals = append(slices.Clone(f.State.Reveals), "tidings", "grimoire", "stone", "scriptorium", "contracts", "council")
	h := New(nil, Backend{})
	for _, mode := range []string{"story", "names"} {
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		}
		tabs := []Button{}
		for _, b := range h.wellButtons(f) {
			if b.Label != "" {
				tabs = append(tabs, b)
			}
		}
		if len(tabs) < 6 {
			t.Fatalf("%s: only %d tabs show, want all six", mode, len(tabs))
		}
		for i, a := range tabs {
			for _, b := range tabs[i+1:] {
				if Overlaps(a.Rect, b.Rect) {
					t.Errorf("%s: tab %q at %+v runs under %q at %+v", mode, a.Label, a.Rect, b.Label, b.Rect)
				}
			}
			if a.Rect.X+a.Rect.W > InfoWell.X+InfoWell.W-5 {
				t.Errorf("%s: tab %q runs past the well", mode, a.Label)
			}
		}
		ui.SetNamer(nil)
	}
}
