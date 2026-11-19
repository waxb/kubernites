package hud

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	IntentHints        = "live:hints"
	IntentLiveChoices  = "live:choices"
	OverlayLiveChoices = "livechoices"
	menuHints          = "menu:hints"
	menuLiveChoices    = "menu:livechoices"
	choicesPrefix      = "livechoices"
	choicesShown       = 6
	hintsTip           = "ON: each remedy shows the kubectl command a person would run, and each order echoes its command in the Speaking Stone. OFF: the cards keep their orders and the cluster's words; the ledger lists every write either way."
	resetChoicesTip    = "List the contexts whose level and passed warnings are remembered, and forget one or all of them: unticking READ ONLY then warns again."
)

var liveOnlyMenu = map[string]bool{"menu:save": true, "menu:coins": true, "menu:experience": true, "menu:calm": true, TrialsButton: true, TutorialButton: true, MenuPanels: true}

type LiveChoice struct {
	Name    string
	Server  string
	Level   string
	Classes []string
}

func (h *HUD) SetLiveChoices(list []LiveChoice) {
	h.lw.choices = list
}

func (h *HUD) liveMenu(labels [][3]string) [][3]string {
	if h.live == nil {
		return labels
	}
	hints := "HINTS: OFF"
	if h.live.Hints {
		hints = "HINTS: ON"
	}
	out := make([][3]string, 0, len(labels)+2)
	for _, l := range labels {
		if liveOnlyMenu[l[0]] || !LiveReadButton(l[0]) {
			continue
		}
		if l[0] == "menu:title" {
			out = append(out, [3]string{menuHints, hints, hintsTip}, [3]string{menuLiveChoices, "RESET LIVE CHOICES", resetChoicesTip})
		}
		out = append(out, l)
		if l[0] == "menu:names" {
			out = append(out, h.stoneLabel())
		}
	}
	return out
}

func (h *HUD) pressLiveChoices(rest string, f Frame) []Intent {
	click := Intent{Kind: IntentSound, Command: "click"}
	switch {
	case rest == "back":
		h.OverlayID = OverlayMenu
		return []Intent{click}
	case rest == "all":
		return []Intent{click, {Kind: IntentLiveChoices, Command: "forget-all"}}
	case strings.HasPrefix(rest, "forget:"):
		return []Intent{click, {Kind: IntentLiveChoices, Command: "forget", Target: strings.TrimPrefix(rest, "forget:")}}
	}
	return nil
}

var ChoicesBox = ui.Rect{X: Width/2 - 160, Y: Overlay.Y + 8, W: 320, H: Overlay.H - 16}

func choiceY(i int) int {
	return ChoicesBox.Y + menuTop + 16 + i*34
}

func forgetRect(y int) ui.Rect {
	return ui.Rect{X: ChoicesBox.X + ChoicesBox.W - 20 - 64, Y: y - 3, W: 64, H: 16}
}

func choiceLevelY(y int) int {
	f := forgetRect(y)
	return f.Y + f.H + 3
}

func (h *HUD) liveChoiceButtons() []Button {
	r := ChoicesBox
	out := []Button{}
	for i, c := range h.lw.choices {
		if i >= choicesShown {
			break
		}
		out = append(out, Button{ID: choicesPrefix + ":forget:" + c.Name, Rect: forgetRect(choiceY(i)), Label: "FORGET", Tip: "Forget " + c.Name + ": unticking READ ONLY on it warns again"})
	}
	all := Button{ID: choicesPrefix + ":all", Rect: ui.Rect{X: r.X + 20, Y: r.Y + r.H - 52, W: r.W - 40, H: 18}, Label: "FORGET ALL", Tip: "Forget every remembered context"}
	if len(h.lw.choices) == 0 {
		all.Disabled, all.Tip = true, "No context is remembered"
	}
	back := Button{ID: choicesPrefix + ":back", Rect: ui.Rect{X: r.X + 20, Y: r.Y + r.H - 28, W: r.W - 40, H: 18}, Label: "BACK", Tip: "Back to the MENU [Esc]"}
	return append(out, all, back, closeButton(r))
}

func (h *HUD) drawLiveChoices(dst *ebiten.Image, f Frame) {
	r := ChoicesBox
	ui.DarkPanel(dst, r)
	h.Font.DrawCentered(dst, "LIVE CHOICES", ui.Rect{X: r.X, Y: r.Y + 6, W: r.W, H: 14}, 2, ui.Palette.Gold)
	if len(h.lw.choices) == 0 {
		for i, l := range h.Font.WrapLines("No context is remembered: every unticked READ ONLY warns first.", r.W-40, 1, 3) {
			h.Font.Draw(dst, l, r.X+20, r.Y+menuTop+16+i*ui.LineHeight, 1, ui.Palette.TextDim)
		}
	}
	for i, c := range h.lw.choices {
		if i >= choicesShown {
			liveFont.Draw(dst, fmt.Sprintf("and %d more", len(h.lw.choices)-choicesShown), r.X+20, choiceY(i), 1, ui.Palette.TextDim)
			break
		}
		y := choiceY(i)
		liveFont.Draw(dst, ui.Clip(c.Name, 34), r.X+20, y, 1, ui.Palette.Text)
		liveFont.Draw(dst, c.Level, r.X+20, choiceLevelY(y), 1, LevelColour(c.Level))
		if len(c.Classes) > 0 {
			liveFont.Draw(dst, plural(len(c.Classes), "CLASS WARNING", "CLASS WARNINGS")+" PASSED", r.X+20+liveFont.Width(c.Level, 1)+8, choiceLevelY(y), 1, ui.Palette.TextDim)
		}
	}
	h.DrawOverlayButtons(dst, h.liveChoiceButtons())
}

func init() {
	simpleIntents[menuHints] = Intent{Kind: IntentHints}
	prefixHandlers[choicesPrefix] = (*HUD).pressLiveChoices
	Overlays[OverlayLiveChoices] = OverlaySpec{func(h *HUD, _ Frame) []Button { return h.liveChoiceButtons() }, (*HUD).drawLiveChoices}
}

func (h *HUD) drawHistoryEntry(dst *ebiten.Image, f Frame, c view.ChronicleView, l ChronLayout) {
	x, w := l.Question.X, l.Question.W
	head := ui.Clip(c.Subject, 48) + "   STRUCK " + c.Struck
	if c.Healed != "" {
		head += "   MENDED " + c.Healed
	}
	h.Font.Draw(dst, head, x, l.Question.Y, 1, ui.Palette.GoldBright)
	h.DrawVig(dst, ChronicleVig(c, f), l.Picture, l.Scale)
	y := l.Answers
	h.Font.Draw(dst, ui.Bridge("THE CLUSTER'S WORDS", "THE CLUSTER'S REASON"), x, y, 1, ui.Palette.Gold)
	y += ui.LineHeight
	for _, row := range h.Font.WrapLines(c.Explanation, w, 1, 3) {
		h.Font.Draw(dst, row, x, y, 1, ui.Palette.Text)
		y += ui.LineHeight
	}
	if c.MendLine != "" {
		h.Font.Draw(dst, ui.Clip(c.MendLine, ui.MaxChars(w, 1)), x, y+2, 1, ui.Palette.Green)
	}
	h.drawShelf(dst, f, c, l)
}
