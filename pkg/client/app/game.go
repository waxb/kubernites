package app

import (
	"errors"
	"image/color"
	"log/slog"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/sound"
	"github.com/waxb/kubernites/pkg/live/write"
)

const (
	SceneTitle   = "title"
	SceneRealm   = "realm"
	TickSeconds  = 1.0 / 60
	SaveInterval = 30.0
)

var (
	ErrQuit    = errors.New("quit")
	background = color.RGBA{24, 52, 74, 255}
)

type Game struct {
	Opts       Options      `json:"-"`
	Log        *slog.Logger `json:"-"`
	Lib        *gfx.Library `json:"-"`
	audio      *sound.Manager
	Scene      string       `json:"-"`
	Title      hud.Title    `json:"-"`
	Demo       *Session     `json:"-"`
	Sess       *Session     `json:"-"`
	Pending    client.Realm `json:"-"`
	Pointer    Pointer      `json:"-"`
	SaveTick   float64      `json:"-"`
	Final      FinalState   `json:"-"`
	Capture    *Capture     `json:"-"`
	Settings   Settings     `json:"-"`
	Controls   Controls     `json:"-"`
	Reforged   bool         `json:"-"`
	Fade       float64      `json:"-"`
	FadeDir    int          `json:"-"`
	window     windowControl
	Dev        *DevRecorder `json:"-"`
	slot       int
	checkedAge string
	Trial      TrialPlay      `json:"-"`
	Yard       TutorialPlay   `json:"-"`
	Chapters   hud.TitleMap   `json:"-"`
	Picker     hud.LivePicker `json:"-"`
	Opening    *liveOpening   `json:"-"`
	LiveLimit  time.Duration  `json:"-"`
	asking     *liveAsking
	campaign   *Session
	choices    write.Memory
}

func NewGame(r client.Realm, opts Options, lib *gfx.Library, audio *sound.Manager) *Game {
	g := &Game{Opts: opts, Log: opts.LoggerOrDiscard(), Lib: lib, audio: audio, Scene: SceneTitle, Pending: r}
	g.Title.HasSave, g.Title.Version, g.Title.LiveOnly = opts.HasSave, VersionLabel(), opts.LiveOnly
	g.Picker.Lib = lib
	g.Picker.Forced = opts.ReadOnly
	settings, err := LoadSettings(opts.SettingsFile())
	if err != nil {
		g.Log.Warn("settings ignored", "err", err)
	}
	g.Settings = settings
	g.window = opts.WindowControl()
	if settings.Fullscreen {
		g.window.SetFullscreen(true)
	}
	ApplyNames(settings.TrueNames)
	g.applyVolumes()
	controls, err := LoadControls(ControlsPath(opts.SavePath))
	if err != nil {
		g.Log.Warn("controls ignored", "err", err)
	}
	g.Controls = controls
	g.Pointer.Edge = controls.EdgeScroll
	g.Pointer.toLogical = g.ToLogical
	if opts.DevDir != "" && opts.Capture == "" {
		if g.Dev, err = OpenDevRecorder(opts.DevDir, VersionLabel(), g.Log); err != nil {
			g.Log.Warn("playtest recording off", "err", err)
		}
	}
	return g
}

func (g *Game) Update() error {
	if ebiten.IsWindowBeingClosed() {
		g.Shutdown()
		return ebiten.Termination
	}
	if g.Capture != nil {
		return g.Capture.step(g)
	}
	now := g.Clock()
	in := g.ApplyFullscreenKey(g.Pointer.Gather(now))
	var err error
	if g.Scene == SceneTitle {
		err = g.UpdateTitle(in)
	} else {
		err = g.UpdateRealm(in)
	}
	g.audio.Update()
	if errors.Is(err, ErrQuit) {
		g.Shutdown()
		return ebiten.Termination
	}
	return err
}

