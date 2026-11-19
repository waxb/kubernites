package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const synthTTL = 40.0

func synthFreight(now float64, tier, per int) view.FreightView {
	last := float64(int(now/8)) * 8
	fv := view.FreightView{QuayCap: 400, Docking: view.DockingView{Ships: uint64(now/8) + 1, Last: last, LastCargo: float64(4 * per), Next: last + 8}}
	fv.Tier, fv.Ship = tier, ShipClassOf(tier)
	for k := range FreightKinds {
		kv := view.KindView{Kind: FreightKinds[k], Name: FreightKinds[k] + "s", Open: true, SizeMix: [4]float64{.4, .3, .2, .1}, TTL: synthTTL}
		for c := 0; c < 4; c++ {
			age := now - (last - float64(c)*8)
			if age < 0 || age >= synthTTL {
				continue
			}
			kv.Cohorts = append(kv.Cohorts, view.CohortView{Count: float64(per), Age: age, Remaining: synthTTL - age, TTL: synthTTL, Crowned: c == 1 && k == 0})
			kv.Waiting += float64(per)
		}
		fv.Kinds = append(fv.Kinds, kv)
		fv.Waiting += kv.Waiting
	}
	return fv
}

func harborScene(t testing.TB) (Scene, *Renderer) {
	sc, _ := sceneFor(t, .5)
	sc.Pods, sc.Transients, sc.Ghost = nil, nil, nil
	sc.ArrivalRate, sc.Queue, sc.GateOpen = 0, 3, true
	sc.Time, sc.RealmTime = 20, 20
	sc.Freight = synthFreight(20, 1, 6)
	return sc, NewRenderer(nil)
}

func cameraOn(sc Scene, r *Renderer) render.Camera {
	h := r.HarborOf()
	h.Replan(sc.Site)
	q := h.Plans[0].Quay
	return render.NewCamera(480, 60).CenterOn(q.X, q.Y, 960, 540)
}
