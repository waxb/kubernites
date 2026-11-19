package hud

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	bannerW      = 32
	bannerH      = 48
	bannerShadow = 6
	stripLabelY  = 2
	stripBarsY   = 12
)

var stripBar = map[string]color.RGBA{
	"learned": ui.Palette.GoldBright, "available": ui.Palette.Cream, "locked": ui.Palette.Muted,
}

func (h *HUD) drawBannerColumn(dst *ebiten.Image, l KMapLayout) {
	col := kmapBannerCol
	ui.Fill(dst, col, ui.Tint(mapInk, 26))
	h.drawRowLines(dst, l, col.X, col.X+col.W, KnowledgeView.Y)
	for i, row := range l.Rows {
		branch := view.KnowledgeBranches[i]
		y := KnowledgeView.Y + row.Start*KMapTile
		by := y + (row.Size*KMapTile-bannerH-ui.LineHeight)/2
		bx := col.X + (col.W-bannerW)/2
		if !h.Lib.Draw(dst, gfx.KMapBannerName(branch), float64(bx), float64(by), 1, render.SpriteOptions{}) {
			ui.Fill(dst, ui.Rect{X: bx, Y: by, W: bannerW, H: bannerH}, ui.Palette.RedDark)
		}
		name := Upper(branch)
		w, _ := h.Font.Measure(name, 1)
		h.Font.Draw(dst, name, col.X+(col.W-w)/2, by+bannerH+2, 1, mapInk)
	}
	edge := col.X + col.W
	ui.Fill(dst, ui.Rect{X: edge - 3, Y: KmapInner.Y, W: 1, H: KmapInner.H}, mapInk)
	ui.Fill(dst, ui.Rect{X: edge - 1, Y: KmapInner.Y, W: 1, H: KmapInner.H}, mapInk)
	for i := 0; i < bannerShadow; i++ {
		ui.Fill(dst, ui.Rect{X: edge + i, Y: KnowledgeView.Y, W: 1, H: KnowledgeView.H}, ui.Tint(mapInk, uint8(60-i*9)))
	}
	title := ui.Rect{X: KmapInner.X, Y: KmapInner.Y, W: kmapBannerW - 3, H: kmapRibbonH}
	w, _ := h.Font.Measure(kmapTitle, 1)
	h.Font.Draw(dst, kmapTitle, title.X+(title.W-w)/2, title.Y+5, 1, mapRedInk)
}

func EraLabel(i int, trueNames bool) string {
	if trueNames {
		return eraNames[i].k8sEnv
	}
	return eraNames[i].story
}

func EraStrip(i int, trueNames bool) string {
	if trueNames {
		return eraNames[i].k8sStrip
	}
	return eraNames[i].tale
}

func eraInk(i, current int) color.RGBA {
	switch {
	case i == current:
		return mapRedInk
	case i > current:
		return ui.Tint(mapInk, 150)
	}
	return mapInk
}

func (h *HUD) drawRibbons(dst *ebiten.Image, l KMapLayout, f Frame) {
	clip := clipTo(dst, kmapRibbonRow)
	current := eraIndex(f.State.Age)
	for i := range l.Eras {
		span := h.eraRect(l, i)
		r := ui.Rect{X: span.X + 6, Y: kmapRibbonRow.Y, W: span.W - 12, H: kmapRibbonH}
		if !h.Lib.NineSlice(clip, "kmap-ribbon", r) {
			ui.Fill(clip, r, ui.Palette.Parchment)
			ui.Stroke(clip, r, mapInk)
		}
		seen, ok := clipRect(r, kmapRibbonRow)
		if !ok {
			continue
		}
		label := EraLabel(i, f.TrueNames)
		if i == current {
			label = "* " + label + " *"
		}
		w, _ := h.Font.Measure(label, 1)
		x := min(max(seen.X+(seen.W-w)/2, r.X+10), r.X+r.W-10-w)
		h.Font.Draw(clip, label, x, r.Y+5, 1, eraInk(i, current))
	}
}

func StripSegments(l KMapLayout) []ui.Rect {
	out := make([]ui.Rect, 0, len(l.Eras))
	total := max(l.Width, 1)
	s := KnowledgeStrip
	for _, e := range l.Eras {
		x0 := s.X + e.Start*s.W/total
		x1 := s.X + (e.Start+e.Size)*s.W/total
		out = append(out, ui.Rect{X: x0, Y: s.Y, W: x1 - x0, H: s.H})
	}
	return out
}

func (h *HUD) drawEraStrip(dst *ebiten.Image, l KMapLayout, f Frame) {
	s := KnowledgeStrip
	ui.Fill(dst, s, ui.Palette.ParchmentDark)
	current := eraIndex(f.State.Age)
	for i, r := range StripSegments(l) {
		fill := ui.Tint(ui.Palette.Parchment, 150)
		if i == current {
			fill = ui.Tint(ui.Palette.GoldBright, 170)
		}
		ui.Fill(dst, r.Inset(1), fill)
		tag := EraStrip(i, f.TrueNames)
		w, _ := h.Font.Measure(tag, 1)
		h.Font.Draw(dst, tag, r.X+(r.W-w)/2, r.Y+stripLabelY, 1, eraInk(i, current))
		if h.Hovering && h.Hover.ID == KmapPrefix+KmapEraPrefix+view.KnowledgeEras[i] {
			ui.Stroke(dst, r, ui.Palette.Cream)
		}
	}
	h.drawStripBars(dst, l)
	ui.Stroke(dst, s, mapInk)
	h.drawStripWindow(dst, l)
	text := learnedText(f.State.Grimoire)
	w, _ := h.Font.Measure(text, 1)
	h.Font.Draw(dst, text, KmapInner.X+(kmapBannerW-3-w)/2, s.Y+6, 1, mapInk)
}

func (h *HUD) drawStripBars(dst *ebiten.Image, l KMapLayout) {
	s := KnowledgeStrip
	total := max(l.Width, 1)
	rows := max(l.Height, 1)
	for _, p := range l.Plaques {
		x := s.X + p.Col*s.W/total
		w := max(PlaqueTiles*s.W/total, 2)
		y := s.Y + stripBarsY + p.Row*(s.H-stripBarsY-2)/rows
		ui.Fill(dst, ui.Rect{X: x, Y: y, W: w, H: 2}, stripBar[plaqueState(p)])
	}
}

func (h *HUD) drawStripWindow(dst *ebiten.Image, l KMapLayout) {
	s := KnowledgeStrip
	width := float64(max(l.Width*KMapTile, 1))
	x := s.X + int(h.Kmap.Scroll.Pos*float64(s.W)/width)
	w := min(int(float64(KnowledgeView.W)*float64(s.W)/width), s.W)
	win := ui.Rect{X: min(x, s.X+s.W-w), Y: s.Y - 1, W: w, H: s.H + 2}
	ui.Stroke(dst, win, mapRedInk)
	ui.Stroke(dst, win.Inset(1), ui.Tint(mapRedInk, 120))
}
