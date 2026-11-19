package world

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

func TestPortLevelReadsTheHaven(t *testing.T) {
	for tier := uint8(0); tier <= 3; tier++ {
		if got := PortLevel(westHaven(tier, true)); got != int(tier)+1 {
			t.Errorf("an open haven at tier %d stands at level %d", tier, got)
		}
		if got := PortLevel(westHaven(tier, false)); got != 0 {
			t.Errorf("a closed haven stands at level %d", got)
		}
	}
	if got := PortLevel(westHaven(9, true)); got != art.PortLevels {
		t.Errorf("a haven past the top tier stands at level %d", got)
	}
}

func TestClosedPortsAreBareCoastWithAForSaleStake(t *testing.T) {
	site := coastSite(40, 32, 8)
	p := PlanHaven(site, westHaven(2, false))
	if len(p.Pier) != 0 || len(p.Deck) != 0 || len(p.Berths) != 0 || p.Level != 0 {
		t.Fatalf("a closed port lays %d planks, %d deck tiles, %d berths at level %d", len(p.Pier), len(p.Deck), len(p.Berths), p.Level)
	}
	if len(p.Props) != 1 || p.Props[0].Kind != StakeKind || p.Props[0].Deck || TileOf(p.Props[0].At) != TileOf(p.Quay) {
		t.Fatalf("a closed port shows %+v, want one stake on its quay", p.Props)
	}
	r := NewRenderer(nil)
	hit := r.propHit(p, p.Props[0], 100, 100, 1)
	if hit == nil || hit.Kind != HitHaven || hit.ID != p.ID {
		t.Fatalf("the stake of a closed port opens %+v", hit)
	}
}

func TestPortsGrowTheirBuildingsLevelByLevel(t *testing.T) {
	site := coastSite(40, 32, 8)
	want := []map[string]bool{
		{"sign": true, "house": true, "crates": true, "bollard": true},
		{"sign": true, "house": true, "crates": true, "bollard": true, "crane": true},
		{"sign": true, "house": true, "crates": true, "bollard": true, "crane": true, "warehouse": true},
		{"sign": true, "house": true, "crates": true, "bollard": true, "crane": true, "warehouse": true, "lighthouse": true},
	}
	for tier, kinds := range want {
		p := PlanHaven(site, westHaven(uint8(tier), true))
		got := map[string]bool{}
		for _, prop := range p.Props {
			got[prop.Kind] = true
		}
		for k := range kinds {
			if !got[k] {
				t.Errorf("level %d lacks its %s: %v", tier+1, k, got)
			}
		}
		for k := range got {
			if !kinds[k] {
				t.Errorf("level %d already shows a %s", tier+1, k)
			}
		}
		if p.Level != tier+1 {
			t.Errorf("tier %d plans level %d", tier, p.Level)
		}
	}
}

func TestPortPiecesDrawByLevelAndStageWithTheOldArtBehind(t *testing.T) {
	for level := 1; level <= art.PortLevels; level++ {
		for age := 0; age < art.AgeTiers; age++ {
			names := PortPieceNames(HavenProp{Kind: "house"}, level, age)
			if names[0] != (art.PortLook{Kind: "house", Level: level, Age: age}).Name() {
				t.Fatalf("level %d age %d draws %v", level, age, names)
			}
			deck := PortPieceNames(HavenProp{Kind: "quay", Variant: 2}, level, age)
			if deck[0] != (art.PortLook{Kind: "deck", Level: level, Age: age, Variant: 2}).Name() || !strings.HasPrefix(deck[len(deck)-1], "haven/quay/") {
				t.Fatalf("a deck tile at level %d age %d draws %v", level, age, deck)
			}
		}
	}
	if names := PortPieceNames(HavenProp{Kind: "crates", Variant: 1}, 2, 1); len(names) != 1 || names[0] != "haven/crates/1" {
		t.Fatalf("crates draw %v", names)
	}
}

