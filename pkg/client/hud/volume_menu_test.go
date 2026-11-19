package hud

import "testing"

func TestTheMenuSetsMasterMusicAndSoundsVolume(t *testing.T) {
	f := fullFrame(t)
	f.Volumes = Volumes{Master: 1, Music: 0.6, Sounds: 0.8}
	h := New(nil, Backend{})
	h.Open(OverlayMenu)
	buttons := h.MenuButtons(f)
	for id, want := range map[string]string{"menu:vol:master": "MASTER 100%", "menu:vol:music": "MUSIC 60%", "menu:vol:sounds": "SOUNDS 80%"} {
		if b := buttonByID(buttons, id); b.Label != want {
			t.Errorf("%s label = %q, want %q", id, b.Label, want)
		}
	}
	for _, id := range []string{"menu:vol:master:-", "menu:vol:master:+", "menu:vol:music:-", "menu:vol:music:+", "menu:vol:sounds:-", "menu:vol:sounds:+"} {
		b := buttonByID(buttons, id)
		if b.ID == "" || !MenuBox.Contains(b.Rect.X+b.Rect.W-1, b.Rect.Y+b.Rect.H-1) {
			t.Errorf("%s = %+v, not inside the menu", id, b)
		}
	}
	it, ok := hasIntent(clickOn(t, h, f, "menu:vol:music:+"), IntentVolume)
	if !ok || it.Command != "music:+" {
		t.Fatalf("MUSIC + asks for %+v", it)
	}
	if h.Overlay() != OverlayMenu {
		t.Fatal("the menu stays open while the player sets a volume")
	}
}
