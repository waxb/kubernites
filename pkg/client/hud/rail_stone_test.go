package hud

import (
	"slices"
	"testing"
)

func railIDs(buttons []Button) []string {
	out := []string{}
	for _, b := range buttons {
		out = append(out, b.ID)
	}
	return out
}

func TestTheRailStaysInSightBesideTheShortStone(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := railFrame(t, 3)
	bare := railIDs(h.RailButtons(f))
	if len(bare) != 3 {
		t.Fatalf("with nothing open the rail shows %v, want three notices", bare)
	}
	h.Open(OverlayStone)
	h.Stone.Tall = false
	if !h.ShortStone() {
		t.Fatal("the Stone did not open short")
	}
	if got := railIDs(h.RailButtons(f)); !slices.Equal(got, bare) {
		t.Errorf("beside the short Stone the rail shows %v, want the same notices as with nothing open: %v", got, bare)
	}
	if rail := h.RailRect(f); rail.W == 0 || Overlaps(rail, h.StoneRect()) {
		t.Errorf("the rail %+v is hidden or lies across the short Stone %+v", rail, h.StoneRect())
	}
	if !h.noticeSeen(f, f.Notices[0].ID) {
		t.Errorf("a notice on the rail beside the short Stone does not count as seen")
	}
	h.Stone.Tall = true
	if got := h.RailButtons(f); len(got) != 0 {
		t.Errorf("the tall Stone leaves the rail in sight: %v", railIDs(got))
	}
	h.Stone.Tall = false
	h.Open(OverlayChronicles)
	if got := h.RailButtons(f); len(got) != 0 {
		t.Errorf("an open panel leaves the rail in sight: %v", railIDs(got))
	}
}
