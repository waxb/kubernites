package world

import (
	"math"

	"github.com/waxb/kubernites/pkg/view"
)

const (
	pierBase     = 3
	BerthGap     = 1.0
	plankSide    = .3
	laneOut      = 1.6
	headMoor     = 1.0
	DeckHalf     = 2
	DeckDepth    = 3
	yardSize     = 2
	MaxBerths    = 3
	FreshSeconds = 4.0
	maxTier      = 3
	seaReach     = 8
	maxPierScan  = 6
	fallbackName = "Gate Harbor"
)

type Berth struct {
	Moor  Vec
	Plank Vec
	Lane  Vec
}

type HavenPlan struct {
	ID     string
	Name   string
	Open   bool
	Tier   int
	Level  int
	Share  float64
	Out    Vec
	Side   Vec
	Quay   Vec
	Head   Vec
	Sea    Vec
	Pier   []Vec
	Deck   []Point
	Berths []Berth
	Posts  []Vec
	Yards  []Vec
	Piles  [][]Vec
	Props  []HavenProp
	Since  float64
}

type HavenProp struct {
	Kind    string
	Variant int
	At      Vec
	Tier    int
	Deck    bool
	Flip    bool
}

type propSpot struct {
	kind          string
	variant, tier int
	side, depth   float64
}

var deckProps = []propSpot{
	{kind: "sign", side: DeckHalf, depth: 0},
	{kind: "crates", side: -DeckHalf, depth: 2},
	{kind: "crane", tier: 1, side: -DeckHalf, depth: 1},
	{kind: houseKind, side: DeckHalf, depth: 2},
	{kind: "crates", variant: 1, tier: 1, side: DeckHalf, depth: 1},
	{kind: "crates", variant: 2, tier: 2, side: -1, depth: 2},
	{kind: "lighthouse", tier: maxTier, side: -DeckHalf, depth: 0},
}

var bigPieces = map[string]bool{"crane": true, "warehouse": true, "lighthouse": true, houseKind: true}

func HavenTier(h view.Haven) int {
	if !h.Open {
		return 0
	}
	return min(int(h.Tier), maxTier)
}

func PierLength(h view.Haven) int {
	if !h.Open {
		return 0
	}
	return max(min(pierBase+HavenTier(h), h.Pier), 0)
}

func BerthCount(h view.Haven) int {
	if !h.Open {
		return 0
	}
	return min(HavenTier(h)+1, MaxBerths)
}

func VecOf(p view.Point) Vec { return Vec{float64(p.X), float64(p.Y)} }

func TileOf(v Vec) Point { return Point{int(math.Round(v.X)), int(math.Round(v.Y))} }

func WetAt(site view.Site, v Vec) bool {
	p := TileOf(v)
	return Wet(site.Tile(p.X, p.Y))
}

func PlanHaven(site view.Site, h view.Haven) HavenPlan {
	out := VecOf(h.Out)
	p := HavenPlan{ID: h.ID, Name: h.Name, Open: h.Open, Tier: HavenTier(h), Level: PortLevel(h), Share: h.Share, Out: out, Side: VecOf(h.Along()),
		Quay: VecOf(h.Quay), Sea: VecOf(h.Sea), Since: max(h.OpenedAt, h.TierAt)}
	if !h.Open {
		p.Head, p.Props = p.Quay, stakeProps(p)
		return p
	}
	length := PierLength(h)
	for k := 1; k <= length; k++ {
		p.Pier = append(p.Pier, offset(p.Quay, out, float64(k)))
	}
	p.Head = offset(p.Quay, out, float64(length))
	p.Deck = deckTiles(site, p)
	p.Berths = berths(site, p, h.Pier, BerthCount(h))
	p.Yards = yards(h)
	p.Props = HavenProps(p)
	p.Piles = PileSpots(p)
	return p
}

func yards(h view.Haven) []Vec {
	ys := h.Yards()
	return []Vec{YardAnchor(ys[1]), YardAnchor(ys[0])}
}

func YardAnchor(corner view.Point) Vec {
	return Vec{float64(corner.X) + yardSize - .5, float64(corner.Y) + yardSize - .5}
}

func pieceAt(kind string, v Vec) Vec {
	if bigPieces[kind] {
		return Vec{v.X + .5, v.Y + .5}
	}
	return v
}

func deckTiles(site view.Site, p HavenPlan) []Point {
	out := make([]Point, 0, (2*DeckHalf+1)*DeckDepth)
	for j := 0; j < DeckDepth; j++ {
		for i := -DeckHalf; i <= DeckHalf; i++ {
			t := TileOf(offset(offset(p.Quay, p.Side, float64(i)), p.Out, -float64(j)))
			if !Wet(site.Tile(t.X, t.Y)) && !Occupied(site, t.X, t.Y) {
				out = append(out, t)
			}
		}
	}
	return out
}

