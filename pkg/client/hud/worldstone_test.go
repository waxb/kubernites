package hud

import (
	"slices"
	"testing"
)

func TestTheStoneOnThePlazaOpensTheSpeakingStoneLikeEnter(t *testing.T) {
	f := stagedFrame(t)
	f.State.Reveals = slices.DeleteFunc(slices.Clone(f.State.Reveals), func(id string) bool { return id == "stone" })
	h := New(nil, (&fakeRunner{}).backend())
	if f.StoneStands() || h.OpenStoneFromWorld(f) != nil || h.Overlay() == OverlayStone {
		t.Fatal("the Stone answers before it is revealed")
	}
	f.State.Reveals = append(f.State.Reveals, "stone")
	if f.StoneStands() {
		t.Fatal("the Stone stands before its intro shows it")
	}
	f.Guide.Intro = &Intro{ID: "stone", Title: "The Speaking Stone", Target: SurfaceStone, Via: ViaOpen}
	if !f.StoneStands() {
		t.Fatal("the Stone does not stand while its intro teaches it")
	}
	intents := h.OpenStoneFromWorld(f)
	if h.Overlay() != OverlayStone {
		t.Fatalf("clicking the Stone opens %q", h.Overlay())
	}
	if it, ok := hasIntent(intents, IntentAction); !ok || it.Action != Ack("stone") {
		t.Fatalf("clicking the Stone does not count as trying it: %+v", intents)
	}
}
