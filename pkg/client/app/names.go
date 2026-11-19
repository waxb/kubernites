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
	if g.Sess != nil && g.Sess.Live {
		on := !g.liveNames()
		g.Settings.LiveNames = &on
		ApplyNames(on)
		g.Sess.TrueNames = on
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

func (g *Game) liveNames() bool {
	return g.Settings.LiveNames == nil || *g.Settings.LiveNames
}
