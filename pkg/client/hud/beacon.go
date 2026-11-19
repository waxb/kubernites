package hud

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var objectiveBuilds = map[string]string{
	"raise-keep": "keep", "build-codex": "codex", "build-chancery": "chancery", "build-quartermaster": "quartermaster",
	"build-stewards": "stewards-hall", "build-guildhall": "guildhall", "archive": "archive", view.ReserveKeepObjective: "keep",
}

var subPages = map[string]struct{ page, parent string }{
	"build:": {PageBuild, "cmd:build"}, "summon:": {PageSummon, "cmd:summon"},
}

func TargetOnPage(target, page string) string {
	for prefix, sub := range subPages {
		if strings.HasPrefix(target, prefix) && page != sub.page {
			return sub.parent
		}
	}
	return target
}

func BeaconButton(f Frame, page string) string {
	path := NavPath(ObjectiveGoal(f), NavState{Page: page})
	if len(path) == 0 || MapTarget(path[0]) {
		return ""
	}
	return path[0]
}

func ServeBeckons(f Frame) bool {
	serving := ObjectiveGoal(f) == view.ParcelPile || f.Teaching(view.ParcelPile) || f.Teaching(MeterGold)
	return serving && HandsNeeded(f.State.Freight) && parcelsWaiting(f.State.Freight) >= 1
}

func (h *HUD) Beckons(f Frame) bool {
	reopened := h.gate.reopened && HandsNeeded(f.State.Freight) && parcelsWaiting(f.State.Freight) >= 1
	return ServeBeckons(f) || reopened && f.GoalAt == nil
}

func (h *HUD) drawBeacon(dst *ebiten.Image, r ui.Rect) {
	a := uint8(90 + 165*pulse(h.Now, h.Reduce))
	ui.Stroke(dst, r.Inset(-1), ui.Tint(ui.Palette.GoldBright, a))
	ui.Stroke(dst, r.Inset(-2), ui.Tint(ui.Palette.Gold, a/2))
}
