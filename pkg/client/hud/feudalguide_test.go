package hud

import "testing"

func feudalIntroFrame(t *testing.T, id, target, via string) Frame {
	t.Helper()
	f := caesarFrame(t)
	f.Guide = Guide{Staged: true, Introduced: []string{SurfaceMinimap, SurfaceGate, SurfaceTidings, MeterGold},
		Intro: &Intro{ID: id, Title: id, Try: "Look.", Target: target, Via: via}}
	return f
}

func TestEveryFeudalIntroHasAHoleOnScreen(t *testing.T) {
	for _, c := range []struct{ id, target string }{{"overlays", SurfaceOverlays}, {"qos", SurfaceQoS}, {"codex-bloat", MeterCodex}} {
		f := feudalIntroFrame(t, c.id, c.target, ViaAck)
		h := New(nil, (&fakeRunner{}).backend())
		hole, ok := h.IntroHole(f)
		if !ok || !OnScreen(hole) {
			t.Errorf("the %s intro dims the whole screen with no hole: %+v %v", c.id, hole, ok)
		}
		if w := h.WalkGuide(f); !w.OK() {
			t.Errorf("the %s intro: %+v", c.id, w)
		}
	}
}

func TestTheMapButtonCyclesTheOverlays(t *testing.T) {
	f := feudalIntroFrame(t, "overlays", SurfaceOverlays, ViaAck)
	h := New(nil, (&fakeRunner{}).backend())
	b, ok := HitID(h.Buttons(f), LensCycleID)
	if !ok || !Within(b.Rect, Minimap) {
		t.Fatalf("the MAP button sits on the minimap: %+v %v", b, ok)
	}
	clickOn(t, h, f, LensCycleID)
	if h.Lens() == LensNone {
		t.Fatal("pressing MAP shows an overlay")
	}
	f.Guide.Intro = nil
	f.Guide.Introduced = []string{SurfaceMinimap}
	if _, ok := HitID(h.Buttons(f), LensCycleID); !ok {
		t.Error("the MAP button vanished when its card was set aside: a shown control stays")
	}
	if _, ok := HitID(New(nil, (&fakeRunner{}).backend()).Buttons(f), LensCycleID); ok {
		t.Error("the MAP button shows before its intro was ever shown")
	}
}
