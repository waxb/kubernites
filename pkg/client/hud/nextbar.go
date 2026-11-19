package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	nextRow    = FreightTip + "next"
	boostRow   = FreightTip + "boost"
	readyShare = "READY"
	barShade   = 140
	barGlow    = 90
)

var boostNames = map[string]string{view.BonusFrenzy: "GILDED TIDE"}

func NextShown(f Frame) bool {
	return f.State.Next != nil && f.State.Next.Label != ""
}

func NextLabel(n *view.NextView) string {
	return "NEXT: " + n.Label
}

func NextShare(n *view.NextView) string {
	if n.Ready || n.Progress >= 1 {
		return readyShare
	}
	return fmt.Sprintf("%d%%", int(min(max(n.Progress, 0), 1)*100))
}

func NextTip(n *view.NextView, st view.GameState) string {
	lines := []string{"The next step within reach: " + n.Label + "."}
	if n.Hint != "" {
		lines = append(lines, "How: "+n.Hint+".")
	}
	if payoff := nextPayoff(n, st); payoff != "" {
		lines = append(lines, payoff)
	}
	if charter := nextCharter(n, st); charter != "" {
		lines = append(lines, charter)
	}
	if n.Gold > 0 {
		have, need := Compact(min(st.Gold, n.Gold)), Compact(n.Gold)
		lines = append(lines, ui.Bridge(have+" of "+need+" gold.", "$"+have+" of $"+need+"."))
	}
	if n.Wisdom > 0 {
		lines = append(lines, fmt.Sprintf("%s of %s wisdom.", Compact(min(st.Wisdom, n.Wisdom)), Compact(n.Wisdom)))
	}
	if n.Hands > 0 {
		lines = append(lines, fmt.Sprintf("Needs %s on your Keeps; %s more are missing.", foodOfCores(n.Hands), foodCount(n.Missing*view.RationsPerCPU)))
	}
	if n.Command != "" {
		lines = append(lines, "Speak: "+n.Command)
	}
	return strings.Join(append(lines, "Click to go there."), "\n")
}

func ActiveBoost(f Frame) (view.BoostView, bool) {
	for _, b := range f.State.Golden.Boosts {
		if b.Left > 0 {
			return b, true
		}
	}
	return view.BoostView{}, false
}

func BoostLabel(b view.BoostView) string {
	name, ok := boostNames[b.Kind]
	if !ok {
		name = "GILDED " + strings.ToUpper(b.Kind)
	}
	return fmt.Sprintf("%s x%s", name, Compact(b.Multiplier))
}

func boostTip(b view.BoostView) string {
	return fmt.Sprintf("A gleaming box's gift: every box delivered is worth x%s for %s more.", Compact(b.Multiplier), Duration(b.Left))
}

func boardExtra(f Frame) int {
	n := 0
	if f.BoardCompact {
		return n
	}
	if TideShown(f) {
		n++
	}
	if NextShown(f) {
		n++
	}
	if _, ok := ActiveBoost(f); ok {
		n++
	}
	return n
}

func (h *HUD) boardExtraButtons(f Frame, row func(int) ui.Rect) []Button {
	out := []Button{}
	i := boardRows
	if TideShown(f) {
		out = append(out, Button{ID: tideRow, Rect: row(i), Tip: TideTip()})
		i++
	}
	if NextShown(f) {
		out = append(out, Button{ID: nextRow, Rect: row(i), Tip: NextTip(f.State.Next, f.State)})
		i++
	}
	if b, ok := ActiveBoost(f); ok {
		out = append(out, Button{ID: boostRow, Rect: row(i), Tip: boostTip(b)})
	}
	return out
}

func (h *HUD) drawBoardExtra(dst *ebiten.Image, f Frame, r ui.Rect, x, y int) {
	if TideShown(f) {
		h.drawTideRow(dst, f, r, x, y)
		y += ui.LineHeight
	}
	if NextShown(f) {
		n := f.State.Next
		clr := ui.Palette.Gold
		switch {
		case h.NextQuiet(f):
			clr = ui.Palette.TextDim
		case n.Ready:
			clr = ui.Palette.Green
		}
		h.drawBarRow(dst, r, x, y, NextLabel(n), NextShare(n), n.Progress, clr)
		y += ui.LineHeight
	}
	if b, ok := ActiveBoost(f); ok {
		share := 0.0
		if view.FrenzySeconds > 0 {
			share = b.Left / view.FrenzySeconds
		}
		h.drawBarRow(dst, r, x, y, BoostLabel(b), Duration(b.Left), share, ui.Palette.GoldBright)
	}
}

func (h *HUD) drawBarRow(dst *ebiten.Image, r ui.Rect, x, y int, label, value string, share float64, clr color.RGBA) {
	bar := ui.Rect{X: r.X + BoardPad - 2, Y: y - 2, W: r.W - 2*BoardPad + 4, H: ui.LineHeight}
	vw, _ := h.Font.Measure(value, 1)
	if lw, _ := h.Font.Measure(label, 1); share >= 1 && lw+vw+2*boardBarGap > bar.W {
		value, vw = "", 0
	}
	ui.Fill(dst, bar, ui.Tint(ui.Palette.Ink, barShade))
	if done := int(float64(bar.W) * min(max(share, 0), 1)); done > 0 {
		ui.Fill(dst, ui.Rect{X: bar.X, Y: bar.Y, W: done, H: bar.H}, ui.Tint(clr, barGlow))
	}
	drawClipped(dst, label, x, y, bar.W-vw-2*boardBarGap, ui.Palette.Cream)
	h.Font.Draw(dst, value, r.X+r.W-BoardPad-vw, y, 1, clr)
}

const IntentGo = "go"

func (h *HUD) PressNext(f Frame) []Intent {
	n := f.State.Next
	click := Intent{Kind: IntentSound, Command: "click"}
	if n == nil {
		return []Intent{click}
	}
	if n.Command != "" {
		h.OpenStone(n.Command)
		return []Intent{click}
	}
	target := n.Target
	if n.Kind == view.NextPage {
		h.Grimoire = n.ID
	}
	if n.Build != "" {
		target = "build:" + n.Build
	}
	path := NavPath(target, h.NavState(f))
	if len(path) == 0 {
		return []Intent{click}
	}
	if id, ok := strings.CutPrefix(path[0], "haven:"); ok {
		return []Intent{{Kind: IntentHarbor, Target: id}, click}
	}
	if MapTarget(path[0]) {
		return []Intent{{Kind: IntentGo, Target: path[0]}, click}
	}
	if n.Build != "" && path[len(path)-1] == target && (len(path) == 1 || len(path) == 2 && path[0] == "cmd:build") {
		h.PageID = PageBuild
		return append(pressBuild(h, n.Build, f), click)
	}
	h.navigate(path[0])
	return []Intent{click}
}

func (h *HUD) navigate(id string) {
	if page, ok := pageButtons[id]; ok {
		h.PageID = page
		return
	}
	if o, ok := overlayButtons[id]; ok {
		h.toggleOverlay(o)
		return
	}
	switch {
	case id == "cmd:back":
		h.PageID = pageParent(h.PageID)
	case strings.HasPrefix(id, "tab:"):
		h.TabID = strings.TrimPrefix(id, "tab:")
	}
}
