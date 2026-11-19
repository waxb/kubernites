package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func TestTheDoorPortersCardReseatsItWithAnOwnedClass(t *testing.T) {
	door := model.PodView{Namespace: "default", Name: "porter-lone-tower", Node: view.TowerName, Image: "porter:1.0", Static: true, Crew: "porters", State: model.HearthGuttering}
	f := fullFrame(t)
	f.State.Soulstones = []view.SoulstoneEntry{{Image: "porter:1.0", Owned: true}, {Image: "porter:1.1", Owned: true}, {Image: "porter:2.0"}}
	f.Selection = Selection{Kind: SelectHearth, ID: "default/porter-lone-tower", Pod: door}
	a, ok := ActionByKey(SelectionActions(f), keyReseat)
	if !ok || a.Disabled || len(a.Choices) != 1 || a.Choices[0].Act.Action != view.ActReseatDoor || a.Choices[0].Act.Image != "porter:1.1" {
		t.Fatalf("reseat = %+v", a)
	}
	crew := door
	crew.Static, crew.Name = false, "porters-7d9f-a"
	f.Selection = Selection{Kind: SelectHearth, ID: "default/porters-7d9f-a", Pod: crew}
	if _, ok := ActionByKey(SelectionActions(f), keyReseat); ok {
		t.Fatal("a crew's worker changes class through its crew, not RESEAT")
	}
	f.State.Soulstones = f.State.Soulstones[:1]
	f.Selection = Selection{Kind: SelectHearth, ID: "default/porter-lone-tower", Pod: door}
	if a, _ := ActionByKey(SelectionActions(f), keyReseat); !a.Disabled {
		t.Fatal("with one class owned there is nothing to re-seat to")
	}
}
