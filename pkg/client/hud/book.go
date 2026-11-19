package hud

import (
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	OverlayBook    = "book"
	bookPrefix     = "book"
	BookGo         = "book:go:"
	BookBack       = "book:back"
	BookHome       = "book:home"
	BookSearch     = "book:search"
	BookMark       = "book:mark"
	BookRibbon     = "book:ribbon:"
	BookShelf      = "book:shelf:"
	BookClose      = "overlay:close"
	BookSound      = "page"
	bookUnmet      = "Not in your Almanac yet: meet it in the realm first."
	bookMaxMarks   = 8
	bookTurnStep   = .1
	bookTurnFrames = 10
)

var BookBox = ui.Rect{X: 40, Y: TopH + 4, W: Width - 60, H: Height - TopH - 8}

type bookTurn struct {
	at  float64
	Dir int `json:"-"`
}

type bookState struct {
	PageID  string `json:"-"`
	history []string
	Scroll  [2]int `json:"-"`
	Search  string `json:"-"`
	Typing  bool   `json:"-"`
	ribbons []string
	shelf   string
	Turn    bookTurn `json:"-"`
}

func (b bookState) at(page string) bookState {
	next := b
	next.PageID, next.Scroll, next.Typing = page, [2]int{}, false
	return next
}

func (b bookState) visited(page string) bookState {
	next := b.at(page)
	next.history = append(slices.Clone(b.history), b.PageID)
	return next
}

func Holds(overlay string) bool {
	return overlay == OverlayBook || overlay == OverlayWelcome
}

func (h *HUD) BookPage() string { return h.Book.PageID }

func bookMet(f Frame, id string) bool {
	return slices.Contains(f.State.Almanac, id) || id != "" && id == introPageID(f)
}

func metPages(f Frame) []view.Page {
	out := []view.Page{}
	for _, p := range view.Content.Pages() {
		if bookMet(f, p.ID) {
			out = append(out, p)
		}
	}
	return out
}

func (h *HUD) OpenBook(page string, f Frame) []Intent {
	if page != "" && !bookMet(f, page) {
		h.Toast(bookUnmet, true)
		return []Intent{{Kind: IntentSound, Command: "error"}}
	}
	switch {
	case h.OverlayID != OverlayBook:
		h.Book = h.Book.at(page)
		h.Book.history = nil
	case h.Book.PageID != page:
		h.Book = h.Book.visited(page)
		h.Book.Turn = bookTurn{at: h.Now, Dir: 1}
	}
	h.OverlayID = OverlayBook
	return []Intent{{Kind: IntentSound, Command: BookSound}}
}

func (h *HUD) pressBook(rest string, f Frame) []Intent {
	switch {
	case strings.HasPrefix(rest, "go:"):
		return h.OpenBook(strings.TrimPrefix(rest, "go:"), f)
	case strings.HasPrefix(rest, "ribbon:"):
		return h.OpenBook(strings.TrimPrefix(rest, "ribbon:"), f)
	case strings.HasPrefix(rest, "shelf:"):
		return h.OpenShelf(strings.TrimPrefix(rest, "shelf:"), "", f)
	case rest == "back":
		return h.BookBack()
	case rest == "home":
		return h.OpenShelf("", "", f)
	case rest == "search":
		intents := h.OpenShelf("", h.Book.Search, f)
		h.Book.Typing = true
		return intents
	case rest == "mark":
		h.Book = h.Book.marked()
	}
	return []Intent{{Kind: IntentSound, Command: "click"}}
}

func (h *HUD) OpenShelf(shelf, search string, f Frame) []Intent {
	intents := h.OpenBook("", f)
	next := h.Book
	next.shelf, next.Search, next.Typing = shelf, search, false
	h.Book = next
	return intents
}

func (h *HUD) BookBack() []Intent {
	if len(h.Book.history) == 0 {
		return nil
	}
	last := h.Book.history[len(h.Book.history)-1]
	next := h.Book.at(last)
	next.history = slices.Clone(h.Book.history[:len(h.Book.history)-1])
	next.Turn = bookTurn{at: h.Now, Dir: -1}
	h.Book = next
	return []Intent{{Kind: IntentSound, Command: BookSound}}
}

func (b bookState) marked() bookState {
	next := b
	if b.PageID == "" {
		return next
	}
	if i := slices.Index(b.ribbons, b.PageID); i >= 0 {
		next.ribbons = slices.Delete(slices.Clone(b.ribbons), i, i+1)
		return next
	}
	if len(b.ribbons) >= bookMaxMarks {
		return next
	}
	next.ribbons = append(slices.Clone(b.ribbons), b.PageID)
	return next
}

func (h *HUD) bookKeys(in Input, f Frame) ([]Intent, bool) {
	if h.Book.Typing {
		return h.searchKeys(in, f), len(in.Keys) > 0 || len(in.Chars) > 0
	}
	switch {
	case in.Pressed(ebiten.KeyEscape) && h.Book.Search != "":
		h.Book = h.Book.Searched("")
	case in.Pressed(ebiten.KeyEscape):
		h.OverlayID = OverlayNone
	case in.Pressed(ebiten.KeyBackspace):
		return h.BookBack(), true
	}
	return nil, len(in.Keys) > 0
}

func (b bookState) Searched(query string) bookState {
	next := b
	next.Search = query
	if query == "" {
		next.Typing = false
	}
	return next
}

func (h *HUD) searchKeys(in Input, f Frame) []Intent {
	query := []rune(h.Book.Search)
	for _, r := range in.Chars {
		if r >= ' ' && len(query) < 24 {
			query = append(query, r)
		}
	}
	switch {
	case in.Pressed(ebiten.KeyEscape):
		h.Book = h.Book.Searched("")
		return nil
	case in.Pressed(ebiten.KeyBackspace) && len(query) > 0:
		query = query[:len(query)-1]
	case in.Pressed(ebiten.KeyEnter):
		h.Book = h.Book.Searched(string(query))
		if found := searchMet(f, string(query)); len(found) > 0 {
			return h.OpenBook(found[0].ID, f)
		}
		return nil
	}
	next := h.Book.Searched(string(query))
	next.Typing = true
	h.Book = next
	return nil
}

func searchMet(f Frame, query string) []view.Page {
	out := []view.Page{}
	for _, p := range view.Content.Search(query) {
		if bookMet(f, p.ID) {
			out = append(out, p)
		}
	}
	return out
}

func (h *HUD) bookWheel(in Input, f Frame, step int) {
	for side := range 2 {
		if !BookPageRect(side).Contains(in.MX, in.MY) {
			continue
		}
		lines := h.BookSide(f, side)
		next := h.Book
		next.Scroll[side] = min(max(next.Scroll[side]+step, 0), BookMaxScroll(lines, BookPageRect(side).H))
		h.Book = next
	}
}

func (h *HUD) linkButtons(f Frame, side int) []Button {
	out := []Button{}
	for _, l := range h.visibleLines(f, side) {
		for _, s := range l.line.Segs {
			if s.link == "" {
				continue
			}
			tip := ""
			if p, ok := view.FindPage(s.link); ok {
				tip = p.Purpose
			}
			out = append(out, Button{ID: BookGo + s.link, Rect: ui.Rect{X: l.x + s.cell*ui.Advance*l.line.Scale, Y: l.y, W: len([]rune(s.Text)) * ui.Advance * l.line.Scale, H: ui.LineHeight * l.line.Scale}, Tip: tip})
		}
	}
	return out
}
