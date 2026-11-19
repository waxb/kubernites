package app

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/world"
)

func TestSelectionKindsAgreeBetweenTheWorldAndTheCard(t *testing.T) {
	if world.HitForeman != hud.SelectForeman || world.HitCrew != hud.SelectCrew || world.HitHearth != hud.SelectHearth {
		t.Fatal("a shift or banner picked in the world selects nothing on the card")
	}
}

func TestKeepChosenDropsAVanishedForeman(t *testing.T) {
	crews := []model.CrewChain{{Namespace: "default", Name: "porters", Foremen: []model.Foreman{{Namespace: "default", Name: "porters-7d9f"}}}}
	kept := []hud.Selection{{Kind: hud.SelectCrew, ID: "default/porters"}, {Kind: hud.SelectForeman, ID: "default/porters-7d9f"}, {Kind: "structure", ID: "x"}}
	for _, sel := range kept {
		if got := KeepChosen(sel, crews); got.Kind != sel.Kind || got.ID != sel.ID {
			t.Errorf("%+v was dropped", sel)
		}
	}
	if !KeepChosen(hud.Selection{Kind: hud.SelectForeman, ID: "default/porters-54cd"}, crews).Empty() {
		t.Error("a shift whose ReplicaSet is gone stays selected")
	}
}

func TestChosenFindsTheLevel(t *testing.T) {
	pods := []model.PodView{{Namespace: "default", Name: "porters-7d9f-a"}}
	if sel, ok := Chosen(hud.Intent{Target: hud.SelectHearth, Command: "default/porters-7d9f-a"}, pods); !ok || sel.Pod.Name != "porters-7d9f-a" || sel.Kind != world.HitHearth {
		t.Fatalf("hut = %+v", sel)
	}
	if _, ok := Chosen(hud.Intent{Target: hud.SelectHearth, Command: "default/gone"}, pods); ok {
		t.Fatal("a gone hut was chosen")
	}
	if sel, ok := Chosen(hud.Intent{Target: hud.SelectForeman, Command: "default/porters-7d9f"}, nil); !ok || sel.ID != "default/porters-7d9f" {
		t.Fatalf("foreman = %+v", sel)
	}
	if _, ok := Chosen(hud.Intent{Target: "nothing"}, nil); ok {
		t.Fatal("chose nothing")
	}
}
