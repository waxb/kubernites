package world

import (
	"slices"
	"sort"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	cullMargin = 40.0
	ViewWidth  = 960
	ViewHeight = 540
)

type route struct {
	work, bank       Vec
	hasWork, hasBank bool
}

type Figure struct {
	Pod     model.PodView `json:"-"`
	role    string
	Pose    Pose   `json:"-"`
	Mood    string `json:"-"`
	clock   float64
	home    Vec
	deliver bool
	Load    *boxLoad `json:"-"`
	Person  int      `json:"-"`
	Seed    uint32   `json:"-"`
	mark    bool
	Ride    *boarding `json:"-"`
	since   float64
}

func RouteOf(site view.Site) route {
	work, hasWork := doorOf(site, view.StructGuildhall)
	bank, hasBank := doorOf(site, view.StructTreasury)
	if !hasBank {
		bank = work
	}
	return route{work: work, bank: bank, hasWork: hasWork, hasBank: hasBank}
}

func (r *Renderer) Crowd(sc Scene, rt route) []Figure {
	t := r.townOf()
	if r.inset {
		return t.figs
	}
	out := t.figs[:0]
	for _, p := range sc.Pods {
		out = r.peopleOf(sc, p, rt, out)
	}
	t.figs = out
	t.prune(sc.Time)
	return out
}

func (r *Renderer) FigureFor(sc Scene, p model.PodView, rt route) (Figure, bool) {
	switch {
	case p.System:
		return systemFigure(sc, p)
	case p.Terminating:
		return Figure{}, false
	case p.State == model.HearthGuttering:
		return r.sickFigure(sc, p)
	case p.State != model.HearthReady:
		return Figure{}, false
	}
	if home, ok := r.Homes[p.Name]; ok && slices.Contains(sc.Idle, p.Name) {
		start := Vec{home.X + .35, home.Y + .35}
		return Figure{Pod: p, role: p.Sprite, Pose: RestPose(sc.Time, PersonSeed(p), start), Mood: MoodRest, home: home}, true
	}
	if h, ok := r.HaulOf(sc, p); ok {
		return r.carrierFigure(sc, p, h)
	}
	switch {
	case p.Static && !rt.hasWork:
		return r.DoorFigure(sc, p)
	case p.Sprite == ScribeRole:
		return r.scribeFigure(sc, p)
	}
	return r.workerFigure(sc, p, rt)
}

func systemFigure(sc Scene, p model.PodView) (Figure, bool) {
	if p.State != model.HearthReady || p.Terminating {
		return Figure{}, false
	}
	door, ok := doorOf(sc.Site, ComponentStructure(p))
	if !ok {
		return Figure{}, false
	}
	pose := IdlePose(sc.Time, model.Hash(p.Name), Vec{door.X - idleSide, door.Y - idleLift})
	return Figure{Pod: p, role: p.Sprite, Pose: pose, Mood: MoodSteady}, true
}

func (r *Renderer) sickFigure(sc Scene, p model.PodView) (Figure, bool) {
	home, ok := r.Homes[p.Name]
	if !ok {
		return Figure{}, false
	}
	pose := Pose{Pos: Vec{home.X + .45, home.Y + .8}, Action: "idle", Dir: "se", Frame: Frame(sc.Time, 2)}
	return Figure{Pod: p, role: p.Sprite, Pose: pose, Mood: MoodSick, home: home}, true
}

func (r *Renderer) workerFigure(sc Scene, p model.PodView, rt route) (Figure, bool) {
	home, ok := r.Homes[p.Name]
	if !ok {
		return Figure{}, false
	}
	start := Vec{home.X + .35, home.Y + .35}
	rate, working, throttled := Workload(sc, p, rt.hasWork)
	c := r.LoopBetween(sc.Site, start, rt.work, rt.bank)
	f := r.PacedOn(sc, p, start, c, rate, working, throttled)
	f.home, f.deliver, f.since = home, f.Mood != MoodRest && rt.hasBank && f.Ride == nil, DeliveryAgeOf(c, f.clock)
	return f, true
}

type cycle interface {
	Length() float64
	At(clock float64) Pose
}

type StraightLoop struct {
	Home, Work, Bank Vec `json:"-"`
}

func (l StraightLoop) Length() float64 { return LoopLength(l.Home, l.Work, l.Bank) }

