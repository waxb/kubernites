package hud

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func overlaps(a, b ui.Rect) bool {
	return a.X < b.X+b.W && b.X < a.X+a.W && a.Y < b.Y+b.H && b.Y < a.Y+a.H
}

func TestNoLiveChoiceLineRunsUnderAForgetButton(t *testing.T) {
	h, _ := liveHUD(t, false)
	var list []LiveChoice
	for i := range choicesShown + 1 {
		list = append(list, LiveChoice{Name: fmt.Sprintf("gke_lumen-staging_europe-west1_north-%d", i), Level: LevelSeamless, Classes: []string{"evict", "scale", "set image"}[:i%3+1]})
	}
	h.SetLiveChoices(list)
	h.OverlayID = OverlayLiveChoices
	var runs []ui.TextRun
	stop := ui.RecordText(func(r ui.TextRun) { runs = append(runs, r) })
	h.drawLiveChoices(ebiten.NewImage(Width, Height), Frame{})
	stop()
	buttons := h.liveChoiceButtons()
	all := buttons[len(buttons)-3]
	for _, b := range buttons {
		drawn := ui.Rect{X: b.Rect.X, Y: b.Rect.Y, W: b.Rect.W + 1, H: b.Rect.H + 2}
		for _, r := range runs {
			if r.Text == b.Label {
				continue
			}
			if overlaps(r.Rect, drawn) {
				t.Errorf("%q at %+v runs under the %s button drawn at %+v", r.Text, r.Rect, b.Label, drawn)
			}
		}
	}
	for _, r := range runs {
		if r.Text != all.Label && r.Text != "BACK" && r.Rect.Y+r.Rect.H > all.Rect.Y {
			t.Errorf("%q at %+v reaches FORGET ALL at y %d", r.Text, r.Rect, all.Rect.Y)
		}
	}
}
