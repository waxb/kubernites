package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/view"
)

func forumHamletSite() view.Site {
	site := landSite()
	site.Structures[0].Territory, site.Structures[1].Territory = 9, 6
	for y := 2; y <= 6; y++ {
		for x := 13; x <= 17; x++ {
			site.Tiles[y*site.Width+x] = view.TileForum
		}
	}
	site.Forum = &view.Forum{Codex: view.Point{X: 14, Y: 7}, Size: view.Point{X: 2, Y: 2}, Side: view.Point{Y: -1}, Lo: view.Point{X: 13, Y: 2}, Hi: view.Point{X: 17, Y: 6}}
	site.Structures = append(site.Structures, depotAt("parcel", "parcels", 14, 3))
	return site
}

func TestCrewHutsClusterBesideTheirDepotOnTheForum(t *testing.T) {
	site := forumHamletSite()
	depot := structureNamed(site, "depot-parcel")
	if _, owned := site.TerritoryAt(depot.X+1, depot.Y+1); owned {
		t.Fatal("the depot stands on a Keep's land, not on the Forum")
	}
	pods := crewPods("porters", "parcels", "lone-tower", 8)
	spots := hutsOf(t, site, pods)
	if len(spots) != len(pods) {
		t.Fatalf("housed %d of %d", len(spots), len(pods))
	}
	for _, p := range pods {
		at := spots[p.Name]
		assertHome(t, site, p.Name, at, "tower")
		if site.OnForum(at.X, at.Y) {
			t.Fatalf("%s built a hut on the Forum at %v", p.Name, at)
		}
		if gapTo(depot, at) > hutReach {
			t.Errorf("%s at %v strays from the depot on the Forum (gap %.1f)", p.Name, at, gapTo(depot, at))
		}
	}
}

func TestHutsOnAFarKeepFaceTheirDepot(t *testing.T) {
	site := forumHamletSite()
	depot, keep := structureNamed(site, "depot-parcel"), structureNamed(site, "stone")
	target := FootprintCentre(depot)
	home := Dist(FootprintCentre(keep), target)
	pods := crewPods("porters", "parcels", "stone-keep-2", 6)
	spots := hutsOf(t, site, pods)
	if len(spots) != len(pods) {
		t.Fatalf("housed %d of %d", len(spots), len(pods))
	}
	for _, p := range pods {
		at := spots[p.Name]
		assertHome(t, site, p.Name, at, "stone")
		if d := Dist(at.Center(), target); d >= home-1 {
			t.Errorf("%s at %v turns away from its depot (%.1f from it, the Keep %.1f)", p.Name, at, d, home)
		}
	}
}

func TestTheNearestLandIsRememberedUntilTheLandChanges(t *testing.T) {
	site := forumHamletSite()
	h := NewHamlet()
	h.Place(site, crewPods("porters", "parcels", "stone-keep-2", 2), 0)
	if len(h.Nears) != 1 {
		t.Fatalf("nearest land remembered %d times", len(h.Nears))
	}
	h.Covered(7, func(x, y int) bool { return false })
	if len(h.Nears) != 0 {
		t.Fatal("the nearest land outlived a change of cover")
	}
	keep := structureNamed(site, "stone")
	lo, hi := Footprint(keep)
	if _, _, ok := h.Nearest(site, view.Structure{ID: "nowhere"}, lo, hi); ok {
		t.Fatal("a Keep with no land has a nearest tile")
	}
}
