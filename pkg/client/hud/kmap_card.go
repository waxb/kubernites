package hud

import (
	"fmt"
	"image/color"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	RowTitle  = "title"
	RowK8s    = "k8s"
	RowRank   = "rank"
	RowBody   = "body"
	RowTeach  = "teach"
	RowRunes  = "runes"
	RowNeed   = "need"
	RowEarn   = "earn"
	RowDocs   = "docs"
	RowStatus = "status"

	KmapCardW    = 252
	KmapCardPad  = 7
	kmapCardHead = 44
	kmapCardGap  = 6
	kmapLearnH   = 18
	CardBodyMax  = 8
	cardTeachMax = 8
	cardEarnMax  = 3
	cardDocsMax  = 3
	cardK8sMax   = 2
)

type CardRow struct {
	Kind  string
	Text  string
	Met   bool
	Price bool
}

var cardRowLimits = map[string]int{RowBody: CardBodyMax, RowTeach: cardTeachMax, RowEarn: cardEarnMax, RowDocs: cardDocsMax, RowNow: cardNowMax}

func KnowledgeCard(p KMapPlaque, f Frame) []CardRow {
	if p.Fog {
		return fogCard(p, f)
	}
	pg := p.Page
	title := pg.Name
	if f.TrueNames {
		title = LabelOf(pg).Tag
	}
	k8s := pg.K8s
	if ui.StoryNaming() {
		k8s = ""
	}
	rows := []CardRow{{Kind: RowTitle, Text: strings.ToUpper(title)}, {Kind: RowK8s, Text: k8s}}
	if pg.MaxRank > 1 {
		rows = append(rows, CardRow{Kind: RowRank, Text: fmt.Sprintf("%d of %d ranks learned", pg.Rank, pg.MaxRank)})
	}
	rows = append(rows, nowRows(pg)...)
	rows = append(rows, CardRow{Kind: RowBody, Text: pg.Description})
	if !ui.StoryNaming() {
		rows = append(rows, teachRows(pg)...)
	}
	if pg.State != "unlocked" {
		rows = append(rows, needRows(pg, f)...)
		rows = append(rows, unlockRows(pg, f)...)
		rows = append(rows, earnRows(pg.Earn)...)
	}
	if docs := DocsText(pg); docs != "" && !ui.StoryNaming() {
		rows = append(rows, CardRow{Kind: RowDocs, Text: docs})
	}
	return append(rows, statusRow(pg, f)...)
}

func fogCard(p KMapPlaque, f Frame) []CardRow {
	users := []string{}
	for _, pg := range f.State.Grimoire {
		if pg.State != "hidden" && slices.Contains(pg.Requires, p.Page.ID) {
			users = append(users, pg.Name)
		}
	}
	rows := []CardRow{{Kind: RowTitle, Text: "AN UNREVEALED PAGE"}}
	if p.Page.Opens != "" {
		rows = append(rows, CardRow{Kind: RowBody, Text: p.Page.Opens})
	} else {
		rows = append(rows, CardRow{Kind: RowBody, Text: "Its page opens as the realm grows. Keep playing to reveal it."})
	}
	if len(users) > 0 {
		rows = append(rows, CardRow{Kind: RowNeed, Text: "Needed by " + JoinNames(users)})
	}
	return rows
}

func teachRows(pg view.GrimoirePage) []CardRow {
	if len(pg.Ranks) == 0 {
		return []CardRow{{Kind: RowTeach, Text: pg.Lesson}}
	}
	next := pg.Ranks[min(pg.Rank, len(pg.Ranks)-1)]
	lesson := next.Lesson
	if pg.State == "unlocked" || lesson == "" {
		lesson = pg.Lesson
	}
	rows := []CardRow{{Kind: RowTeach, Text: lesson}}
	if len(next.Runes) > 0 && pg.State != "unlocked" {
		rows = append(rows, CardRow{Kind: RowRunes, Text: "Teaches " + strings.Join(runeLabels(next.Runes), ", ")})
	}
	return rows
}

func needRows(pg view.GrimoirePage, f Frame) []CardRow {
	gold, wisdom := f.State.Gold, f.State.Wisdom
	rows := []CardRow{
		costRow(pg.Cost.Gold, gold, "gold"),
		costRow(pg.Cost.Wisdom, wisdom, "wisdom"),
	}
	for _, req := range pg.Requires {
		rows = append(rows, RequirementRow(req, f))
	}
	return rows
}

