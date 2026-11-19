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
	leftW        = 300
	rightW       = 388
	stripH       = 30
	stripGap     = 3
	headerH      = 13
	columnTop    = 58
	loomTop      = 34
	flashTime    = 0.15
	tierFlash    = 0.6
	loomHeading  = "THE LOOM"
	loomTitleGap = 6
	threadIconW  = 20
	threadsInset = 24
	headingScale = 2
)

var patternIcons = map[string]string{
	"royal-treasury": "icon/loom-treasury", "swift-builders": "icon/loom-builders", "heraldic-fame": "icon/loom-fame",
	"scribes-charter": "icon/archive", "deep-ledger": "icon/archive", "siegecraft": "icon/siege", "ancestral-renown": "icon/renown",
}

func patternIcon(id string) string {
	if icon, ok := patternIcons[id]; ok {
		return icon
	}
	return "icon/thread"
}

func AgeLeft() ui.Rect {
	return ui.Rect{X: Overlay.X + 8, Y: Overlay.Y + columnTop - 30, W: leftW, H: Overlay.H - 62}
}

func ageRight() ui.Rect {
	return ui.Rect{X: Overlay.X + Overlay.W - 8 - rightW, Y: Overlay.Y + loomTop, W: rightW, H: Overlay.H - loomTop - 34}
}

func AgeBottom() ui.Rect {
	return ui.Rect{X: Overlay.X + 8, Y: Overlay.Y + Overlay.H - 28, W: Overlay.W - 16, H: 22}
}

func AgeBottomRule() int { return AgeBottom().Y - 4 }

type LoomRow struct {
	Header  bool
	Tier    int
	Pattern view.PatternView
	Index   int
	Rect    ui.Rect
}

func LoomLayout(patterns []view.PatternView) []LoomRow {
	r := ageRight()
	y, tier, out := r.Y, -1, []LoomRow{}
	for i, p := range patterns {
		if p.Tier != tier {
			tier = p.Tier
			out = append(out, LoomRow{Header: true, Tier: tier, Pattern: p, Index: -1, Rect: ui.Rect{X: r.X, Y: y, W: r.W, H: headerH}})
			y += headerH
		}
		h := stripH + (len(descLines(p.Description, r.W))-1)*ui.LineHeight
		out = append(out, LoomRow{Tier: tier, Pattern: p, Index: i, Rect: ui.Rect{X: r.X, Y: y, W: r.W, H: h}})
		y += h + stripGap
	}
	return out
}

func descLines(text string, stripWidth int) []string {
	var font ui.Font
	return font.WrapLines(text, stripWidth-38, 1, 2)
}

func teaserRect() ui.Rect {
	r := ageRight()
	return ui.Rect{X: r.X, Y: r.Y, W: r.W, H: stripH + 10}
}

func loomOpen(f Frame) bool {
	return f.Revealed("loom")
}

func (h *HUD) ageButtons(f Frame) []Button {
	out := []Button{}
	if loomOpen(f) {
		for _, row := range LoomLayout(f.State.Reforge.Patterns) {
			if !row.Header {
				out = append(out, LoomButton(row.Rect, row.Pattern))
			}
		}
	}
	b := AgeBottom()
	primary := ui.Rect{X: b.X + b.W - 204, Y: b.Y + 2, W: 200, H: 18}
	switch a := f.State.AgeUp; {
	case f.State.Reforge.Available:
		out = append(out, Button{ID: "reforge:open", Rect: primary, Label: "REFORGE THE REALM", Selected: true,
			Tip: "Begin again on a new island. Only what is declared survives."})
	case a.NextAge != "":
		out = append(out, Button{ID: "age:advance", Rect: primary, Label: "ADVANCE TO THE " + strings.ToUpper(view.Content.AgeTitle(a.NextAge)),
			Disabled: !a.Available, Tip: advanceTip(a)})
	}
	if road, ok := roadAgeButton(f); ok {
		out = append(out, road)
	}
	if more, ok := h.dawnMoreButton(f); ok {
		out = append(out, more)
	}
	out = append(out, h.kubeAgeButtons(f)...)
	return append(out, closeButton(Overlay))
}

