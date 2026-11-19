package app

import (
	"errors"
	"fmt"
	"image"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client"
	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	worldRegion = "world"
	lineSlack   = 3
	runGap      = "  "
	screenTPS   = 600
)

var ErrNoScreen = errors.New("no screen: run the test under app.RunScreen to take screenshots")

type RegionText struct {
	Region string
	Lines  []string
}

type ScreenText struct {
	Regions []RegionText
}

func (s ScreenText) Region(name string) []string {
	for _, r := range s.Regions {
		if r.Region == name {
			return r.Lines
		}
	}
	return nil
}

type shotRequest struct {
	draw func(screen *ebiten.Image)
	out  chan *image.RGBA
}

type screenGame struct {
	run   func() int
	start sync.Once
	code  int
	done  chan struct{}
	shots chan shotRequest
	calls chan func()
}

func onScreenThread(f func()) {
	g := screen.Load()
	if g == nil {
		f()
		return
	}
	done := make(chan struct{})
	g.calls <- func() {
		defer close(done)
		f()
	}
	<-done
}

var screen atomic.Pointer[screenGame]

func NewScreenProbe(r client.Realm) *GuideProbe {
	return &GuideProbe{S: NewSession(HeldRealm{r}, gfx.Load(art.Build())), dst: ebiten.NewImage(hud.Width, hud.Height)}
}

func StoryMode() func() {
	ApplyNames(false)
	return func() { ui.SetNamer(nil) }
}

func NamesMode() func() {
	ApplyNames(true)
	return func() { ui.SetNamer(nil) }
}

func (p *GuideProbe) Screen() ScreenText {
	var runs []ui.TextRun
	onScreenThread(func() {
		stop := ui.RecordText(func(r ui.TextRun) { runs = append(runs, r) })
		p.dst.Clear()
		ui.InRegion(worldRegion, func() { p.S.Draw(p.dst, true) })
		stop()
	})
	return ScreenOf(runs)
}

func (p *GuideProbe) ScreenSecond() GuideCheck {
	var check GuideCheck
	onScreenThread(func() { check = p.Second() })
	return check
}

func (p *GuideProbe) Tips() []hud.TipCard {
	return p.S.Hud.TipCards(p.S.Frame(true))
}

func (p *GuideProbe) Shot(path string) error {
	g := screen.Load()
	if g == nil {
		return ErrNoScreen
	}
	out := make(chan *image.RGBA, 1)
	g.shots <- shotRequest{draw: func(dst *ebiten.Image) { p.S.Draw(dst, true) }, out: out}
	return WritePNG(path, <-out)
}

func ScreenOf(runs []ui.TextRun) ScreenText {
	order := []string{}
	byRegion := map[string][]ui.TextRun{}
	for _, r := range runs {
		if r.Text == "" {
			continue
		}
		if _, seen := byRegion[r.Region]; !seen {
			order = append(order, r.Region)
		}
		byRegion[r.Region] = append(byRegion[r.Region], r)
	}
	out := ScreenText{}
	for _, name := range order {
		out.Regions = append(out.Regions, RegionText{Region: name, Lines: linesOf(byRegion[name])})
	}
	return out
}

func linesOf(runs []ui.TextRun) []string {
	sorted := append([]ui.TextRun(nil), runs...)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i].Rect, sorted[j].Rect
		if Abs(a.Y-b.Y) > lineSlack {
			return a.Y < b.Y
		}
		return a.X < b.X
	})
	out := []string{}
	lastY := 0
	for i, r := range sorted {
		if i > 0 && Abs(r.Rect.Y-lastY) <= lineSlack {
			out[len(out)-1] += runGap + r.Spelled
			continue
		}
		out = append(out, r.Spelled)
		lastY = r.Rect.Y
	}
	return out
}

func RunScreen(run func() int) int {
	g := &screenGame{run: run, done: make(chan struct{}), shots: make(chan shotRequest), calls: make(chan func())}
	screen.Store(g)
	ebiten.SetWindowSize(hud.Width, hud.Height)
	ebiten.SetWindowTitle("Kubernites screen transcript")
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetTPS(screenTPS)
	ebiten.SetScreenClearedEveryFrame(false)
	if err := ebiten.RunGame(g); err != nil && !errors.Is(err, ebiten.Termination) {
		fmt.Fprintln(os.Stderr, "screen transcript:", err)
		return 1
	}
	return g.code
}

func (g *screenGame) Update() error {
	g.start.Do(func() {
		go func() {
			g.code = g.run()
			close(g.done)
		}()
	})
	for {
		select {
		case <-g.done:
			return ebiten.Termination
		case call := <-g.calls:
			call()
		default:
			return nil
		}
	}
}

func (g *screenGame) Draw(dst *ebiten.Image) {
	select {
	case req := <-g.shots:
		dst.Clear()
		req.draw(dst)
		img := image.NewRGBA(dst.Bounds())
		dst.ReadPixels(img.Pix)
		req.out <- img
	default:
	}
}

func (g *screenGame) Layout(int, int) (int, int) { return hud.Width, hud.Height }

func RegionLabel(region string) string {
	switch {
	case strings.HasPrefix(region, hud.PanelRegion):
		return "Panel " + strings.TrimPrefix(region, hud.PanelRegion)
	case strings.HasPrefix(region, hud.WindowRegion):
		return windowLabels.of(strings.TrimPrefix(region, hud.WindowRegion))
	case strings.HasPrefix(region, hud.ConsoleRegion):
		return consoleLabels.of(strings.TrimPrefix(region, hud.ConsoleRegion))
	}
	return layerLabels.of(region)
}

type labels map[string]string

func (l labels) of(id string) string {
	if v, ok := l[id]; ok {
		return v
	}
	return id
}

var (
	windowLabels  = labels{hud.PanelObjective: "Objective card", hud.PanelHarbor: "Harbor board", hud.PanelLesson: "Lesson card", hud.PanelNotices: "Rail", hud.PanelCalamity: "Calamity card"}
	consoleLabels = labels{"strip": "Realm panel", "minimap": "Minimap", "card": "Selection card", "well": "Event log", "commands": "Command grid"}
	layerLabels   = labels{worldRegion: "World", "speech": "Speech bubble", "topbar": "Top bar", "alerts": "Alarms", "intro": "Intro card", "toasts": "Toast", "banner": "Banner",
		"hint": "Hint", "guide": "Guide", "purse": "Purse tab", "coins": "Coins", "tooltip": "Tooltip", "worldtip": "World tip", "deltas": "Deltas", "siege": "Siege strip", "survey": "Survey", "legend": "Legend", "spot": "Objective spot", "lens": "Lens", hud.TeacherRegion: "Teacher card", titleRegion: "Title page", mapRegion: "Chapter map", liveRegion: "Live cluster"}
)
