package hud

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	bookVCut = 5
)

var (
	bookGilt    = color.RGBA{196, 152, 70, 255}
	bookDim     = color.RGBA{8, 10, 12, 170}
	bookMarkRed = color.RGBA{150, 36, 36, 255}
	bookMarkOff = color.RGBA{196, 158, 132, 255}
	pageIcons   = map[string]string{"gold": ui.IconGold, "wisdom": ui.IconWisdom, "food": ui.IconFood, "housing": ui.IconHousing}
	bookLiteral = ui.Font{Literal: true}
)

var bookSpriteIcons = map[string]string{
	"treasury": "gold", "owl": "wisdom", "hut": "pod", "keep": "node",
}

var bookTextIcons = map[string]string{
	"treasury": ui.IconGold, "owl": ui.IconWisdom, "fields": ui.IconFood, "land": ui.IconHousing,
}

func (h *HUD) drawBook(dst *ebiten.Image, f Frame) {
	ui.Fill(dst, ui.Rect{Y: TopH, W: Width, H: Height - TopH}, bookDim)
	buttons := h.BookButtons(f)
	for _, b := range buttons {
		if b.ID != BookMark && !strings.HasPrefix(b.ID, BookGo) && !strings.HasPrefix(b.ID, trialPrefix+":") {
			h.drawBookTab(dst, b)
		}
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(bookPx, bookPx)
	op.GeoM.Translate(float64(BookBox.X), float64(BookBox.Y))
	dst.DrawImage(bookArt(), op)
	if p, ok := view.FindPage(h.Book.PageID); ok && h.Book.PageID != "" {
		h.drawPortrait(dst, f, p)
	}
	for side := range 2 {
		h.drawBookSide(dst, f, side)
	}
	for _, b := range buttons {
		if b.ID == BookMark {
			h.drawMark(dst, b)
		}
	}
	h.drawTrialButtons(dst, h.BookTrialButtons())
	h.drawTurn(dst)
}

func (h *HUD) drawMark(dst *ebiten.Image, b Button) {
	hot := h.Hovering && h.Hover.ID == b.ID
	clr := bookMarkOff
	if h.marked() || hot {
		clr = bookMarkRed
	}
	drawRibbon(dst, b.Rect, clr, hot)
}

func drawRibbon(dst *ebiten.Image, r ui.Rect, clr color.RGBA, hot bool) {
	body := r.H - bookVCut
	ui.Fill(dst, ui.Rect{X: r.X - 1, Y: r.Y, W: r.W + 2, H: body + 1}, bookOutline)
	ui.Fill(dst, ui.Rect{X: r.X, Y: r.Y, W: r.W, H: body}, clr)
	mid := r.W / 2
	for k := range bookVCut {
		y := r.Y + body + k
		ui.Fill(dst, ui.Rect{X: r.X, Y: y, W: mid - k, H: 1}, clr)
		ui.Fill(dst, ui.Rect{X: r.X + mid + k, Y: y, W: r.W - mid - k, H: 1}, clr)
	}
	ui.Fill(dst, ui.Rect{X: r.X + r.W - 2, Y: r.Y, W: 2, H: body}, tone(clr, .7, 0))
	if hot {
		ui.Fill(dst, ui.Rect{X: r.X, Y: r.Y, W: r.W, H: 2}, ui.Palette.GoldBright)
	}
}

func (h *HUD) drawPortrait(dst *ebiten.Image, f Frame, p view.Page) {
	side := f.gameSide()
	page := BookPageRect(side)
	frame := ui.Rect{X: page.X, Y: h.contentTop(f, side), W: bookFrameW, H: BookFrameH}
	canvas := clipTo(dst, page)
	ui.Fill(canvas, frame.Offset(2, 2), ui.Tint(bookSoft, 110))
	ui.Fill(canvas, frame, bookGilt)
	ui.Stroke(canvas, frame, bookHead)
	ui.Stroke(canvas, frame.Inset(3), tone(bookGilt, .7, 0))
	inner := frame.Inset(4)
	age := gfx.AgeTier(f.State.Age)
	if world.DrawPortrait(h.Lib, canvas, inner, p.Sprite, f.Site, age, h.Now) || world.DrawPortrait(h.Lib, canvas, inner, art.PageIconName(p.ID), f.Site, age, h.Now) {
		return
	}
	ui.Fill(canvas, inner, ui.Tint(ui.Palette.ParchmentDark, 200))
	if h.Lib.Fitted(canvas, gfx.BuildIcon(p.Sprite, 1), inner.Inset(8)) || h.Lib.Icon(canvas, bookSpriteIcons[p.Sprite], inner.X+inner.W/2-8, inner.Y+inner.H/2-8) {
		return
	}
	icon, ok := bookTextIcons[p.Sprite]
	if !ok {
		icon = ui.IconWisdom
	}
	bookLiteral.Draw(canvas, icon, inner.X+inner.W/2-2*ui.GlyphWidth*2, inner.Y+inner.H/2-ui.GlyphHeight*2, 4, BookInk)
}

func (h *HUD) drawBookSide(dst *ebiten.Image, f Frame, side int) {
	lines := h.visibleLines(f, side)
	for _, l := range lines {
		switch {
		case l.line.titled():
			drawTitle(dst, l)
			continue
		case l.line.Cap != 0:
			drawIlluminated(dst, l.line.Cap, l.x-bookCapCells*ui.Advance, l.y)
		}
		for _, s := range l.line.Segs {
			x := l.x + s.cell*ui.Advance*l.line.Scale
			clr := l.line.clr
			if s.link != "" {
				clr = bookLink
				lead := ui.IconLead(s.Text) * ui.Advance * l.line.Scale
				w := len([]rune(s.Text))*ui.Advance*l.line.Scale - lead
				ui.Fill(dst, ui.Rect{X: x + lead, Y: l.y + ui.CapHeight*l.line.Scale + 1, W: w - 1, H: 1}, h.linkUnderline(s.link))
			}
			bookLiteral.Draw(dst, s.Text, x, l.y, l.line.Scale, clr)
		}
	}
	h.drawMore(dst, f, side, len(lines))
}

func drawTitle(dst *ebiten.Image, l placedLine) {
	drawIlluminated(dst, l.line.Cap, l.x, l.y)
	tx := l.x + BookCapW + 4
	room := ui.MaxChars(l.w-(tx-l.x), 2)
	rule := l.y + l.line.tall - 4
	if len(l.line.Segs) == 1 {
		bookLiteral.Draw(dst, ui.Clip(l.line.Segs[0].Text, room), tx, l.y+bookCapH-2*ui.CapHeight-1, 2, l.line.clr)
	} else {
		for i, s := range l.line.Segs {
			bookLiteral.Draw(dst, ui.Clip(s.Text, room), tx, l.y+2+2*ui.LineHeight*i, 2, l.line.clr)
		}
	}
	ui.Fill(dst, ui.Rect{X: l.x, Y: rule, W: l.w, H: 1}, bookHead)
	ui.Fill(dst, ui.Rect{X: l.x, Y: rule + 2, W: l.w * 2 / 3, H: 1}, bookGilt)
}

func (h *HUD) linkUnderline(link string) color.Color {
	if h.Hovering && h.Hover.ID == BookGo+link {
		return bookHead
	}
	return ui.Tint(bookLink, 150)
}

func drawIlluminated(dst *ebiten.Image, letter rune, x, y int) {
	box := ui.Rect{X: x, Y: y, W: BookCapW, H: bookCapH}
	ui.Fill(dst, box.Offset(1, 1), ui.Tint(bookSoft, 120))
	ui.Fill(dst, box, ui.Palette.GoldBright)
	ui.Stroke(dst, box, bookHead)
	ui.Stroke(dst, box.Inset(2), bookGilt)
	for _, c := range [][2]int{{3, 3}, {box.W - 5, 3}, {3, box.H - 5}, {box.W - 5, box.H - 5}} {
		ui.Fill(dst, ui.Rect{X: box.X + c[0], Y: box.Y + c[1], W: 2, H: 2}, bookLink)
	}
	scale := 3
	lx := box.X + (box.W-ui.GlyphWidth*scale)/2
	ly := box.Y + (box.H-ui.CapHeight*scale)/2
	bookLiteral.DrawShadow(dst, string(letter), lx, ly, scale, bookHead)
}

func (h *HUD) drawMore(dst *ebiten.Image, f Frame, side, shown int) {
	r := BookPageRect(side)
	total := len(h.BookSide(f, side))
	if h.Book.Scroll[side] > 0 {
		bookLiteral.Draw(dst, "^", r.X+r.W-6, r.Y-6, 1, bookSoft)
	}
	if h.Book.Scroll[side]+shown < total {
		bookLiteral.Draw(dst, "v", r.X+r.W-6, r.Y+r.H-4, 1, bookSoft)
	}
}