func TestAPortsZoneIsItsDeckAndPierOrTheQuayItWouldTake(t *testing.T) {
	site := coastSite(40, 32, 8)
	p := PlanHaven(site, westHaven(2, true))
	zone := map[Point]bool{}
	for _, d := range ZoneTiles(p) {
		zone[d] = true
	}
	for _, d := range p.Deck {
		if !zone[d] {
			t.Errorf("the zone misses the deck tile %v", d)
		}
	}
	for _, v := range p.Pier {
		if !zone[TileOf(v)] {
			t.Errorf("the zone misses the pier tile %v", v)
		}
	}
	if len(zone) != len(p.Deck)+len(p.Pier) {
		t.Errorf("the zone holds %d tiles, want %d", len(zone), len(p.Deck)+len(p.Pier))
	}
	closed := PlanHaven(site, westHaven(0, false))
	tiles := ZoneTiles(closed)
	if len(tiles) != (2*DeckHalf+1)*DeckDepth {
		t.Fatalf("a closed port marks %d tiles", len(tiles))
	}
	for _, d := range tiles {
		if Wet(site.Tile(d.X, d.Y)) {
			t.Errorf("a closed port marks water at %v", d)
		}
	}
}

func TestOnlyTheSelectedPortLightsUpAndItReagesWithTheStage(t *testing.T) {
	site := coastSite(40, 32, 8)
	p := PlanHaven(site, westHaven(1, true))
	r := NewRenderer(nil)
	if !r.portArtOf(Scene{Selected: p.ID}, p).selected || r.portArtOf(Scene{Selected: "keep-1"}, p).selected {
		t.Fatal("selection does not follow the port id")
	}
	r.portArtOf(Scene{Age: 0, Time: 1}, p)
	busy := false
	for tick := 0.0; tick < AgeStagger+AgeWork+1; tick += .1 {
		a := r.portArtOf(Scene{Age: 2, Time: 2 + tick}, p)
		busy = busy || a.busy
	}
	if a := r.portArtOf(Scene{Age: 2, Time: 10}, p); a.age != 2 || a.level != 2 || !busy {
		t.Fatalf("after promotion the port shows age %d level %d (rebuilt: %v)", a.age, a.level, busy)
	}
}

func TestASelectedPortDrawsWithEveryBuildingPickable(t *testing.T) {
	dst := ebiten.NewImage(960, 540)
	defer dst.Deallocate()
	for tier := uint8(0); tier <= 3; tier++ {
		sc, cam := tieredSite(t, tier)
		r := NewRenderer(testLib())
		sc.Selected, sc.Age = sc.Site.Havens[0].ID, int(tier)
		cam.Zoom = 2
		q := sc.Site.Havens[0].Quay
		cam = cam.CenterOn(float64(q.X), float64(q.Y), 960, 540)
		for i := 0; i < 3; i++ {
			sc.Time += .1
			r.Draw(dst, cam, sc)
		}
		kinds := map[string]bool{}
		for _, h := range r.HitList {
			if h.Kind == HitHaven && h.ID == sc.Selected {
				rest, _ := strings.CutPrefix(h.Shapes[0].Name, art.PortPrefix)
				kind, _, _ := strings.Cut(rest, "/")
				kinds[kind] = true
			}
		}
		if !kinds["house"] || !kinds["sign"] || tier >= 1 && !kinds["crane"] || tier >= 3 && !kinds["lighthouse"] {
			t.Errorf("tier %d: the selected port picks through %v", tier, kinds)
		}
	}
}

func TestEachPortMooresAPostPerOwnBerth(t *testing.T) {
	gate := HavenPlan{Open: true, Pier: []Vec{{7, 16}, {6, 16}, {5, 16}, {4, 16}}, Side: Vec{0, 1}, Out: Vec{-1, 0}}
	shut := HavenPlan{Open: false, Pier: gate.Pier, Side: gate.Side, Out: gate.Out}
	port := gate
	havens := []view.Haven{{Open: true, Moored: 1}, {Open: false, Moored: 3}, {Open: true, Moored: 9}}
	posts := PortPosts([]HavenPlan{gate, shut, port}, havens, 10)
	if len(posts[0]) != 1 || len(posts[1]) != 0 || len(posts[2]) != len(BerthSpots(port)) {
		t.Fatalf("posts per port = %d, %d, %d", len(posts[0]), len(posts[1]), len(posts[2]))
	}
	old := PortPosts([]HavenPlan{gate, shut, port}, []view.Haven{{Open: true}, {}, {Open: true}}, 7)
	if len(old[0]) != 6 || len(old[2]) != 1 {
		t.Fatalf("a realm without per-port berths shares its %d berths as %d and %d", 7, len(old[0]), len(old[2]))
	}
}
