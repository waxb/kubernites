package hud

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func depotBoardFrame() Frame {
	f := boardFrame()
	f.Selection = Selection{Kind: "structure", ID: "depot-1", Structure: view.Structure{ID: "depot-1", Kind: view.StructGuildhall, State: view.StateReady,
		Label: "Guildhall", Object: view.ObjectRef{Kind: "Service", Name: "parcels"}}}
	f.Pods[2].Ready, f.Pods[2].State = false, model.HearthKindling
	return f
}

func TestTheDepotFanLightsItsReadyEndpoints(t *testing.T) {
	f := depotBoardFrame()
	h := New(nil, Backend{})
	h.Press(ChainButton, f)
	if _, ok := h.BoardView(f); ok {
		t.Fatal("a depot selection opened the crew board")
	}
	d, ok := h.DepotView(f)
	if !ok || d.ReadyCount() != 2 || len(d.Pods) != 3 || len(d.Huts) != 3 {
		t.Fatalf("depot view ok=%v ready=%d pods=%d", ok, d.ReadyCount(), len(d.Pods))
	}
	if _, ok := crewButtonMap(h.CrewButtons(f))[CrewBoardButton]; ok {
		t.Fatal("the depot board keeps only its fan: CREW moved to the grid")
	}
	if gridCell(t, h, f, CrewBoardButton) != 3 {
		t.Fatalf("the depot's grid has no CREW: %v", gridIDs(h, f))
	}
	lanes := d.Lanes()
	if len(lanes) != 4 || !lanes[0].Lit {
		t.Fatalf("lanes = %+v", lanes)
	}
	for i, lane := range lanes[1:] {
		if lane.Pod != d.Pods[i].Name || lane.Lit != d.Pods[i].Ready {
			t.Errorf("lane %+v for %s ready=%v", lane, d.Pods[i].Name, d.Pods[i].Ready)
		}
	}
	h.Reduce = true
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.DrawCrewCard(dst, f)
	if got := h.Press(CrewBoardButton, f); len(got) == 0 || got[0].Kind != IntentChoose || got[0].Target != SelectCrew {
		t.Fatalf("CREW BOARD = %+v", got)
	}
}

func TestTheDepotBoardSaysWhatWaits(t *testing.T) {
	f := depotBoardFrame()
	h := New(nil, Backend{})
	h.Press(ChainButton, f)
	d, _ := h.DepotView(f)
	if got := DepotInfo(d); got != "PARCELS: 14 WAIT ON THE QUAY, 1.2/S CARRIED OF 2.0" {
		t.Fatalf("info = %q", got)
	}
	d.Known = false
	if got := DepotInfo(d); got == "" {
		t.Fatal("an unknown flow says nothing")
	}
	many := depotBoardFrame()
	for i := range 9 {
		name := "porters-7d9f-x" + string(rune('a'+i))
		many.Pods = append(many.Pods, model.PodView{Namespace: "default", Name: name, State: model.HearthReady, Ready: true, Crew: "porters"})
		many.Crews[0].Foremen[0].Pods = append(many.Crews[0].Foremen[0].Pods, name)
	}
	dm, _ := h.DepotView(many)
	if rects, full := dm.HutRects(); full || len(rects) != 12 {
		t.Fatalf("12 endpoints drawn full=%v n=%d", full, len(rects))
	}
	dst := ebiten.NewImage(Width, Height)
	defer dst.Deallocate()
	h.DrawCrewCard(dst, many)
}
