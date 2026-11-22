package app

import (
	"github.com/waxb/kubernites/pkg/client/names"
	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	namerCache = 4096
	KubeScene  = "+k8s"
)

func ApplyNames(on bool) {
	if !on {
		ui.SetStoryNamer(newStoryNamer())
		return
	}
	ui.SetNamer(names.Kubernetes(namerCache))
}

func (g *Game) toggleNames() {
	if g.namesAreLive() {
		on := !g.liveNames()
		g.Settings.LiveNames = &on
		ApplyNames(on)
		if g.Sess != nil {
			g.Sess.TrueNames = on
		}
		if err := SaveSettings(g.SettingsPath(), g.Settings); err != nil {
			g.Log.Warn("could not save settings", "err", err)
		}
		return
	}
	g.Settings.TrueNames = !g.Settings.TrueNames
	ApplyNames(g.Settings.TrueNames)
	if g.Sess != nil {
		g.Sess.TrueNames = g.Settings.TrueNames
	}
	if err := SaveSettings(g.SettingsPath(), g.Settings); err != nil {
		g.Log.Warn("could not save settings", "err", err)
	}
}

func (g *Game) namesAreLive() bool {
	if g.Sess != nil {
		return g.Sess.Live
	}
	return g.Title.LiveOnly
}

func (g *Game) liveNames() bool {
	return g.Settings.LiveNames == nil || *g.Settings.LiveNames
}
