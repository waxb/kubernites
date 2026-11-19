package hud

import "testing"

func TestTheCrewPanelHasALargeCloseButtonThatLeavesIt(t *testing.T) {
	f := boardFrame()
	h, v := openBoard(t, f)
	x := checkClose(t, "crew board", v.box, h.CrewButtons(f))
	out := h.Press(x.ID, f)
	if _, ok := hasIntent(out, IntentSelect); !ok {
		t.Fatalf("the crew board's X leaves the crew: %+v", out)
	}
}

func TestTheCrewsListHasALargeCloseButtonThatClosesIt(t *testing.T) {
	f := listFrame(t)
	h := New(nil, (&fakeRunner{}).backend())
	clickOn(t, h, f, CmdCrew)
	x := checkClose(t, "crews list", crewsRect(f, len(CrewsRows(f))), h.crewsButtons(f))
	h.Press(x.ID, f)
	if h.CrewsList.Open {
		t.Fatal("the list's X leaves it open")
	}
}
