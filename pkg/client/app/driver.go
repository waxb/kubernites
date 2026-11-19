package app

import (
	"bytes"
	"cmp"
	"compress/gzip"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client"
	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	DriverShots    = "shots"
	DriverActions  = "actions.md"
	waitCap        = 3600.0
	ticksPerSecond = 60
	waitNudge      = 1e-6
	settleTicks    = 600
	keyPan         = panSpeed / 2
	idleX, idleY   = -10, -10
	overlayBody    = "overlay:body"
	NoTip          = "(no tooltip here: click it to select it and read its card)"
	viewMoved      = "the view moved: look again to see what is on screen"
)

var (
	abovePanels = map[string]bool{"intro": true, "spot": true, "banner": true, "toasts": true, "hint": true, "guide": true, "tooltip": true, "worldtip": true}
	panKeys     = map[ebiten.Key][2]float64{
		ebiten.KeyArrowLeft: {-1, 0}, ebiten.KeyA: {-1, 0}, ebiten.KeyArrowRight: {1, 0}, ebiten.KeyD: {1, 0},
		ebiten.KeyArrowUp: {0, -1}, ebiten.KeyW: {0, -1}, ebiten.KeyArrowDown: {0, 1}, ebiten.KeyS: {0, 1},
	}
	errStoneClosed = errors.New("the Speaking Stone is not open: click it on the plaza, or press its button or Enter")
	errStoneAsleep = errors.New("the Speaking Stone has not awakened yet")
	digitRun       = regexp.MustCompile(`[0-9][0-9.,]*`)
)

type DriverOptions struct {
	Seed      uint64
	Story     bool
	Out       string
	Shots     bool
	AutoPause bool
	Chapter   string
	Title     bool
	Save      string
}

type LookRegion struct {
	Region string   `json:"region"`
	Label  string   `json:"label"`
	Lines  []string `json:"lines"`
}

type LookButton struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Text    string `json:"text,omitempty"`
	Region  string `json:"region"`
	Enabled bool   `json:"enabled"`

	CoveredBy string `json:"covered_by,omitempty"`
}

type LookObject struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

type Look struct {
	Time    string       `json:"time"`
	Seconds float64      `json:"seconds"`
	Regions []LookRegion `json:"regions"`
	Buttons []LookButton `json:"buttons"`
	World   []LookObject `json:"world"`
	Panel   string       `json:"panel"`
	Lit     string       `json:"lit,omitempty"`
	Shot    string       `json:"shot,omitempty"`
}

type Changes struct {
	Time   string       `json:"time"`
	New    []LookRegion `json:"new"`
	Closed []string     `json:"closed"`
	Output []string     `json:"output,omitempty"`
	Note   string       `json:"note,omitempty"`
}

type Tooltip struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Tip     string `json:"tip"`
	Purpose string `json:"purpose,omitempty"`
	How     string `json:"how,omitempty"`
	Why     string `json:"why,omitempty"`
	Page    string `json:"page"`
}

type Driver struct {
	mu       sync.Mutex
	G        *Game                      `json:"-"`
	Dst      *ebiten.Image              `json:"-"`
	Opts     DriverOptions              `json:"-"`
	Seen     map[string]map[string]bool `json:"-"`
	bridged  map[string]int
	shots    int
	inThread bool
	quit     bool
}

