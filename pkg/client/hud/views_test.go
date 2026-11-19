package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func depotFrame() Frame {
	depot := view.Structure{ID: "depot-1", Kind: view.StructGuildhall, State: view.StateReady, Label: "Guildhall",
		Object: view.ObjectRef{Kind: "Service", Name: "parcels"}}
	crew := model.CrewChain{Namespace: "default", Name: "porters", Depot: "parcels", Desired: 2, Ready: 2,
		Foremen: []model.Foreman{{Namespace: "default", Name: "porters-abc", Desired: 2, Newest: true, Pods: []string{"porters-abc-1"}}}}
	return Frame{Selection: Selection{Kind: "structure", ID: depot.ID, Structure: depot}, Crews: []model.CrewChain{crew}, Land: true}
}

func viewIDs(buttons []Button) map[string]Button {
	out := map[string]Button{}
	for _, b := range buttons {
		out[b.ID] = b
	}
	return out
}

func TestSelectingADepotShowsOnlyItsCard(t *testing.T) {
	h := New(nil, Backend{})
	f := depotFrame()
	if !h.ChainShown(f) || h.depotBoardOn(f, f.Crews[0]) || h.LandShown(f) {
		t.Fatal("selecting a depot opens the crew board behind it, not the depot board, and tints no land")
	}
	ids := viewIDs(h.CommandButtons(f))
	if _, ok := ids[ChainButton]; !ok {
		t.Fatalf("no BOARD on the depot's grid: %v", ids)
	}
	if _, ok := ids[LandButton]; !ok {
		t.Fatalf("no LAND on the depot's grid: %v", ids)
	}
	h.Press(ChainButton, f)
	if !h.depotBoardOn(f, f.Crews[0]) || !viewIDs(h.CommandButtons(f))[ChainButton].Selected {
		t.Fatal("the stall cell did not open the depot board")
	}
	h.Press(ChainButton, f)
	if h.depotBoardOn(f, f.Crews[0]) {
		t.Fatal("a second CHAIN did not close it")
	}
	h.Press(LandButton, f)
	if !h.LandShown(f) {
		t.Fatal("LAND did not tint the land")
	}
	h.Press(LandButton, f)
	if h.LandShown(f) {
		t.Fatal("a second LAND did not clear the tint")
	}
}

func TestTheChainStaysOpenWhileYouWalkItsLevels(t *testing.T) {
	h := New(nil, Backend{})
	f := depotFrame()
	h.Press(ChainButton, f)
	crew := f
	crew.Selection = Selection{Kind: SelectCrew, ID: "default/porters"}
	if !h.ChainShown(crew) {
		t.Fatal("picking the crew row closed the chain")
	}
	if _, ok := viewIDs(h.CommandButtons(crew))["crew:describe"]; !ok {
		t.Fatal("the crew's grid carries its orders")
	}
	other := f
	other.Selection = Selection{Kind: "structure", ID: "keep-1", Structure: view.Structure{ID: "keep-1", Kind: view.StructKeep}}
	other.Crews = nil
	if h.ChainShown(other) || h.LandShown(other) {
		t.Fatal("another selection inherited the open views")
	}
	h.Press(CrewPrefix+":close", f)
	if h.depotBoardOn(f, f.Crews[0]) {
		t.Fatal("the chain's X did not close it")
	}
}

func TestNoLandButtonWithoutLand(t *testing.T) {
	h := New(nil, Backend{})
	f := depotFrame()
	f.Land, f.Crews = false, nil
	for _, b := range h.CommandButtons(f) {
		if b.ID == LandButton || b.ID == ChainButton {
			t.Fatalf("a card with no land or crew offers %s", b.ID)
		}
	}
	h.Press(LandButton, f)
	if h.LandShown(f) {
		t.Fatal("LAND tinted land the card does not have")
	}
}

func TestTheGuideOpensTheChainBeforeItsButton(t *testing.T) {
	h := New(nil, Backend{})
	f := crewFrame(Selection{Kind: SelectHearth, ID: "default/porters-7d9f-a", Pod: crewPod("porters-7d9f-a")})
	if got := NavPath("crew:scale:up", h.NavState(f)); len(got) != 2 || got[0] != ChainButton || got[1] != "crew:scale:up" {
		t.Fatalf("path to SCALE with the chain shut = %v", got)
	}
	h.Press(ChainButton, f)
	if got := NavPath("crew:scale:up", h.NavState(f)); len(got) != 1 || got[0] != "crew:scale:up" {
		t.Fatalf("path to SCALE with the chain open = %v", got)
	}
	if got := NavPath("crew:default/porters", h.NavState(f)); len(got) == 0 || got[0] == ChainButton {
		t.Fatalf("a crew banner target walked through CHAIN: %v", got)
	}
}

func TestIdenticalToastsMergeWithACount(t *testing.T) {
	h := New(nil, Backend{})
	for range 3 {
		h.Toast("The ledgers wait", false)
	}
	h.Toast("Other", true)
	ts := h.Toasts()
	if len(ts) != 2 || ts[0].Text != "The ledgers wait" || ts[0].Count != 3 || ToastLabel(ts[0]) != "The ledgers wait (x3)" || ToastLabel(ts[1]) != "Other" {
		t.Fatalf("toasts %+v", ts)
	}
	h.Now += ToastLife + 1
	h.expireToasts()
	h.Toast("The ledgers wait", false)
	if ts := h.Toasts(); len(ts) != 1 || ts[0].Count != 1 {
		t.Fatalf("an expired toast still counted: %+v", ts)
	}
	if Counted("a", 1) != "a" || Counted("a", 0) != "a" || Counted("a", 4) != "a (x4)" {
		t.Fatal("counted text")
	}
}

func TestLeavingASelectionForgetsItsViews(t *testing.T) {
	h := New(nil, Backend{})
	f := depotFrame()
	h.Update(Input{}, f)
	h.Press(ChainButton, f)
	h.Press(LandButton, f)
	crew := f
	crew.Selection = Selection{Kind: SelectCrew, ID: "default/porters"}
	h.Update(Input{}, crew)
	if !h.ChainShown(crew) {
		t.Fatal("walking to the crew closed its chain")
	}
	h.Update(Input{}, Frame{})
	h.Update(Input{}, f)
	if h.depotBoardOn(f, f.Crews[0]) || h.LandShown(f) {
		t.Fatal("selecting the depot again reopened its views")
	}
}
