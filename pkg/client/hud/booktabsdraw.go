package hud

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/art"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

var (
	tabLeather    = color.RGBA{176, 98, 58, 255}
	tabLeatherLit = color.RGBA{198, 122, 76, 255}
	tabEmboss     = color.RGBA{132, 70, 42, 255}
	tabTan        = color.RGBA{196, 150, 112, 255}
	tabTanLit     = color.RGBA{214, 176, 140, 255}
	tabTanInk     = color.RGBA{120, 80, 54, 255}
	tabMuted      = color.RGBA{150, 112, 84, 255}
)

var shelfIcons = map[string][]string{
	"buildings": {"...##...", "..####..", ".######.", "########", ".#....#.", ".#.##.#.", ".#.##.#."},
	"workers":   {"...##...", "...##...", "..####..", ".#.##.#.", "...##...", "..#..#..", "..#..#.."},
	"resources": {"...##...", "..#..#..", ".######.", "##.##.##", "########", "########", ".######."},
	"powers":    {"....##..", "...##...", "..##....", ".#####..", "...##...", "..##....", ".##....."},
	"troubles":  {"...#....", "..##....", "..###.#.", ".#####..", "##.###.#", "##..####", ".######."},
	"realm":     {".######.", ".#####..", ".######.", ".#......", ".#......", ".#......", "###....."},
}

var controlIcons = map[string][]string{
	BookBack:   {"..#.....", ".#......", "########", ".#......", "..#....."},
	BookHome:   {"########", "........", "########", "........", "########"},
	BookSearch: {".###....", "#...#...", "#...#...", ".###....", "....##..", ".....##."},
	BookClose:  {"#....#", ".#..#.", "..##..", "..##..", ".#..#.", "#....#"},
}

func fatBits(dst *ebiten.Image, rows []string, x, y int, clr color.RGBA) {
	for j, row := range rows {
		for i, c := range row {
			if c == '#' {
				ui.Fill(dst, ui.Rect{X: x + i*bookPx, Y: y + j*bookPx, W: bookPx, H: bookPx}, clr)
			}
		}
	}
}

func bitsSize(rows []string) (int, int) {
	if len(rows) == 0 {
		return 0, 0
	}
	return len(rows[0]) * bookPx, len(rows) * bookPx
}

func drawTabShape(dst *ebiten.Image, r ui.Rect, fill, lit color.RGBA) {
	p := bookPx
	ui.Fill(dst, ui.Rect{X: r.X + p, Y: r.Y, W: r.W - 2*p, H: r.H}, bookOutline)
	ui.Fill(dst, ui.Rect{X: r.X, Y: r.Y + p, W: r.W, H: r.H - p}, bookOutline)
	ui.Fill(dst, ui.Rect{X: r.X + p, Y: r.Y + p, W: r.W - 2*p, H: r.H - p}, fill)
	ui.Fill(dst, ui.Rect{X: r.X + 2*p, Y: r.Y + p, W: r.W - 4*p, H: p}, lit)
}

func (h *HUD) drawBookTab(dst *ebiten.Image, b Button) {
	hot := h.Hovering && h.Hover.ID == b.ID && !b.Disabled
	r := b.Rect
	if b.Selected || hot {
		r = r.Offset(0, -bookPx)
	}
	switch {
	case strings.HasPrefix(b.ID, BookShelf+ShelfCategory):
		fill, ink := tabLeather, tabEmboss
		if b.Disabled {
			fill, ink = tabMuted, tone(tabMuted, .85, 0)
		}
		drawTabShape(dst, r, fill, tone(fill, 1.15, 10))
		drawTabIcon(dst, r, shelfIcons[strings.TrimPrefix(b.ID, BookShelf+ShelfCategory)], ink)
	case strings.HasPrefix(b.ID, BookShelf+ShelfBranch):
		h.drawFlag(dst, b, r)
	case strings.HasPrefix(b.ID, BookRibbon):
		drawTabShape(dst, r, tabTan, tabTanLit)
		h.drawRibbonTab(dst, r, strings.TrimPrefix(b.ID, BookRibbon))
	default:
		ink := tabTanInk
		if b.Disabled {
			ink = tabMuted
		}
		drawTabShape(dst, r, tabTan, tabTanLit)
		drawTabIcon(dst, r, controlIcons[b.ID], ink)
	}
}

func drawTabIcon(dst *ebiten.Image, r ui.Rect, rows []string, ink color.RGBA) {
	w, h := bitsSize(rows)
	top := r.H - artTabTuck*bookPx
	fatBits(dst, rows, r.X+(r.W-w)/2, r.Y+(top-h)/2+bookPx/2, ink)
}

func (h *HUD) drawRibbonTab(dst *ebiten.Image, r ui.Rect, id string) {
	if icon, ok := pageIcons[id]; ok {
		bookLiteral.Draw(dst, icon, r.X+(r.W-2*ui.Advance)/2, r.Y+5, 1, tabTanInk)
		return
	}
	bookLiteral.Draw(dst, pageInitial(id), r.X+(r.W-ui.GlyphWidth*2)/2, r.Y+4, 2, tabTanInk)
}

func pageInitial(id string) string {
	p, ok := view.FindPage(id)
	if !ok || p.Title == "" {
		return "?"
	}
	title := []rune(strings.ToUpper(strings.TrimPrefix(ui.Name(p.Title), "The ")))
	return string(title[0])
}

func (h *HUD) drawFlag(dst *ebiten.Image, b Button, r ui.Rect) {
	if b.Selected || h.Hovering && h.Hover.ID == b.ID {
		r = ui.Rect{X: b.Rect.X + 2*bookPx, Y: b.Rect.Y, W: b.Rect.W, H: b.Rect.H}
	}
	alpha := float32(1)
	if b.Disabled {
		alpha = .55
	}
	name := art.KMapFlagName(strings.TrimPrefix(b.ID, BookShelf+ShelfBranch))
	if h.Lib != nil && h.Lib.Has(name) {
		img, reg, _ := h.Lib.Lookup(name)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(r.X-reg.AnchorX), float64(r.Y-reg.AnchorY))
		op.ColorScale.ScaleAlpha(alpha)
		dst.DrawImage(img.SubImage(reg.Rect()).(*ebiten.Image), op)
		return
	}
	ui.Fill(dst, r, ui.Palette.RedDark)
}
