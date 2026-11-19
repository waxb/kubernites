package hud

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/ui"
)

type cachedImage struct {
	img *ebiten.Image
	key string
}

func (c *cachedImage) get(key string, w, h int, paint func(*ebiten.Image)) *ebiten.Image {
	if c.img != nil && c.key == key {
		if ui.Recording() {
			c.img.Clear()
			paint(c.img)
		}
		return c.img
	}
	if c.img != nil {
		c.img.Deallocate()
	}
	c.img, c.key = ebiten.NewImage(max(w, 1), max(h, 1)), key
	paint(c.img)
	return c.img
}

func (h *HUD) canvasKey(l KMapLayout, f Frame) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%p|%s|%v|%d|%d|", h.Lib, f.State.Age, f.TrueNames, l.Width, l.Height)
	for _, p := range l.Plaques {
		pg := p.Page
		fmt.Fprintf(&b, "%s:%s:%s:%d:%d:%v:%d:%d:%s;", pg.ID, pg.Name, pg.State, pg.Rank, pg.MaxRank, p.Fog, p.Col, p.Row, strings.Join(pg.Requires, ","))
	}
	return b.String()
}

func (h *HUD) drawMapCanvas(view *ebiten.Image, l KMapLayout, f Frame) {
	w, ht := l.Width*KMapTile, l.Height*KMapTile
	img := h.Kmap.canvas.get(h.canvasKey(l, f), w, ht, func(dst *ebiten.Image) { h.paintCanvas(dst, l, f) })
	p := h.viewPen()
	if w-int(h.Kmap.Scroll.Pos) < KnowledgeView.W {
		h.tileParchment(view, KnowledgeView)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(p.x), float64(p.y))
	view.DrawImage(img, op)
}

func (h *HUD) paintCanvas(dst *ebiten.Image, l KMapLayout, f Frame) {
	origin := pen{}
	full := ui.Rect{W: l.Width * KMapTile, H: l.Height * KMapTile}
	roads := KnowledgeRoads(l)
	h.tileParchment(dst, full)
	h.drawEras(dst, l, f, origin)
	h.drawDecor(dst, l, roads, origin)
	h.drawRowLines(dst, l, 0, full.W, 0)
	h.drawRoads(dst, roads, origin)
	for _, p := range l.Plaques {
		h.drawPlaque(dst, p, origin.at(p.Rect()), f.TrueNames)
	}
	h.drawFog(dst, l, f, origin)
}

func cardKey(p KMapPlaque, card ui.Rect, lines [][]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s|%v|%s|%d|%d|", p.Page.ID, p.Fog, plaqueState(p), card.W, card.H)
	for _, ls := range lines {
		b.WriteString(strings.Join(ls, "\n"))
		b.WriteByte(0)
	}
	return b.String()
}
