package world

import (
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	HitStructure = "structure"
	HitHearth    = "hearth"
	HitWalker    = "walker"
	HitFamiliar  = "familiar"
)

type Ghost struct {
	Kind  string
	X, Y  int
	W, H  int
	Valid bool
	Class string
}

type Errand struct {
	Kind  string
	Role  string
	From  Vec
	To    Vec
	Start float64
	Life  float64
	Pod   string
	Tier  int
}

type Transient struct {
	Kind  string
	At    Vec
	Pod   string
	Start float64
	Life  float64
}

type Omen struct {
	Pod    string
	Start  float64
	Arrive float64
}

type Scene struct {
	Site        view.Site
	Pods        []model.PodView
	Metrics     view.MetricsSnapshot
	ArrivalRate float64
	Queue       float64
	DayPhase    float64
	Time        float64
	Real        float64
	Selected    string
	Ghost       *Ghost
	Transients  []Transient
	Omen        *Omen
	Reduce      bool
	GateOpen    bool
	Beckon      bool
	Hint        bool
	Served      float64
	Dropped     float64
	HandServed  float64
	RealmTime   float64
	Errands     []Errand
	ServedAt    float64
	Ledger      Ledger
	Surge       bool
	Lens        Lens
	Caesar      model.Caesar
	Callouts    []Callout
	Avoid       []ui.Rect
	Panels      []ui.Rect
	Goal        *Vec
	Plot        *Ghost
	Frictions   []Friction
	Borders     *Borders
	Tints       []Tint
	Causes      []view.CauseTag
	Incidents   []IncidentMark
	Chests      []ChestMark
	Watch       *view.WatchView
	Preview     []BorderMark
	Spots       []SpotMark
	Freight     view.FreightView
	Chapter     bool
	HoldSays    bool
	Hover       string
	Golden      *Gilded
	Crews       []model.CrewChain
	Age         int
	Berths      int
	Debuts      []Debut
	Rollouts    model.Rollouts
	Keeps       []view.KeepView
	Storage     view.StorageState
	Stone       bool
	Stood       map[string]float64
	Fallen      []string
	Idle        []string
	Pins        map[string]Point
	Heraldry    *Heraldry
	Plaques     map[string]Plaque
	Focus       string
	Speech      *Speech
}

type Plaque struct {
	Title string
	Line  string
}

type Hit struct {
	Kind      string
	ID        string
	Pet       string
	Structure view.Structure
	Pod       model.PodView
	Rect      ui.Rect
	Serial    int
	Shapes    []Shape
	Grow      float64
}

const (
	personWorker  = "worker"
	fallenPerson  = "fallen"
	debutPerson   = "debut"
	foremanPerson = "foreman"
)

type Drawable struct {
	person     string
	Depth      float64                 `json:"-"`
	Draw       func(dst *ebiten.Image) `json:"-"`
	Hit        *Hit                    `json:"-"`
	stand      standing
	box        footprint
	SeeThrough bool `json:"-"`
}

type Renderer struct {
	Lib      *gfx.Library
	HitList  []Hit `json:"-"`
	watch    watchIndex
	deckHits []Hit
	Homes    map[string]Vec `json:"-"`
	kindled  map[string]float64
	shores   map[string]Vec
	town     *town
	glow     *ebiten.Image
	Land     *landCache `json:"-"`
	harbor   *Harbor
	huts     map[hutKey]string
	life     *life
	post     CampPlan
	hamlet   *Hamlet
	beds     [gardenKinds]*ebiten.Image
	gates    []GateMark
	foremen  *Foremen
	labor    *labor
	ages     *ageClock
	looks    map[lookKey][]string
	decor    *forumDecor
	debuts   map[string]float64
	shares   map[hideKey]float64
	marks    []CauseMark
	placed   []ui.Rect
	boxes    []DrawnBox
	prices   map[string]keptPrice
	stores   []StorageMark
	couriers courierMemo
	fields   *keepLand
	stone    stoneState
	living   *living
	focus    FocusMark
	inset    bool
	clip     image.Rectangle
	clipping bool
	items    []Drawable
	hitBuf   []Hit
	builtHit []Hit
	depth    depthSort
}

func NewRenderer(lib *gfx.Library) *Renderer {
	return &Renderer{Lib: lib, Homes: map[string]Vec{}, kindled: map[string]float64{}, shores: map[string]Vec{}}
}

