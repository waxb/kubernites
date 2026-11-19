package hud

import "testing"

func TestTheDepotChooserIsRetired(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := chooserFrame(t)
	h.Press(CmdDepot, f)
	if h.Overlay() != OverlayNone {
		t.Errorf("DEPOT still opens %q; Services are built as stalls from BUILD", h.Overlay())
	}
	for _, id := range []string{"depots"} {
		if _, ok := Overlays[id]; ok {
			t.Errorf("the %q overlay is still registered", id)
		}
		if _, ok := prefixHandlers[id]; ok {
			t.Errorf("the %q buttons still have a handler", id)
		}
	}
	openChooser(t, h, f, "clerks")
	if len(h.raiseButtons(f)) == 0 {
		t.Fatalf("CREW no longer opens its dialog: %q", h.Overlay())
	}
}
