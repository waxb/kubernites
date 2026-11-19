package hud

import (
	"fmt"
	"math"
	"strings"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const volumePrefix = "menu:vol:"

type Volumes struct {
	Master float64
	Music  float64
	Sounds float64
}

var volumeRows = []struct{ key, label, tip string }{
	{"master", "MASTER", "Every sound in Kubernites, music and effects together"},
	{"music", "MUSIC", "The music's volume"},
	{"sounds", "SOUNDS", "The sound effects' volume: coins, ships, alarms and clicks"},
}

func (v Volumes) of(key string) float64 {
	switch key {
	case "master":
		return v.Master
	case "music":
		return v.Music
	}
	return v.Sounds
}

func volumeButtons(y int, v Volumes) []Button {
	out := []Button{}
	for i, r := range volumeRows {
		ry := y + i*22
		label := fmt.Sprintf("%s %d%%", r.label, int(math.Round(v.of(r.key)*100)))
		out = append(out,
			Button{ID: volumePrefix + r.key, Rect: ui.Rect{X: MenuBox.X + 20, Y: ry, W: MenuBox.W - 88, H: 18}, Label: label, Tip: r.tip},
			Button{ID: volumePrefix + r.key + ":-", Rect: ui.Rect{X: MenuBox.X + MenuBox.W - 64, Y: ry, W: 20, H: 18}, Label: "-", Tip: "Quieter"},
			Button{ID: volumePrefix + r.key + ":+", Rect: ui.Rect{X: MenuBox.X + MenuBox.W - 40, Y: ry, W: 20, H: 18}, Label: "+", Tip: "Louder"},
		)
	}
	return out
}

func volumeIntents(cmd string, click Intent) []Intent {
	if !strings.HasSuffix(cmd, ":-") && !strings.HasSuffix(cmd, ":+") {
		return []Intent{click}
	}
	return []Intent{click, {Kind: IntentVolume, Command: cmd}}
}
