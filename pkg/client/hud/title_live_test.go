package hud

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestALiveOnlyTitleOffersLiveClusterSettingsAndQuit(t *testing.T) {
	ti := Title{LiveOnly: true, HasSave: true}
	ids := []string{}
	for _, b := range ti.Buttons() {
		ids = append(ids, b.ID)
	}
	if !slices.Equal(ids, []string{"title:live", TitleSettingsID, "title:quit"}) {
		t.Fatalf("a live-only title offers %v", ids)
	}
	for _, k := range []ebiten.Key{ebiten.KeyN, ebiten.Key1, ebiten.Key2, ebiten.KeyEnter} {
		if got := ti.Update(Input{MX: -10, MY: -10, Keys: []ebiten.Key{k}}); len(got) > 0 {
			t.Errorf("key %v on a live-only title leads to %v", k, got)
		}
	}
	if got := ti.Update(Input{MX: -10, MY: -10, Keys: []ebiten.Key{ebiten.Key3}}); len(got) != 1 || got[0].Kind != IntentLive {
		t.Errorf("3 on a live-only title gives %v, want LIVE CLUSTER", got)
	}
}

func TestTheFullTitleStillOffersEveryChoice(t *testing.T) {
	ti := Title{HasSave: true}
	ids := []string{}
	for _, b := range ti.Buttons() {
		ids = append(ids, b.ID)
	}
	if !slices.Equal(ids, []string{"title:new", "title:load", TitleLearn, "title:live", "title:quit"}) {
		t.Fatalf("the full title offers %v", ids)
	}
}
