package app

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	stageSettleStep  = 0.5
	stageSettleLimit = 240.0
	CaptureSeed      = 42
	CaptureFrames    = 90
	LateFrames       = 8
	PlayTicks        = 6
	frameMark        = "@"
)

var CaptureEpoch = time.Date(2026, 9, 1, 6, 0, 0, 0, time.UTC)

type Capture struct {
	path    string
	Scene   string `json:"-"`
	Frames  int    `json:"-"`
	input   hud.Input
	Done    bool                            `json:"-"`
	Err     error                           `json:"-"`
	Grab    func(*ebiten.Image) *image.RGBA `json:"-"`
	hook    GrabHook
	window  image.Point
	late    func(*Game)
	lateAt  int
	live    bool
	at      int
	game    *Game
	settled bool
	fresh   bool
	play    *playing
}

type Play struct {
	Until func(*Session) bool `json:"-"`
	Limit float64             `json:"-"`
	Then  float64             `json:"-"`
	Each  func(*Session)      `json:"-"`
}

var PlayScenes = map[string]Play{}

type playing struct {
	Play
	ticks, after int
	met          bool
}

func (p *playing) frame(s *Session, in hud.Input) bool {
	p.met = p.met || float64(p.ticks)*TickSeconds >= p.Limit || p.Until != nil && p.Until(s)
	if p.met && float64(p.after)*TickSeconds >= p.Then {
		return false
	}
	for range PlayTicks {
		s.Update(in, TickSeconds, &Pointer{}, true)
	}
	if p.Each != nil {
		p.Each(s)
	}
	p.ticks += PlayTicks
	if p.met {
		p.after += PlayTicks
	}
	return true
}

type LateHook struct {
	Run    func(*Game) `json:"-"`
	Before int         `json:"-"`
}

var LateHooks = map[string]LateHook{}

var CaptureScenes = map[string]func(*Game) hud.Input{}

func NewCapture(path, scene string, g *Game) (*Capture, error) {
	if scene == "" {
		scene = "realm"
	}
	if base, ok := strings.CutSuffix(scene, KubeScene); ok {
		scene, g.Settings.TrueNames = base, true
		ApplyNames(true)
	}
	scene, at, err := SceneFrame(scene)
	if err != nil {
		return nil, err
	}
	setup, ok := CaptureScenes[scene]
	if !ok {
		return nil, fmt.Errorf("unknown capture scene %q", scene)
	}
	g.Opts.SavePath, g.Settings.Panels = "", nil
	c := &Capture{path: path, Scene: scene, Grab: readScreen, at: at, game: g}
	c.input = setup(g)
	c.live = LiveScenes[scene]
	if hook, ok := GrabHooks[scene]; ok {
		c.hook = hook()
	}
	if size, ok := WindowScenes[scene]; ok {
		c.window, c.hook = size, windowGrab(size)
	}
	if hook, ok := LateHooks[scene]; ok {
		c.late, c.lateAt = hook.Run, at-hook.Before
	}
	if play, ok := PlayScenes[scene]; ok {
		c.play, c.live = &playing{Play: play}, true
	}
	return c, nil
}

func SceneFrame(scene string) (string, int, error) {
	base, frame, ok := strings.Cut(scene, frameMark)
	if !ok {
		return scene, CaptureFrames, nil
	}
	at, err := strconv.Atoi(frame)
	if err != nil || at < 1 {
		return "", 0, fmt.Errorf("capture scene %q: the frame after %s is a positive number", scene, frameMark)
	}
	return base, at, nil
}

func (c *Capture) step(g *Game) error {
	if c.fresh && !c.Done {
		return nil
	}
	c.fresh = true
	return c.Update(g)
}