func berths(site view.Site, p HavenPlan, pier, want int) []Berth {
	out := []Berth{}
	for _, side := range []float64{0, -1, 1} {
		if len(out) >= want {
			break
		}
		b := BerthAt(p, side, pier)
		if WetAt(site, b.Moor) && WetAt(site, b.Lane) {
			out = append(out, b)
		}
	}
	return out
}

func BerthAt(p HavenPlan, side float64, pier int) Berth {
	lane := offset(offset(p.Quay, p.Out, float64(max(pier, len(p.Pier)))+laneOut), p.Side, side*BerthGap)
	if side == 0 {
		moor := offset(p.Head, p.Out, headMoor)
		return Berth{Moor: moor, Plank: offset(p.Head, p.Out, plankSide), Lane: offset(lane, p.Out, headMoor)}
	}
	return Berth{Moor: offset(p.Head, p.Side, side*BerthGap), Plank: offset(p.Head, p.Side, side*plankSide), Lane: lane}
}

func (p HavenPlan) SailTime() float64 {
	if len(p.Berths) == 0 {
		return SailSeconds
	}
	b := p.Berths[0]
	return SailTime(Dist(p.Sea, b.Lane) + Dist(b.Lane, b.Moor))
}

func Fresh(p HavenPlan, realmTime float64) bool {
	age := realmTime - p.Since
	return p.Open && p.Since > 0 && age >= 0 && age < FreshSeconds
}

func HavenProps(p HavenPlan) []HavenProp {
	if !p.Open {
		return stakeProps(p)
	}
	out := []HavenProp{}
	flip := p.Out.Y != 0
	for _, s := range deckProps {
		if s.tier <= p.Tier {
			at := pieceAt(s.kind, offset(offset(p.Quay, p.Side, s.side), p.Out, -s.depth))
			out = append(out, HavenProp{Kind: s.kind, Variant: s.variant, At: at, Tier: s.tier, Deck: true, Flip: flip && s.kind == "crane"})
		}
	}
	if p.Tier >= 2 {
		for i, y := range p.Yards {
			out = append(out, HavenProp{Kind: "warehouse", Variant: i, At: y, Tier: 2})
		}
	}
	if len(p.Pier) == 0 {
		return out
	}
	for _, side := range []float64{-1, 1} {
		out = append(out, HavenProp{Kind: "bollard", At: offset(p.Head, p.Side, side*plankSide*1.4), Deck: true})
	}
	return out
}

func ShipHeading(v Vec) string {
	if math.Abs(v.X) >= math.Abs(v.Y) {
		if v.X >= 0 {
			return "se"
		}
		return "nw"
	}
	if v.Y > 0 {
		return "sw"
	}
	return "ne"
}

func SiteHavens(site view.Site) []view.Haven {
	if site.NoHarbor {
		return nil
	}
	if len(site.Havens) > 0 {
		return site.Havens
	}
	if h, ok := fallbackHaven(site); ok {
		return []view.Haven{h}
	}
	return nil
}

func fallbackHaven(site view.Site) (view.Haven, bool) {
	if site.Width == 0 {
		return view.Haven{}, false
	}
	for _, dy := range []int{3, -3, 4, -4, 5, -5, 2, -2, 6, -6} {
		if h, ok := westCoast(site, site.Gate.Y+dy); ok {
			return h, true
		}
	}
	return view.Haven{}, false
}

func westCoast(site view.Site, y int) (view.Haven, bool) {
	x := site.Gate.X
	for x > 0 && !Wet(site.Tile(x-1, y)) {
		x--
	}
	if x <= 0 || Wet(site.Tile(x, y)) {
		return view.Haven{}, false
	}
	water := 0
	for water < maxPierScan && Wet(site.Tile(x-water-1, y)) {
		water++
	}
	h := view.Haven{ID: view.GateHavenID, Name: fallbackName, Quay: view.Point{X: x, Y: y}, Out: view.Point{X: -1}, Pier: water,
		Sea: view.Point{X: max(x-water-seaReach, 0), Y: y}, Open: true, Share: 1}
	return h, len(deckTiles(site, PlanHaven(site, h))) == (2*DeckHalf+1)*DeckDepth
}

func HarborFocus(site view.Site) (Vec, bool) {
	havens := SiteHavens(site)
	if len(havens) == 0 {
		return Vec{}, false
	}
	p := PlanHaven(site, havens[0])
	return Lerp(p.Head, offset(p.Quay, p.Out, -DeckDepth), .5), true
}
