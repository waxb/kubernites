package world

import (
	"sort"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	freeEase     = .05
	maxCairns    = 24
	maxRubble    = 12
	rubbleMargin = 1
)

type livingPiece struct {
	img *ebiten.Image
	reg render.Region
}

type keepFolk struct {
	s      view.Structure
	view   view.KeepView
	known  bool
	free   float64
	fruit  float64
	hungry bool
	huts   int
	crews  []string
	quiet  float64
}

type keepEase struct {
	free  float64
	fruit float64
}

type cairnMark struct {
	at    Vec
	since float64
	seed  uint32
}

type rubbleMark struct {
	s     view.Structure
	since float64
}

type living struct {
	pieces   map[string]livingPiece
	primed   bool
	last     float64
	folk     []keepFolk
	eased    map[string]keepEase
	known    map[string]view.Structure
	rubble   []rubbleMark
	cairns   []cairnMark
	bare     map[string]Vec
	fallen   map[string]Vec
	imp      string
	worn     map[Point]float64
	plans    livingPlans
	bound    map[string]bool
	scaffold map[string]bool
	ivy      map[ivyKey]*ebiten.Image
	glows    []Vec
	roads    roadLamps
	site     Point
	footed   wornTiles
	cleared  map[int32]bool
	base     func(x, y int) bool
	baseSig  uint64
}

func (r *Renderer) livingOf() *living {
	if r.living == nil {
		r.living = &living{
			pieces: map[string]livingPiece{}, eased: map[string]keepEase{}, known: map[string]view.Structure{},
			bare: map[string]Vec{}, worn: map[Point]float64{}, ivy: map[ivyKey]*ebiten.Image{},
		}
	}
	return r.living
}

func (l *living) piece(key string, paint func() art.Piece) livingPiece {
	if p, ok := l.pieces[key]; ok {
		return p
	}
	src := paint()
	b := src.Img.Bounds()
	p := livingPiece{img: newImage(src.Img), reg: render.Region{W: b.Dx(), H: b.Dy(), AnchorX: src.AX, AnchorY: src.AY}}
	l.pieces[key] = p
	return p
}

func (p livingPiece) draw(dst *ebiten.Image, sx, sy, zoom float64, o render.SpriteOptions) {
	render.DrawAt(dst, p.img, p.reg, sx, sy, zoom, o)
}

func (r *Renderer) stepLiving(sc Scene) {
	l := r.livingOf()
	dt := 0.0
	if l.primed {
		dt = min(max(sc.Time-l.last, 0), 1)
	}
	l.last, l.primed = sc.Time, true
	if at := (Point{sc.Site.Width, sc.Site.Height}); at != l.site {
		l.reset(at)
	}
	h := r.hamletOf()
	r.coverHamlet(h, sc.Site)
	l.plans.sync(sc, l.baseSig, l.base)
	r.coverHamlet(h, sc.Site)
	l.syncKeeps(sc, dt)
	l.syncRubble(sc)
	l.syncCairns(r, sc)
	l.bound = boundPods(sc.Crews)
	h.bound = l.bound
	l.scaffold = ScaffoldedPods(sc.Crews)
}

func (l *living) reset(at Point) {
	l.site = at
	l.known, l.bare, l.rubble, l.cairns, l.cleared = map[string]view.Structure{}, map[string]Vec{}, nil, nil, nil
	l.plans = livingPlans{}
}

func boundPods(crews []model.CrewChain) map[string]bool {
	out := map[string]bool{}
	for _, c := range crews {
		if !HasBinding(c) {
			continue
		}
		for _, f := range c.Foremen {
			for _, name := range f.Pods {
				out[name] = true
			}
		}
	}
	return out
}

type nodeTally struct {
	huts      int
	throttled int
	crews     map[string]bool
}

func tallyNodes(sc Scene) (map[string]*nodeTally, bool) {
	out := map[string]*nodeTally{}
	stalled := false
	for _, p := range sc.Pods {
		if p.System {
			continue
		}
		if p.State == model.HearthPending {
			stalled = true
			continue
		}
		if p.Node == "" || p.Terminating {
			continue
		}
		t := out[p.Node]
		if t == nil {
			t = &nodeTally{crews: map[string]bool{}}
			out[p.Node] = t
		}
		t.huts++
		if sc.Metrics.Pods[p.Namespace+"/"+p.Name].Throttled {
			t.throttled++
		}
		if name := crewName(p); name != "" {
			t.crews[name] = true
		}
	}
	return out, stalled
}

