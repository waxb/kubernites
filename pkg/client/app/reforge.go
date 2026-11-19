package app

import (
	"github.com/waxb/kubernites/pkg/client"
	"path/filepath"
)

const (
	fadeSeconds  = 0.8
	SwapCooldown = 1.0
)

func (g *Game) OpenSession(r client.Realm) *Session {
	s := g.settledSession(r)
	g.opened(s)
	return s
}

func (g *Game) settledSession(r client.Realm) *Session {
	s := NewSession(r, g.Lib)
	s.Reduce = g.Settings.ReduceEffects
	s.TrueNames = g.Settings.TrueNames
	s.volumes = g.volumes()
	s.Hud.SetCoinFlights(g.Settings.CoinFlights)
	s.Hud.SetPanels(g.Settings.Panels)
	s.Hud.SetTrackerFolded(g.Settings.DecreeFolded)
	s.Hud.SetTaught(g.Settings.Taught)
	s.Hud.SetPauseFor(g.Settings.PauseFor)
	s.Hud.SetStoneMode(g.Settings.Stone)
	s.AutoPauseOn = g.Opts.AutoPause
	if lv, ok := r.(liveView); ok {
		s.Writer = g.writerFor(r)
		s.hints = g.Settings.LiveHints
		s.goLive(lv)
	}
	return s
}

func (g *Game) beginReforge() {
	if g.Sess == nil || g.FadeDir != 0 || (g.Reforged && g.Sess.Clock < SwapCooldown) {
		return
	}
	g.FadeDir = 1
}

func (g *Game) reforge() {
	s := g.Sess
	if s == nil {
		return
	}
	if err := g.Save(); err != nil {
		s.Hud.Toast("The realm was not saved, so it was not reforged.", true)
		return
	}
	next, err := campaignHook.Reforge(s.Realm)
	if err != nil {
		s.Hud.Toast("The realm cannot be reforged: "+err.Error(), true)
		return
	}
	BackupSave(g.Opts.SavePath, g.Log)
	g.saveSet().Reset()
	g.Sess = g.OpenSession(next)
	g.Sess.Hud.Banner("THE REALM IS REFORGED")
	g.Reforged, g.Fade = true, 1
	g.SaveTick = 0
	g.Save()
}

func (g *Game) toggleReduce() {
	g.Settings.ReduceEffects = !g.Settings.ReduceEffects
	if g.Sess != nil {
		g.Sess.Reduce = g.Settings.ReduceEffects
	}
	if err := SaveSettings(g.SettingsPath(), g.Settings); err != nil {
		g.Log.Warn("could not save settings", "err", err)
	}
}

func (g *Game) SettingsPath() string {
	if g.Capture != nil {
		return ""
	}
	return g.Opts.SettingsFile()
}

func (g *Game) LiveDir() string {
	path := g.SettingsPath()
	if path == "" {
		return ""
	}
	return filepath.Dir(path)
}

func (g *Game) StepFade() {
	switch {
	case g.FadeDir > 0:
		g.Fade = min(g.Fade+TickSeconds/fadeSeconds, 1)
		if g.Fade >= 1 {
			g.FadeDir = -1
			g.reforge()
		}
	default:
		g.Fade = max(g.Fade-TickSeconds/fadeSeconds, 0)
		if g.Fade == 0 {
			g.FadeDir = 0
		}
	}
}
