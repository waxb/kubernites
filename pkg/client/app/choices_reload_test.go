package app

import (
	"slices"
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/live/livetest"
	"github.com/waxb/kubernites/pkg/live/write"
)

func TestOpeningResetLiveChoicesShowsWhatAnotherWindowSaved(t *testing.T) {
	rig := connectAt(t, hud.LevelReadOnly, livetest.Options{})
	other := write.Remembered{Context: write.Context{Name: "other-window", Server: "https://other:6443", User: "u", Identity: "x"}, Level: write.Writes, Warned: write.Writes}
	mem, err := write.LoadMemory(rig.dir)
	if err != nil {
		t.Fatal(err)
	}
	mem.Keep(other)
	if err := write.SaveMemory(rig.dir, mem); err != nil {
		t.Fatal(err)
	}
	for _, it := range rig.s.Hud.Press("menu:livechoices", rig.s.Frame(true)) {
		if err := rig.g.RealmIntent(it); err != nil {
			t.Fatal(err)
		}
	}
	if rig.s.Hud.OverlayID != hud.OverlayLiveChoices {
		t.Fatalf("RESET LIVE CHOICES did not open: overlay %q", rig.s.Hud.OverlayID)
	}
	if !slices.ContainsFunc(rig.s.Hud.LiveChoices(), func(c hud.LiveChoice) bool { return c.Name == "other-window" }) {
		t.Fatalf("the list misses what another window saved: %+v", rig.s.Hud.LiveChoices())
	}
}
