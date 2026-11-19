package hud

import (
	"strings"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

func listFrame(t *testing.T) Frame {
	t.Helper()
	f := chooserFrame(t)
	f.Metrics.Pods = nil
	return f
}

func TestTheCrewsListNamesEveryCrewAndRaisesTheRest(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := listFrame(t)
	clickOn(t, h, f, CmdCrew)
	rows := CrewsRows(f)
	if len(rows) != 3 || !rows[0].Standing || rows[1].Standing || rows[1].trade.Crew != "clerks" {
		t.Fatalf("rows = %+v", rows)
	}
	for _, id := range []string{crewsCloseKey, CrewsKey + ":open:default/porters", CrewsKey + ":scale:default/porters:down",
		CrewsKey + ":scale:default/porters:up", CrewsKey + ":raise:clerks"} {
		if _, ok := HitID(h.Buttons(f), id); !ok {
			t.Errorf("the list has no %s", id)
		}
	}
	if len(h.CrewRects(f)) != 0 {
		t.Error("the list opened on top of a crew board")
	}
	h.Update(Input{MX: 0, MY: 0, Click: true}, f)
	clickOn(t, h, f, crewsCloseKey)
	if h.CrewsList.Open {
		t.Error("the close button leaves the list open")
	}
}

func TestTheCrewsListSpeaksBothModes(t *testing.T) {
	defer ui.SetNamer(nil)
	f := listFrame(t)
	c := f.Crews[0]
	if crewKindWord(c) != "CREW" || crewsTitle(f) != "THE CREWS" {
		t.Errorf("story words %q %q", crewKindWord(c), crewsTitle(f))
	}
	ui.SetNamer(names.Kubernetes(64))
	if crewKindWord(c) != "DEPLOYMENT" || crewsTitle(f) != "DEPLOYMENTS" {
		t.Errorf("NAMES words %q %q", crewKindWord(c), crewsTitle(f))
	}
}

func twoCrewsBehindOneStall(t *testing.T) Frame {
	t.Helper()
	f := depotBoardFrame()
	second := crewChain()
	second.Name = "porters-b"
	second.Foremen = []model.Foreman{{Namespace: "default", Name: "porters-b-1", Hash: "1", Revision: 1, Desired: 1, Newest: true, Image: "porter:1.0"}}
	f.Crews = append(f.Crews, second)
	return f
}

func TestOneCrewBehindAStallOpensItsBoardAndTwoOpenTheFilteredList(t *testing.T) {
	one := depotBoardFrame()
	h := New(nil, Backend{})
	h.Update(Input{}, one)
	if _, ok := h.BoardView(one); !ok || h.crewsShown(one) {
		t.Fatal("one crew behind the stall opens that crew's board, not the list")
	}
	two := twoCrewsBehindOneStall(t)
	h = New(nil, Backend{})
	h.Update(Input{}, two)
	if _, ok := h.BoardView(two); ok {
		t.Fatal("two crews behind the stall must not open one crew's board")
	}
	rows := CrewsRows(two)
	if !h.crewsShown(two) || len(rows) != 2 || rows[0].Chain.Name != "porters" || rows[1].Chain.Name != "porters-b" {
		t.Fatalf("the filtered list shows %+v", rows)
	}
	if !strings.Contains(crewsTitle(two), "parcels") {
		t.Errorf("the list names the stall: %q", crewsTitle(two))
	}
}

func TestARealmRowOpensThatCrewsBoard(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := listFrame(t)
	f.Selection = Selection{}
	b, ok := HitID(h.Buttons(f), guildKey+":porters")
	if !ok {
		t.Fatal("the YOUR REALM crew row is not clickable")
	}
	if !CardBox.Contains(b.Rect.X, b.Rect.Y) {
		t.Errorf("the row sits at %+v outside the card", b.Rect)
	}
	out := clickOn(t, h, f, guildKey+":porters")
	if it, ok := hasIntent(out, IntentChoose); !ok || it.Target != SelectCrew || it.Command != "default/porters" {
		t.Fatalf("a realm row opens its crew's board: %+v", out)
	}
}