func advanceTip(a view.AgeUpView) string {
	if !a.Available {
		return "Meet every requirement first"
	}
	return "Every requirement is met: advance to the " + view.Content.AgeTitle(a.NextAge)
}

func LoomButton(r ui.Rect, p view.PatternView) Button {
	b := Button{ID: "loom:" + p.ID, Rect: r, Tip: p.Name + ": " + p.Description}
	switch {
	case p.Locked:
		b.Disabled, b.Tip = true, fmt.Sprintf("Tier %s opens when %s threads are woven", romanOf(p.Tier-1), Compact(p.OpensAt))
	case !p.Repeatable && p.Rank >= p.MaxRank:
		b.Disabled, b.Tip = true, p.Name+" is fully woven"
	case !p.Affordable:
		b.Disabled, b.Tip = true, fmt.Sprintf("Needs %s threads", Compact(p.Cost))
	}
	return b
}

func (h *HUD) loomKeys(in Input, f Frame) ([]Intent, bool) {
	patterns := f.State.Reforge.Patterns
	if !loomOpen(f) || len(patterns) == 0 {
		return nil, false
	}
	switch {
	case in.Pressed(ebiten.KeyArrowUp):
		h.loomRow = (h.loomRow + len(patterns) - 1) % len(patterns)
	case in.Pressed(ebiten.KeyArrowDown):
		h.loomRow = (h.loomRow + 1) % len(patterns)
	case in.Pressed(ebiten.KeyEnter):
		p := patterns[min(h.loomRow, len(patterns)-1)]
		b := LoomButton(ui.Rect{}, p)
		if b.Disabled {
			h.Toast(b.Tip, true)
			return []Intent{{Kind: IntentSound, Command: "error"}}, true
		}
		return []Intent{act(weave(p.ID))}, true
	default:
		return nil, false
	}
	return nil, true
}

func (h *HUD) DrawAge(dst *ebiten.Image, f Frame) {
	ui.DarkPanel(dst, Overlay)
	left, right := AgeLeft(), ageRight()
	h.Font.DrawShadow(dst, h.AgeTitle(f), left.X+2, Overlay.Y+6, 2, ui.Palette.Gold)
	h.drawDawnFlash(dst)
	h.drawAgeLeft(dst, f, left)
	title, _ := h.loomTitle(f, right)
	h.Font.DrawShadow(dst, loomHeading, title.X, title.Y, title.H/ui.CapHeight, ui.Palette.Gold)
	h.drawThreads(dst, f.State.Reforge, right.X+right.W-threadsInset, Overlay.Y+6)
	if loomOpen(f) {
		for _, row := range LoomLayout(f.State.Reforge.Patterns) {
			h.drawLoomRow(dst, row, f.State.Reforge.Spent)
		}
	} else {
		h.drawTeaser(dst, teaserRect())
	}
	h.drawAgeBottom(dst, f)
	h.drawKubeAge(dst, f)
	for _, b := range h.ageButtons(f) {
		if !strings.HasPrefix(b.ID, "loom:") && b.ID != DawnMore {
			h.drawButton(dst, b)
		}
	}
}

func (h *HUD) loomTitle(f Frame, right ui.Rect) (ui.Rect, int) {
	x, y := right.X+2, Overlay.Y+6
	left := h.threadsLeft(f.State.Reforge, right.X+right.W-threadsInset)
	scale := headingScale
	if w, _ := h.Font.Measure(loomHeading, scale); x+w+loomTitleGap > left {
		scale, y = 1, y+ui.CapHeight/2
	}
	w, _ := h.Font.Measure(loomHeading, scale)
	return ui.Rect{X: x, Y: y, W: w, H: ui.CapHeight * scale}, left
}

func threadsText(rf view.ReforgeView) string {
	text := Compact(rf.Threads) + " THREADS"
	if rf.RunThreads > 0 {
		text += "  +" + Compact(rf.RunThreads) + " THIS RUN"
	}
	return text
}

func (h *HUD) threadsLeft(rf view.ReforgeView, rightEdge int) int {
	w, _ := h.Font.Measure(threadsText(rf), 1)
	return rightEdge - w - threadIconW
}