func NewDriver(o DriverOptions) (*Driver, error) {
	if o.Out != "" {
		if err := os.MkdirAll(filepath.Join(o.Out, DriverShots), 0o755); err != nil {
			return nil, fmt.Errorf("create the play folder: %w", err)
		}
		if err := os.WriteFile(filepath.Join(o.Out, DriverActions), nil, 0o644); err != nil {
			return nil, fmt.Errorf("start the play log: %w", err)
		}
	}
	r := campaignHook.FoundRealm(o.Seed, CaptureEpoch)
	if o.Title && o.Chapter != "" {
		return nil, errors.New("start at the title page or in a chapter, not both")
	}
	if o.Save != "" && (o.Title || o.Chapter != "") {
		return nil, errors.New("start from a save, the title page or a chapter, not two of them")
	}
	if o.Save != "" {
		saved, err := driverSave(o.Save)
		if err != nil {
			return nil, err
		}
		r = saved
	}
	g := NewGame(r, Options{AutoPause: o.AutoPause}, gfx.Load(art.Build()), sound.NewSilent())
	g.Settings.TrueNames = !o.Story
	ApplyNames(!o.Story)
	switch {
	case o.Title:
		g.Yard = TutorialPlay{Record: campaignHook.NewChapterRecord(), Loaded: true}
	case o.Save != "":
		StartRealm(g, r)
	default:
		g.StartNew()
	}
	if o.Chapter != "" {
		if _, err := campaignHook.StartChapter(o.Chapter); err != nil {
			return nil, err
		}
		g.Sess.Hud.Open(hud.OverlayNone)
		g.Yard = TutorialPlay{Record: campaignHook.NewChapterRecord(), Loaded: true}
		g.StartChapter(o.Chapter)
		g.Fade = 0
	}
	d := &Driver{G: g, Dst: ebiten.NewImage(hud.Width, hud.Height), Opts: o, Seen: map[string]map[string]bool{}}
	d.logLine(fmt.Sprintf("# Play log\n\nseed %d, story mode %v\n", o.Seed, o.Story))
	if o.Chapter != "" {
		d.logLine(fmt.Sprintf("chapter %s\n", o.Chapter))
	}
	if o.Title {
		d.logLine("the title page\n")
	}
	if o.Save != "" {
		d.logLine(fmt.Sprintf("from the save %s\n", o.Save))
	}
	return d, nil
}

func driverSave(path string) (client.Realm, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read the save: %w", err)
	}
	if strings.HasSuffix(path, ".gz") {
		zr, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("unzip the save: %w", err)
		}
		if raw, err = io.ReadAll(zr); err != nil {
			return nil, fmt.Errorf("unzip the save: %w", err)
		}
	}
	return campaignHook.ReadSave(raw)
}

func (d *Driver) Sess() *Session { return d.G.Sess }

func (d *Driver) Beats() []hud.Beat { return d.Sess().Hud.Beats() }

func (d *Driver) WatchAttention() { d.Sess().Hud.SetAttention(true) }

func (d *Driver) Attention() hud.AttnLog {
	s := d.Sess()
	return s.Hud.Attention().Closed(s.Clock, s.State.RealmTime)
}

func (d *Driver) Age() string { return d.Sess().State.Age }

func (d *Driver) clock() (string, float64) {
	if d.atTitle() {
		return titleTime, 0
	}
	t := d.Sess().State.RealmTime
	return Mmss(t), t
}

func Mmss(t float64) string {
	sec := int(t)
	return fmt.Sprintf("%02d:%02d", sec/60, sec%60)
}

func (d *Driver) onThread(f func()) {
	if d.inThread {
		f()
		return
	}
	onScreenThread(func() {
		d.inThread = true
		defer func() { d.inThread = false }()
		f()
	})
}

func (d *Driver) Step(in hud.Input, dt float64) {
	d.onThread(func() {
		var err error
		if d.atTitle() {
			err = d.G.UpdateTitle(in)
		} else {
			err = d.G.StepRealm(in, dt)
		}
		switch {
		case errors.Is(err, ErrQuit):
			d.quit = true
			d.logLine("- the game closes (QUIT)")
		case err != nil:
			d.logLine("- error: " + err.Error())
		}
	})
}

func (d *Driver) Settle() {
	if d.atTitle() {
		return
	}
	d.onThread(func() {
		s := d.Sess()
		for i := 0; (s.Lens.Glide.On || s.Lens.Zooming) && i < settleTicks; i++ {
			s.Clock += TickSeconds
			s.StepLens()
		}
	})
}

func (d *Driver) moved(cam render.Camera) string {
	if d.atTitle() || d.Sess().Cam == cam {
		return ""
	}
	return viewMoved
}

