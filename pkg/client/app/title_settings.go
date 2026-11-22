package app

import "github.com/waxb/kubernites/pkg/client/hud"

func (g *Game) settingIntent(it hud.Intent) bool {
	switch it.Kind {
	case hud.IntentMute:
		g.audio.SetMuted(!g.audio.Muted())
	case hud.IntentReduce:
		g.toggleReduce()
	case hud.IntentEdge:
		g.ToggleEdge()
	case hud.IntentStone:
		g.toggleStone()
	case hud.IntentHints:
		g.toggleHints()
	case hud.IntentFullscreen:
		g.ToggleFullscreen()
	case hud.IntentVolume:
		g.stepVolume(it.Command)
	case hud.IntentNames:
		if g.Scene != SceneTitle {
			return false
		}
		g.toggleNames()
	default:
		return false
	}
	return true
}

func (g *Game) titleNames() bool {
	if g.namesAreLive() {
		return g.liveNames()
	}
	return g.Settings.TrueNames
}

func (g *Game) stoneKubectl() bool {
	return hud.StoneKubectlWith(g.Settings.Stone, g.titleNames())
}

func (g *Game) titleSettings() hud.TitleSettings {
	return hud.TitleSettings{
		Muted:        g.audio.Muted(),
		Reduce:       g.Settings.ReduceEffects,
		TrueNames:    g.titleNames(),
		Edge:         g.Pointer.Edge,
		Fullscreen:   g.window.IsFullscreen(),
		Hints:        g.Settings.LiveHints,
		StoneKubectl: g.stoneKubectl(),
		Volumes:      g.volumes(),
	}
}
