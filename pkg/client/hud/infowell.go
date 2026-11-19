package hud

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var wellTabs = []struct{ tab, reveal, surface, tip string }{
	{TabEvents, "tidings", SurfaceTidings, "Tidings: what the realm's servants report (kubectl get events)"},
	{TabTech, "grimoire", "cmd:grimoire", "The Grimoire: Kubernetes lore to learn"},
	{TabKube, "stone", SurfaceStone, "The Speaking Stone: type kubectl [Enter]"},
	{TabYAML, "scriptorium", SurfaceScriptorium, "The Scriptorium: write decrees, kubectl apply [Shift+Enter]"},
}

var (
	storyTabs = map[string]string{TabEvents: "TIDINGS", TabKube: "STONE", TabYAML: "DECREE"}
	kubeTabs  = map[string]string{TabEvents: "EVENTS"}
)

const (
	tabMinW    = 40
	tabPad     = 10
	wellTabGap = 4
)

func TabLabel(tab string) string {
	if story, ok := storyTabs[tab]; ok && ui.StoryNaming() {
		return story
	}
	if kube, ok := kubeTabs[tab]; ok {
		return kube
	}
	return tab
}

const (
	tidingMaxLines = 2
)

func WellBody() ui.Rect {
	return ui.Rect{X: InfoWell.X + 5, Y: InfoWell.Y + 23, W: InfoWell.W - 10, H: InfoWell.H - 28}
}

func eventRows() int {
	return ui.VisibleLines(WellBody().H-4, 1)
}

func (h *HUD) visibleTabs(f Frame) []string {
	out := []string{}
	for _, t := range wellTabs {
		if f.Revealed(t.reveal) && f.Shown(t.surface) {
			out = append(out, t.tab)
		}
	}
	return out
}

func (h *HUD) activeTab(f Frame) string {
	tabs := h.visibleTabs(f)
	for _, t := range tabs {
		if t == h.TabID {
			return t
		}
	}
	if len(tabs) > 0 {
		return tabs[0]
	}
	return ""
}

var tabFits = [][2]int{{tabMinW, tabPad}, {0, tabPad}, {0, 8}, {0, 6}, {0, 4}}

func (h *HUD) tabWidths(labels []string, room int) []int {
	out := make([]int, len(labels))
	for _, fit := range tabFits {
		used := 0
		for i, label := range labels {
			out[i] = max(fit[0], h.Font.Width(label, 1)+fit[1])
			used += out[i] + wellTabGap
		}
		if used-wellTabGap <= room {
			break
		}
	}
	return out
}

func (h *HUD) wellButtons(f Frame) []Button {
	out := []Button{}
	x := InfoWell.X + 5
	active := h.activeTab(f)
	council, councilled := councilWellButton(f)
	end := InfoWell.X + InfoWell.W - 5
	if councilled {
		end = council.Rect.X - wellTabGap
	}
	shown, labels := []int{}, []string{}
	for i, t := range wellTabs {
		if f.Revealed(t.reveal) && f.Shown(t.surface) {
			shown, labels = append(shown, i), append(labels, TabLabel(t.tab))
		}
	}
	for i, w := range h.tabWidths(labels, end-x) {
		t := wellTabs[shown[i]]
		out = append(out, Button{ID: "tab:" + t.tab, Rect: ui.Rect{X: x, Y: InfoWell.Y + 4, W: w, H: 14}, Label: labels[i], Tip: t.tip, Selected: active == t.tab})
		x += w + wellTabGap
	}
	if councilled {
		out = append(out, council)
	}
	switch active {
	case TabEvents:
		out = append(out, h.tidingButtons(f)...)
	case TabKube:
		out = append(out, Button{ID: "well:stone", Rect: WellBody(), Tip: "Open the Speaking Stone [Enter]"})
	case TabTech:
		out = append(out, Button{ID: "well:tech", Rect: WellBody(), Tip: "Open the Grimoire"})
	case TabYAML:
		out = append(out, Button{ID: "well:yaml", Rect: WellBody(), Tip: "Open the Scriptorium [Shift+Enter]"})
	case TabTrade:
		out = append(out, Button{ID: ContractsOpen, Rect: WellBody(), Tip: tradeOpenHint})
	}
	return out
}