func (d *Driver) Capture() []ui.TextRun {
	if d.atTitle() {
		return d.captureTitle()
	}
	var runs []ui.TextRun
	d.onThread(func() {
		d.Sess().Hud.SetAppearInstant(true)
		d.Sess().Talk.whole = true
		defer func() { d.Sess().Hud.SetAppearInstant(false); d.Sess().Talk.whole = false }()
		stop := ui.RecordText(func(r ui.TextRun) { runs = append(runs, r) })
		d.Dst.Fill(background)
		ui.InRegion(worldRegion, func() { d.G.drawSession(d.Dst) })
		stop()
		d.Sess().Hud.NoteDrawn(worldSurfaces(d.Sess().Renderer.Drawn()), d.Sess().State.RealmTime)
	})
	return d.unhidden(runs)
}

func (d *Driver) unhidden(runs []ui.TextRun) []ui.TextRun {
	s := d.Sess()
	if s.Hud.Overlay() == hud.OverlayNone {
		return runs
	}
	panel, ok := ui.Rect{}, false
	for _, b := range s.Hud.Buttons(s.Frame(true)) {
		if b.ID == overlayBody {
			panel, ok = b.Rect, true
		}
	}
	if !ok {
		return runs
	}
	out := make([]ui.TextRun, 0, len(runs))
	for _, r := range runs {
		if abovePanels[r.Region] || strings.HasPrefix(r.Region, hud.PanelRegion) || !panel.Contains(r.Rect.X+r.Rect.W/2, r.Rect.Y+r.Rect.H/2) {
			out = append(out, r)
		}
	}
	return out
}

func lookRegions(st ScreenText) []LookRegion {
	out := make([]LookRegion, 0, len(st.Regions))
	for _, r := range st.Regions {
		out = append(out, LookRegion{Region: r.Region, Label: RegionLabel(r.Region), Lines: r.Lines})
	}
	return out
}

func maskedLine(line string) string {
	return strings.Join(strings.Fields(digitRun.ReplaceAllString(line, "#")), " ")
}

func (d *Driver) diff(st ScreenText, key func(string) string) ([]LookRegion, []string) {
	fresh := []LookRegion{}
	present := map[string]bool{}
	for _, r := range st.Regions {
		present[r.Region] = true
		known := map[string]bool{}
		for l := range d.Seen[r.Region] {
			known[key(l)] = true
		}
		now := map[string]bool{}
		lines := []string{}
		for _, l := range r.Lines {
			now[l] = true
			if !known[key(l)] {
				lines = append(lines, l)
			}
		}
		d.Seen[r.Region] = now
		if len(lines) > 0 {
			fresh = append(fresh, LookRegion{Region: r.Region, Label: RegionLabel(r.Region), Lines: lines})
		}
	}
	closed := []string{}
	for region := range d.Seen {
		if !present[region] {
			closed = append(closed, RegionLabel(region))
			delete(d.Seen, region)
		}
	}
	sort.Strings(closed)
	return fresh, closed
}

func exactLine(line string) string { return line }

func (d *Driver) changes() Changes {
	return d.changesBy(exactLine)
}

func (d *Driver) changesBy(key func(string) string) Changes {
	fresh, closed := d.diff(PlayText(d.Capture()), key)
	at, _ := d.clock()
	return Changes{Time: at, New: fresh, Closed: closed}
}

func (d *Driver) Look(shot bool) Look {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.atTitle() {
		return d.lookTitle(shot)
	}
	runs := d.Capture()
	st := PlayText(runs)
	d.diff(st, exactLine)
	at, secs := d.clock()
	f := d.Sess().Frame(true)
	look := Look{Time: at, Seconds: secs, Regions: lookRegions(st), Buttons: d.Buttons(f, runs), World: d.World(f), Panel: d.Sess().Hud.Overlay(), Lit: d.Sess().Hud.Lit(f)}
	if shot {
		look.Shot = d.shot(at)
	}
	line := fmt.Sprintf("- `%s` look", at)
	if look.Shot != "" {
		line += ", shot " + filepath.Base(look.Shot)
	}
	d.logRegions(line, look.Regions)
	return look
}

