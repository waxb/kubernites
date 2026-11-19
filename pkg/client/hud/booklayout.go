package hud

import (
	"image/color"
	"strings"
	"unicode"

	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	bookCapLines   = 3
	bookCapCells   = 5
	BookCapW       = bookCapCells*ui.Advance - 4
	bookCapH       = bookCapLines*ui.LineHeight - 1
	bookFrameW     = 128
	BookFrameH     = 158
	BookFrameCells = 23
	bookTitleH     = bookCapH + 7
)

var (
	BookInk     = color.RGBA{58, 44, 32, 255}
	bookSoft    = color.RGBA{104, 84, 60, 255}
	bookHead    = color.RGBA{128, 40, 34, 255}
	bookLink    = color.RGBA{36, 74, 112, 255}
	bookLeather = color.RGBA{74, 40, 30, 255}
)

type bookSeg struct {
	Text string `json:"-"`
	link string
	cell int
}

type BookLine struct {
	Segs   []bookSeg `json:"-"`
	Scale  int       `json:"-"`
	clr    color.RGBA
	Indent int  `json:"-"`
	Cap    rune `json:"-"`
	tall   int
	rule   bool
}

type placedLine struct {
	line BookLine
	x, y int
	w    int
}

func (l BookLine) Height() int {
	if l.tall > 0 {
		return l.tall
	}
	return ui.LineHeight * max(l.Scale, 1)
}

func titleLine(text string, indent int) BookLine {
	runes := []rune(text)
	if len(runes) == 0 {
		return BlankLine()
	}
	return BookLine{Segs: []bookSeg{{Text: string(runes[1:])}}, Cap: runes[0], Scale: 2, clr: bookHead, Indent: indent, tall: bookTitleH, rule: true}
}

func TitleFit(text string, indent, cells int) BookLine {
	l := titleLine(strings.ToUpper(text), indent)
	if len(l.Segs) == 0 {
		return l
	}
	room := ((cells-indent)*ui.Advance - BookCapW - 4) / (2 * ui.Advance)
	rows := titleRows(titlePieces(string([]rune(text)[1:])), room)
	if len(rows) < 2 {
		return l
	}
	next := l
	next.Segs = make([]bookSeg, len(rows))
	for i, r := range rows {
		next.Segs[i] = bookSeg{Text: strings.ToUpper(r)}
	}
	next.tall = bookTitleH + 2*ui.LineHeight*(len(rows)-1)
	return next
}

type titlePiece struct {
	text  string
	space bool
}

func titlePieces(text string) []titlePiece {
	out := []titlePiece{}
	for i, word := range strings.Fields(text) {
		parts := wordParts(word)
		for j, part := range parts {
			out = append(out, titlePiece{text: part, space: i > 0 && j == 0})
		}
	}
	return out
}

func wordParts(word string) []string {
	out := []string{}
	start := 0
	runes := []rune(word)
	for i := 1; i < len(runes); i++ {
		if runes[i-1] == '-' || unicode.IsUpper(runes[i]) && unicode.IsLower(runes[i-1]) {
			out = append(out, string(runes[start:i]))
			start = i
		}
	}
	return append(out, string(runes[start:]))
}

func titleRows(pieces []titlePiece, room int) []string {
	rows := []string{}
	cur := ""
	for _, p := range pieces {
		join := p.text
		if p.space && cur != "" {
			join = " " + p.text
		}
		if cur != "" && len([]rune(cur+join)) > room {
			rows, cur = append(rows, cur), p.text
			continue
		}
		cur += join
	}
	return append(rows, cur)
}

func (l BookLine) titled() bool { return l.Cap != 0 && l.Scale == 2 }

func linesHeight(lines []BookLine) int {
	total := 0
	for _, l := range lines {
		total += l.Height()
	}
	return total
}

func TextLine(text string, clr color.RGBA, scale int) BookLine {
	return BookLine{Segs: []bookSeg{{Text: text}}, Scale: scale, clr: clr}
}

func linkLine(text, link string) BookLine {
	return BookLine{Segs: []bookSeg{{Text: text, link: link}}, Scale: 1, clr: bookLink}
}

func BlankLine() BookLine { return BookLine{Scale: 1} }

func (h *HUD) BookSide(f Frame, side int) []BookLine {
	cells := ui.MaxChars(BookPageRect(side).W, 1)
	role := side
	if f.KubernetesFirst() {
		role = 1 - side
	}
	switch {
	case h.Book.PageID != "":
		p, _ := view.FindPage(h.Book.PageID)
		if role == 0 {
			return GamePage(p, f, cells)
		}
		return kubePage(p, f, cells)
	case h.Searching():
		query := h.Book.Search
		if h.Book.Typing {
			query += "_"
		}
		return searchSide(searchMet(f, h.Book.Search), query, role)
	}
	return contentsSide(booksOnShelf(f, h.Book.shelf), shelfTitle(h.Book.shelf), role)
}

