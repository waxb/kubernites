package world

import (
	"sort"
	"strings"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	HitForeman     = "foreman"
	HitCrew        = "crew"
	WhistleLife    = 1.8
	ThreadLife     = 3.0
	WalkOffLife    = 3.2
	foremanBurst   = 1.5
	maxForemanSays = 2
	postRow        = 1.5
	postStep       = 1.0
	walkOffX       = -1.4
	walkOffY       = 2.6
	ownerPrefix    = "ReplicaSet/"
)

type Post struct {
	Crew    model.CrewChain
	Banner  Vec
	Foremen []Vec
}

type Departure struct {
	Foreman model.Foreman `json:"-"`
	From    Vec           `json:"-"`
	Since   float64       `json:"-"`
}

type ForemanSay struct {
	Foreman  model.Foreman
	Pod      string
	Desired  int
	At       Vec
	Start    float64
	Count    int
	Replaced bool
	Now      float64
}

type Foremen struct {
	primed   bool
	last     float64
	active   map[string]map[string]bool
	standing map[string]Vec
	Whistles map[string]float64 `json:"-"`
	gone     []Departure
	says     []ForemanSay
	Marks    []tallyMark `json:"-"`
}

func NewForemen() *Foremen {
	return &Foremen{active: map[string]map[string]bool{}, standing: map[string]Vec{}, Whistles: map[string]float64{}}
}

func OwnerRS(p model.PodView) string {
	name, ok := strings.CutPrefix(p.Owner, ownerPrefix)
	if !ok {
		return ""
	}
	return p.Namespace + "/" + name
}

func activePods(pods []model.PodView) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, p := range pods {
		rs := OwnerRS(p)
		if rs == "" || p.Terminating || p.State == model.HearthBanished {
			continue
		}
		if out[rs] == nil {
			out[rs] = map[string]bool{}
		}
		out[rs][p.Name] = true
	}
	return out
}

type Homes func(pod string) (Vec, bool)

func PostsOf(site view.Site, chains []model.CrewChain, pods []model.PodView, homes Homes) []Post {
	out := []Post{}
	for _, c := range chains {
		s, ok := postSite(site, c, pods)
		if !ok {
			continue
		}
		base, step := FrontOf(s, hutCentre(c, pods, homes))
		p := Post{Crew: c, Banner: base}
		for i := range c.Standing() {
			p.Foremen = append(p.Foremen, Vec{X: base.X + step.X*float64(i+1), Y: base.Y + step.Y*float64(i+1)})
		}
		out = append(out, p)
	}
	return out
}

func postSite(site view.Site, c model.CrewChain, pods []model.PodView) (view.Structure, bool) {
	if d, ok := DepotOf(site, c.Depot); ok && c.Depot != "" {
		return d, true
	}
	for _, p := range pods {
		if p.Node == "" || !ownedBy(c, p) {
			continue
		}
		if k, ok := KeepForNode(site, p.Node); ok {
			return k, true
		}
	}
	return view.Structure{}, false
}

func hutCentre(c model.CrewChain, pods []model.PodView, homes Homes) *Vec {
	sum, n := Vec{}, 0.0
	for _, p := range pods {
		if !ownedBy(c, p) || homes == nil {
			continue
		}
		if v, ok := homes(p.Name); ok {
			sum, n = Vec{X: sum.X + v.X, Y: sum.Y + v.Y}, n+1
		}
	}
	if n == 0 {
		return nil
	}
	return &Vec{X: sum.X / n, Y: sum.Y / n}
}

func FrontOf(s view.Structure, huts *Vec) (Vec, Vec) {
	if f, ok := stallFrameOf(s); ok {
		return f.post()
	}
	lo, hi := Footprint(s)
	mid := Vec{X: float64(lo.X+hi.X)/2 + .5, Y: float64(lo.Y+hi.Y)/2 + .5}
	if huts != nil && huts.X-huts.Y > mid.X-mid.Y {
		return Vec{X: float64(hi.X) + postRow, Y: float64(hi.Y) + .5}, Vec{X: 0, Y: -postStep}
	}
	return Vec{X: float64(lo.X) + .5, Y: float64(hi.Y) + postRow}, Vec{X: postStep, Y: 0}
}

func ownedBy(c model.CrewChain, p model.PodView) bool {
	rs := OwnerRS(p)
	for _, f := range c.Foremen {
		if f.ID() == rs {
			return true
		}
	}
	return false
}

type Birth func(pod string) (float64, Vec, bool)

