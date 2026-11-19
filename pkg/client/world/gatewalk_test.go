package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

func TestBoxesWalkQuayGateDepot(t *testing.T) {
	sc, r := harborScene(t)
	r.HarborOf().Replan(sc.Site)
	porter := model.PodView{Namespace: "default", Name: "porters-a", Sprite: "porter", State: model.HearthReady}
	if _, ok := gateOf(sc.Site); !ok {
		sc.Site.Structures = append(sc.Site.Structures, view.Structure{ID: "gate-1", Kind: view.StructGate, X: 30, Y: 24, W: 2, H: 2, State: view.StateReady})
	}
	gate, _ := gateOf(sc.Site)
	h, ok := r.HaulOf(sc, porter)
	if !ok || !h.gated || h.via != Door(gate) {
		t.Fatalf("the porter's haul skips the Gate: %+v %v", h, ok)
	}
	for i := range sc.Site.Structures {
		if sc.Site.Structures[i].Kind == view.StructGate {
			sc.Site.Structures[i].State = view.StateConstruction
		}
	}
	if h, _ := r.HaulOf(sc, porter); h.gated {
		t.Fatal("a Gate under construction sorts boxes")
	}
}

func TestAGateLoopCarriesThroughTheGateAndDeliversAtTheDepot(t *testing.T) {
	line := func(a, b Vec) trip { return walkTrip([]Vec{a, b}) }
	home, quay, gate, depot := Vec{0, 0}, Vec{10, 0}, Vec{10, 10}, Vec{0, 10}
	loop := NewGateLoop(line(home, quay), line(quay, gate), line(gate, depot), line(depot, home))
	carried, delivered := 0, 0
	for _, leg := range loop.Legs {
		if leg.carry {
			carried++
		}
		if leg.deliver {
			delivered++
			if leg.pts[len(leg.pts)-1] != depot {
				t.Errorf("the box is delivered at %v, not the depot", leg.pts[len(leg.pts)-1])
			}
		}
	}
	if carried != 2 || delivered != 1 {
		t.Fatalf("carried on %d legs, delivered on %d", carried, delivered)
	}
}
