package app

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"

	"github.com/waxb/kubernites/pkg/client"
	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/sound"
)

const (
	WindowWidth  = 1440
	WindowHeight = 810
)

type Options struct {
	SavePath     string
	Logger       *slog.Logger
	Capture      string
	CaptureScene string
	HasSave      bool
	NewRealm     func() client.Realm
	DevDir       string
	RestoredNote string
	AutoPause    bool
	LiveOnly     bool
	Live         LiveSource
	SettingsDir  string
	ReadOnly     bool
	Window       windowControl `json:"-"`
}

func (o Options) LoggerOrDiscard() *slog.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (o Options) RealmOrNew() client.Realm {
	if o.Capture != "" {
		return campaignHook.FoundRealm(CaptureSeed, CaptureEpoch)
	}
	if o.NewRealm != nil {
		return o.NewRealm()
	}
	return campaignHook.NewRealm()
}

func Run(r client.Realm, opts Options) error {
	if r == nil && opts.Capture == "" && !opts.LiveOnly {
		return fmt.Errorf("run the client: no realm given")
	}
	if r == nil && !opts.LiveOnly {
		r = opts.RealmOrNew()
	}
	lib := gfx.Load(art.Build())
	lib.SetAsync(opts.Capture == "")
	snd := OpenAudio(opts.Capture != "")
	defer CloseAudio(snd, opts.LoggerOrDiscard())
	g := NewGame(r, opts, lib, snd)
	if opts.Capture != "" {
		c, err := NewCapture(opts.Capture, opts.CaptureScene, g)
		if err != nil {
			return err
		}
		g.Capture = c
		snd.SetMuted(true)
	}
	ebiten.SetWindowTitle("Kubernites")
	ebiten.SetWindowSize(WindowWidth, WindowHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetWindowSizeLimits(hud.Width/2, hud.Height/2, -1, -1)
	if err := ebiten.RunGame(g); err != nil {
		return fmt.Errorf("run the client: %w", err)
	}
	if g.Capture != nil {
		return g.Capture.Err
	}
	return nil
}

func OpenAudio(headless bool) *sound.Manager {
	if headless || !hostAudioProbe().hasDevice() {
		return sound.NewSilent()
	}
	if ctx := audio.CurrentContext(); ctx != nil {
		return sound.New(ctx)
	}
	return sound.New(audio.NewContext(sound.SampleRate))
}

func CloseAudio(m *sound.Manager, log *slog.Logger) {
	if err := m.Err(); err != nil {
		log.Warn("audio playback problem", "err", err)
	}
	if err := m.Close(); err != nil {
		log.Warn("close audio", "err", err)
	}
}
