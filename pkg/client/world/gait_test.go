package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
)

func TestEachTradeMovesAtItsOwnPace(t *testing.T) {
	seen := map[float64]string{}
	for _, role := range gfx.TradeRoles {
		g := TradeGait(role)
		if other, dup := seen[g]; dup {
			t.Errorf("%s and %s share the gait %v", role, other, g)
		}
		seen[g] = role
	}
	if TradeGait("courier") <= TradeGait("porter") || TradeGait("hauler") >= TradeGait("clerk") || TradeGait("reeve") != 1 {
		t.Fatalf("gaits: courier %v porter %v clerk %v hauler %v", TradeGait("courier"), TradeGait("porter"), TradeGait("clerk"), TradeGait("hauler"))
	}
}

func strideGain(r *Renderer, p model.PodView) float64 {
	loop := NewRouteLoop([]Vec{{0, 0}, {40, 0}}, []Vec{{40, 0}, {0, 0}}, []Vec{{0, 0}, {0, 1}})
	sc := Scene{Time: 1}
	start := r.PacedOn(sc, p, Vec{}, loop, 1, true, false).clock
	for sc.Time < 6 {
		sc.Time += .1
	}
	return r.PacedOn(sc, p, Vec{}, loop, 1, true, false).clock - start
}

func TestACourierOutrunsAHaulerOnTheSameRoad(t *testing.T) {
	r := NewRenderer(nil)
	courier := strideGain(r, model.PodView{Namespace: "default", Name: "couriers-a", Sprite: "courier", State: model.HearthReady})
	hauler := strideGain(r, model.PodView{Namespace: "default", Name: "haulers-a", Sprite: "hauler", State: model.HearthReady})
	if courier <= hauler*1.5 {
		t.Fatalf("courier covered %v, hauler %v", courier, hauler)
	}
}

func TestASealwrightCarriesASealedBoxAndOthersTheirKind(t *testing.T) {
	for role, want := range map[string]string{"sealwright": "sealed", "clerk": "ledger", "porter": "parcel", "courier": "dispatch"} {
		kind, _ := TradeKindIndex(role)
		if got := LoadKind(role, kind); got != want {
			t.Errorf("%s carries %q, want %q", role, got, want)
		}
	}
}