func (f *Foremen) Step(posts []Post, pods []model.PodView, born Birth, now float64) {
	active := activePods(pods)
	standing, owners := map[string]Vec{}, map[string]model.Foreman{}
	for _, p := range posts {
		for i, fm := range p.Crew.Standing() {
			standing[fm.ID()], owners[fm.ID()] = p.Foremen[i], fm
		}
	}
	if f.primed {
		for id, fm := range owners {
			if LostHut(f.active[id], active[id], fm.Desired) {
				f.Whistles = WithTime(f.Whistles, id, now)
			}
		}
		f.raise(pods, owners, standing, born, now)
	}
	f.gone = f.departures(posts, standing, now)
	f.says = f.pruneSays(now)
	f.active, f.standing, f.primed, f.last = active, standing, true, now
}

func LostHut(before, now map[string]bool, desired int) bool {
	kept := 0
	for name := range before {
		if now[name] {
			kept++
		}
	}
	return kept < len(before) && kept < desired
}

func (f *Foremen) raise(pods []model.PodView, owners map[string]model.Foreman, standing map[string]Vec, born Birth, now float64) {
	for _, p := range pods {
		fm, ok := owners[OwnerRS(p)]
		if !ok || born == nil {
			continue
		}
		if t, _, housed := born(p.Name); housed && t > f.last && t <= now {
			f.Raised(fm, p.Name, fm.Desired, standing[fm.ID()], now)
		}
	}
}

func (f *Foremen) departures(posts []Post, standing map[string]Vec, now float64) []Departure {
	out := []Departure{}
	for _, d := range f.gone {
		if now-d.Since < WalkOffLife {
			out = append(out, d)
		}
	}
	if !f.primed {
		return out
	}
	for _, p := range posts {
		for _, fm := range p.Crew.History() {
			at, was := f.standing[fm.ID()]
			if _, still := standing[fm.ID()]; was && !still {
				out = append(out, Departure{Foreman: fm, From: at, Since: now})
			}
		}
	}
	return out
}

func WithTime(m map[string]float64, key string, t float64) map[string]float64 {
	next := make(map[string]float64, len(m)+1)
	for k, v := range m {
		next[k] = v
	}
	next[key] = t
	return next
}

func (f *Foremen) Whistling(id string, now float64) bool {
	t, ok := f.Whistles[id]
	return ok && now >= t && now-t < WhistleLife
}

func (f *Foremen) Departures() []Departure { return f.gone }

func (f *Foremen) Standing(id string) (Vec, bool) {
	v, ok := f.standing[id]
	return v, ok
}

func (f *Foremen) Raised(fm model.Foreman, pod string, desired int, at Vec, now float64) {
	replaced := f.Whistling(fm.ID(), now) || now-f.Whistles[fm.ID()] < ThreadLife
	for i := len(f.says) - 1; i >= 0; i-- {
		s := f.says[i]
		if s.Foreman.ID() == fm.ID() && now-s.Start < foremanBurst {
			next := append([]ForemanSay(nil), f.says...)
			next[i] = ForemanSay{Foreman: fm, Pod: pod, Desired: desired, At: at, Start: now, Count: s.Count + 1, Replaced: replaced || s.Replaced}
			f.says = next
			return
		}
	}
	f.says = append(append([]ForemanSay(nil), f.says...), ForemanSay{Foreman: fm, Pod: pod, Desired: desired, At: at, Start: now, Count: 1, Replaced: replaced})
}

func (f *Foremen) pruneSays(now float64) []ForemanSay {
	out := []ForemanSay{}
	for _, s := range f.says {
		if now-s.Start < CalloutLife && now >= s.Start {
			out = append(out, s)
		}
	}
	return out
}

func (f *Foremen) Says() []ForemanSay {
	out := append([]ForemanSay(nil), f.says...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Start > out[j].Start })
	return out[:min(len(out), maxForemanSays)]
}

func SayText(s ForemanSay) (string, string) {
	head := "the shift of " + s.Foreman.Name + " (ReplicaSet)"
	tail := "raised a new hut to keep " + Itoa(s.Desired)
	if s.Count > 1 {
		tail = "raised " + Itoa(s.Count) + " new huts to keep " + Itoa(s.Desired)
	}
	return head, tail
}

func SayTextK8s(s ForemanSay) (string, string) {
	head := "ReplicaSet " + s.Foreman.Name + ": SuccessfulCreate"
	tail := "Created pod: " + s.Pod
	if s.Count > 1 {
		tail += ", and " + Itoa(s.Count-1) + " more"
	}
	return head, tail
}
