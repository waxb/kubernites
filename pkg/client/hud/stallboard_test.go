package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

func TestAStallOpensTheBoardOfTheCrewBehindIt(t *testing.T) {
	defer ui.SetNamer(nil)
	f := depotBoardFrame()
	h := New(nil, Backend{})
	h.Update(Input{}, f)
	v, ok := h.BoardView(f)
	if !ok || v.Chain.Name != "porters" {
		t.Fatalf("a stall opened no crew board (ok=%v)", ok)
	}
	if _, ok := h.DepotView(f); ok {
		t.Fatal("the stall shows the depot board and the crew board together")
	}
	if title, _ := boardTitle(v, f); title != "parcels depot -> crew porters" {
		t.Errorf("story title = %q", title)
	}
	ui.SetNamer(names.Kubernetes(64))
	if title, _ := boardTitle(v, f); title != "parcels Service -> porters Deployment" {
		t.Errorf("NAMES title = %q", title)
	}
}

func TestAStallBoardScalesTheDeploymentAndEchoesKubectl(t *testing.T) {
	f := depotBoardFrame()
	h := New(nil, Backend{})
	h.Update(Input{}, f)
	a, from := actionOf(h.Press(StepUpButton, f))
	if a.Action != view.ActScaleCrew || a.Name != "porters" || a.Replicas != crewChain().Desired+1 || from != FromCrewCard {
		t.Fatalf("+1 from the stall's board = %+v from %q", a, from)
	}
	if got := CommandOf(a); got != "kubectl scale deployment porters --replicas=4" {
		t.Fatalf("echo = %q", got)
	}
}

func TestTheStallCellSwitchesBetweenItsBoardAndTheCrewBoard(t *testing.T) {
	f := depotBoardFrame()
	h := New(nil, Backend{})
	h.Update(Input{}, f)
	h.Press(ChainButton, f)
	if _, ok := h.DepotView(f); !ok {
		t.Fatal("the stall cell does not show the depot board")
	}
	if _, ok := h.BoardView(f); ok {
		t.Fatal("the depot board and the crew board show together")
	}
	h.Press(ChainButton, f)
	if _, ok := h.BoardView(f); !ok {
		t.Fatal("pressing the stall cell again does not return to the crew board")
	}
}

func TestClosingAStallBoardLeavesTheStall(t *testing.T) {
	f := depotBoardFrame()
	h := New(nil, Backend{})
	h.Update(Input{}, f)
	if _, ok := hasIntent(h.Press(CrewPrefix+":close", f), IntentSelect); !ok {
		t.Fatal("closing the stall's crew board does not deselect the stall")
	}
}