func unlockRows(pg view.GrimoirePage, f Frame) []CardRow {
	rows := []CardRow{}
	for _, b := range f.State.Buildables {
		if b.Locked && strings.Contains(b.Reason, "Learn "+pg.Name+" ") {
			rows = append(rows, CardRow{Kind: RowNeed, Text: fmt.Sprintf("Unlocks %s: the next costs %s gold", b.Name, Compact(b.Cost)), Met: f.State.Gold >= b.Cost, Price: true})
		}
	}
	return rows
}

func costRow(cost, have float64, what string) CardRow {
	if have >= cost {
		return CardRow{Kind: RowNeed, Text: fmt.Sprintf("Costs %s %s", Compact(cost), what), Met: true, Price: true}
	}
	return CardRow{Kind: RowNeed, Text: fmt.Sprintf("Costs %s %s: you have %s", Compact(cost), what, Compact(have)), Price: true}
}

func RequirementRow(id string, f Frame) CardRow {
	for _, pg := range f.State.Grimoire {
		if pg.ID != id {
			continue
		}
		switch {
		case satisfied(pg):
			return CardRow{Kind: RowNeed, Text: pg.Name + " learned", Met: true}
		case pg.State == "hidden":
			return CardRow{Kind: RowNeed, Text: "An unrevealed page, not yet learned"}
		}
		return CardRow{Kind: RowNeed, Text: "Learn " + pg.Name + " first"}
	}
	return CardRow{Kind: RowNeed, Text: "Learn " + id + " first"}
}

func earnRows(earn string) []CardRow {
	rows := []CardRow{}
	for _, part := range splitEarn(earn) {
		rows = append(rows, CardRow{Kind: RowEarn, Text: part})
	}
	return rows
}

func splitEarn(earn string) []string {
	earn = strings.TrimSpace(earn)
	if earn == "" {
		return nil
	}
	if i := strings.Index(earn, " Gold: "); i > 0 {
		return []string{earn[:i], earn[i+1:]}
	}
	return []string{earn}
}

func statusRow(pg view.GrimoirePage, f Frame) []CardRow {
	switch pg.State {
	case "unlocked":
		return []CardRow{{Kind: RowStatus, Text: "LEARNED", Met: true}}
	case "locked":
		return []CardRow{{Kind: RowStatus, Text: "LOCKED: learn what it needs first"}}
	}
	return nil
}

