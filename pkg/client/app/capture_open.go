package app

import "github.com/waxb/kubernites/pkg/client/hud"

var openScenes = map[string]bool{"open-new": true, "open-learn": true}

func init() {
	CaptureScenes["open-new"] = setupOpenNew
	CaptureScenes["open-learn"] = setupOpenLearn
}

func setupOpenNew(g *Game) hud.Input {
	g.Scene = SceneTitle
	g.StartNew()
	return hud.Input{MX: -10, MY: -10}
}

func setupOpenLearn(g *Game) hud.Input {
	g.Scene = SceneTitle
	g.StartChapter("ch1")
	return hud.Input{MX: -10, MY: -10}
}
