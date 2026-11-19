package hud

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	welcomeCopyright = "Copyright (c) 1370-2026 by the Chancery - All Rites Reserved"
	welcomeNotice    = "This realm is a Demo: you can play it fully functional through its first two Ages: the Dark Age and the Feudal Age. " +
		"The full release adds the Castle Age and the Age of Empire, workshops, custom-item quests and more. " +
		"Ships land boxes on your quay; you carry them for gold, and a box that waits too long spoils. " +
		"Please press Realm information for further information."
	welcomeLearn        = "New to Kubernetes itself? LEARN KUBERNETES on the title page teaches it in chapters."
	welcomeUnregistered = "This scroll will not appear in the registered realm."
	welcomePress        = "Please press button Nr. 1, 2 or 3 to found your realm!"
	welcomeKept         = "Your choice is kept in the save. The MENU changes it, and its CALM REALM switch: no Rush Tide, gentler troubles."
)

func (h *HUD) drawWelcome(dst *ebiten.Image, f Frame) {
	r := WelcomeBox
	ui.DarkPanel(dst, r)
	switch h.welcomePage {
	case WelcomeInfo:
		h.drawWelcomeInfo(dst, f)
	case WelcomeRoad:
		h.drawWelcomeRoad(dst, f)
	default:
		h.drawWelcomeChoice(dst, f)
	}
	for _, b := range h.welcomeButtons(f) {
		switch {
		case b.ID == "welcome:begin":
			h.drawLiteralPrimary(dst, b)
		case b.ID == "welcome:back" && b.Label != "X":
			h.drawPrimary(dst, b)
		case experienceButton(b.ID):
			h.drawLiteralButton(dst, b)
		default:
			h.drawButton(dst, b)
		}
	}
}

func (h *HUD) drawWelcomeChoice(dst *ebiten.Image, f Frame) {
	r := WelcomeBox
	h.icon(dst, "icon/node", r.X+WelcomeInset, r.Y+8)
	title := "KUBERNITES"
	if f.Version != "" {
		title += " version " + f.Version
	}
	h.Font.DrawShadow(dst, title, r.X+WelcomeInset+24, r.Y+10, 2, ui.Palette.GoldBright)
	h.Font.Draw(dst, welcomeCopyright, r.X+WelcomeInset, r.Y+30, 1, ui.Palette.Cream)
	y := r.Y + 46
	for _, row := range h.Font.WrapLines(welcomeNotice, r.W-2*WelcomeInset, 1, 7) {
		h.Font.Draw(dst, row, r.X+WelcomeInset, y, 1, ui.Palette.Text)
		y += ui.LineHeight
	}
	Literal.Draw(dst, welcomeLearn, r.X+WelcomeInset, y+3, 1, ui.Palette.GoldBright)
	h.Font.Draw(dst, ui.Bridge(welcomeUnregistered, "This notice will not appear in the registered cluster."), r.X+WelcomeInset, WelcomeInfoRect().Y-14, 1, ui.Palette.Cream)
	h.drawWelcomeStrip(dst)
}

func (h *HUD) drawWelcomeStrip(dst *ebiten.Image) {
	strip := welcomeStrip()
	ui.Well(dst, strip)
	h.Font.DrawCentered(dst, welcomePress, ui.Rect{X: strip.X, Y: strip.Y + 8, W: strip.W, H: 12}, 1, ui.Palette.GoldBright)
	rects := welcomeChoiceRects()
	h.icon(dst, "icon/gold", rects[0].X-28, rects[0].Y+3)
	h.icon(dst, "icon/wisdom", rects[2].X+rects[2].W+12, rects[2].Y+3)
	for i, e := range experiences {
		rows := Literal.WrapLines(e.caption, rects[i].W, 1, 4)
		for j, row := range rows {
			Literal.Draw(dst, row, rects[i].X, rects[i].Y+rects[i].H+6+j*ui.LineHeight, 1, ui.Palette.TextDim)
		}
	}
	rows := h.Font.WrapLines(welcomeKept, strip.W-16, 1, 2)
	for j, row := range rows {
		h.Font.Draw(dst, row, strip.X+8, strip.Y+strip.H-6-(len(rows)-j)*ui.LineHeight, 1, ui.Palette.TextDim)
	}
}