func JoinNames(names []string) string {
	switch len(names) {
	case 0:
		return "a later page"
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

func (h *HUD) CardFont(kind string) ui.Font {
	if kind == RowDocs {
		return ui.Font{Literal: true}
	}
	return h.Font
}

func (h *HUD) cardLines(rows []CardRow) [][]string {
	out := make([][]string, len(rows))
	width := KmapCardW - 2*KmapCardPad
	for i, r := range rows {
		prefix := ""
		if r.Kind == RowNeed && !r.Price {
			prefix = map[bool]string{true: "+ ", false: "- "}[r.Met]
		}
		limit, ok := cardRowLimits[r.Kind]
		if !ok {
			limit = 2
		}
		switch r.Kind {
		case RowTitle:
			out[i] = []string{ui.Clip(r.Text, ui.MaxChars(width-kmapCardHead+KmapCardPad, 1))}
			continue
		case RowK8s:
			out[i] = h.Font.WrapLines(r.Text, width-kmapCardHead+KmapCardPad, 1, cardK8sMax)
			continue
		}
		out[i] = h.CardFont(r.Kind).WrapLines(prefix+r.Text, width, 1, limit)
	}
	return out
}

func rowGap(prev, kind string) int {
	switch {
	case prev == "" || prev == kind:
		return 0
	case kind == RowRank || kind == RowRunes:
		return 1
	}
	return 5
}

func (h *HUD) cardHeight(rows []CardRow, lines [][]string, learn bool) int {
	y := kmapCardHead
	prev := RowK8s
	for i, r := range rows {
		if r.Kind == RowTitle || r.Kind == RowK8s {
			continue
		}
		y += rowGap(prev, r.Kind) + len(lines[i])*ui.LineHeight
		prev = r.Kind
	}
	if learn {
		y += 6 + kmapLearnH
	}
	return y + KmapCardPad
}

func cardPlace(plaque ui.Rect, height int, others []ui.Rect) ui.Rect {
	v := KnowledgeView
	inView := func(r ui.Rect) ui.Rect {
		r.X = min(max(r.X, v.X), v.X+v.W-r.W)
		r.Y = max(min(max(r.Y, v.Y), v.Y+v.H-r.H), v.Y)
		return r
	}
	top := plaque.Y - 8
	candidates := []ui.Rect{
		inView(ui.Rect{X: plaque.X + plaque.W + kmapCardGap, Y: top, W: KmapCardW, H: height}),
		inView(ui.Rect{X: plaque.X - kmapCardGap - KmapCardW, Y: top, W: KmapCardW, H: height}),
		inView(ui.Rect{X: plaque.X + plaque.W/2 - KmapCardW/2, Y: plaque.Y + plaque.H + kmapCardGap, W: KmapCardW, H: height}),
		inView(ui.Rect{X: plaque.X + plaque.W/2 - KmapCardW/2, Y: plaque.Y - kmapCardGap - height, W: KmapCardW, H: height}),
	}
	own, _ := clipRect(plaque, KnowledgeView)
	return ClearAvoiding(candidates, []ui.Rect{own}, others)
}

func (h *HUD) otherPlaques(p KMapPlaque, f Frame) []ui.Rect {
	out := []ui.Rect{}
	for _, q := range h.Knowledge(f).Plaques {
		if q.Page.ID == p.Page.ID {
			continue
		}
		if r, ok := clipRect(h.ScreenOf(q.Rect()), KnowledgeView); ok {
			out = append(out, r)
		}
	}
	return out
}

func learnable(p KMapPlaque) bool {
	return !p.Fog && p.Page.State == "available"
}

func (h *HUD) CardRect(p KMapPlaque, f Frame) (ui.Rect, []CardRow, [][]string) {
	rows := KnowledgeCard(p, f)
	lines := h.cardLines(rows)
	_, almanacked := GrimoireAlmanac(f, p)
	return cardPlace(h.ScreenOf(p.Rect()), h.cardHeight(rows, lines, learnable(p) || almanacked), h.otherPlaques(p, f)), rows, lines
}

func (h *HUD) CardButtons(f Frame, l KMapLayout) []Button {
	p, ok := h.focusPlaque(l)
	if !ok {
		return nil
	}
	card, _, _ := h.CardRect(p, f)
	out := []Button{{ID: KmapCardID, Rect: card}}
	r := ui.Rect{X: card.X + KmapCardPad, Y: card.Y + card.H - KmapCardPad - kmapLearnH, W: card.W - 2*KmapCardPad, H: kmapLearnH}
	if page, ok := GrimoireAlmanac(f, p); ok {
		out = append(out, Button{ID: BookGo + page.ID, Rect: ui.Rect{X: r.X + r.W - cardAlmanacW, Y: r.Y, W: cardAlmanacW, H: r.H}, Label: "ALMANAC", Tip: "Open the Almanac: " + page.Title, Page: page.ID})
		r.W -= cardAlmanacW + 4
	}
	if !learnable(p) {
		if b, ok := applyButton(p, r); ok {
			out = append(out, b)
		}
		return out
	}
	if b, ok := applyButton(p, ui.Rect{X: r.X + r.W - applyW, Y: r.Y, W: applyW, H: r.H}); ok {
		out = append(out, b)
		r.W -= applyW + 4
	}
	pg := p.Page
	short := shortfall(pg, f.State.Gold, f.State.Wisdom)
	label := learnLabel(pg)
	if (ui.Font{}).Width(label, 1) > r.W-4 {
		label = learnWord(pg)
	}
	return append(out, Button{ID: "learn:" + pg.ID, Rect: r, Label: label, Disabled: short != "", Tip: short})
}

func GrimoireAlmanac(f Frame, p KMapPlaque) (view.Page, bool) {
	if p.Fog {
		return view.Page{}, false
	}
	return firstMetPage(f, []string{view.SubjectGrimoire}, []string{p.Page.ID})
}

func (h *HUD) handbookCard(f Frame) (ui.Rect, []Button, bool) {
	if h.OverlayID != OverlayGrimoire {
		return ui.Rect{}, nil, false
	}
	buttons := h.CardButtons(f, h.Knowledge(f))
	if len(buttons) == 0 {
		return ui.Rect{}, nil, false
	}
	return buttons[0].Rect, buttons[1:], true
}

func (h *HUD) onMap(f Frame, target ui.Rect, s paintedSurface) paintedSurface {
	if h.OverlayID != OverlayGrimoire || !within(target, KnowledgeView) {
		return s
	}
	card, buttons, open := h.handbookCard(f)
	if open && slices.ContainsFunc(buttons, func(b Button) bool { return b.Rect == target }) {
		return s
	}
	s.Rect, _ = clipRect(s.Rect, KnowledgeView)
	if open {
		s.Rect = beside(s.Rect, Shadowed(card))
	}
	return s
}

func (h *HUD) drawCard(dst *ebiten.Image, f Frame, l KMapLayout) {
	p, ok := h.focusPlaque(l)
	if !ok {
		return
	}
	card, rows, lines := h.CardRect(p, f)
	if ui.Recording() {
		ui.Fill(dst, card.Offset(2, 3), ui.Palette.Shadow)
		ui.InRegion(PanelRegion+h.OverlayID+"-card", func() {
			h.paintCard(dst, p, card, rows, lines)
			h.drawCardButtons(dst, f, l)
		})
		return
	}
	img := h.Kmap.card.get(cardKey(p, card, lines), card.W, card.H, func(img *ebiten.Image) {
		h.paintCard(img, p, ui.Rect{W: card.W, H: card.H}, rows, lines)
	})
	ui.Fill(dst, card.Offset(2, 3), ui.Palette.Shadow)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(card.X), float64(card.Y))
	dst.DrawImage(img, op)
	h.drawCardButtons(dst, f, l)
}

func (h *HUD) drawCardButtons(dst *ebiten.Image, f Frame, l KMapLayout) {
	for _, b := range h.CardButtons(f, l)[1:] {
		h.drawButton(dst, b)
	}
}

func (h *HUD) paintCard(dst *ebiten.Image, p KMapPlaque, card ui.Rect, rows []CardRow, lines [][]string) {
	ui.Fill(dst, card, ui.Tint(ui.Palette.ConsoleEdge, 245))
	ui.Stroke(dst, card, ui.Palette.Gold)
	ui.Stroke(dst, card.Inset(2), ui.Tint(ui.Palette.Gold, 90))
	h.drawCardHead(dst, p, card, rows, lines)
	h.drawCardRows(dst, card, rows, lines)
}

func (h *HUD) drawCardHead(dst *ebiten.Image, p KMapPlaque, card ui.Rect, rows []CardRow, lines [][]string) {
	state := plaqueState(p)
	h.Lib.Draw(dst, gfx.KMapMedallionName(state), float64(card.X+KmapCardPad-1), float64(card.Y+KmapCardPad-1), 1, render.SpriteOptions{})
	if p.Fog || !h.Lib.Draw(dst, gfx.PageIconName(p.Page.ID), float64(card.X+KmapCardPad+14), float64(card.Y+KmapCardPad+14), 1, render.SpriteOptions{}) {
		h.Font.DrawShadow(dst, "?", card.X+KmapCardPad+10, card.Y+KmapCardPad+7, 2, ui.Palette.Cream)
	}
	x := card.X + kmapCardHead
	h.Font.DrawShadow(dst, lines[0][0], x, card.Y+KmapCardPad+2, 1, ui.Palette.GoldBright)
	if len(rows) > 1 && rows[1].Kind == RowK8s {
		for i, line := range lines[1] {
			h.Font.Draw(dst, line, x, card.Y+KmapCardPad+14+i*ui.LineHeight, 1, ui.Palette.Blue)
		}
	}
}

var cardColors = map[string]color.RGBA{
	RowRank: ui.Palette.TextDim, RowBody: ui.Palette.Cream, RowTeach: ui.Palette.Text, RowRunes: ui.Palette.Purple,
	RowEarn: ui.Palette.TextDim, RowDocs: ui.Palette.Blue, RowNow: ui.Palette.Green,
}

func cardColor(r CardRow) color.RGBA {
	switch r.Kind {
	case RowNeed, RowStatus:
		if r.Met {
			return ui.Palette.Green
		}
		return ui.Palette.RedText
	}
	return cardColors[r.Kind]
}

func (h *HUD) drawCardRows(dst *ebiten.Image, card ui.Rect, rows []CardRow, lines [][]string) {
	y := card.Y + kmapCardHead
	prev := RowK8s
	for i, r := range rows {
		if r.Kind == RowTitle || r.Kind == RowK8s {
			continue
		}
		y += rowGap(prev, r.Kind)
		if rowGap(prev, r.Kind) > 1 {
			ui.Fill(dst, ui.Rect{X: card.X + KmapCardPad, Y: y - 3, W: card.W - 2*KmapCardPad, H: 1}, ui.Tint(ui.Palette.Gold, 60))
		}
		for _, line := range lines[i] {
			h.CardFont(r.Kind).Draw(dst, line, card.X+KmapCardPad, y, 1, cardColor(r))
			y += ui.LineHeight
		}
		prev = r.Kind
	}
}

var realRunes = map[string]string{
	"war-horns": "HorizontalPodAutoscaler", "great-keeps": "great-keep Nodes", "citadel-keeps": "citadel-keep Nodes",
	"familiars": "sidecar containers", "drain": "kubectl drain",
}

func runeLabels(runes []string) []string {
	if !ui.Naming() {
		return runes
	}
	out := make([]string, len(runes))
	for i, r := range runes {
		if real, ok := realRunes[r]; ok {
			out[i] = real
			continue
		}
		out[i] = r
	}
	return out
}
