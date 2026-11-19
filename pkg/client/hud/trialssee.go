package hud

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/lexicon"
	"github.com/waxb/kubernites/pkg/view"
)

const seeItTitle = "SEE IT IN KUBERNETES"

var seeLiteral = ui.Font{Literal: true}

func storyYAML(lines []string) []string {
	return strings.Split(lexicon.StoryManifest(strings.Join(lines, "\n")), "\n")
}

func (h *HUD) drawSeeIt(dst *ebiten.Image, k view.TrialKube, x, y, w, bottom int) int {
	seeLiteral.Draw(dst, seeItTitle, x, y, 1, ui.Palette.Blue)
	y += ui.LineHeight + 2
	cells := ui.MaxChars(w, 1)
	story := !ui.Naming()
	for _, cmd := range k.Commands {
		if story {
			seeLiteral.Draw(dst, seeLiteral.Clip(lexicon.StoryText(cmd), cells), x, y, 1, ui.Palette.Text)
			y += ui.LineHeight
		}
		seeLiteral.Draw(dst, seeLiteral.Clip("$ "+cmd, cells), x, y, 1, ui.Palette.GoldBright)
		y += ui.LineHeight + 2
	}
	y = h.drawYAML(dst, k.YAML, x, y+2, story) + 4
	for _, row := range seeLiteral.Wrap(k.Rule, w, 1) {
		if y+ui.LineHeight > bottom {
			return y
		}
		seeLiteral.Draw(dst, row, x, y, 1, ui.Palette.Text)
		y += ui.LineHeight
	}
	if y+ui.LineHeight <= bottom {
		seeLiteral.Draw(dst, k.Doc, x, y+2, 1, ui.Palette.Blue)
		y += ui.LineHeight + 2
	}
	return y
}

func (h *HUD) drawYAML(dst *ebiten.Image, lines []string, x, y int, story bool) int {
	right := x
	if story {
		for i, l := range storyYAML(lines) {
			seeLiteral.Draw(dst, l, x, y+i*ui.LineHeight, 1, ui.Palette.Text)
		}
		right = x + trialYAMLW + trialColGap
	}
	for i, l := range lines {
		seeLiteral.Draw(dst, l, right, y+i*ui.LineHeight, 1, ui.Palette.GoldBright)
	}
	return y + len(lines)*ui.LineHeight
}

func seeItHeight(k view.TrialKube, w int) int {
	y := ui.LineHeight + 2
	for range k.Commands {
		if !ui.Naming() {
			y += ui.LineHeight
		}
		y += ui.LineHeight + 2
	}
	y += 2 + len(k.YAML)*ui.LineHeight + 4
	y += len(seeLiteral.Wrap(k.Rule, w, 1)) * ui.LineHeight
	return y + ui.LineHeight + 2
}