func (r *Renderer) Draw(dst *ebiten.Image, cam render.Camera, sc Scene) {
	r.boxes = r.boxes[:0]
	screen := dst.Bounds()
	r.drawTerrain(dst, cam, sc, screen)
	r.stepLiving(sc)
	r.drawForumGround(dst, cam, sc)
	r.drawKeepLand(dst, cam, sc)
	r.drawLivingGround(dst, cam, sc)
	r.HarborOf().Step(sc)
	r.drawDecks(dst, cam, sc)
	r.DrawTints(dst, cam, sc)
	r.drawLens(dst, cam, sc)
	r.drawSelectionGround(dst, cam, sc)
	r.drawLifeGround(dst, cam, sc)
	r.drawRollRoads(dst, cam, sc)
	restore := r.clipTo(screen)
	items := r.sceneItems(cam, sc)
	restore()
	stale := len(r.hitBuf)
	r.hitBuf = r.hitBuf[:0]
	for _, it := range items {
		if it.Draw != nil {
			it.Draw(dst)
		}
		if it.Hit != nil {
			r.hitBuf = append(r.hitBuf, *it.Hit)
		}
	}
	clearTail(r.hitBuf, stale)
	stale = len(r.HitList)
	r.HitList = append(append(r.HitList[:0], r.deckHits...), r.hitBuf...)
	clearTail(r.HitList, stale)
	r.drawForemanThreads(dst, cam, sc)
	r.drawLifeOverlay(dst, cam, sc)
	r.DrawOmenXray(dst, cam, sc)
	r.drawIncidents(dst, cam, sc)
	r.drawChests(dst, cam, sc)
	r.drawOverlays(dst, cam, sc)
	drawNight(dst, sc.DayPhase)
	r.drawLifeGlow(dst, cam, sc)
	r.drawLivingGlow(dst, cam, sc)
}

func (r *Renderer) sceneItems(cam render.Camera, sc Scene) []Drawable {
	stale := len(r.items)
	items := r.stoneItems(r.collect(cam, sc), cam, sc)
	items = append(items, r.lifeItems(cam, sc)...)
	items = append(items, r.livingItems(cam, sc)...)
	items = append(items, r.BorderItems(cam, sc)...)
	items = append(items, r.forumItems(cam, sc)...)
	items = append(items, r.foremanItems(cam, sc)...)
	r.depth.lift(items, cam.Scale())
	r.depth.sort(items)
	clearTail(items, stale)
	r.items = items
	return items
}

func clearTail[T any](s []T, stale int) {
	if stale > len(s) && stale <= cap(s) {
		clear(s[len(s):stale])
	}
}

func (r *Renderer) Home(pod string) (Vec, bool) {
	v, ok := r.Homes[pod]
	return v, ok
}

func elevationAt(site view.Site, v Vec) float64 {
	return float64(site.ElevationAt(int(math.Floor(v.X)), int(math.Floor(v.Y))))
}

func ScreenPointOf(cam render.Camera, site view.Site, v Vec) (float64, float64) {
	return cam.GridToScreenElevated(v.X, v.Y, elevationAt(site, v))
}

func ScreenOf(cam render.Camera, site view.Site, v Vec) (float64, float64) {
	return ScreenPointOf(cam, site, v)
}

func SpriteRect(lib *gfx.Library, name string, sx, sy, zoom float64, fallbackW, fallbackH int) ui.Rect {
	_, reg, ok := lib.Lookup(name)
	if !ok {
		w, h := float64(fallbackW)*zoom, float64(fallbackH)*zoom
		return ui.Rect{X: int(sx - w/2), Y: int(sy - h), W: int(w), H: int(h)}
	}
	return ui.Rect{
		X: int(sx - float64(reg.AnchorX)*zoom), Y: int(sy - float64(reg.AnchorY)*zoom),
		W: int(float64(reg.W) * zoom), H: int(float64(reg.H) * zoom),
	}
}

func daylight(phase float64) float64 {
	return .5 - .5*math.Cos(2*math.Pi*phase)
}

const nightDepth = .55

func Darkness(phase float64) float64 {
	return min(max((.62-daylight(phase))/.62, 0), 1) * nightDepth
}
