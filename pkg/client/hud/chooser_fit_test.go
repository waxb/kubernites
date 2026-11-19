package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestEveryChooserOptionReadsWholeInBothModes(t *testing.T) {
	for _, mode := range []string{"story", "names"} {
		if mode == "names" {
			ui.SetNamer(names.Kubernetes(64))
		}
		h := New(nil, Backend{})
		h.Ops.Chooser = chooserState{title: "WARD STONE KEEP 1", x: Width - 100, Options: TaintChoices("stone-keep-1", model.NodeView{})}
		h.OverlayID = OverlayChooser
		box := h.ChooserRect()
		if box.X < 0 || box.X+box.W > Width {
			t.Errorf("%s: the chooser at %+v leaves the screen", mode, box)
		}
		for _, b := range h.chooserButtons(Frame{}) {
			if IsClose(b) {
				continue
			}
			if shown, room := ui.Name(b.Label), ui.MaxChars(b.Rect.W-8, 1); len([]rune(shown)) > room {
				t.Errorf("%s: %q is cut to %d characters", mode, shown, room)
			}
		}
		ui.SetNamer(nil)
	}
}