func (h *HUD) drawWelcomeInfo(dst *ebiten.Image, f Frame) {
	r := WelcomeBox
	h.Font.DrawCentered(dst, ui.Bridge("A REALM RUN ON MAGIC", "THE REALM IS A CLUSTER"), ui.Rect{X: r.X, Y: r.Y + 10, W: r.W, H: 16}, 2, ui.Palette.GoldBright)
	y := r.Y + 38
	spend := welcomeSpend()
	for _, row := range append(append([]string(nil), WelcomeStory...), spend[0], spend[1], "") {
		h.Font.Draw(dst, row, r.X+WelcomeInset, y, 1, ui.Palette.Text)
		y += ui.LineHeight
	}
	for _, row := range h.Font.WrapLines(WelcomeGoal(f), r.W-2*WelcomeInset, 1, 3) {
		h.Font.Draw(dst, row, r.X+WelcomeInset, y, 1, ui.Palette.GoldBright)
		y += ui.LineHeight
	}
	y += 6
	h.Font.DrawShadow(dst, "WHAT IS WHAT", r.X+24, y, 1, ui.Palette.Gold)
	y += ui.LineHeight + 2
	for i, row := range WelcomeLegend {
		x := r.X + 24 + (i%2)*legendColumn
		h.Font.Draw(dst, row[0], x, y, 1, ui.Palette.Cream)
		h.Font.Draw(dst, legendMeaning(row), x+legendNameW, y, 1, ui.Palette.TextDim)
		if i%2 == 1 {
			y += ui.LineHeight
		}
	}
}

func legendMeaning(row [3]string) string { return row[2] }

func (h *HUD) drawWelcomeRoad(dst *ebiten.Image, f Frame) {
	r := WelcomeBox
	h.Font.DrawCentered(dst, "THE ROAD OF AGES", ui.Rect{X: r.X, Y: r.Y + 10, W: r.W, H: 16}, 2, ui.Palette.GoldBright)
	y := r.Y + 38
	for i, a := range f.State.Road.Ages {
		in := i < view.DemoAges
		h.Font.DrawShadow(dst, fmt.Sprintf("%s %s", romans[i], a.Name), r.X+WelcomeInset, y, 1, ui.Palette.Gold)
		tag, clr := "THE FULL RELEASE", ui.Palette.TextDim
		if in {
			tag, clr = "IN THE DEMO", ui.Palette.Green
		}
		tw := h.Font.Width(tag, 1)
		h.Font.Draw(dst, tag, r.X+r.W-WelcomeInset-tw, y, 1, clr)
		y += ui.LineHeight
		for _, row := range h.Font.WrapLines(a.Theme+". "+a.Goal+".", r.W-2*WelcomeInset, 1, 2) {
			h.Font.Draw(dst, row, r.X+WelcomeInset+12, y, 1, ui.Palette.Text)
			y += ui.LineHeight
		}
		y += 8
	}
	note := ui.Bridge("The Dark and Feudal Ages are the demo. The Castle Age and the Age of Empire, workshops and custom-item quests come with the full release.",
		"Dev and Staging are the demo. UAT, Production, workshops and custom-item quests come with the full release.")
	for _, row := range h.Font.WrapLines(note, r.W-2*WelcomeInset, 1, 2) {
		h.Font.Draw(dst, row, r.X+WelcomeInset, y, 1, ui.Palette.GoldBright)
		y += ui.LineHeight
	}
}

func experienceButton(id string) bool {
	_, ok := experienceOf(id)
	return ok
}

func (h *HUD) drawLiteralButton(dst *ebiten.Image, b Button) {
	ui.Button(dst, Literal, b.Rect, b.Label, ui.ButtonState{Hovered: h.Hovering && h.Hover.ID == b.ID})
}

func (h *HUD) drawLiteralPrimary(dst *ebiten.Image, b Button) {
	hot := h.Hovering && h.Hover.ID == b.ID
	face := ui.Palette.Gold
	if hot {
		face = ui.Palette.GoldBright
	}
	ui.Fill(dst, b.Rect.Offset(1, 2), ui.Palette.Shadow)
	ui.Fill(dst, b.Rect, face)
	ui.Stroke(dst, b.Rect, ui.Palette.Ink)
	Literal.DrawCentered(dst, b.Label, b.Rect, 1, ui.Palette.Ink)
}
