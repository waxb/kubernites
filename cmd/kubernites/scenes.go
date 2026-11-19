package main

import (
	"github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/hud"
)

func init() {
	app.CaptureScenes["live-title"] = func(g *app.Game) hud.Input {
		g.Scene = app.SceneTitle
		return hud.Input{MX: -10, MY: -10}
	}
}
