package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheQuietLineFitsTheCardOnceItsNamesAreSpoken(t *testing.T) {
	defer ui.SetNamer(nil)
	for _, namer := range []func(string) string{nil, names.Kubernetes(64)} {
		ui.SetNamer(namer)
		c := LiveCard{Context: "kind-a", Server: "https://127.0.0.1:6443", Level: LevelReadOnly}
		for _, k := range []string{view.TagUnhoused, view.TagOOMKilled, view.CalamityNoDepot, view.CalamityNoCrew, view.CalamitySpoiling, view.TagCrashLoop, view.TagImagePull} {
			c.Quiet = append(c.Quiet, QuietTrouble{ID: k, Kind: k, Cause: "x"})
		}
		line, ok := c.quietLine()
		if !ok {
			t.Fatal("no QUIET line for seven quiet troubles")
		}
		if w := (ui.Font{}).Width(line.text, 1); w > CardBox.W-16 {
			t.Errorf("names %v: %q is drawn %d pixels wide as %q, the card has %d", ui.Naming(), line.text, w, ui.Name(line.text), CardBox.W-16)
		}
	}
}
