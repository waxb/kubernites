package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestTheConsoleRegistersOnlyThePiecesItDrawsWhenItsStripIsHidden(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.State.Tutorial = true
	h.Update(Input{MX: -10, MY: -10}, f)
	if h.StripShown(f) {
		t.Fatal("the console strip shows in the tutorial frame: the test proves nothing")
	}
	drawn := []ui.Rect{}
	for _, r := range h.consoleRects(f) {
		drawn = append(drawn, Shadowed(r))
	}
	if len(drawn) < 2 {
		t.Fatalf("the tutorial frame's console draws %d pieces: the test proves nothing", len(drawn))
	}
	gap := ui.Rect{X: Minimap.X + Minimap.W + 20, Y: Minimap.Y + 10, W: 40, H: 20}
	for _, r := range drawn {
		if Overlaps(gap, r) {
			t.Fatalf("the probe %+v lies on a drawn console piece %+v: the test proves nothing", gap, r)
		}
	}
	for _, s := range consoleSurfaces(h, f) {
		if Overlaps(s.Rect, gap) {
			t.Fatalf("the console registers %+v over the open world between its pieces %v: a world callout in that gap counted as overlapping the console", s.Rect, drawn)
		}
	}
}