func (l StraightLoop) DeliveryAge(clock float64) float64 {
	return DeliveryAgeAt(clock, l.Home, l.Work, l.Bank)
}

type delivering interface {
	DeliveryAge(clock float64) float64
}

func DeliveryAgeOf(c cycle, clock float64) float64 {
	if d, ok := c.(delivering); ok {
		return d.DeliveryAge(clock)
	}
	return floatLife
}
func (l StraightLoop) At(clock float64) Pose { return CyclePose(clock, l.Home, l.Work, l.Bank) }

func (r *Renderer) paced(sc Scene, p model.PodView, start, work, bank Vec, rate float64, working, throttled bool) Figure {
	return r.PacedOn(sc, p, start, r.LoopBetween(sc.Site, start, work, bank), rate, working, throttled)
}

func (r *Renderer) LoopBetween(site view.Site, home, work, bank Vec) cycle {
	h := r.HarborOf()
	if !h.crossesWater(site, home, work, bank) {
		return StraightLoop{Home: home, Work: work, Bank: bank}
	}
	return NewTripLoop(h.Trip(site, home, work), h.Trip(site, work, bank), h.Trip(site, bank, home))
}

func (r *Renderer) PacedOn(sc Scene, p model.PodView, start Vec, c cycle, rate float64, working, throttled bool) Figure {
	seed := PersonSeed(p)
	st := r.strideOf(sc, p, seed, throttled, !working)
	dt := st.tick(sc.Time)
	st.feel(dt, throttled)
	if st.ride != nil && r.stillRiding(sc, st) {
		return r.riderFigure(sc, p, st)
	}
	prev := st.clock
	working = Wakes(st.resting, rate, working)
	pace := PaceOf(rate, working, st.strain >= strainOn)
	st.walk(dt, pace.Speed*TradeGait(p.Sprite), c.Length(), !working)
	f := Figure{Pod: p, role: p.Sprite, Mood: pace.Mood, clock: st.clock}
	if st.resting {
		f.Pose = RestPose(sc.Time, seed, start)
		return f
	}
	if r.boardFerry(sc, PersonKey(p), st, c, prev) {
		return r.riderFigure(sc, p, st)
	}
	if f.Mood == MoodRest {
		f.Mood = MoodSteady
	}
	f.Pose = c.At(st.clock)
	return f
}

func RestPose(t float64, seed uint32, home Vec) Pose {
	return Pose{Pos: Vec{home.X + .15, home.Y + .45}, Action: "idle", Dir: "se", Frame: Frame(t+seedOffset(seed), 2)}
}

func WalkerBudget(zoom float64) int {
	switch {
	case zoom < .75:
		return 120
	case zoom < 1.5:
		return 200
	}
	return 260
}

var cullW, cullH = float64(ViewWidth), float64(ViewHeight)

func onScreen(sx, sy, zoom float64) bool {
	m := cullMargin * max(zoom, 1)
	return sx >= -m && sx <= cullW+m && sy >= -m && sy <= cullH+2*m
}

func Visible(cam render.Camera, site view.Site, figs []Figure, selected string) []Figure {
	zoom := cam.Scale()
	shown := make([]int, 0, len(figs))
	for i := range figs {
		if sx, sy := ScreenPointOf(cam, site, figs[i].Pose.Pos); onScreen(sx, sy, zoom) {
			shown = append(shown, i)
		}
	}
	if budget := WalkerBudget(zoom); len(shown) > budget {
		rank := make([]uint32, len(figs))
		for _, i := range shown {
			rank[i] = visibleRank(figs[i], selected)
		}
		sort.SliceStable(shown, func(a, b int) bool { return rank[shown[a]] < rank[shown[b]] })
		shown = shown[:budget]
	}
	out := make([]Figure, len(shown))
	for k, i := range shown {
		out[k] = figs[i]
	}
	return out
}

func visibleRank(f Figure, selected string) uint32 {
	if isPodKey(f.Pod, selected) {
		return 0
	}
	return f.seedOf()
}

func isPodKey(p model.PodView, key string) bool {
	ns := len(p.Namespace)
	return len(key) == ns+1+len(p.Name) && key[ns] == '/' && key[:ns] == p.Namespace && key[ns+1:] == p.Name
}

func PodKey(p model.PodView) string {
	return p.Namespace + "/" + p.Name
}