func (h *HUD) drawWell(dst *ebiten.Image, f Frame) {
	for _, b := range h.wellButtons(f) {
		if b.Label != "" {
			h.drawButton(dst, b)
		}
	}
	body := WellBody()
	ui.Well(dst, body)
	switch h.activeTab(f) {
	case "":
	case TabTech:
		h.drawTechSummary(dst, f, body)
	case TabKube:
		h.drawStoneSummary(dst, body)
	case TabYAML:
		h.drawScriptSummary(dst, body)
	case TabTrade:
		h.drawTradeSummary(dst, f, body)
	default:
		h.DrawTidings(dst, f, body)
	}
}

type tidingRow struct {
	index int
	y     int
	lines []string
	cut   bool
}

func (h *HUD) TidingTop(f Frame, body ui.Rect) int {
	if f.AutonomyToday > 0 {
		return body.Y + 3 + ui.LineHeight + 2
	}
	return body.Y + 3
}

func (h *HUD) tidingRows(f Frame, body ui.Rect) []tidingRow {
	scroll := h.scroll.Clamp(len(f.Events), 1)
	out := []tidingRow{}
	y := h.TidingTop(f, body)
	for i := scroll.Offset; i < len(f.Events); i++ {
		text := Label(f.Events[i].Text)
		lines := h.Font.WrapLines(text, body.W-14, 1, tidingLines(f.Events[i]))
		if y+len(lines)*ui.LineHeight > body.Y+body.H {
			break
		}
		cut := len(h.Font.Wrap(text, body.W-14, 1)) > len(lines)
		out = append(out, tidingRow{index: i, y: y, lines: lines, cut: cut})
		y += len(lines)*ui.LineHeight + 2
	}
	return out
}

func tidingLines(t Tiding) int {
	if t.Detail != "" || t.Kind != "event" {
		return eventRows() - 1
	}
	return tidingMaxLines
}

func (h *HUD) tidingButtons(f Frame) []Button {
	body := WellBody()
	out := []Button{}
	for _, r := range h.tidingRows(f, body) {
		rect := ui.Rect{X: body.X + 3, Y: r.y, W: body.W - 6, H: len(r.lines) * ui.LineHeight}
		switch {
		case f.Events[r.index].Detail != "":
			out = append(out, Button{ID: fmt.Sprintf("tiding:%d", r.index), Rect: rect, Tip: "Read what just happened"})
		case r.cut:
			out = append(out, Button{ID: fmt.Sprintf("info:tiding:%d", r.index), Rect: rect, Tip: Label(f.Events[r.index].Text)})
		}
	}
	return out
}