func storyText(text string) string { return ui.Decorate(ui.Name(text)) }

func besideFrame(cells int) func(y int) (int, int) {
	return func(y int) (int, int) {
		if y < BookFrameH+2 {
			return BookFrameCells, cells - BookFrameCells
		}
		return 0, cells
	}
}

func GamePage(p view.Page, f Frame, cells int) []BookLine {
	fl := bookFlow{slot: besideFrame(cells)}.with(TitleFit(ui.Name(p.Title), BookFrameCells, cells))
	fl = fl.para(spansOf(p.Purpose, f, storyText), BookInk)
	fl = fl.para(spansOf(p.How, f, storyText), bookSoft).with(BlankLine())
	fl = fl.para(spansOf(p.Game, f, storyText), BookInk)
	if p.Why != "" {
		fl = fl.with(BlankLine()).heading(storyText("WHY IT MATTERS"))
		fl = fl.para(spansOf(p.Why, f, storyText), BookInk)
	}
	return fl.lines
}

func kubePage(p view.Page, f Frame, cells int) []BookLine {
	title := []rune(strings.ToUpper(p.K8sTitle))
	if len(title) == 0 {
		title = []rune(" ")
	}
	out := []BookLine{{Segs: []bookSeg{{Text: string(title[1:])}}, Cap: title[0], Scale: 1, clr: bookHead, Indent: bookCapCells}}
	out = append(out, WrapSpans(spansOf(p.K8s, f, ui.Decorate), cells, BookInk, bookCapLines-1)...)
	if p.Doc != "" {
		out = append(out, BlankLine(), TextLine("Read more at the source:", bookSoft, 1))
		for _, chunk := range chunks(p.Doc, cells) {
			out = append(out, TextLine(chunk, bookLink, 1))
		}
	}
	return append(out, BookBridge(p, f, cells)...)
}

func contentsSide(pages []view.Page, title string, side int) []BookLine {
	if side == 1 {
		out := []BookLine{titleLine("KUBERNETES", 0), BlankLine()}
		for _, p := range pages {
			out = append(out, linkLine(p.K8sTitle, p.ID))
		}
		return out
	}
	out := []BookLine{titleLine(ui.Name(title), 0), BlankLine()}
	if len(pages) == 0 {
		return append(out, TextLine(storyText("Nothing met yet: the pages appear as the realm meets them."), bookSoft, 1))
	}
	stage := ""
	for _, p := range pages {
		if p.Stage != stage {
			stage = p.Stage
			out = append(out, TextLine(storyText(stageHeading(stage)), bookSoft, 1))
		}
		out = append(out, linkLine(storyText(p.Title), p.ID))
	}
	return out
}

func stageHeading(stage string) string {
	if stage == "" {
		return "ALWAYS OPEN"
	}
	if name, ok := stageAges[stage]; ok {
		return name
	}
	return strings.ToUpper(stage)
}

var stageAges = map[string]string{view.StageDev: "DARK AGE", view.StageStaging: "FEUDAL AGE"}

func searchSide(found []view.Page, query string, side int) []BookLine {
	if side == 1 {
		out := []BookLine{titleLine("KUBERNETES", 0), BlankLine()}
		for _, p := range found {
			out = append(out, linkLine(p.K8sTitle, p.ID))
		}
		return out
	}
	out := []BookLine{titleLine(ui.Name("SEARCH"), 0), TextLine("for: "+query, bookSoft, 1), BlankLine()}
	if len(found) == 0 {
		return append(out, TextLine(storyText("No page you have met says that."), bookSoft, 1))
	}
	for _, p := range found {
		out = append(out, linkLine(storyText(p.Title), p.ID))
	}
	return out
}

func spansOf(text string, f Frame, name func(string) string) []view.TextSpan {
	out := []view.TextSpan{}
	for _, s := range view.Parse(text) {
		link := s.Link
		if link != "" && !bookMet(f, link) {
			link = ""
		}
		out = append(out, view.TextSpan{Text: name(s.Text), Link: link})
	}
	return out
}

type word struct {
	text string
	link string
}

func wordsOf(spans []view.TextSpan) []word {
	out := []word{}
	for _, s := range spans {
		for i, w := range strings.Split(s.Text, " ") {
			if i > 0 {
				out = append(out, word{text: " ", link: ""})
			}
			if w != "" {
				out = append(out, word{text: w, link: s.Link})
			}
		}
	}
	return out
}

