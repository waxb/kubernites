package hud

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	RowNow      = "now"
	cardNowMax  = 4
	callBadge   = 11
	callLift    = 4
	nowLead     = "For your realm: "
	stonePageID = "speaking-stone"
)

func nowRows(pg view.GrimoirePage) []CardRow {
	if pg.Now == "" || pg.State == "unlocked" {
		return nil
	}
	return []CardRow{{Kind: RowNow, Text: nowLead + pg.Now}}
}

func PageCalls(p KMapPlaque) bool {
	return learnable(p) && p.Page.Now != ""
}

func pageProgress(pg view.GrimoirePage, st view.GameState) float64 {
	share := func(have, need float64) float64 {
		if need <= 0 {
			return 1
		}
		return min(have/need, 1)
	}
	return min(share(st.Gold, pg.Cost.Gold), share(st.Wisdom, pg.Cost.Wisdom))
}

func NextPage(st view.GameState) (view.GrimoirePage, bool) {
	var best view.GrimoirePage
	found, top := false, -1.0
	for _, pg := range st.Grimoire {
		if pg.State != "available" || pg.ID == stonePageID || pg.Cost.Gold+pg.Cost.Wisdom <= 0 {
			continue
		}
		if p := pageProgress(pg, st); p > top || p == top && pg.Cost.Gold < best.Cost.Gold {
			best, top, found = pg, p, true
		}
	}
	return best, found
}

func CallingPage(st view.GameState) string {
	id, top := "", -1.0
	for _, pg := range st.Grimoire {
		if pg.State != "available" || pg.Now == "" {
			continue
		}
		if p := pageProgress(pg, st); p > top {
			id, top = pg.ID, p
		}
	}
	return id
}

func pageTitle(pg view.GrimoirePage) string {
	if pg.MaxRank > 1 && pg.Rank < len(pg.Ranks) {
		return fmt.Sprintf("%s %s: %s", pg.Name, romanOf(pg.Rank), pg.Ranks[pg.Rank].Title)
	}
	return pg.Name
}

func NextPageLines(f Frame) []string {
	pg, ok := NextPage(f.State)
	if !ok {
		return nil
	}
	have := fmt.Sprintf("Next page: %s: %s/%s wisdom, %s/%s gold.", pageTitle(pg), Compact(min(f.State.Wisdom, pg.Cost.Wisdom)), Compact(pg.Cost.Wisdom),
		Compact(min(f.State.Gold, pg.Cost.Gold)), Compact(pg.Cost.Gold))
	if pageProgress(pg, f.State) >= 1 {
		have = fmt.Sprintf("Next page: %s is yours to learn in the Grimoire.", pageTitle(pg))
	}
	if pg.Now == "" {
		return []string{have}
	}
	return []string{have, nowLead + pg.Now}
}

func pageNowOf(st view.GameState, id string) string {
	for _, pg := range st.Grimoire {
		if pg.ID == id {
			return pg.Now
		}
	}
	return ""
}

func (h *HUD) focusCalling(f Frame, l KMapLayout) {
	id := h.Grimoire
	if id == "" {
		id = CallingPage(f.State)
	}
	p, ok := l.Find(id)
	if !ok || id == "" {
		return
	}
	if _, seen := clipRect(h.ScreenOf(p.Rect()), KnowledgeView); seen && h.Grimoire == "" {
		return
	}
	r := p.Rect()
	h.Kmap.Scroll = KmapScroll{Pos: float64(r.X + r.W/2 - KnowledgeView.W/2)}.clamp(KmapLimit(l))
}

func (h *HUD) drawPageCall(dst *ebiten.Image, p KMapPlaque, r ui.Rect) {
	if !PageCalls(p) {
		return
	}
	badge := ui.Rect{X: r.X + r.W - callBadge - 1, Y: r.Y - callLift, W: callBadge, H: callBadge}
	ui.Fill(dst, badge.Offset(1, 1), ui.Tint(ui.Palette.Ink, 160))
	ui.Fill(dst, badge, ui.Palette.GoldBright)
	ui.Stroke(dst, badge, ui.Palette.Ink)
	h.Font.DrawCentered(dst, "!", badge, 1, ui.Palette.Ink)
}

func withPageWay(b Button, f Frame) Button {
	pg, ok := NextPage(f.State)
	if !ok {
		return b
	}
	next := b
	next.Tip = strings.Join(append([]string{b.Tip}, NextPageLines(f)...), "\n")
	switch p := pageProgress(pg, f.State); {
	case p >= 1 && pg.Now != "":
		next.Cost = "!"
	case p < 1:
		next.Note = fmt.Sprintf("%d%%", int(p*100))
		next.Tip += fmt.Sprintf("\n%s: you hold %s of its price", next.Note, next.Note)
	}
	return next
}
