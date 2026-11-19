package hud

import "testing"

func TestMenuOffersFullscreen(t *testing.T) {
	f := fullFrame(t)
	h := New(nil, Backend{})
	h.Open(OverlayMenu)
	for _, on := range []bool{false, true} {
		f.Fullscreen = on
		want := map[bool]string{false: "FULLSCREEN: OFF", true: "FULLSCREEN: ON"}[on]
		b := buttonByID(h.MenuButtons(f), "menu:fullscreen")
		if b.Label != want || !MenuBox.Contains(b.Rect.X+b.Rect.W-1, b.Rect.Y+b.Rect.H-1) {
			t.Fatalf("fullscreen button = %+v", b)
		}
	}
	if _, ok := hasIntent(clickOn(t, h, f, "menu:fullscreen"), IntentFullscreen); !ok {
		t.Fatal("FULLSCREEN does not ask for fullscreen")
	}
	if h.Overlay() != OverlayNone {
		t.Fatal("a menu choice should close the menu")
	}
}

func TestMenuButtonsDoNotOverlap(t *testing.T) {
	f := fullFrame(t)
	h := New(nil, Backend{})
	buttons := h.MenuButtons(f)
	for i, a := range buttons {
		for _, b := range buttons[i+1:] {
			if Overlaps(a.Rect, b.Rect) {
				t.Fatalf("%s overlaps %s", a.ID, b.ID)
			}
		}
	}
}
