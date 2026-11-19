package world

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/view"
)

func coastSite(w, h, shore int) view.Site {
	tiles := make([]view.TileKind, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			k := view.TileGrass
			switch {
			case x < shore-1:
				k = view.TileWater
			case x == shore-1:
				k = view.TileShallows
			case x == shore:
				k = view.TileSand
			}
			tiles[y*w+x] = k
		}
	}
	return view.Site{Width: w, Height: h, Tiles: tiles, Gate: view.Point{X: shore + 2, Y: h / 2}, Plaza: view.Point{X: shore + 10, Y: h / 2}}
}

func westHaven(tier uint8, open bool) view.Haven {
	return view.Haven{ID: "haven-0", Name: "Gate Harbor", Quay: view.Point{X: 8, Y: 16}, Out: view.Point{X: -1, Y: 0}, Pier: 6,
		Sea: view.Point{X: 0, Y: 16}, Open: open, Tier: tier, Share: 1}
}

func TestPierGrowsWithTierUpToItsWater(t *testing.T) {
	cases := []struct {
		tier uint8
		open bool
		pier int
		want int
	}{
		{0, true, 6, 3}, {1, true, 6, 4}, {2, true, 6, 5}, {3, true, 6, 6},
		{3, true, 4, 4}, {3, false, 6, 0}, {0, true, 2, 2}, {2, true, 0, 0},
	}
	for _, c := range cases {
		h := westHaven(c.tier, c.open)
		h.Pier = c.pier
		if got := PierLength(h); got != c.want {
			t.Errorf("tier %d open %v pier %d: length %d, want %d", c.tier, c.open, c.pier, got, c.want)
		}
	}
}

func TestHavenPlanLaysThePierOverWaterAndTheDeckOnLand(t *testing.T) {
	site := coastSite(40, 32, 8)
	for tier := uint8(0); tier <= 3; tier++ {
		p := PlanHaven(site, westHaven(tier, true))
		if len(p.Pier) != PierLength(westHaven(tier, true)) {
			t.Fatalf("tier %d: %d planks", tier, len(p.Pier))
		}
		for i, v := range p.Pier {
			if !WetAt(site, v) {
				t.Errorf("tier %d plank %d on land at %v", tier, i, v)
			}
			if i > 0 && Dist(v, p.Quay) <= Dist(p.Pier[i-1], p.Quay) {
				t.Errorf("planks run out to sea: %v then %v", p.Pier[i-1], v)
			}
		}
		if len(p.Deck) != (2*DeckHalf+1)*DeckDepth {
			t.Errorf("tier %d deck tiles = %d", tier, len(p.Deck))
		}
		for _, d := range p.Deck {
			if Wet(site.Tile(d.X, d.Y)) {
				t.Errorf("deck over water at %v", d)
			}
		}
		if got, want := len(p.Berths), min(int(tier)+1, MaxBerths); got != want {
			t.Errorf("tier %d berths = %d, want %d", tier, got, want)
		}
		for _, b := range p.Berths {
			if !WetAt(site, b.Moor) || !WetAt(site, b.Lane) {
				t.Errorf("tier %d berth %+v is not afloat", tier, b)
			}
			if Dist(b.Plank, b.Moor) > BerthGap+.01 {
				t.Errorf("gangplank %v too far from the ship at %v", b.Plank, b.Moor)
			}
		}
	}
}

func TestClosedHavensAreEmptyJettyStubs(t *testing.T) {
	site := coastSite(40, 32, 8)
	p := PlanHaven(site, westHaven(3, false))
	if len(p.Berths) != 0 {
		t.Errorf("a closed haven berths %d ships", len(p.Berths))
	}
	for _, prop := range HavenProps(p) {
		if prop.Tier > 0 {
			t.Errorf("closed haven shows %s", prop.Kind)
		}
	}
}

func TestHarborPiecesAppearByTier(t *testing.T) {
	site := coastSite(40, 32, 8)
	kinds := func(tier uint8) map[string]int {
		out := map[string]int{}
		for _, prop := range HavenProps(PlanHaven(site, westHaven(tier, true))) {
			out[prop.Kind]++
		}
		return out
	}
	cases := []struct {
		tier    uint8
		has     []string
		lacks   []string
		bollard int
	}{
		{0, []string{"sign", "crates", "bollard"}, []string{"crane", "warehouse", "lighthouse"}, 2},
		{1, []string{"crane"}, []string{"warehouse", "lighthouse"}, 2},
		{2, []string{"crane", "warehouse"}, []string{"lighthouse"}, 2},
		{3, []string{"crane", "warehouse", "lighthouse"}, nil, 2},
	}
	for _, c := range cases {
		got := kinds(c.tier)
		for _, k := range c.has {
			if got[k] == 0 {
				t.Errorf("tier %d lacks %s: %v", c.tier, k, got)
			}
		}
		for _, k := range c.lacks {
			if got[k] != 0 {
				t.Errorf("tier %d already has %s", c.tier, k)
			}
		}
		if got["bollard"] < c.bollard {
			t.Errorf("tier %d bollards = %d", c.tier, got["bollard"])
		}
	}
}

