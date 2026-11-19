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
	maxIncidents = 5
	verbRows     = 2
)

func (h *HUD) drawWoven(dst *ebiten.Image, r ui.Rect, muted bool) {
	if !h.Lib.NineSlice(dst, "woven-strip", r) {
		ui.Fill(dst, r, ui.Palette.ConsoleEdge)
		ui.Fill(dst, ui.Rect{X: r.X, Y: r.Y, W: r.W, H: 2}, ui.Palette.Parchment)
		ui.Fill(dst, ui.Rect{X: r.X, Y: r.Y + r.H - 2, W: r.W, H: 2}, ui.Palette.Parchment)
	}
	if muted {
		ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, 150))
	}
}

func (h *HUD) drawLoomRow(dst *ebiten.Image, row LoomRow, spent float64) {
	if row.Header {
		h.drawTierHeader(dst, row, spent)
		return
	}
	h.drawStrip(dst, row)
}

func TierHeader(p view.PatternView, spent float64) (string, bool) {
	if !p.Locked {
		return "TIER " + romanOf(p.Tier-1), false
	}
	return fmt.Sprintf("TIER %s: weave %s threads to open (%s/%s)", romanOf(p.Tier-1), Compact(p.OpensAt), Compact(min(spent, p.OpensAt)), Compact(p.OpensAt)), true
}

func (h *HUD) drawTierHeader(dst *ebiten.Image, row LoomRow, spent float64) {
	text, locked := TierHeader(row.Pattern, spent)
	clr := ui.Palette.Gold
	switch {
	case locked:
		clr = ui.Palette.TextDim
	case h.events-h.Juice.Tiers[row.Tier] < tierFlash:
		clr = ui.Palette.Cream
	}
	h.Font.Draw(dst, text, row.Rect.X+2, row.Rect.Y+2, 1, clr)
}

func (h *HUD) drawStrip(dst *ebiten.Image, row LoomRow) {
	r, p := row.Rect, row.Pattern
	maxed := !p.Repeatable && p.Rank >= p.MaxRank
	h.drawWoven(dst, r, maxed || p.Locked)
	if row.Index == h.loomRow || (h.Hovering && h.Hover.ID == "loom:"+p.ID) {
		ui.Stroke(dst, r, ui.Palette.GoldBright)
	}
	h.icon(dst, patternIcon(p.ID), r.X+8, r.Y+7)
	name := ui.Palette.Gold
	if maxed || p.Locked {
		name = ui.Palette.TextDim
	}
	h.Font.Draw(dst, strings.ToUpper(p.Name), r.X+30, r.Y+5, 1, name)
	for i, row := range descLines(p.Description, r.W) {
		h.Font.Draw(dst, row, r.X+30, r.Y+16+i*ui.LineHeight, 1, ui.Palette.Text)
	}
	h.drawRank(dst, p, r)
	switch {
	case p.Locked:
		h.drawTag(dst, "LOCKED", r, ui.Palette.TextDim)
	case maxed:
		h.drawTag(dst, "MAX", r, ui.Palette.TextDim)
	case !p.Affordable:
		h.drawCost(dst, p.Cost, r, ui.Palette.RedText)
	default:
		h.drawCost(dst, p.Cost, r, ui.Palette.GoldBright)
	}
}

func (h *HUD) drawRank(dst *ebiten.Image, p view.PatternView, r ui.Rect) {
	if p.Locked {
		return
	}
	if p.Repeatable {
		h.Font.Draw(dst, fmt.Sprintf("RANK %d", p.Rank), r.X+r.W-96, r.Y+5, 1, ui.Palette.Cream)
		return
	}
	h.drawPips(dst, p, r.X+r.W-66, r.Y+7)
}

func (h *HUD) drawTag(dst *ebiten.Image, tag string, r ui.Rect, clr color.RGBA) {
	w, _ := h.Font.Measure(tag, 1)
	h.Font.Draw(dst, tag, r.X+r.W-8-w, r.Y+5, 1, clr)
}

func (h *HUD) drawCost(dst *ebiten.Image, cost float64, r ui.Rect, clr color.RGBA) {
	text := Compact(cost)
	w, _ := h.Font.Measure(text, 1)
	h.Font.Draw(dst, text, r.X+r.W-8-w, r.Y+5, 1, clr)
}