func (h *HUD) PressTiding(rest string, f Frame) []Intent {
	var i int
	if _, err := fmt.Sscan(rest, &i); err != nil || i < 0 || i >= len(f.Events) {
		return nil
	}
	e := f.Events[i]
	h.Lesson(e.Title, e.Detail)
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func AutonomyLine(n int) string {
	if n == 1 {
		return ui.Bridge("Your servants did 1 thing for you today", "Kubernetes handled 1 thing for you today")
	}
	return ui.Bridge(fmt.Sprintf("Your servants did %d things for you today", n), fmt.Sprintf("Kubernetes handled %d things for you today", n))
}

func (h *HUD) DrawTidings(dst *ebiten.Image, f Frame, body ui.Rect) {
	if f.AutonomyToday > 0 {
		h.Font.Draw(dst, ui.Clip(AutonomyLine(f.AutonomyToday), ui.MaxChars(body.W-8, 1)), body.X+4, body.Y+3, 1, ui.Palette.Blue)
	}
	if len(f.Events) == 0 {
		h.Font.Draw(dst, "All is quiet in the realm.", body.X+4, h.TidingTop(f, body)+1, 1, ui.Palette.TextDim)
		return
	}
	h.scroll = h.scroll.Clamp(len(f.Events), 1)
	for _, r := range h.tidingRows(f, body) {
		e := f.Events[r.index]
		ui.Fill(dst, ui.Rect{X: body.X + 3, Y: r.y + 1, W: 3, H: 7}, tidingColor(e))
		hot := h.Hovering && h.Hover.ID == fmt.Sprintf("tiding:%d", r.index)
		for i, l := range r.lines {
			y := r.y + i*ui.LineHeight
			h.Font.Draw(dst, l, body.X+9, y, 1, tidingText(e))
			if hot {
				w, _ := h.Font.Measure(l, 1)
				ui.Fill(dst, ui.Rect{X: body.X + 9, Y: y + ui.CapHeight + 1, W: w, H: 1}, tidingText(e))
			}
			if at := h.Font.Span(l, 2); i == len(r.lines)-1 && e.Reason != "" && !ui.StoryNaming() && at+h.Font.Width(e.Reason, 1) <= body.W-14 {
				h.Font.Draw(dst, e.Reason, body.X+9+at, y, 1, ui.Palette.TextDim)
			}
		}
	}
}

func tidingText(t Tiding) color.RGBA {
	switch {
	case t.Detail != "":
		return ui.Palette.Cream
	case t.Kind == "omen":
		return ui.Palette.GoldBright
	case t.Warning:
		return ui.Palette.RedText
	}
	return ui.Palette.Text
}

func tidingColor(t Tiding) color.RGBA {
	switch {
	case t.Detail != "":
		return ui.Palette.Green
	case t.Warning:
		return ui.Palette.Red
	case t.Kind == "omen":
		return ui.Palette.GoldBright
	case t.Kind == "event":
		return ui.Palette.Blue
	}
	return ui.Palette.Gold
}

func (h *HUD) drawTechSummary(dst *ebiten.Image, f Frame, body ui.Rect) {
	open := []string{}
	for _, p := range f.State.Grimoire {
		if p.State == "available" {
			open = append(open, fmt.Sprintf("%s  %s", p.Name, costText(p.Cost.Gold, p.Cost.Wisdom)))
		}
	}
	h.Font.Draw(dst, techSummaryTitle(f.State.Grimoire), body.X+4, body.Y+4, 1, ui.Palette.GoldBright)
	y := body.Y + 16
	for _, s := range open[:min(len(open), 5)] {
		h.Font.Draw(dst, ui.Clip(s, ui.MaxChars(body.W-8, 1)), body.X+4, y, 1, ui.Palette.Text)
		y += ui.LineHeight
	}
	h.Font.Draw(dst, "Click to open the Grimoire", body.X+4, body.Y+body.H-11, 1, ui.Palette.TextDim)
}

func techSummaryTitle(pages []view.GrimoirePage) string {
	learned, shown := learnedCount(pages)
	return fmt.Sprintf("PAGES LEARNED %d/%d", learned, shown)
}

func costText(gold, wisdom float64) string {
	switch {
	case gold > 0 && wisdom > 0:
		return Gold(gold) + " " + Wisdom(wisdom)
	case wisdom > 0:
		return Wisdom(wisdom)
	}
	return Gold(gold)
}

func (h *HUD) drawStoneSummary(dst *ebiten.Image, body ui.Rect) {
	y := body.Y + 3
	for _, r := range h.stoneSummaryRows(body) {
		Literal.Draw(dst, r.Text, body.X+4, y, 1, r.clr)
		y += ui.LineHeight
	}
	h.Font.Draw(dst, "> _   press Enter to speak", body.X+4, body.Y+body.H-11, 1, ui.Palette.GoldBright)
}

func (h *HUD) stoneSummaryRows(body ui.Rect) []stoneRow {
	room, width := ui.VisibleLines(body.H-16, 1), ui.MaxChars(body.W-8, 1)
	out := []stoneRow{}
	for i := len(h.Stone.Output) - 1; i >= 0; i-- {
		l := h.Stone.Output[i]
		parts := wrapStoneText(ConsoleText(l), width)
		if len(out)+len(parts) > room {
			if len(out) == 0 {
				parts = parts[len(parts)-room:]
			} else {
				break
			}
		}
		rows := make([]stoneRow, 0, len(parts)+len(out))
		for _, part := range parts {
			rows = append(rows, stoneRow{Text: part, clr: l.color(), line: i})
		}
		out = append(rows, out...)
	}
	return out
}

func (h *HUD) drawScriptSummary(dst *ebiten.Image, body ui.Rect) {
	s := h.Script
	h.Font.Draw(dst, "THE SCRIPTORIUM", body.X+4, body.Y+4, 1, ui.Palette.Gold)
	h.Font.Draw(dst, "Draft: "+s.Template, body.X+4, body.Y+17, 1, ui.Palette.Text)
	if len(s.Result) > 0 {
		clr := ui.Palette.Green
		if !s.OK {
			clr = ui.Palette.RedText
		}
		for i, l := range h.Font.WrapLines(strings.Join(s.Result, " "), body.W-8, 1, 3) {
			h.Font.Draw(dst, l, body.X+4, body.Y+30+i*ui.LineHeight, 1, clr)
		}
	}
	h.Font.Draw(dst, "Click to write a decree", body.X+4, body.Y+body.H-11, 1, ui.Palette.TextDim)
}
