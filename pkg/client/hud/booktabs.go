package hud

import (
	"slices"
	"strconv"
	"strings"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	ShelfCategory = "category:"
	ShelfBranch   = "branch:"
	shelfMarks    = "marks"
	artTabW       = 14
	artTabH       = 13
	artTabGap     = 1
	artTabTuck    = 2
	artTabInset   = 8
	artFlagTop    = 30
	artFlagGap    = 3
	bookPagePad   = 8
	bookMarkW     = 12
	bookMarkLong  = 40
	bookMarkStub  = 18
	bookCloseW    = CloseFace + 2*ClosePad
)

func BookPageRect(side int) ui.Rect {
	l, r := pageSpan(side)
	inset := (artBorder+1)*bookPx + bookPagePad
	if side == 0 {
		r -= 7
	} else {
		l += 7
	}
	y := PageTopY() + inset
	return ui.Rect{X: ArtX(l) + inset, Y: y, W: (r-l+1)*bookPx - 2*inset, H: pageBottomY() - inset - y}
}

func topTab(side, i int) ui.Rect {
	x := artPageL + artTabInset + i*(artTabW+artTabGap)
	if side == 1 {
		x = artPageR - artTabInset - artTabW - i*(artTabW+artTabGap)
	}
	return ui.Rect{X: ArtX(x), Y: artY(artPageT - artTabH + artTabTuck), W: artTabW * bookPx, H: artTabH * bookPx}
}

func flagRect(i int) ui.Rect {
	x := ArtX(ArtCoverR+1) - art.FlagTuck
	return ui.Rect{X: x, Y: artY(artFlagTop) + i*(art.FlagH+2*artFlagGap+4), W: art.FlagW, H: art.FlagH}
}

func closeTab() ui.Rect {
	r := topTab(1, 0)
	return ui.Rect{X: r.X + r.W - bookCloseW, Y: r.Y, W: bookCloseW, H: max(r.H, bookCloseW)}
}

func (h *HUD) BookButtons(f Frame) []Button {
	out := []Button{{ID: BookClose, Rect: closeTab(), Face: closeTab().Inset(ClosePad), Label: "X", Tip: "Close the Almanac [Esc]"}}
	out = append(out, h.controlTabs(f)...)
	out = append(out, h.shelfTabs(f)...)
	if h.Book.PageID != "" {
		out = append(out, Button{ID: BookMark, Rect: h.markRect(), Tip: h.markTip()})
	}
	for side := range 2 {
		out = append(out, h.linkButtons(f, side)...)
	}
	return append(out, h.BookTrialButtons()...)
}

func (h *HUD) controlTabs(f Frame) []Button {
	out := []Button{
		{ID: BookBack, Rect: topTab(1, 1), Tip: "Back to the page before [Backspace]", Disabled: len(h.Book.history) == 0},
		{ID: BookHome, Rect: topTab(1, 2), Tip: "Contents: every page your realm has met", Selected: h.Book.PageID == "" && h.Book.shelf == "" && !h.Searching()},
		{ID: BookSearch, Rect: topTab(1, 3), Tip: "Search the pages you have met: type a word, Enter opens the first", Selected: h.Searching()},
	}
	for _, id := range h.Book.ribbons {
		p, ok := view.FindPage(id)
		if !ok || !bookMet(f, id) {
			continue
		}
		out = append(out, Button{ID: BookRibbon + id, Rect: topTab(1, len(out)+1), Tip: "Ribbon: " + ui.Name(p.Title), Selected: h.Book.PageID == id})
	}
	return out
}

func (h *HUD) Searching() bool { return h.Book.PageID == "" && (h.Book.Typing || h.Book.Search != "") }

func (h *HUD) shelfTabs(f Frame) []Button {
	out := []Button{}
	for i, c := range shelfCategories {
		n := len(booksOnShelf(f, ShelfCategory+c))
		out = append(out, Button{ID: BookShelf + ShelfCategory + c, Rect: topTab(0, i), Tip: shelfTitle(ShelfCategory+c) + shelfCount(n), Disabled: n == 0, Selected: h.onShelf(ShelfCategory + c)})
	}
	for i, b := range ShelfBranches {
		n := len(booksOnShelf(f, ShelfBranch+b))
		out = append(out, Button{ID: BookShelf + ShelfBranch + b, Rect: flagRect(i), Tip: shelfTitle(ShelfBranch+b) + shelfCount(n), Disabled: n == 0, Selected: h.onShelf(ShelfBranch + b)})
	}
	return out
}

func (h *HUD) onShelf(shelf string) bool { return h.Book.PageID == "" && h.Book.shelf == shelf }

func shelfCount(n int) string {
	if n == 0 {
		return ": nothing met yet"
	}
	if n == 1 {
		return ": 1 page"
	}
	return ": " + strconv.Itoa(n) + " pages"
}

func shelfTitle(shelf string) string {
	switch {
	case strings.HasPrefix(shelf, ShelfCategory):
		return Upper(strings.TrimPrefix(shelf, ShelfCategory))
	case strings.HasPrefix(shelf, ShelfBranch):
		return Upper(strings.TrimPrefix(shelf, ShelfBranch))
	}
	return "CONTENTS"
}

func booksOnShelf(f Frame, shelf string) []view.Page {
	out := []view.Page{}
	for _, p := range metPages(f) {
		if onShelf(p, shelf) {
			out = append(out, p)
		}
	}
	return out
}

func onShelf(p view.Page, shelf string) bool {
	switch {
	case strings.HasPrefix(shelf, ShelfCategory):
		return pageCategory(p) == strings.TrimPrefix(shelf, ShelfCategory)
	case strings.HasPrefix(shelf, ShelfBranch):
		return pageBranch(p) == strings.TrimPrefix(shelf, ShelfBranch)
	}
	return true
}

func (h *HUD) marked() bool { return slices.Contains(h.Book.ribbons, h.Book.PageID) }

func (h *HUD) markTip() string {
	if h.marked() {
		return "Take the ribbon out of this page"
	}
	return "Hang a ribbon on this page: it gets a tab on the top edge"
}

func (h *HUD) markRect() ui.Rect {
	x := ArtX(artPageR) - bookMarkW - 2
	length := bookMarkStub
	if h.marked() {
		length = bookMarkLong
	}
	return ui.Rect{X: x, Y: PageTopY() - 2, W: bookMarkW, H: length}
}
