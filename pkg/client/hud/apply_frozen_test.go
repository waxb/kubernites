package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func TestOnACrewOpensTheCrewOfTheFrozenHut(t *testing.T) {
	f := boardFrame()
	clerks := crewChain()
	clerks.Name, clerks.Image, clerks.Container, clerks.Depot = "clerks", "clerk:1.0", "clerk", "ledgers"
	f.Crews = []model.CrewChain{crewChain(), clerks}
	hut := model.PodView{Namespace: "default", Name: "clerks-5d8f-x2kq9", Owner: "ReplicaSet/clerks-5d8f", State: model.HearthReady, Crew: "clerks"}
	f.Pods = append(f.Pods, hut)
	f.State.Calamities = []view.CalamityView{{ID: view.CalamityFrozen + ":" + hut.Name, Kind: view.CalamityFrozen, Object: "pod/default/" + hut.Name}}
	if c, ok := applyCrew(f, CrewModeProbes); !ok || c.Name != "clerks" {
		t.Fatalf("ON A CREW with a frozen clerk hut opens the %q", c.Name)
	}
	f.State.Calamities = []view.CalamityView{{ID: view.CalamityWarm + ":clerks", Kind: view.CalamityWarm}}
	if c, ok := applyCrew(f, CrewModeProbes); !ok || c.Name != "clerks" {
		t.Fatalf("ON A CREW with a warming clerks card opens the %q", c.Name)
	}
	f.State.Calamities = nil
	if c, ok := applyCrew(f, CrewModeProbes); !ok || c.Name != "porters" {
		t.Fatalf("ON A CREW with no card opens the first crew, not the %q", c.Name)
	}
}
