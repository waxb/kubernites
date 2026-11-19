package world

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

func gatePlan(tier uint8) (view.Site, HavenPlan) {
	site := coastSite(40, 32, 8)
	return site, PlanHaven(site, westHaven(tier, true))
}

func TestBerthSpotsWalkThePierFromTheHead(t *testing.T) {
	_, p := gatePlan(1)
	spots := BerthSpots(p)
	if len(p.Pier) != 4 || len(spots) != 6 {
		t.Fatalf("a pier of %d planks has %d berth spots", len(p.Pier), len(spots))
	}
	first := p.Pier[len(p.Pier)-2]
	if Dist(spots[0], first) > postSide+1e-9 || Dist(spots[1], first) > postSide+1e-9 || spots[0] == spots[1] {
		t.Fatalf("the first berths stand at %v and %v, beside plank %v", spots[0], spots[1], first)
	}
	if Dist(spots[5], p.Pier[0]) > postSide+1e-9 {
		t.Fatalf("the last berth stands at %v, far from the first plank %v", spots[5], p.Pier[0])
	}
}

func TestPostsFillTheGateHarborFirstThenTheOpenPorts(t *testing.T) {
	gate := HavenPlan{Open: true, Pier: []Vec{{7, 16}, {6, 16}, {5, 16}}, Side: Vec{0, 1}, Out: Vec{-1, 0}}
	shut := HavenPlan{Open: false, Pier: gate.Pier, Side: gate.Side, Out: gate.Out}
	port := HavenPlan{Open: true, Pier: gate.Pier, Side: gate.Side, Out: gate.Out}
	posts := PostsPerHaven([]HavenPlan{gate, shut, port}, 6)
	if len(posts[0]) != 4 || len(posts[1]) != 0 || len(posts[2]) != 2 {
		t.Fatalf("posts per haven = %d, %d, %d", len(posts[0]), len(posts[1]), len(posts[2]))
	}
	if got := PostsPerHaven([]HavenPlan{gate}, 0); len(got[0]) != 0 {
		t.Fatalf("no berth open but %d posts", len(got[0]))
	}
	if got := PostsPerHaven([]HavenPlan{gate}, 99); len(got[0]) != 4 {
		t.Fatalf("a full pier holds %d posts", len(got[0]))
	}
}

func TestEveryOtherBerthMooresAShipOverWater(t *testing.T) {
	site, p := gatePlan(3)
	base := len(p.Berths)
	next := WithPosts(site, p, BerthSpots(p))
	if len(next.Posts) != 10 || len(next.Berths) != base+4 {
		t.Fatalf("%d posts moor %d ships beside the %d at the head", len(next.Posts), len(next.Berths)-base, base)
	}
	for _, b := range next.Berths[base:] {
		if !WetAt(site, b.Moor) || !WetLine(site, b.Lane, b.Moor) {
			t.Fatalf("a ship moors on land at %v", b.Moor)
		}
	}
	if len(p.Posts) != 0 || len(p.Berths) != base {
		t.Fatal("WithPosts changed the plan it was given")
	}
	dry := coastSite(40, 32, 8)
	for y := 0; y < dry.Height; y++ {
		for x := 0; x < 8; x++ {
			if y == 15 || y == 17 {
				dry.Tiles[y*dry.Width+x] = view.TileSand
			}
		}
	}
	if got := WithPosts(dry, p, BerthSpots(p)); len(got.Berths) != base {
		t.Fatalf("ships moor beside a pier walled in by sand: %d", len(got.Berths)-base)
	}
}

func TestOpenBerthsStandOnTheGateHarborWithLamps(t *testing.T) {
	site := coastSite(40, 32, 8)
	site.Havens = []view.Haven{westHaven(1, true)}
	r := NewRenderer(nil)
	r.HarborOf().Step(Scene{Site: site, Berths: 3})
	p := r.HarborOf().Plans[0]
	if len(p.Posts) != 3 || len(r.berthLamps()) != 3 {
		t.Fatalf("3 berths: %d posts, %d lamps", len(p.Posts), len(r.berthLamps()))
	}
	for _, l := range r.berthLamps() {
		if !l.lit || r.HarborOf().Lift(l.at) != art.HavenDeck {
			t.Fatalf("a berth lamp at %v is unlit or off the pier", l.at)
		}
	}
	r.HarborOf().Step(Scene{Site: site, Berths: 1})
	if got := len(r.HarborOf().Plans[0].Posts); got != 1 {
		t.Fatalf("closing berths left %d posts", got)
	}
}

func TestOnlyTheHarborsBuildingsAndSignOpenIt(t *testing.T) {
	site := coastSite(40, 32, 8)
	site.Havens = []view.Haven{westHaven(3, true)}
	r := NewRenderer(nil)
	r.HarborOf().Step(Scene{Site: site})
	p := r.HarborOf().Plans[0]
	opens := map[string]bool{}
	for _, prop := range p.Props {
		if hit := r.propHit(p, prop, 100, 100, 1); hit != nil {
			if hit.Kind != HitHaven || hit.ID != p.ID || hit.Rect.W == 0 || len(hit.Shapes) != 1 {
				t.Fatalf("the %s opens %+v", prop.Kind, hit)
			}
			opens[prop.Kind] = true
		}
	}
	if !opens["sign"] || !opens["warehouse"] || !opens["lighthouse"] || opens["crates"] || opens["bollard"] {
		t.Fatalf("props that open the harbor: %v", opens)
	}
	closed := p
	closed.Open = false
	if hit := r.propHit(closed, HavenProp{Kind: "sign"}, 0, 0, 1); hit != nil {
		t.Fatal("a closed haven's sign opens a harbor card")
	}
	if !HarborBuilding("haven/warehouse/1") || HarborBuilding("haven/crates/0") || HarborBuilding("building/sign/0") {
		t.Fatal("harbor building names")
	}
	for name, building := range map[string]bool{
		"port/house/4/0/0": true, "port/warehouse/4/2/1": true, "port/sign/1/0/0": true, "port/stake/1/0/0": true,
		"port/deck/4/0/0": false, "port/pier-x/2/0/1": false, "port/pier-y/3/1/2": false, "port/bollard/1/0/0": false, "haven/quay/0": false,
	} {
		if HarborBuilding(name) != building {
			t.Errorf("HarborBuilding(%q) = %v, want %v: only the port's buildings, its sign and its stake open it", name, !building, building)
		}
	}
}

func TestOpenBerthsDrawTheirPostsOnThePier(t *testing.T) {
	site := coastSite(40, 32, 8)
	site.Havens = []view.Haven{westHaven(1, true)}
	r := NewRenderer(nil)
	r.HarborOf().Step(Scene{Site: site, Berths: 4})
	p := r.HarborOf().Plans[0]
	cam := render.NewCamera(480, 60).CenterOn(p.Head.X, p.Head.Y, ViewWidth, ViewHeight)
	items := r.postItems(nil, cam, Scene{Site: site}, p)
	if len(items) != 4 {
		t.Fatalf("4 berths draw %d posts", len(items))
	}
	dst := ebiten.NewImage(ViewWidth, ViewHeight)
	defer dst.Deallocate()
	for _, it := range items {
		it.Draw(dst)
	}
	far := render.NewCamera(480, 60).CenterOn(p.Head.X+200, p.Head.Y+200, ViewWidth, ViewHeight)
	if got := r.postItems(nil, far, Scene{Site: site}, p); len(got) != 0 {
		t.Fatalf("%d posts drawn off the screen", len(got))
	}
}