func (g *Game) Shutdown() {
	g.Save()
	g.MarkClosed()
	g.Dev.Close()
	g.stopWatching()
	if g.Sess != nil {
		g.Sess.stepForwards()
	}
}

func (g *Game) Clock() float64 {
	if g.Sess != nil && g.Scene == SceneRealm {
		return g.Sess.Clock
	}
	if g.Demo != nil {
		return g.Demo.Clock
	}
	return 0
}

func (g *Game) UpdateTitle(in hud.Input) error {
	g.audio.PlayMusic(sound.MusicTitle)
	g.ensureDemo()
	if g.Demo != nil {
		g.Demo.Advance(TickSeconds)
		g.Demo.Cam = g.Demo.Cam.Pan(math.Cos(g.Demo.Clock/9)*0.25, math.Sin(g.Demo.Clock/7)*0.12).Clamp(g.Demo.Site.Width, hud.Width, hud.Height)
	}
	g.StepLive()
	var intents []hud.Intent
	switch {
	case g.Chapters.Active():
		g.Chapters.Reduce = g.Settings.ReduceEffects
		intents = g.Chapters.Update(in, TickSeconds)
	case g.Picker.Active():
		intents = g.Picker.Update(in)
	default:
		intents = g.Title.Update(in)
	}
	for _, it := range intents {
		if err := g.TitleIntent(it); err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) TitleIntent(it hud.Intent) error {
	if g.liveIntent(it) {
		return nil
	}
	switch it.Kind {
	case hud.IntentSound:
		g.audio.PlaySFX(sound.SFX(it.Command))
		return nil
	case hud.IntentQuit:
		return ErrQuit
	case hud.IntentLoad:
		if g.Sess == nil {
			g.Sess = g.OpenSession(g.Pending)
		}
		g.Scene = SceneRealm
	case hud.IntentNew:
		g.StartNew()
	case hud.IntentTutorial:
		g.TutorialIntent(it)
		return nil
	}
	g.audio.PlaySFX(sound.SFXClick)
	return nil
}

func (g *Game) StartNew() {
	r := g.Pending
	if g.Opts.HasSave || g.Sess != nil {
		BackupSave(g.Opts.SavePath, g.Log)
		g.saveSet().Reset()
		r = g.Opts.RealmOrNew()
	}
	g.Sess = g.OpenSession(r)
	g.Sess.BeginIntro()
	g.Sess.Hud.Open(hud.OverlayWelcome)
	g.Opts.HasSave = true
	g.Title.HasSave = true
	g.Scene = SceneRealm
	g.SaveTick = 0
}

func (g *Game) UpdateRealm(in hud.Input) error {
	return g.StepRealm(in, TickSeconds)
}

func (g *Game) StepRealm(in hud.Input, dt float64) error {
	s := g.Sess
	s.Hud.SetEdgeScroll(g.Pointer.Edge)
	s.Fullscreen = g.window.IsFullscreen()
	if g.Dev != nil && !s.Hud.StoneTerminal() {
		in = g.Dev.intercept(in, s)
	}
	rest, sfx := s.Update(in, dt, &g.Pointer, g.audio.Muted())
	if g.Dev != nil {
		g.Dev.Observe(s)
	}
	pointerShape(s.Cursor || s.Hud.HelpMode())
	if s.Hud.TaughtChanged() {
		g.savePanels()
	}
	g.HushDuringCalm()
	g.audio.PlayAll(sfx...)
	g.voice(s.TakeVoice())
	g.PlayMusic()
	for _, it := range rest {
		if err := g.RealmIntent(it); err != nil {
			return err
		}
	}
	if g.Scene != SceneRealm || g.Sess == nil {
		return nil
	}
	g.StepTrial()
	g.audio.PlayAll(g.StepChapter()...)
	g.StepFade()
	if g.AutosaveDue(in, dt) {
		g.Save()
	}
	return nil
}

func (g *Game) voice(say *workerLine) {
	switch {
	case say == nil:
	case say.Stop:
		g.audio.StopSpeech()
	default:
		g.audio.Speak(say.Voice, say.Text)
	}
}

func (g *Game) HushDuringCalm() {
	st := g.Sess.State
	g.audio.Hush(time.Duration(st.Calm / realmSpeed(st.Speed) * float64(time.Second)))
}

func (g *Game) PlayMusic() {
	st := g.Sess.State
	g.audio.PlayMusic(sound.ThemeFor(st.Age, st.Pilgrims.DayPhase, Crisis(g.Sess.Pods) || st.Siege.Phase == "active"))
}

func (g *Game) RealmIntent(it hud.Intent) error {
	switch it.Kind {
	case hud.IntentSave:
		if g.InChapter() {
			g.Sess.Hud.Toast(chapterSaveNote, false)
			break
		}
		if g.InTrial() {
			g.Sess.Hud.Toast(trialSaveNote, false)
			break
		}
		if err := g.Save(); err == nil {
			g.Sess.Hud.Toast("The realm is saved", false)
		}
	case hud.IntentMute:
		g.audio.SetMuted(!g.audio.Muted())
	case hud.IntentTitle:
		g.leaveTrial()
		g.leaveChapter(false)
		g.Save()
		g.leaveLive()
		g.Scene = SceneTitle
	case hud.IntentReduce:
		g.toggleReduce()
	case hud.IntentEdge:
		g.ToggleEdge()
	case hud.IntentStone:
		g.toggleStone()
	case hud.IntentHints:
		g.toggleHints()
	case hud.IntentLiveChoices:
		if it.Command == "forget-all" {
			g.forgetAllChoices()
		} else {
			g.forgetChoice(it.Target)
		}
		g.pushChoices()
	case hud.IntentNames:
		if g.InChapter() {
			g.Sess.Hud.Toast(chapterNamesNote, false)
			break
		}
		g.toggleNames()
	case hud.IntentFullscreen:
		g.ToggleFullscreen()
	case hud.IntentPausePolicy:
		g.cyclePausePolicy()
	case hud.IntentCoinFlights:
		g.cycleCoinFlights()
	case hud.IntentVolume:
		g.stepVolume(it.Command)
	case hud.IntentPanels:
		g.savePanels()
	case hud.IntentReforge:
		if !g.InTrial() && !g.InChapter() {
			g.beginReforge()
		}
	case hud.IntentTrial:
		if g.InChapter() {
			g.Sess.Hud.Toast(chapterTrialNote, true)
			break
		}
		g.trialIntent(it)
	case hud.IntentTutorial:
		g.TutorialIntent(it)
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(background)
	if g.Scene == SceneTitle {
		g.drawTitle(screen)
	} else if g.Sess != nil {
		g.drawSession(screen)
		if g.Dev != nil {
			g.Dev.capture(screen)
			g.Dev.draw(screen)
		}
	}
	g.drawScaleHint(screen)
	if g.Capture != nil {
		g.Capture.draw(screen)
	}
}

func (g *Game) drawTitle(screen *ebiten.Image) {
	g.drawTitlePage(screen)
	g.drawTitleLayers(screen)
}

func (g *Game) drawTitlePage(screen *ebiten.Image) {
	g.ensureDemo()
	if g.Demo != nil {
		sc := g.Demo.Scene()
		g.Demo.Renderer.Draw(screen, g.Demo.Cam, sc)
	}
	g.Title.Draw(screen)
}

func (g *Game) drawTitleLayers(screen *ebiten.Image) {
	g.Chapters.Draw(screen, g.Lib)
	g.Picker.Draw(screen)
}

func (g *Game) drawSession(screen *ebiten.Image) {
	g.Sess.Draw(screen, g.audio.Muted())
	g.drawPlacementHint(screen)
	g.drawFade(screen)
}

func (g *Game) Layout(int, int) (int, int) {
	return hud.Width, hud.Height
}
