package hud

import (
	"fmt"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

const (
	TitleSettingsID      = "title:settings"
	titleSettingsBack    = "title:settings:back"
	titleSettingsCaption = "Sound, volumes, effects, fullscreen, names, the Speaking Stone and hints."
	titleSettingsHead    = 152
	titleSettingsTop     = 176
	titleSettingsGap     = 4
	titleStepW           = 20
)

var titleClick = Intent{Kind: IntentSound, Command: "click"}

type TitleSettings struct {
	Muted        bool
	Reduce       bool
	TrueNames    bool
	Edge         bool
	Fullscreen   bool
	Hints        bool
	StoneKubectl bool
	Volumes      Volumes
}

func (t *Title) SettingsOpen() bool {
	return t.settings
}

func (t *Title) openSettings(on bool) {
	t.settings, t.hover, t.hovered, t.held = on, Button{}, false, 0
}

func (t *Title) settingsButtons() []Button {
	s := t.Settings
	x, w, y := titleLeft(), titleButtonW, titleSettingsTop
	out := []Button{}
	row := func(id, label, tip string) {
		out = append(out, Button{ID: id, Rect: ui.Rect{X: x, Y: y, W: w, H: titleRowH}, Label: label, Tip: tip})
		y += titleRowH + titleSettingsGap
	}
	row("menu:mute", soundLabel(s.Muted), soundTip)
	for _, r := range volumeRows {
		label := fmt.Sprintf("%s %d%%", r.label, int(math.Round(s.Volumes.of(r.key)*100)))
		out = append(out,
			Button{ID: volumePrefix + r.key, Rect: ui.Rect{X: x, Y: y, W: w - 2*titleStepW - 8, H: titleRowH}, Label: label, Tip: r.tip},
			Button{ID: volumePrefix + r.key + ":-", Rect: ui.Rect{X: x + w - 2*titleStepW - 4, Y: y, W: titleStepW, H: titleRowH}, Label: "-", Tip: "Quieter"},
			Button{ID: volumePrefix + r.key + ":+", Rect: ui.Rect{X: x + w - titleStepW, Y: y, W: titleStepW, H: titleRowH}, Label: "+", Tip: "Louder"},
		)
		y += titleRowH + titleSettingsGap
	}
	row("menu:reduce", effectsLabel(s.Reduce), effectsTip)
	row("menu:fullscreen", fullscreenLabel(s.Fullscreen), fullscreenTip)
	row("menu:edge", edgeLabel(s.Edge), edgeTip)
	row("menu:names", namesLabel(s.TrueNames), namesTip)
	stone := stoneMenuRow(s.StoneKubectl)
	row(stone[0], stone[1], stone[2])
	row(menuHints, hintsLabel(s.Hints), hintsTip)
	row(titleSettingsBack, "BACK", "Back to the title [Esc]")
	return out
}

func (t *Title) updateSettings(in Input) []Intent {
	if in.Pressed(ebiten.KeyEscape) {
		t.openSettings(false)
		return []Intent{titleClick}
	}
	if !in.Click || !t.hovered || t.hover.Disabled {
		return nil
	}
	id := t.hover.ID
	if id == titleSettingsBack {
		t.openSettings(false)
		return []Intent{titleClick}
	}
	if cmd, ok := strings.CutPrefix(id, volumePrefix); ok {
		if strings.HasSuffix(cmd, ":-") || strings.HasSuffix(cmd, ":+") {
			return []Intent{{Kind: IntentVolume, Command: cmd}}
		}
		return []Intent{titleClick}
	}
	if it, ok := simpleIntents[id]; ok {
		return []Intent{it}
	}
	return nil
}