func (h *HUD) drawThreads(dst *ebiten.Image, rf view.ReforgeView, rightEdge, y int) {
	text := threadsText(rf)
	w, _ := h.Font.Measure(text, 1)
	h.icon(dst, "icon/thread", rightEdge-w-threadIconW, y-2)
	h.Font.Draw(dst, text, rightEdge-w, y+2, 1, ui.Palette.GoldBright)
	if rf.NextThreadAt != "" {
		next := "next thread at: " + rf.NextThreadAt
		nw, _ := h.Font.Measure(next, 1)
		h.Font.Draw(dst, next, rightEdge-nw, y+14, 1, ui.Palette.TextDim)
	}
}

func (h *HUD) drawAgeLeft(dst *ebiten.Image, f Frame, r ui.Rect) {
	y := h.DrawDawn(dst, f, r, h.DrawStageTheme(dst, f, r))
	switch a := f.State.AgeUp; {
	case a.NextAge != "":
		h.Font.Draw(dst, "To reach the "+view.Content.AgeTitle(a.NextAge)+":", r.X, y, 1, ui.Palette.Text)
		y = h.drawRequirements(dst, a.Requirements, r.X, y+16, r.W)
		h.drawAgeVig(dst, f, y)
	case !f.State.Reforge.Available:
		lines := h.FoundationLines(f)
		y = h.drawTextLines(dst, lines, r.X, y, r.W, ui.Palette.Text)
	}
	h.drawDawnPicture(dst, f, y)
	if f.Revealed("loom") {
		h.DrawMastery(dst, f.State.Mastery, ui.Rect{X: r.X, Y: max(y+8, r.Y), W: r.W, H: r.Y + r.H - y - 8})
	}
}

var foundationCaption = fmt.Sprintf("Hold Two Nines for %d minutes in the Feudal Age to Reforge.", view.FoundationMinutes)

func (h *HUD) FoundationLines(f Frame) []string {
	held := fmt.Sprintf("Held now: %s (%s delivered fresh).", countWord(f.State.SLO.Nines, "nine"), Nines(f.State.SLO.Current))
	if h.DawnLive(f) && h.Dawn.More {
		return nil
	}
	if h.DawnLive(f) {
		return []string{held}
	}
	return []string{fmt.Sprintf("The foundation: hold Two Nines for %d minutes in the Feudal Age.", view.FoundationMinutes), held}
}

const requirementLines = 4

func (h *HUD) drawRequirements(dst *ebiten.Image, reqs []view.Requirement, x, y, w int) int {
	for _, req := range reqs {
		mark, clr := "[ ]", ui.Palette.Red
		if req.Met {
			mark, clr = "[x]", ui.Palette.Green
		}
		h.Font.Draw(dst, mark, x, y, 1, clr)
		for _, row := range h.Font.WrapLines(req.Text, w-24, 1, requirementLines) {
			h.Font.Draw(dst, row, x+24, y, 1, ui.Palette.Text)
			y += ui.LineHeight
		}
		y += 2
	}
	return y
}

func (h *HUD) drawTextLines(dst *ebiten.Image, lines []string, x, y, w int, clr color.RGBA) int {
	for _, l := range lines {
		for _, row := range h.Font.Wrap(l, w, 1) {
			h.Font.Draw(dst, row, x, y, 1, clr)
			y += ui.LineHeight
		}
	}
	return y
}

func (h *HUD) drawAgeBottom(dst *ebiten.Image, f Frame) {
	b := AgeBottom()
	ui.Fill(dst, ui.Rect{X: b.X, Y: AgeBottomRule(), W: b.W, H: 1}, ui.Palette.Gold)
	rf := f.State.Reforge
	switch {
	case rf.Available:
		caption := fmt.Sprintf("weave %d%%   +%s threads   %d nines", int(rf.Weave*100+.5), Compact(rf.Earn), rf.Nines)
		h.Font.Draw(dst, caption, b.X+2, b.Y+7, 1, ui.Palette.Text)
	case f.State.AgeUp.NextAge != "":
		h.Font.Draw(dst, "Meet every requirement to advance.", b.X+2, b.Y+7, 1, ui.Palette.TextDim)
	default:
		h.Font.Draw(dst, foundationCaption, b.X+2, b.Y+7, 1, ui.Palette.TextDim)
	}
}
