package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func crowdedAcademy() []view.SoulstoneEntry {
	out := []view.SoulstoneEntry{}
	for _, tag := range []string{"1.0", "1.1", "2.0", "3.0", "latest"} {
		out = append(out, view.SoulstoneEntry{Image: "porter:" + tag, Name: "Porters, class of " + tag, Owned: tag != "latest", Visible: true, Cost: 100})
	}
	out = append(out, view.SoulstoneEntry{Image: "porter:2.1", Name: "Porters, class of 2.1", Owned: true, Visible: true, Cursed: true, Description: "Nobody examined them."})
	for _, tag := range []string{"1.0", "1.1", "2.0"} {
		out = append(out, view.SoulstoneEntry{Image: "clerk:" + tag, Name: "Clerks, class of " + tag, Owned: tag != "2.0", Visible: true, Cost: 100})
	}
	return out
}

func TestTheAcademyShowsTheClassTheStepNamesWithoutScrolling(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.State.Soulstones = crowdedAcademy()
	f.State.Objectives = []view.ObjectiveView{{ID: "crates", Progress: "ready: pull clerk:2.0 from the REGISTRY", Target: "cmd:soulvault", Button: view.VaultPrefix + "clerk:2.0"}}
	f.Guide.Target, f.Guide.Button = "cmd:soulvault", view.VaultPrefix+"clerk:2.0"
	if len(f.State.Soulstones) <= academyVisible() {
		t.Fatalf("the fixture fits on one page (%d rows): it proves nothing", academyVisible())
	}
	h.Open(OverlaySoulvault)
	if _, ok := HitID(h.AcademyButtons(f), "vault:clerk:2.0"); !ok {
		t.Fatalf("clerk:2.0 is not on the registry's first page: %v", buttonIDs(h.AcademyButtons(f)))
	}
	if goal := h.readyGoal(f); goal != "vault:clerk:2.0" {
		t.Errorf("inside the registry the guide points at %q, not the class it names", goal)
	}
	if w := h.WalkReady(f); !w.OK() {
		t.Errorf("the ready line cannot be followed inside the registry: %+v", w)
	}
}

func TestARumouredClassIsNotCalledEndowed(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := fullFrame(t)
	f.State.Soulstones = []view.SoulstoneEntry{{Image: "porter:2.1", Name: "Porters, class of 2.1", Owned: true, Visible: true, Cursed: true}}
	for _, b := range h.AcademyButtons(f) {
		if b.ID == "vault:owned" && (b.Label == "ENDOWED" || b.Selected) {
			t.Errorf("the rumoured porter:2.1 reads %q as if it had been pulled", b.Label)
		}
	}
}
