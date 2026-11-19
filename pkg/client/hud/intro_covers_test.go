package hud

import "testing"

func TestAHiddenIntroCardDoesNotHideTheCrewsList(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := calamityFrame(t, noDepot)
	f.Guide = Guide{Intro: &Intro{ID: "rail", Title: "Matters at hand", Try: "Glance at the rail.", Target: SurfaceRail, Via: ViaAck}}
	f = settled(h, f)
	f.Clock += troubleWait
	h.Update(Input{MX: -5, MY: -5}, f)
	if _, up := h.ShownCalamity(f); !up {
		t.Fatal("the trouble card is not up")
	}
	h.CrewsList = CrewsListState{Open: true, sel: selKey(f.Selection)}
	if len(CrewsRows(f)) == 0 {
		t.Fatal("the fixture has no crews to list: it proves nothing")
	}
	if h.IntroShown(f) {
		t.Fatal("an intro card landed beside a trouble card")
	}
	if !h.crewsShown(f) {
		t.Fatal("the crews list the player opened vanished behind an intro card that is not even drawn")
	}
}
