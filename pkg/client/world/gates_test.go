package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func crampedSite() view.Site {
	site := hamletSite()
	site.Structures[1].Territory = 4
	return site
}

func landFor(t *testing.T, site view.Site, keep string) int {
	t.Helper()
	h := NewHamlet()
	h.Place(site, nil, 0)
	return len(h.list(site, h.areas[keep]))
}

func TestACrowdedKeepPacksHutsTightlyBeforeTurningAnyoneAway(t *testing.T) {
	site := crampedSite()
	room := landFor(t, site, "stone")
	if room < 4 {
		t.Fatalf("the cramped Keep has %d tiles", room)
	}
	pods := crewPods("porters", "", "stone-keep-2", room)
	h := NewHamlet()
	h.Place(site, pods, 0)
	spots := h.Spots()
	if len(spots) != room || len(h.Homeless) != 0 {
		t.Fatalf("housed %d of %d with %d tiles of land (homeless %d)", len(spots), len(pods), room, len(h.Homeless))
	}
	tight := false
	for a, pa := range spots {
		assertHome(t, site, a, pa, "stone")
		for b, pb := range spots {
			if a != b && pa == pb {
				t.Fatalf("%s and %s share %v", a, b, pa)
			}
			tight = tight || a < b && chebyshev(view.Point(pa), view.Point(pb)) == 1
		}
	}
	if !tight {
		t.Error("a full Keep packs its huts wall to wall")
	}
	for name, s := range h.SpotList {
		if s.Sown && h.occAt(s.Garden) == occGarden {
			t.Errorf("%s keeps a garden on a full Keep", name)
		}
	}
}

func TestHouseholdsWithoutLandWaitAtTheirKeepsGate(t *testing.T) {
	site := crampedSite()
	room := landFor(t, site, "stone")
	pods := crewPods("porters", "", "stone-keep-2", room+3)
	h := NewHamlet()
	h.Place(site, pods, 0)
	gates := h.Gates(site, pods)
	keep := structureNamed(site, "stone")
	if len(gates) != 1 || gates[0].Count != 3 || gates[0].Node != "stone-keep-2" || gates[0].At != GateTile(keep).Center() {
		t.Fatalf("gates = %+v", gates)
	}
	if _, hutted := h.SpotList[gates[0].First.Name]; hutted {
		t.Errorf("the first waiting household %s has a hut", gates[0].First.Name)
	}
	for name, p := range h.Spots() {
		if p == GateTile(keep) {
			t.Errorf("%s built on the gate", name)
		}
	}
	homes := withGates(h.Homes, pods, h, gates)
	waiting := 0
	for _, p := range pods {
		if _, hutted := h.SpotList[p.Name]; hutted {
			continue
		}
		waiting++
		if Dist(homes[p.Name], gates[0].At) > gateJitter {
			t.Errorf("%s waits %.1f from the gate", p.Name, Dist(homes[p.Name], gates[0].At))
		}
	}
	if waiting != 3 || GateLabel(3) != "+3" {
		t.Errorf("waiting = %d", waiting)
	}
	if NewHamlet().Gates(site, pods) != nil {
		t.Error("nobody waits before anyone was placed")
	}
}

func TestTheGateMarkerSelectsTheFirstWaitingHouseholdAndItsPeopleWalkFromThere(t *testing.T) {
	site := crampedSite()
	room := landFor(t, site, "stone")
	pods := crewPods("porters", "", "stone-keep-2", room+2)
	sc := Scene{Site: site, Pods: pods, Time: 1, Queue: 2, ArrivalRate: 1, Metrics: view.MetricsSnapshot{Pods: map[string]view.PodMetrics{}}}
	for _, p := range pods {
		sc.Metrics.Pods[PodKey(p)] = view.PodMetrics{ServedRate: 0}
	}
	gate := GateTile(structureNamed(site, "stone")).Center()
	cam := render.NewCamera(480, 60).CenterOn(gate.X, gate.Y, ViewWidth, ViewHeight)
	cam.Zoom = 2
	r := NewRenderer(gfxLibrary())
	dst := ebiten.NewImage(ViewWidth, ViewHeight)
	defer dst.Deallocate()
	r.Draw(dst, cam, sc)
	if len(r.gates) != 1 || r.gates[0].Count < 2 || r.gates[0].Count != len(pods)-len(r.hamlet.SpotList) {
		t.Fatalf("gates = %+v with %d huts for %d households", r.gates, len(r.hamlet.SpotList), len(pods))
	}
	var marker Hit
	for _, h := range r.HitList {
		if h.Kind == HitSignpost {
			marker = h
		}
	}
	if marker.Pod.Name != r.gates[0].First.Name {
		t.Fatalf("the marker selects %q, want %q", marker.Pod.Name, r.gates[0].First.Name)
	}
	if got, ok := r.Pick(marker.Rect.X+marker.Rect.W/2, marker.Rect.Y+2); !ok || got.Kind != HitSignpost {
		t.Errorf("picking the marker gives %+v", got)
	}
	for _, f := range figuresOf(r.Crowd(sc, RouteOf(sc.Site)), r.gates[0].First.Name) {
		if Dist(f.Pose.Pos, gate) > 1.5 {
			t.Errorf("a waiting household's person rests %.1f from the gate", Dist(f.Pose.Pos, gate))
		}
	}
	for _, it := range r.HearthItems(cam, sc) {
		if it.Hit != nil && it.Hit.Kind == HitHearth && it.Hit.Pod.Name == r.gates[0].First.Name {
			t.Error("a waiting household has no hut")
		}
	}
	r.drawGateLabels(dst, cam, sc)
}
