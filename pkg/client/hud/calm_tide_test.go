package hud

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/view"
)

func TestACalmRealmShowsNoRushTide(t *testing.T) {
	h := New(nil, Backend{})
	f := fullFrame(t)
	f.State.Reveals = append(slices.Clone(f.State.Reveals), SiegeReveal)
	f.State.Objectives = []view.ObjectiveView{{ID: CrewServesObjective, Done: true}}
	f.State.CalmRealm = true
	if _, ok := HitID(h.CommandButtons(f), "cmd:siege"); ok {
		t.Fatal("a calm realm shows the surge button")
	}
	h.Update(Input{Keys: []ebiten.Key{ebiten.KeyG}}, f)
	if h.Overlay() == OverlaySiege {
		t.Fatal("G opens the surge panel in a calm realm")
	}
	f.State.CalmRealm = false
	if _, ok := HitID(h.CommandButtons(f), "cmd:siege"); !ok {
		t.Fatal("the surge button never comes back once the calm is switched off")
	}
}