func (c *Capture) Update(g *Game) error {
	if c.Done {
		return ebiten.Termination
	}
	if c.play != nil && g.Sess != nil && g.Scene == SceneRealm {
		if c.play.frame(g.Sess, c.input) {
			return nil
		}
		c.play = nil
	}
	c.Frames++
	if g.Scene == SceneTitle {
		g.ensureDemo()
		if g.Demo != nil {
			g.Demo.Advance(TickSeconds)
		}
		g.Title.Update(c.input)
		return nil
	}
	if openScenes[c.Scene] {
		return g.UpdateRealm(c.input)
	}
	s := g.Sess
	if c.live {
		s.Update(c.input, TickSeconds, &Pointer{}, true)
		return nil
	}
	s.Advance(TickSeconds)
	g.StepChapter()
	s.Hud.Update(c.input, s.Frame(true))
	if c.late != nil && c.Frames == c.lateAt {
		c.late(g)
	}
	if !c.settled || c.Frames == c.at-1 {
		c.settled = true
		SettleStage(s, c.input)
	}
	return nil
}

func SettleStage(s *Session, in hud.Input) {
	s.Hud.Update(in, s.Frame(true))
	for waited := 0.0; waited < stageSettleLimit && s.Hud.StagePending(s.Frame(true)); waited += stageSettleStep {
		s.Clock += stageSettleStep
		s.Hud.Update(in, s.Frame(true))
	}
}

func (c *Capture) draw(screen *ebiten.Image) {
	c.fresh = false
	if c.Done {
		return
	}
	src, ready := screen, c.Frames >= c.at
	if c.hook != nil {
		src, ready = c.hook(screen, c.Frames)
	}
	if !ready {
		return
	}
	c.Done = true
	c.Err = WritePNG(c.path, c.Grab(src))
	if c.Err == nil && os.Getenv(LintEnv) != "" && c.game != nil {
		c.Err = c.lint(c.game)
	}
}

func readScreen(screen *ebiten.Image) *image.RGBA {
	b := screen.Bounds()
	img := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	screen.ReadPixels(img.Pix)
	return img
}

func WritePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create capture: %w", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		return fmt.Errorf("encode capture: %w", err)
	}
	return f.Close()
}

func StartRealm(g *Game, r client.Realm) *Session {
	g.Sess = g.OpenSession(r)
	g.Scene = SceneRealm
	return g.Sess
}

func FocusKeep(s *Session) {
	if k, ok := FirstKeep(s.Site); ok {
		s.CenterOn(float64(k.X)+1, float64(k.Y)+3)
	}
}

var (
	WindowScenes = map[string]image.Point{}
	WindowSizes  = map[string]image.Point{"-window": {X: WindowWidth, Y: WindowHeight}, "-small": {X: hud.Width / 2, Y: hud.Height / 2}}
	FollowOrder  = "kubectl create deployment web --image=porter:1.0"
)

func windowGrab(size image.Point) GrabHook {
	return func(screen *ebiten.Image, frame int) (*ebiten.Image, bool) {
		if frame < CaptureFrames {
			return nil, false
		}
		out := ebiten.NewImage(size.X, size.Y)
		out.Fill(ui.Palette.ConsoleEdge)
		scale := FitScale(size.X, size.Y)
		out.DrawImage(screen, &ebiten.DrawImageOptions{GeoM: FitGeoM(size.X, size.Y), Filter: FinalFilter(scale)})
		return out, true
	}
}

type GrabHook func(screen *ebiten.Image, frame int) (*ebiten.Image, bool)

var GrabHooks = map[string]func() GrabHook{}

var LiveScenes = map[string]bool{}

func DepotCentre(site view.Site, service string) (world.Vec, bool) {
	sum, n := world.Vec{}, 0
	for _, st := range site.Structures {
		if st.Object.Kind == ServiceObject && (service == "" || st.Object.Name == service) {
			a := world.Anchor(st)
			sum, n = world.Vec{X: sum.X + a.X, Y: sum.Y + a.Y}, n+1
		}
	}
	if n == 0 {
		return world.Vec{}, false
	}
	return world.Vec{X: sum.X / float64(n), Y: sum.Y / float64(n)}, true
}

const ServiceObject = "Service"