func (d *Driver) shot(at string) string {
	if !d.Opts.Shots || d.Opts.Out == "" || screen.Load() == nil {
		return ""
	}
	d.shots++
	name := fmt.Sprintf("%s-%03d.png", strings.ReplaceAll(at, ":", "m"), d.shots)
	path := filepath.Join(d.Opts.Out, DriverShots, name)
	g := screen.Load()
	out := make(chan *image.RGBA, 1)
	g.shots <- shotRequest{draw: func(dst *ebiten.Image) {
		if d.atTitle() {
			dst.Fill(background)
			d.G.drawTitle(dst)
			return
		}
		d.Sess().Hud.SetAppearInstant(true)
		defer d.Sess().Hud.SetAppearInstant(false)
		dst.Fill(background)
		d.G.drawSession(dst)
	}, out: out}
	if err := WritePNG(path, <-out); err != nil {
		d.logLine("- screenshot failed: " + err.Error())
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

func (d *Driver) Click(id string) (Changes, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.atTitle() {
		return d.clickTitle(id)
	}
	f := d.Sess().Frame(true)
	x, y, err := d.Target(id, f, false)
	if err != nil {
		d.logLine(fmt.Sprintf("- `%s` click `%s`: %v", Mmss(d.Sess().State.RealmTime), id, err))
		return Changes{}, err
	}
	cam := d.Sess().Cam
	d.Step(hud.Input{MX: x, MY: y}, 0)
	d.Step(hud.Input{MX: x, MY: y, Click: true}, 0)
	d.Step(hud.Input{MX: x, MY: y}, 0)
	ch := d.changes()
	d.Settle()
	ch.Note = d.moved(cam)
	d.logAction("click", id, ch)
	return ch, nil
}

func (d *Driver) Hover(id string) (Tooltip, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.atTitle() {
		return d.hoverTitle(id)
	}
	f := d.Sess().Frame(true)
	x, y, err := d.Target(id, f, true)
	if err != nil {
		return Tooltip{}, err
	}
	d.Step(hud.Input{MX: idleX, MY: idleY}, 0)
	d.Step(hud.Input{MX: x, MY: y}, 0)
	st := PlayText(d.Capture())
	tip := Tooltip{ID: id, Tip: strings.Join(append(st.Region("tooltip"), st.Region("worldtip")...), "\n"), Page: hud.NoPage}
	for _, c := range d.Sess().Hud.TipCards(f) {
		if c.ID == id {
			tip = Tooltip{ID: id, Label: named(c.Label), Tip: cmp.Or(tip.Tip, named(c.Tip)), Purpose: named(c.Purpose), How: named(c.How), Why: named(c.Why), Page: c.Page}
		}
	}
	if tip.Tip == "" {
		tip.Tip = NoTip
	}
	d.logHover(tip)
	return tip, nil
}

func named(text string) string { return ui.Plain(ui.Name(text)) }

func (d *Driver) Type(text string) (Changes, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.atTitle() {
		return Changes{}, errTitleOpen
	}
	s := d.Sess()
	switch {
	case !(hud.Frame{State: s.State, Guide: s.shownGuide()}).StoneStands():
		return Changes{}, errStoneAsleep
	case s.Hud.Overlay() != hud.OverlayStone:
		return Changes{}, errStoneClosed
	}
	s.Hud.FocusStone()
	before, cam := s.Hud.StoneWritten(), s.Cam
	d.Step(hud.Input{Chars: []rune(text)}, 0)
	d.Step(hud.Input{Keys: []ebiten.Key{ebiten.KeyEnter}}, 0)
	ch := d.changes()
	d.Settle()
	ch.Note = d.moved(cam)
	out := s.Hud.StoneOutput()
	ch.Output = append([]string(nil), out[max(len(out)-(s.Hud.StoneWritten()-before), 0):]...)
	d.logAction("type", text, ch)
	return ch, nil
}

func (d *Driver) Key(name string) (Changes, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	k, mods, err := parseChord(name)
	if err != nil {
		return Changes{}, err
	}
	if d.atTitle() {
		return d.keyTitle(name, k, mods), nil
	}
	cam := d.Sess().Cam
	d.Step(hud.Input{MX: idleX, MY: idleY, Keys: []ebiten.Key{k}, Chars: keyChar(k, mods), Shift: mods.shift, Ctrl: mods.ctrl, Alt: mods.alt}, 0)
	d.Step(hud.Input{MX: idleX, MY: idleY}, 0)
	if mods == (chordMods{}) {
		d.pan(k)
	}
	ch := d.changes()
	d.Settle()
	ch.Note = d.moved(cam)
	d.logAction("key", name, ch)
	return ch, nil
}

func (d *Driver) pan(k ebiten.Key) {
	if d.atTitle() {
		return
	}
	dir, ok := panKeys[k]
	s := d.Sess()
	if !ok || s.Hud.KeyFocus() || s.Typing {
		return
	}
	d.onThread(func() {
		s.Lens.Glide = Glide{}
		s.Cam = s.Cam.Pan(dir[0]*keyPan, dir[1]*keyPan).Clamp(max(s.Site.Width, s.Site.Height), hud.Width, hud.ConsoleY)
	})
}

func (d *Driver) Wait(seconds float64) Changes {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.atTitle() {
		return d.waitTitle(seconds)
	}
	s := d.Sess()
	seconds = min(max(seconds, 0), waitCap)
	start, ticks := s.State.RealmTime, int(math.Round(seconds/TickSeconds))
	merged := map[string]*LookRegion{}
	order := []string{}
	closed := map[string]bool{}
	note := func(ch Changes) {
		for _, r := range ch.New {
			if merged[r.Region] == nil {
				merged[r.Region], order = &LookRegion{Region: r.Region, Label: r.Label}, append(order, r.Region)
			}
			merged[r.Region].Lines = append(merged[r.Region].Lines, r.Lines...)
		}
		for _, c := range ch.Closed {
			closed[c] = true
		}
	}
	for done := 0; done < ticks; done += ticksPerSecond {
		d.onThread(func() {
			for range min(ticksPerSecond, ticks-done) {
				d.Step(hud.Input{MX: idleX, MY: idleY}, TickSeconds)
			}
			note(d.changesBy(maskedLine))
		})
	}
	s.Poll = pollSeconds
	d.Step(hud.Input{MX: idleX, MY: idleY}, min(seconds, waitNudge))
	note(d.changesBy(maskedLine))
	out := Changes{Time: Mmss(s.State.RealmTime), New: []LookRegion{}, Closed: []string{}, Note: d.Held(seconds, start)}
	for _, r := range order {
		out.New = append(out.New, *merged[r])
	}
	for c := range closed {
		out.Closed = append(out.Closed, c)
	}
	sort.Strings(out.Closed)
	d.logAction("wait", fmt.Sprint(seconds), out)
	return out
}

func (d *Driver) Held(seconds, start float64) string {
	s := d.Sess()
	switch {
	case seconds < 1 || s.State.RealmTime > start:
		return ""
	case s.Paused:
		return "time is held: the game is paused; press top:pause to go on"
	case hud.Holds(s.Hud.Overlay()):
		return "time is held while the " + s.Hud.Overlay() + " panel is open: close it"
	case s.ScrollOpen():
		return "time is held while Borg's scroll is open: press its button to read on"
	}
	return "no realm time passed"
}

func (d *Driver) logAction(verb, arg string, ch Changes) {
	var b strings.Builder
	fmt.Fprintf(&b, "- `%s` %s `%s`", ch.Time, verb, arg)
	for _, r := range ch.New {
		fmt.Fprintf(&b, "\n  - %s: %s", r.Label, strings.Join(r.Lines, " / "))
	}
	if ch.Note != "" {
		fmt.Fprintf(&b, "\n  - note: %s", ch.Note)
	}
	if len(ch.Closed) > 0 {
		fmt.Fprintf(&b, "\n  - closed: %s", strings.Join(ch.Closed, ", "))
	}
	for _, o := range ch.Output {
		fmt.Fprintf(&b, "\n  - stone: %s", o)
	}
	for _, press := range d.bridgePresses() {
		fmt.Fprintf(&b, "\n  - %s", press)
	}
	d.logLine(b.String())
}

func (d *Driver) logLine(line string) {
	if d.Opts.Out == "" {
		return
	}
	f, err := os.OpenFile(filepath.Join(d.Opts.Out, DriverActions), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "play log:", err)
		return
	}
	defer f.Close()
	if _, err := fmt.Fprintln(f, line); err != nil {
		fmt.Fprintln(os.Stderr, "play log:", err)
	}
}