func crewName(p model.PodView) string {
	if p.Crew != "" {
		return p.Crew
	}
	return ""
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (l *living) syncKeeps(sc Scene, dt float64) {
	tally, stalled := tallyNodes(sc)
	views := map[string]view.KeepView{}
	for _, v := range sc.Keeps {
		views[v.ID] = v
	}
	l.folk = l.folk[:0]
	for _, s := range sc.Site.Structures {
		if s.Kind != view.StructKeep || s.State != view.StateReady {
			continue
		}
		v, known := views[s.ID]
		t := tally[s.Object.Name]
		if t == nil {
			t = &nodeTally{}
		}
		f := keepFolk{s: s, view: v, known: known, huts: t.huts, crews: sortedKeys(t.crews), free: .5}
		if known {
			f.hungry = Hungry(v, t.throttled, stalled)
			f.free = FoodFree(v)
		}
		f.free, f.fruit = l.ease(s.ID, f, dt)
		if since, ok := sc.Stood[s.Object.Name]; ok {
			f.quiet = max(sc.RealmTime-since, 0)
		}
		l.folk = append(l.folk, f)
		l.known[s.ID] = s
	}
}

func (l *living) ease(id string, f keepFolk, dt float64) (float64, float64) {
	target := BerryTarget(f.hungry, f.free)
	e, seen := l.eased[id]
	if !seen {
		e = keepEase{free: f.free, fruit: target}
	}
	e.free += min(max(f.free-e.free, -freeEase*dt), freeEase*dt)
	e.fruit = EaseBerries(e.fruit, target, dt)
	l.eased[id] = e
	return e.free, e.fruit
}

func (l *living) keepFolkOf(id string) (keepFolk, bool) {
	for _, f := range l.folk {
		if f.s.ID == id {
			return f, true
		}
	}
	return keepFolk{}, false
}

func (l *living) syncRubble(sc Scene) {
	standing := map[string]bool{}
	for _, s := range sc.Site.Structures {
		standing[s.ID] = true
	}
	ids := make([]string, 0, len(l.known))
	for id := range l.known {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if standing[id] || len(sc.Site.Structures) == 0 {
			continue
		}
		l.rubble = append(l.rubble, rubbleMark{s: l.known[id], since: sc.Time})
		l.clear(l.known[id], sc.Site)
		delete(l.known, id)
		delete(l.eased, id)
	}
	if len(l.rubble) > maxRubble {
		l.rubble = l.rubble[len(l.rubble)-maxRubble:]
	}
}

func (l *living) clear(s view.Structure, site view.Site) {
	if l.cleared == nil {
		l.cleared = map[int32]bool{}
	}
	for y := s.Y - rubbleMargin; y < s.Y+max(s.H, 1)+rubbleMargin; y++ {
		for x := s.X - rubbleMargin; x < s.X+max(s.W, 1)+rubbleMargin; x++ {
			if x >= 0 && y >= 0 && x < site.Width && y < site.Height {
				l.cleared[int32(y*site.Width+x)] = true
			}
		}
	}
}

func (l *living) syncCairns(r *Renderer, sc Scene) {
	if sc.Omen != nil {
		l.imp = sc.Omen.Pod
	}
	live := map[string]bool{}
	for _, p := range sc.Pods {
		live[p.Name] = true
		if home, ok := r.Homes[p.Name]; ok && p.Owner == "" && !p.System {
			l.bare[p.Name] = home
		}
	}
	if sc.Omen == nil && live[l.imp] {
		l.imp = ""
	}
	if gone, ok := l.bare[l.imp]; ok && l.imp != "" && !live[l.imp] {
		l.cairns = append(l.cairns, cairnMark{at: Vec{gone.X + .5, gone.Y + .5}, since: sc.Time, seed: model.Hash(l.imp)})
		l.imp = ""
		if len(l.cairns) > maxCairns {
			l.cairns = l.cairns[len(l.cairns)-maxCairns:]
		}
	}
	l.keepFallen(sc, live)
	for name := range l.bare {
		if !live[name] && name != l.imp {
			delete(l.bare, name)
		}
	}
}