type slotFunc func(line int) (indent, cells int)

func WrapSpans(spans []view.TextSpan, cells int, clr color.RGBA, capLines int) []BookLine {
	return wrapSlots(spans, clr, func(line int) (int, int) {
		if line < capLines {
			return bookCapCells, cells - bookCapCells
		}
		return 0, cells
	})
}

func unitsOf(words []word) [][]word {
	out := [][]word{}
	cur := []word{}
	for _, w := range words {
		if w.text == " " {
			if len(cur) > 0 {
				out = append(out, cur)
			}
			out, cur = append(out, []word{w}), []word{}
			continue
		}
		cur = append(cur, w)
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return out
}

func unitWidth(unit []word) int {
	n := 0
	for _, w := range unit {
		n += len([]rune(w.text))
	}
	return n
}

func wrapSlots(spans []view.TextSpan, clr color.RGBA, slot slotFunc) []BookLine {
	out := []BookLine{}
	indent, width := slot(0)
	cur, used := BookLine{Scale: 1, clr: clr, Indent: indent}, 0
	flush := func() {
		cur.Segs = trimLine(cur.Segs)
		out = append(out, cur)
		indent, width = slot(len(out))
		cur, used = BookLine{Scale: 1, clr: clr, Indent: indent}, 0
	}
	for _, unit := range unitsOf(wordsOf(spans)) {
		space := len(unit) == 1 && unit[0].text == " "
		if space && used == 0 {
			continue
		}
		if n := unitWidth(unit); used+n > width && used > 0 {
			flush()
			if space {
				continue
			}
		}
		for _, w := range unit {
			cur.Segs = appendSeg(cur.Segs, bookSeg{Text: w.text, link: w.link, cell: used})
			used += len([]rune(w.text))
		}
	}
	if used > 0 {
		flush()
	}
	return out
}

type bookFlow struct {
	lines []BookLine
	y     int
	slot  func(y int) (indent, cells int)
}

func (fl bookFlow) para(spans []view.TextSpan, clr color.RGBA) bookFlow {
	y := fl.y
	lines := wrapSlots(spans, clr, func(line int) (int, int) { return fl.slot(y + line*ui.LineHeight) })
	return fl.with(lines...)
}

func (fl bookFlow) with(lines ...BookLine) bookFlow {
	next := fl
	next.lines = append(append([]BookLine(nil), fl.lines...), lines...)
	next.y += linesHeight(lines)
	return next
}

func (fl bookFlow) heading(text string) bookFlow {
	indent, _ := fl.slot(fl.y)
	l := TextLine(text, bookHead, 1)
	l.Indent = indent
	return fl.with(l)
}

func appendSeg(segs []bookSeg, s bookSeg) []bookSeg {
	if n := len(segs); n > 0 && segs[n-1].link == s.link && (s.link != "" || s.Text != " ") {
		last := segs[n-1]
		return append(segs[:n-1:n-1], bookSeg{Text: last.Text + s.Text, link: last.link, cell: last.cell})
	}
	return append(segs, s)
}

func trimLine(segs []bookSeg) []bookSeg {
	for len(segs) > 0 && strings.TrimSpace(segs[len(segs)-1].Text) == "" {
		segs = segs[:len(segs)-1]
	}
	return segs
}

func chunks(text string, cells int) []string {
	runes := []rune(text)
	out := []string{}
	for len(runes) > cells {
		out = append(out, string(runes[:cells]))
		runes = runes[cells:]
	}
	return append(out, string(runes))
}

func BookMaxScroll(lines []BookLine, height int) int {
	total := 0
	for i := len(lines) - 1; i >= 0; i-- {
		total += lines[i].Height()
		if total > height {
			return i + 1
		}
	}
	return 0
}

func (h *HUD) visibleLines(f Frame, side int) []placedLine {
	r := BookPageRect(side)
	lines := h.BookSide(f, side)
	out := []placedLine{}
	y := r.Y
	for _, l := range lines[min(h.Book.Scroll[side], len(lines)):] {
		if y+l.Height() > r.Y+r.H {
			break
		}
		out = append(out, placedLine{line: l, x: r.X + l.Indent*ui.Advance, y: y, w: r.W - l.Indent*ui.Advance})
		y += l.Height()
	}
	return out
}

func (h *HUD) contentTop(f Frame, side int) int {
	lines := h.BookSide(f, side)
	return BookPageRect(side).Y - linesHeight(lines[:min(h.Book.Scroll[side], len(lines))])
}
