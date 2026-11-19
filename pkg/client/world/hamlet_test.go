package world

import (
	"fmt"
	"testing"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

const hutReach = 6.0

func depotAt(kind, service string, x, y int) view.Structure {
	return view.Structure{ID: "depot-" + kind, Kind: view.DepotStructure(kind), X: x, Y: y, W: 3, H: 3, State: view.StateReady,
		Object: view.ObjectRef{Kind: ServiceKind, Name: service}}
}

func hamletSite() view.Site {
	site := landSite()
	site.Structures[0].Territory, site.Structures[1].Territory = 9, 6
	site.Structures = append(site.Structures, depotAt("parcel", "parcels", 15, 6), depotAt("ledger", "ledgers", 17, 24))
	return site
}

func crewPods(crew, depot, node string, n int) []model.PodView {
	out := make([]model.PodView, 0, n)
	for i := range n {
		out = append(out, model.PodView{Namespace: "default", Name: fmt.Sprintf("%s-7d9f-%02d", crew, i), Node: node, Sprite: "porter", Tier: 1,
			State: model.HearthReady, Owner: "ReplicaSet/" + crew + "-7d9f", Crew: crew, Depot: depot})
	}
	return out
}

func hutsOf(t *testing.T, site view.Site, pods []model.PodView) map[string]Point {
	t.Helper()
	h := NewHamlet()
	h.Place(site, pods, 0)
	return h.Spots()
}

func structureNamed(site view.Site, id string) view.Structure {
	for _, s := range site.Structures {
		if s.ID == id {
			return s
		}
	}
	return view.Structure{}
}

func gapTo(s view.Structure, p Point) float64 {
	lo, hi := Footprint(s)
	return FootprintGap(lo, hi, p.X, p.Y)
}

func assertHome(t *testing.T, site view.Site, name string, p Point, keep string) {
	t.Helper()
	if k, ok := site.TerritoryAt(p.X, p.Y); !ok || k.ID != keep {
		t.Errorf("%s at %v stands outside %s's land (%s)", name, p, keep, k.ID)
	}
	if !Homeable(site.Tile(p.X, p.Y)) || RoadLevel(site, p.X, p.Y) > RoadNone {
		t.Errorf("%s at %v stands on %v", name, p, site.Tile(p.X, p.Y))
	}
	if NearAnyStructure(site, p.X, p.Y) || InAnyShadow(site, p.X, p.Y) {
		t.Errorf("%s at %v crowds a building", name, p)
	}
}

func TestCrewHutsClusterAroundTheirDepotOnTheirKeepsLand(t *testing.T) {
	site := hamletSite()
	pods := crewPods("porters", "parcels", "lone-tower", 8)
	spots := hutsOf(t, site, pods)
	depot, keep := structureNamed(site, "depot-parcel"), structureNamed(site, "tower")
	if len(spots) != len(pods) {
		t.Fatalf("housed %d of %d", len(spots), len(pods))
	}
	seen := map[Point]string{}
	for _, p := range pods {
		at := spots[p.Name]
		assertHome(t, site, p.Name, at, "tower")
		if other, ok := seen[at]; ok {
			t.Fatalf("%s and %s share %v", p.Name, other, at)
		}
		seen[at] = p.Name
		if gapTo(depot, at) > hutReach || gapTo(depot, at) >= gapTo(keep, at)+2 {
			t.Errorf("%s at %v strays from the depot (gap %.1f, keep gap %.1f)", p.Name, at, gapTo(depot, at), gapTo(keep, at))
		}
	}
}

func TestHutsOnAnotherKeepHuddleAtThatKeep(t *testing.T) {
	site := hamletSite()
	pods := crewPods("porters", "parcels", "stone-keep-2", 5)
	spots := hutsOf(t, site, pods)
	keep := structureNamed(site, "stone")
	for _, p := range pods {
		at, ok := spots[p.Name]
		if !ok {
			t.Fatalf("%s is unhoused", p.Name)
		}
		assertHome(t, site, p.Name, at, "stone")
		if gapTo(keep, at) > hutReach {
			t.Errorf("%s at %v is far from its own Keep (gap %.1f)", p.Name, at, gapTo(keep, at))
		}
	}
}

func TestALocalDepotGathersItsCrewOnTheSameLand(t *testing.T) {
	site := hamletSite()
	pods := crewPods("clerks", "ledgers", "stone-keep-2", 4)
	spots := hutsOf(t, site, pods)
	depot := structureNamed(site, "depot-ledger")
	for _, p := range pods {
		at := spots[p.Name]
		assertHome(t, site, p.Name, at, "stone")
		if gapTo(depot, at) > hutReach {
			t.Errorf("%s at %v strays from the ledgers depot (gap %.1f)", p.Name, at, gapTo(depot, at))
		}
	}
}

func TestBareHutsStandByTheKeepDoor(t *testing.T) {
	site := hamletSite()
	keep := structureNamed(site, "tower")
	pods := []model.PodView{}
	for i := range 3 {
		pods = append(pods, model.PodView{Namespace: "default", Name: fmt.Sprintf("porter-summoned-%d", i), Node: "lone-tower", Sprite: "porter",
			State: model.HearthReady, Depot: "parcels"})
	}
	spots := hutsOf(t, site, pods)
	_, hi := Footprint(keep)
	for _, p := range pods {
		at := spots[p.Name]
		assertHome(t, site, p.Name, at, "tower")
		if gapTo(keep, at) > DoorRing1+1.5 || at.X+at.Y < hi.X+hi.Y-2 {
			t.Errorf("%s at %v is not by the door (gap %.1f)", p.Name, at, gapTo(keep, at))
		}
	}
}

func TestTheDoorHearthKeepsItsPorchHut(t *testing.T) {
	site := hamletSite()
	keep := structureNamed(site, "tower")
	door := model.PodView{Namespace: "default", Name: "porter-lone-tower", Node: "lone-tower", Sprite: "porter", State: model.HearthReady, Static: true}
	pods := append(crewPods("porters", "", "lone-tower", 10), door)
	want, ok := DoorHut(site, keep)
	if !ok {
		t.Fatal("no door hut on the test site")
	}
	if got := hutsOf(t, site, pods)[door.Name]; got != want {
		t.Errorf("door hut at %v, want %v", got, want)
	}
}

func TestHutsNeverJumpWhenNeighboursComeAndGo(t *testing.T) {
	site := hamletSite()
	h := NewHamlet()
	pods := crewPods("porters", "parcels", "lone-tower", 10)
	h.Place(site, pods, 0)
	before := h.Spots()
	next := append(append([]model.PodView{}, pods[:2]...), pods[5:]...)
	next = append(next, crewPods("fresh", "parcels", "lone-tower", 4)...)
	h.Place(site, next, 1)
	h.Place(site, next, 9)
	after := h.Spots()
	for _, p := range next[:7] {
		if after[p.Name] != before[p.Name] {
			t.Errorf("%s jumped from %v to %v", p.Name, before[p.Name], after[p.Name])
		}
	}
	for _, p := range pods[2:5] {
		if _, ok := after[p.Name]; ok {
			t.Errorf("%s left but its hut stayed", p.Name)
		}
	}
	if len(after) != len(next) {
		t.Errorf("spots = %d, want %d", len(after), len(next))
	}
}

func TestFewHutsKeepGardensBetweenThem(t *testing.T) {
	site := hamletSite()
	spots := hutsOf(t, site, crewPods("porters", "parcels", "lone-tower", 6))
	for a, pa := range spots {
		for b, pb := range spots {
			if a < b && chebyshev(view.Point(pa), view.Point(pb)) < 2 {
				t.Errorf("%s at %v and %s at %v stand wall to wall", a, pa, b, pb)
			}
		}
	}
}

func TestOnlyScheduledHearthsOnKnownKeepsGetHuts(t *testing.T) {
	site := hamletSite()
	pods := []model.PodView{
		{Namespace: "default", Name: "waiting", State: model.HearthPending, Owner: "ReplicaSet/x"},
		{Namespace: "kube-system", Name: "etcd-lone-tower", Node: "lone-tower", State: model.HearthReady, System: true},
		{Namespace: "default", Name: "lost", Node: "no-such-keep", State: model.HearthReady},
	}
	if spots := hutsOf(t, site, pods); len(spots) != 0 {
		t.Errorf("spots = %v", spots)
	}
}

func TestHutsAvoidQuaysAndYards(t *testing.T) {
	site := coastSite(40, 32, 8)
	site.Havens = []view.Haven{westHaven(3, true)}
	site.Structures = []view.Structure{{ID: "tower", Kind: view.StructKeep, Variant: "tower", X: 12, Y: 12, W: 3, H: 3, Territory: 9,
		State: view.StateReady, Object: view.ObjectRef{Kind: "Node", Name: "lone-tower"}}}
	spots := hutsOf(t, site, crewPods("porters", "", "lone-tower", 30))
	plan := PlanHaven(site, site.Havens[0])
	blocked := map[Point]bool{}
	for _, d := range plan.Deck {
		blocked[d] = true
	}
	for _, y := range site.Havens[0].Yards() {
		for dy := range 2 {
			for dx := range 2 {
				blocked[Point{y.X + dx, y.Y + dy}] = true
			}
		}
	}
	for name, p := range spots {
		if blocked[p] {
			t.Errorf("%s stands on the quay at %v", name, p)
		}
	}
}

func TestPlacementIsCachedUntilTheLandChanges(t *testing.T) {
	site := hamletSite()
	h := NewHamlet()
	pods := crewPods("porters", "parcels", "lone-tower", 6)
	h.Place(site, pods, 0)
	h.Place(site, pods, 1)
	if h.Builds != 1 {
		t.Fatalf("builds = %d after a quiet frame", h.Builds)
	}
	moved := site
	moved.Structures = append(append([]view.Structure{}, site.Structures...), view.Structure{ID: "well", Kind: view.StructCodex, X: 30, Y: 30, W: 2, H: 2})
	h.Place(moved, pods, 2)
	if h.Builds != 2 {
		t.Fatalf("builds = %d after a new building", h.Builds)
	}
	crowded := site
	crowded.Structures = append(append([]view.Structure{}, site.Structures...), view.Structure{ID: "hall", Kind: view.StructCodex, X: 3, Y: 3, W: 2, H: 2})
	spot := h.Spots()[pods[0].Name]
	crowded.Structures[len(crowded.Structures)-1].X, crowded.Structures[len(crowded.Structures)-1].Y = spot.X, spot.Y
	h.Place(crowded, pods, 3)
	if got := h.Spots()[pods[0].Name]; got == spot {
		t.Errorf("a hut under a new building must move, still at %v", got)
	}
	assertHome(t, crowded, pods[0].Name, h.Spots()[pods[0].Name], "tower")
}

func TestHomelessHearthsWaitForLandAndTakeItWhenItFrees(t *testing.T) {
	site := hamletSite()
	site.Structures[1].Territory = 3
	pods := crewPods("porters", "", "stone-keep-2", 40)
	h := NewHamlet()
	h.Place(site, pods, 0)
	housed := len(h.Spots())
	if housed == 0 || housed >= len(pods) {
		t.Fatalf("housed %d of %d on a small Keep", housed, len(pods))
	}
	h.Place(site, pods, 1)
	if len(h.Spots()) != housed || len(h.Homeless) != len(pods)-housed {
		t.Fatalf("homeless %d, housed %d", len(h.Homeless), len(h.Spots()))
	}
	left := []model.PodView{}
	gone := 0
	for _, p := range pods {
		if _, ok := h.SpotList[p.Name]; ok && gone < 3 {
			gone++
			continue
		}
		left = append(left, p)
	}
	h.Place(site, left, 2)
	if len(h.Spots()) != housed-3 {
		t.Fatalf("the fading ruins still hold their land: %d huts", len(h.Spots()))
	}
	h.Place(site, left, 2+FadeSeconds+.1)
	if len(h.Spots()) != housed {
		t.Errorf("freed land houses the waiting: %d huts, want %d", len(h.Spots()), housed)
	}
}

func BenchmarkHamletPlace300(b *testing.B) {
	site := hamletSite()
	pods := append(crewPods("porters", "parcels", "lone-tower", 150), crewPods("clerks", "ledgers", "stone-keep-2", 150)...)
	h := NewHamlet()
	h.Place(site, pods, 0)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Place(site, pods, float64(i)/60)
	}
}
