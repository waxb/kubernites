package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func bareForumSite() view.Site {
	site := forumHamletSite()
	site.Structures = site.Structures[:3]
	return site
}

func TestTheKerbRingsTheForumWithPostsAtItsCorners(t *testing.T) {
	site := bareForumSite()
	plan := PlanKerb(site)
	if len(plan.pieces) != 20 {
		t.Fatalf("a 5 by 5 Forum has %d kerb stones, want 20", len(plan.pieces))
	}
	corners := map[Vec]bool{{12.5, 1.5}: false, {17.5, 1.5}: false, {12.5, 6.5}: false, {17.5, 6.5}: false}
	for _, p := range plan.posts {
		if _, ok := corners[p.at]; ok {
			corners[p.at] = true
		}
	}
	for at, seen := range corners {
		if !seen {
			t.Errorf("no post at the corner %v", at)
		}
	}
	if len(plan.posts) <= len(corners) {
		t.Errorf("only %d posts: the long sides have none", len(plan.posts))
	}
}

func TestTheKerbOpensForRoadsAndRunsUnderNoBuilding(t *testing.T) {
	site := bareForumSite()
	site.Tiles[4*site.Width+12] = view.TileRoad
	site.Structures = append(site.Structures, depotAt("parcel", "parcels", 15, 5))
	plan := PlanKerb(site)
	for _, k := range plan.pieces {
		if k.mid == (Vec{12.5, 4}) {
			t.Fatal("the kerb blocks the road into the Forum")
		}
		if x, y := int(k.mid.X+.5), int(k.mid.Y+.5); k.mid.Y > 6 && x >= 15 && x <= 17 && y <= 8 {
			t.Fatalf("a kerb stone runs under the depot at %v", k.mid)
		}
	}
	posts := map[Vec]bool{}
	for _, p := range plan.posts {
		posts[p.at] = true
	}
	if !posts[Vec{12.5, 3.5}] || !posts[Vec{12.5, 4.5}] {
		t.Errorf("the road's opening has no posts on both sides: %v", plan.posts)
	}
}

func TestNoBorderStoneStandsOnOrFacesTheForum(t *testing.T) {
	site := bareForumSite()
	b := BordersOf(site)
	for _, m := range b.Marks {
		if site.OnForum(m.X, m.Y) {
			t.Fatalf("a border stone stands on the Forum at %d,%d", m.X, m.Y)
		}
	}
	for _, a := range b.Areas {
		for _, p := range a.Border {
			wild := false
			for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if k, ok := b.At(p.X+d[0], p.Y+d[1]); !site.OnForum(p.X+d[0], p.Y+d[1]) && (!ok || b.Areas[k].Keep != a.Keep) {
					wild = true
				}
			}
			if !wild {
				t.Fatalf("%s's border at %v faces only the Forum", a.Keep, p)
			}
		}
	}
}

func TestTheRendererDrawsTheKerbOnce(t *testing.T) {
	site := bareForumSite()
	r := NewRenderer(nil)
	cam := render.NewCamera(ViewWidth/2, ViewHeight/2)
	cam.X, cam.Y = (15-4)*16, (15+4)*8
	items := r.forumItems(cam, Scene{Site: site})
	if len(items) == 0 {
		t.Fatal("the Forum drew no kerb")
	}
	first := r.decor
	r.forumItems(cam, Scene{Site: site})
	if r.decor != first {
		t.Fatal("the kerb was planned again for the same Forum")
	}
	far := cam
	far.X += 5000
	if n := len(r.forumItems(far, Scene{Site: site})); n != 0 {
		t.Fatalf("%d kerb stones drawn off screen", n)
	}
}

func TestWaitingPodsCampOnTheForumEvenWhenBuildingsCrowdIt(t *testing.T) {
	site := bareForumSite()
	site.Structures = append(site.Structures, depotAt("parcel", "parcels", 13, 2), depotAt("crate", "crates", 16, 2), depotAt("ledger", "ledgers", 13, 5))
	spots := freeForum(site, maxCamps)
	if len(spots) == 0 {
		t.Fatal("no camp spot on a crowded Forum")
	}
	for _, v := range spots {
		x, y := int(v.X), int(v.Y)
		if !site.OnForum(x, y) || Occupied(site, x, y) {
			t.Fatalf("a camp at %v stands off the open Forum", v)
		}
	}
	for i := 0; i < 3; i++ {
		v := PendingSpot(site, i)
		if !site.OnForum(int(v.X+.5), int(v.Y+.5)) {
			t.Fatalf("waiting Pod %d camps off the Forum at %v", i, v)
		}
	}
}

func TestTheForumTintsAndKnowsItsServices(t *testing.T) {
	site := forumHamletSite()
	tints := ForumTints(site, TintGood)
	if len(tints) != 25 {
		t.Fatalf("%d Forum tints, want 25", len(tints))
	}
	for _, tint := range tints {
		if !site.OnForum(tint.X, tint.Y) || tint.Kind != TintGood {
			t.Fatalf("tint %+v", tint)
		}
	}
	if !OnTheForum(site, structureNamed(site, "depot-parcel")) || OnTheForum(site, structureNamed(site, "tower")) {
		t.Fatal("OnTheForum mistakes the depot or the tower")
	}
}

func TestADepotAnchorFallsBackWhenTheKeepHasNoLand(t *testing.T) {
	site := forumHamletSite()
	h := NewHamlet()
	h.Place(site, nil, 0)
	fallback := hutAnchor{ring0: gateRing0}
	if got := h.depotAnchor(site, view.Structure{ID: "nowhere"}, structureNamed(site, "depot-parcel"), fallback); got != fallback {
		t.Fatalf("a landless Keep anchors at %+v", got)
	}
}
