package world

import (
	"sort"
	"strconv"
	"strings"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

type Harbor struct {
	sig       string
	decks     map[Point]bool
	deckOrder []Point
	deckPlan  map[Point]int
	Plans     []HavenPlan `json:"-"`
	Fleets    []*fleet    `json:"-"`
	docks     docks
	Book      Manifest `json:"-"`
	clock     FreightClock
	Now       float64 `json:"-"`
	Loads     []*Load `json:"-"`
	landed    []*Load
	Yards     []Yard `json:"-"`
	Full      bool   `json:"-"`
	names     map[boxKey][]string
	routes    RouteBook
	landSig   uint64
	carrying  int
	gleams    map[string]Vec
	berths    int
	ferries   ferryBook
	Lanes     []FerryLane `json:"-"`
	riders    map[boatKey][]Figure
	pieceMemo map[pieceKey][]string
}

func (r *Renderer) HarborOf() *Harbor {
	if r.harbor == nil {
		r.harbor = &Harbor{}
	}
	return r.harbor
}

func PlanSig(site view.Site, havens []view.Haven) string {
	var b strings.Builder
	for _, h := range havens {
		fields := []string{h.ID, strconv.FormatBool(h.Open), strconv.Itoa(int(h.Tier)), strconv.Itoa(h.Level), strconv.Itoa(h.Moored), strconv.Itoa(h.Pier),
			strconv.Itoa(h.Quay.X), strconv.Itoa(h.Quay.Y), strconv.Itoa(h.Out.X), strconv.Itoa(h.Out.Y),
			strconv.FormatFloat(max(h.OpenedAt, h.TierAt), 'f', 1, 64)}
		b.WriteString(strings.Join(fields, ","))
		b.WriteString(";")
	}
	b.WriteString(strconv.Itoa(len(site.Structures)))
	return b.String()
}

func (h *Harbor) sigOf(site view.Site, havens []view.Haven) string {
	return PlanSig(site, havens) + "|" + strconv.Itoa(h.berths)
}

func (h *Harbor) Replan(site view.Site) {
	havens := SiteHavens(site)
	sig := h.sigOf(site, havens)
	if sig == h.sig {
		return
	}
	h.sig = sig
	h.decks, h.deckOrder, h.deckPlan = map[Point]bool{}, nil, map[Point]int{}
	h.Plans = make([]HavenPlan, len(havens))
	fleets := make([]*fleet, len(havens))
	byID := map[string]*fleet{}
	for _, f := range h.Fleets {
		byID[f.haven] = f
	}
	for i, hv := range havens {
		h.Plans[i] = PlanHaven(site, hv)
		h.markDecks(h.Plans[i], i)
		if f, ok := byID[hv.ID]; ok {
			fleets[i] = f
			continue
		}
		fleets[i] = &fleet{haven: hv.ID}
	}
	h.Fleets = fleets
	posts := PortPosts(h.Plans, havens, h.berths)
	for i := range h.Plans {
		h.Plans[i] = WithPosts(site, h.Plans[i], posts[i])
	}
}

func (h *Harbor) markDecks(p HavenPlan, plan int) {
	for _, d := range p.Deck {
		h.decks[d], h.deckPlan[d] = true, plan
	}
	h.deckOrder = append(h.deckOrder, p.Deck...)
	sort.SliceStable(h.deckOrder, func(i, j int) bool {
		a, b := h.deckOrder[i], h.deckOrder[j]
		return render.DepthKey(float64(a.X), float64(a.Y)) < render.DepthKey(float64(b.X), float64(b.Y))
	})
	for _, v := range p.Pier {
		h.decks[TileOf(v)] = true
	}
}

func (h *Harbor) Lift(v Vec) float64 {
	if t := TileOf(v); h.decks[t] || h.ferries.decks[t] {
		return art.HavenDeck
	}
	return 0
}

func Weights(plans []HavenPlan) []float64 {
	out := make([]float64, len(plans))
	total, open := 0.0, 0
	for i, p := range plans {
		if p.Open {
			out[i] = max(p.Share, 0)
			total += out[i]
			open++
		}
	}
	if total > 0 {
		return out
	}
	for i, p := range plans {
		if p.Open {
			out[i] = 1 / float64(open)
		}
	}
	return out
}

func (h *Harbor) Step(sc Scene) {
	h.berths = sc.Berths
	h.Replan(sc.Site)
	h.landSig = StructureSig(sc.Site)
	h.SyncLanes(sc.Site)
	h.Now = h.clock.Step(sc.RealmTime, sc.Time)
	if len(h.Plans) == 0 {
		h.Loads, h.landed, h.Yards, h.Full = nil, nil, nil, false
		return
	}
	weights := Weights(h.Plans)
	h.schedule(sc.Freight, h.Now, weights)
	if h.Book.Update(sc.Freight.Kinds, sc.RealmTime, sc.Time, h.picker(weights)) {
		h.Loads = h.Book.Sorted()
		h.landed = h.Book.landed()
		h.Yards = Yards(h.Loads)
	}
	h.countLandings(sc.RealmTime)
	h.Full = QuayFull(sc.Freight)
}

func (h *Harbor) countLandings(seen float64) {
	for i, f := range h.Fleets {
		for j := range f.Ships {
			s := &f.Ships[j]
			if s.Docked && !s.Counted && s.Cargo > 0 && seen >= s.Dock {
				s.Accepted, s.Counted = h.accepted(*s, i), true
			}
		}
	}
}

func (h *Harbor) picker(weights []float64) HavenPicker {
	return func(at float64, key uint32) int {
		if i, _, ok := h.shipOfLanding(at); ok {
			return i
		}
		return HavenFor(int(key%4096), weights)
	}
}

func (h *Harbor) route(site view.Site, from, to Vec) []Vec {
	return h.routes.Route(site, h.landSig, from, to)
}

func QuayFull(fv view.FreightView) bool {
	return fv.QuayCap > 0 && (fv.Waiting >= fv.QuayCap-.5 || fv.Losses.QuayFull > 0)
}

func (h *Harbor) Plan(i int) HavenPlan {
	if len(h.Plans) == 0 {
		return HavenPlan{}
	}
	return h.Plans[min(max(i, 0), len(h.Plans)-1)]
}

func onPier(plan HavenPlan, v Vec) Vec {
	d := (v.X-plan.Quay.X)*plan.Out.X + (v.Y-plan.Quay.Y)*plan.Out.Y
	return offset(plan.Quay, plan.Out, max(d, 0))
}

func PathLength(path []Vec) float64 {
	total := 0.0
	for i := 1; i < len(path); i++ {
		total += Dist(path[i-1], path[i])
	}
	return total
}

func HandSpot(site view.Site) Vec {
	g := GatewayOf(site)
	return offset(g.Door, g.Inward, HelmAhead)
}