func (h *HUD) drawPips(dst *ebiten.Image, p view.PatternView, x, y int) {
	flash := h.events-h.Juice.flash[p.ID] < flashTime
	for i := 0; i < p.MaxRank; i++ {
		pip := ui.Rect{X: x + i*8, Y: y, W: 5, H: 4}
		clr := ui.Palette.Muted
		if i < p.Rank {
			clr = ui.Palette.Gold
			if flash && i == p.Rank-1 {
				clr = ui.Palette.Cream
			}
		}
		ui.Fill(dst, pip, clr)
	}
}

var loomTeaser = fmt.Sprintf("The Loom opens after %d minutes of Two Nines.", view.FoundationMinutes)

func (h *HUD) drawTeaser(dst *ebiten.Image, r ui.Rect) {
	h.drawWoven(dst, r, true)
	h.icon(dst, "icon/thread", r.X+8, r.Y+12)
	h.Font.Draw(dst, "LOCKED", r.X+30, r.Y+8, 1, ui.Palette.TextDim)
	h.Font.Draw(dst, loomTeaser, r.X+30, r.Y+20, 1, ui.Palette.Text)
}

func (h *HUD) DrawMastery(dst *ebiten.Image, m view.MasteryView, r ui.Rect) {
	h.Font.DrawShadow(dst, "WHAT YOU PRACTISED", r.X, r.Y, 1, ui.Palette.Gold)
	y := r.Y + 15
	if len(m.Verbs) == 0 && len(m.Incidents) == 0 && m.Declared == 0 {
		h.Font.Draw(dst, "Nothing practised yet. Speak to the Stone.", r.X, y, 1, ui.Palette.TextDim)
		return
	}
	h.Font.Draw(dst, "WORDS", r.X, y+2, 1, ui.Palette.TextDim)
	y = h.drawVerbChips(dst, m.Verbs, r.X+48, y, r.W-48) + 4
	h.Font.Draw(dst, fmt.Sprintf("CALAMITIES  %d understood", len(m.Incidents)), r.X, y, 1, ui.Palette.Text)
	y += ui.LineHeight
	for i, inc := range m.Incidents {
		if i == maxIncidents {
			h.Font.Draw(dst, fmt.Sprintf("  +%d more", len(m.Incidents)-maxIncidents), r.X, y, 1, ui.Palette.TextDim)
			y += ui.LineHeight
			break
		}
		title, gloss := Glossed(inc.Calamity, inc.K8s)
		h.Font.Draw(dst, "  "+title, r.X, y, 1, ui.Palette.Text)
		h.Font.Draw(dst, gloss, r.X+h.Font.Span("  "+title, 2), y, 1, ui.Palette.Blue)
		y += ui.LineHeight
	}
	decrees := fmt.Sprintf("DECREES  %d declared", m.Declared)
	if len(m.DeclaredKinds) > 0 {
		decrees += ": " + strings.Join(m.DeclaredKinds, ", ")
	}
	h.drawTextLines(dst, []string{decrees}, r.X, y+2, r.W, ui.Palette.Text)
}

func (h *HUD) drawVerbChips(dst *ebiten.Image, verbs []string, x0, y, w int) int {
	if len(verbs) == 0 {
		h.Font.Draw(dst, "none yet", x0, y+2, 1, ui.Palette.TextDim)
		return y + 14
	}
	x, row := x0, 0
	for i, v := range verbs {
		cw := Literal.Span(v, 0) + 7
		if x+cw > x0+w {
			row++
			x, y = x0, y+15
		}
		if row >= verbRows {
			h.Font.Draw(dst, fmt.Sprintf("+%d", len(verbs)-i), x0, y+2, 1, ui.Palette.TextDim)
			return y + 14
		}
		chip := ui.Rect{X: x, Y: y, W: cw, H: 13}
		ui.Stroke(dst, chip, ui.Palette.ChipBorder)
		Literal.Draw(dst, v, x+4, y+3, 1, ui.Palette.Text)
		x += cw + 4
	}
	return y + 15
}
