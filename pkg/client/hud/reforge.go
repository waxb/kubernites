package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/ui"
)

var ReforgeBox = ui.Rect{X: Overlay.X + (Overlay.W-420)/2, Y: Overlay.Y + (Overlay.H-220-boxVigH)/2, W: 420, H: 220 + boxVigH}

func HandCast(pods []model.PodView, weave float64) int {
	n := 0
	for _, p := range pods {
		if !p.System && !p.Terminating {
			n++
		}
	}
	return int(float64(n)*(1-min(max(weave, 0), 1)) + .5)
}

func (h *HUD) ReforgeButtons() []Button {
	r := ReforgeBox
	y := r.Y + r.H - 28
	return []Button{
		{ID: "reforge:cancel", Rect: ui.Rect{X: r.X + 80, Y: y, W: 110, H: 18}, Label: "NOT YET", Tip: "Keep ruling this realm [Esc]"},
		{ID: "reforge:confirm", Rect: ui.Rect{X: r.X + r.W - 190, Y: y, W: 110, H: 18}, Label: "REFORGE", Tip: "Begin again on a new island"},
		closeButtonAs(r, "reforge:close", "Keep ruling this realm [Esc]"),
	}
}

func (h *HUD) pressReforge(rest string, _ Frame) []Intent {
	switch rest {
	case "open":
		h.OverlayID = OverlayReforge
		return []Intent{{Kind: IntentSound, Command: "click"}}
	case "cancel", "close":
		h.OverlayID = OverlayAge
		return []Intent{{Kind: IntentSound, Command: "click"}}
	case "confirm":
		h.OverlayID = OverlayNone
		return []Intent{{Kind: IntentReforge}}
	}
	return nil
}

func (h *HUD) DrawReforge(dst *ebiten.Image, f Frame) {
	h.DrawAge(dst, f)
	ui.Fill(dst, Overlay, ui.Tint(ui.Palette.Shadow, reforgeDim))
	r := ReforgeBox
	ui.DarkPanel(dst, r)
	h.Font.DrawCentered(dst, "REFORGE THE REALM?", ui.Rect{X: r.X, Y: r.Y + 8, W: r.W, H: 16}, 2, ui.Palette.GoldBright)
	rf, m := f.State.Reforge, f.State.Mastery
	kept := []string{"Grimoire pages", "Academy classes", "words and Chronicles", fmt.Sprintf("The Tapestry: %d decrees", m.Declared)}
	lost := []string{"gold, keeps, island", "berths, renown", count(HandCast(f.Pods, rf.Weave), "hand-cast hut")}
	h.drawColumn(dst, "KEPT", kept, r.X+24, r.Y+34, ui.Palette.Green)
	h.drawColumn(dst, "LOST", lost, r.X+r.W/2+16, r.Y+34, ui.Palette.RedText)
	y := r.Y + 104
	h.icon(dst, "icon/thread", r.X+24, y-4)
	h.Font.Draw(dst, "+"+Compact(rf.Earn)+" threads", r.X+44, y, 1, ui.Palette.GoldBright)
	ui.Bar(dst, ui.Rect{X: r.X + 170, Y: y, W: 120, H: 8}, rf.Weave, ui.Palette.Gold)
	h.Font.Draw(dst, fmt.Sprintf("weave %d%%", int(rf.Weave*100+.5)), r.X+298, y, 1, ui.Palette.Text)
	h.Font.DrawCentered(dst, "Only what is declared survives: apply, not run.", ui.Rect{X: r.X, Y: r.Y + 124, W: r.W, H: 10}, 1, ui.Palette.Cream)
	words := fmt.Sprintf("You practised %s, understood %s, declared %s.", count(len(m.Verbs), "word"), count(len(m.Incidents), "calamity"), count(m.Declared, "decree"))
	h.Font.DrawCentered(dst, words, ui.Rect{X: r.X, Y: r.Y + 142, W: r.W, H: 10}, 1, ui.Palette.TextDim)
	h.DrawVig(dst, ReforgeVig(f), ReforgeVigRect(), 1)
	buttons := h.ReforgeButtons()
	h.drawButton(dst, buttons[2])
	h.drawButton(dst, buttons[0])
	ui.Stroke(dst, buttons[0].Rect.Inset(-2), ui.Palette.Cream)
	h.drawPrimary(dst, buttons[1])
}

func (h *HUD) drawColumn(dst *ebiten.Image, title string, rows []string, x, y int, clr color.RGBA) {
	h.Font.Draw(dst, title, x, y, 1, clr)
	for i, row := range rows {
		h.Font.Draw(dst, row, x, y+14+i*ui.LineHeight, 1, ui.Palette.Text)
	}
}

func (h *HUD) drawPrimary(dst *ebiten.Image, b Button) {
	hot := h.Hovering && h.Hover.ID == b.ID
	face := ui.Palette.Gold
	if hot {
		face = ui.Palette.GoldBright
	}
	ui.Fill(dst, b.Rect.Offset(1, 2), ui.Palette.Shadow)
	ui.Fill(dst, b.Rect, face)
	ui.Stroke(dst, b.Rect, ui.Palette.Ink)
	h.Font.DrawCentered(dst, b.Label, b.Rect, 1, ui.Palette.Ink)
}

func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	if strings.HasSuffix(noun, "y") {
		return fmt.Sprintf("%d %sies", n, strings.TrimSuffix(noun, "y"))
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