func TestEveryYardStandsOnTheDeckClearOfTheProps(t *testing.T) {
	site := coastSite(40, 32, 8)
	p := PlanHaven(site, westHaven(3, true))
	deck := map[Point]bool{}
	for _, d := range p.Deck {
		deck[d] = true
	}
	props := map[Point]bool{}
	for _, prop := range p.Props {
		if prop.Deck {
			props[TileOf(prop.At)] = true
		}
	}
	for kind := range FreightKinds {
		for c := 0; c < YardColumns; c++ {
			across, depth := YardSpot(kind, c)
			v := QuayPoint(p, across, depth)
			if !deck[TileOf(v)] {
				t.Errorf("kind %d column %d at %v stands off the deck", kind, c, v)
			}
			if props[TileOf(v)] {
				t.Errorf("kind %d column %d at %v shares a tile with a prop", kind, c, v)
			}
		}
	}
}

func TestHavensFallBackToTheGateCoast(t *testing.T) {
	site := coastSite(40, 32, 8)
	if got := SiteHavens(site); len(got) != 1 || got[0].ID != "haven-0" || !got[0].Open || got[0].Out != (view.Point{X: -1}) {
		t.Fatalf("fallback havens = %+v", got)
	}
	p := PlanHaven(site, SiteHavens(site)[0])
	if len(p.Pier) == 0 || len(p.Berths) != 1 {
		t.Errorf("fallback plan = %+v", p)
	}
	site.Havens = []view.Haven{westHaven(2, true)}
	if got := SiteHavens(site); len(got) != 1 || got[0].Tier != 2 {
		t.Errorf("real havens win: %+v", got)
	}
	if got := SiteHavens(view.Site{}); len(got) != 0 {
		t.Errorf("an empty site has no haven: %+v", got)
	}
}

func TestShipHeadingFollowsTheNearestIsoAxis(t *testing.T) {
	cases := []struct {
		v    Vec
		want string
	}{
		{Vec{1, 0}, "se"}, {Vec{-1, 0}, "nw"}, {Vec{0, 1}, "sw"}, {Vec{0, -1}, "ne"},
		{Vec{1, .4}, "se"}, {Vec{.2, -.9}, "ne"}, {Vec{-.7, .69}, "nw"}, {Vec{0, 0}, "se"},
	}
	for _, c := range cases {
		if got := ShipHeading(c.v); got != c.want {
			t.Errorf("heading of %v = %s, want %s", c.v, got, c.want)
		}
	}
}

func TestFreshHavensRaiseTheirNewPieces(t *testing.T) {
	cases := []struct {
		since, now float64
		open       bool
		want       bool
	}{
		{100, 101, true, true}, {100, 100 + FreshSeconds + .1, true, false}, {0, 2, true, false}, {100, 101, false, false}, {100, 99, true, false},
	}
	for _, c := range cases {
		if got := Fresh(HavenPlan{Open: c.open, Since: c.since}, c.now); got != c.want {
			t.Errorf("fresh(since %v, now %v, open %v) = %v, want %v", c.since, c.now, c.open, got, c.want)
		}
	}
}

func TestWarehousesStandOnTheSimsYards(t *testing.T) {
	site := coastSite(40, 32, 8)
	p := PlanHaven(site, westHaven(2, true))
	want := []Vec{{12.5, 18.5}, {12.5, 15.5}}
	if len(p.Yards) != 2 || p.Yards[0] != want[0] || p.Yards[1] != want[1] {
		t.Fatalf("yards = %v, want %v (+Along first, anchored at the far corner)", p.Yards, want)
	}
	for _, out := range []view.Point{{X: -1}, {X: 1}, {Y: -1}, {Y: 1}} {
		h := westHaven(2, true)
		h.Out = out
		ys := h.Yards()
		got := PlanHaven(site, h).Yards
		if got[0] != YardAnchor(ys[1]) || got[1] != YardAnchor(ys[0]) {
			t.Errorf("out %v: yards %v do not follow Haven.Yards %v", out, got, ys)
		}
		for i, y := range got {
			corner := ys[1-i]
			if y.X != float64(corner.X)+1.5 || y.Y != float64(corner.Y)+1.5 {
				t.Errorf("out %v: yard %d anchor %v is not the far corner of %v", out, i, y, corner)
			}
		}
	}
	site.Structures = []view.Structure{{ID: "x", Kind: view.StructKeep, X: 13, Y: 17, W: 2, H: 2}}
	if got := PlanHaven(site, westHaven(2, true)).Yards; len(got) != 2 {
		t.Errorf("a Keep beside a yard hid a warehouse: %v", got)
	}
	for _, prop := range HavenProps(PlanHaven(site, westHaven(3, true))) {
		if prop.Kind == "crane" && prop.Flip {
			t.Error("a crane on an X pier needs no flip")
		}
	}
	north := westHaven(1, true)
	north.Out = view.Point{Y: -1}
	flipped := false
	for _, prop := range HavenProps(PlanHaven(site, north)) {
		flipped = flipped || prop.Kind == "crane" && prop.Flip
	}
	if !flipped {
		t.Error("a crane on a Y pier points its jib along -Y")
	}
}

func TestHarborLiftsWhatStandsOnTheDeck(t *testing.T) {
	site := coastSite(40, 32, 8)
	site.Havens = []view.Haven{westHaven(0, true)}
	h := &Harbor{}
	h.Replan(site)
	p := h.Plans[0]
	if h.Lift(p.Quay) != art.HavenDeck || h.Lift(p.Pier[0]) != art.HavenDeck {
		t.Error("the quay and pier raise walkers")
	}
	if h.Lift(Vec{30, 30}) != 0 {
		t.Error("open ground is not a deck")
	}
}
