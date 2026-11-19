package hud

import (
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	kmapParchment = 64
	glowPeriod    = 2.6
	glowRest      = 0.45
	glowSwing     = 0.3
	vignetteSteps = 10
	FogTextW      = 200
	FogTextTop    = 34
)

var (
	mapInk      = color.RGBA{92, 64, 40, 255}
	mapRedInk   = color.RGBA{150, 44, 32, 255}
	fogColor    = color.RGBA{118, 108, 92, 255}
	plaqueBlue  = color.RGBA{74, 137, 156, 255}
	fogMessage  = "UNCHARTED"
	FogAsksWord = " asks: "
	fogGoalWord = ", its goal: "
	eraNames    = []struct{ story, k8sEnv, k8sStrip, tale, title, k8sPlain string }{
		{story: "DARK AGE", k8sEnv: "DEV", k8sStrip: "DEV", tale: "DARK", title: "the Dark Age", k8sPlain: "Dev"},
		{story: "FEUDAL AGE", k8sEnv: "STAGING", k8sStrip: "STAGING", tale: "FEUDAL", title: "the Feudal Age", k8sPlain: "Staging"},
		{story: "CASTLE AGE", k8sEnv: "UAT", k8sStrip: "UAT", tale: "CASTLE", title: "the Castle Age", k8sPlain: "UAT"},
		{story: "AGE OF EMPIRE", k8sEnv: "PRODUCTION", k8sStrip: "PROD", tale: "EMPIRE", title: "the Age of Empire", k8sPlain: "Production"},
	}
)

type pen struct{ x, y int }

func (p pen) at(r ui.Rect) ui.Rect { return r.Offset(p.x, p.y) }

func (h *HUD) viewPen() pen {
	return pen{x: KnowledgeView.X - int(math.Round(h.Kmap.Scroll.Pos)), y: KnowledgeView.Y}
}

func clipTo(dst *ebiten.Image, r ui.Rect) *ebiten.Image {
	return dst.SubImage(image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)).(*ebiten.Image)
}

func mixRGBA(a, b color.RGBA, t float64) color.RGBA {
	m := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t) }
	return color.RGBA{m(a.R, b.R), m(a.G, b.G), m(a.B, b.B), 255}
}

func (h *HUD) drawGrimoire(dst *ebiten.Image, f Frame) {
	l := h.Knowledge(f)
	h.tileParchment(dst, KmapInner)
	canvas := clipTo(dst, KnowledgeView)
	h.drawMapCanvas(canvas, l, f)
	h.drawPlaqueMarks(canvas, l, f)
	h.drawFogText(canvas, l, f)
	h.drawVignetteKMap(dst)
	h.drawBannerColumn(dst, l)
	h.drawRibbons(dst, l, f)
	h.drawEraStrip(dst, l, f)
	h.drawKMapFrame(dst)
	h.drawCard(dst, f, l)
	h.drawButton(dst, closeButton(KnowledgeBox))
}

func (h *HUD) tileParchment(dst *ebiten.Image, area ui.Rect) {
	if !h.Lib.Has(gfx.KMapParchmentName()) {
		ui.Fill(dst, area, ui.Palette.Parchment)
		return
	}
	dst = clipTo(dst, area)
	for y := area.Y; y < area.Y+area.H; y += kmapParchment {
		for x := area.X; x < area.X+area.W; x += kmapParchment {
			h.Lib.Draw(dst, gfx.KMapParchmentName(), float64(x), float64(y), 1, render.SpriteOptions{})
		}
	}
}

func eraSpan(l KMapLayout, era int) ui.Rect {
	e := l.Eras[era]
	return ui.Rect{X: e.Start * KMapTile, Y: 0, W: e.Size * KMapTile, H: l.Height * KMapTile}
}

func (h *HUD) eraRect(l KMapLayout, era int) ui.Rect {
	return h.ScreenOf(eraSpan(l, era))
}

func (h *HUD) drawEras(dst *ebiten.Image, l KMapLayout, f Frame, p pen) {
	current := eraIndex(f.State.Age)
	for i := range l.Eras {
		r := p.at(eraSpan(l, i))
		if i == current {
			ui.Fill(dst, r, ui.Tint(ui.Palette.GoldBright, 22))
		}
		if i > 0 {
			for y := r.Y; y < r.Y+r.H; y += 6 {
				ui.Fill(dst, ui.Rect{X: r.X, Y: y, W: 1, H: 3}, ui.Tint(mapInk, 150))
			}
		}
	}
}

func (h *HUD) drawRowLines(dst *ebiten.Image, l KMapLayout, x0, x1, y0 int) {
	for _, row := range l.Rows[1:] {
		y := y0 + row.Start*KMapTile
		for x := x0 + 2; x < x1; x += 4 {
			ui.Fill(dst, ui.Rect{X: x, Y: y, W: 2, H: 1}, ui.Tint(mapInk, 110))
		}
	}
}

func (h *HUD) drawRoads(dst *ebiten.Image, roads []KMapRoadTile, p pen) {
	for _, t := range roads {
		r := p.at(ui.Rect{X: t.X * KMapTile, Y: t.Y * KMapTile, W: KMapTile, H: KMapTile})
		if !h.Lib.Draw(dst, gfx.KMapRoadName(t.State, t.N, t.E, t.S, t.W), float64(r.X), float64(r.Y), 1, render.SpriteOptions{}) {
			ui.Fill(dst, ui.Rect{X: r.X + 5, Y: r.Y + 5, W: 6, H: 6}, mapInk)
		}
	}
}

func plaqueState(p KMapPlaque) string {
	switch {
	case p.Fog:
		return "locked"
	case p.Page.State == "unlocked":
		return "learned"
	case p.Page.State == "available":
		return "available"
	}
	return "locked"
}

func plaqueColor(state string) color.RGBA {
	c := gfx.PlaqueTextColor(state)
	return color.RGBA{c[0], c[1], c[2], 255}
}

func (h *HUD) drawPlaque(dst *ebiten.Image, p KMapPlaque, r ui.Rect, trueNames bool) {
	state := plaqueState(p)
	ui.Fill(dst, r.Offset(2, 3), ui.Tint(mapInk, 80))
	if !h.Lib.NineSlice(dst, gfx.PlaqueName(state), r) {
		ui.Fill(dst, r, ui.Palette.Ink)
		ui.Stroke(dst, r, ui.Palette.Gold)
	}
	h.drawPlaqueIcon(dst, p, r, state)
	line1, line2 := PlaqueLines(p.Page, p.Fog, trueNames)
	c := plaqueColor(state)
	h.Font.DrawShadow(dst, line1, r.X+plaqueTextX, r.Y+plaqueLine1Y, 1, c)
	h.Font.Draw(dst, line2, r.X+plaqueTextX, r.Y+plaqueLine2Y, 1, mixRGBA(c, plaqueBlue, 0.35))
	h.drawRankPips(dst, p.Page, r)
}

func (h *HUD) drawPlaqueIcon(dst *ebiten.Image, p KMapPlaque, r ui.Rect, state string) {
	h.Lib.Draw(dst, gfx.KMapMedallionName(state), float64(r.X+5), float64(r.Y+5), 1, render.SpriteOptions{})
	if p.Fog {
		ui.Fill(dst, r.Inset(3), ui.Tint(fogColor, 120))
		h.Font.DrawShadow(dst, "?", r.X+15, r.Y+13, 2, ui.Palette.Cream)
		return
	}
	alpha := float32(1)
	if state == "locked" {
		alpha = 0.6
	}
	h.Lib.Draw(dst, gfx.PageIconName(p.Page.ID), float64(r.X+20), float64(r.Y+20), 1, render.SpriteOptions{Alpha: alpha})
}

func (h *HUD) drawRankPips(dst *ebiten.Image, pg view.GrimoirePage, r ui.Rect) {
	if pg.MaxRank < 2 {
		return
	}
	top := r.Y + r.H - rankPipBottom - (pg.MaxRank*rankPipStep - rankPipGap)
	for i := 0; i < pg.MaxRank; i++ {
		pip := ui.Rect{X: r.X + r.W - rankPipRight, Y: top + i*rankPipStep, W: rankPipSize, H: rankPipSize}
		edge, fill := ui.Palette.Gold, ui.Palette.Ink
		if i < pg.Rank {
			edge, fill = ui.Palette.GoldBright, ui.Palette.GoldBright
		}
		ui.Fill(dst, pip.Inset(-1), ui.Palette.Ink)
		ui.Fill(dst, pip, edge)
		ui.Fill(dst, pip.Inset(1), fill)
	}
}

const (
	rankPipSize   = 6
	rankPipGap    = 3
	rankPipStep   = rankPipSize + rankPipGap
	rankPipRight  = 11
	rankPipBottom = 5
)

func (h *HUD) drawPlaqueMarks(dst *ebiten.Image, l KMapLayout, f Frame) {
	p := h.viewPen()
	for _, pl := range l.Plaques {
		r := p.at(pl.Rect())
		if r.X+r.W < KnowledgeView.X-8 || r.X > KnowledgeView.X+KnowledgeView.W+8 {
			continue
		}
		if plaqueState(pl) == "learned" {
			h.drawGlow(dst, r, h.glowAlpha(pl))
		}
		h.drawPlaqueFocus(dst, pl, r, affordable(pl, f))
		h.drawPageCall(dst, pl, r)
	}
}

func (h *HUD) drawPlaqueFocus(dst *ebiten.Image, p KMapPlaque, r ui.Rect, ready bool) {
	switch {
	case p.Page.ID == h.Grimoire:
		ui.Stroke(dst, r.Inset(-1), ui.Palette.GoldBright)
		ui.Stroke(dst, r.Inset(-2), ui.Palette.GoldBright)
	case p.Page.ID == h.Kmap.Focus:
		ui.Stroke(dst, r.Inset(-1), ui.Palette.Cream)
	case ready:
		ui.Stroke(dst, r.Inset(-1), ui.Tint(ui.Palette.Cream, 200))
	}
}

func affordable(p KMapPlaque, f Frame) bool {
	return learnable(p) && f.State.Gold >= p.Page.Cost.Gold && f.State.Wisdom >= p.Page.Cost.Wisdom
}

func (h *HUD) glowAlpha(p KMapPlaque) float64 {
	if h.Reduce {
		return glowRest
	}
	phase := float64(p.Col*7+p.Row*3) * 0.37
	return glowRest + glowSwing*math.Sin(h.Now*2*math.Pi/glowPeriod+phase)
}

func (h *HUD) drawGlow(dst *ebiten.Image, r ui.Rect, a float64) {
	for i := 1; i <= 4; i++ {
		alpha := uint8(math.Max(0, math.Min(255, a*float64(230-i*48))))
		ui.Stroke(dst, r.Inset(-i), ui.Tint(ui.Palette.GoldBright, alpha))
	}
}

func (h *HUD) drawFog(dst *ebiten.Image, l KMapLayout, f Frame, p pen) {
	current := eraIndex(f.State.Age)
	for i := range l.Eras {
		if i <= current {
			continue
		}
		r := p.at(eraSpan(l, i))
		ui.Fill(dst, r, ui.Tint(fogColor, 70))
		h.drawMist(dst, r, i)
	}
}

func (h *HUD) drawMist(dst *ebiten.Image, r ui.Rect, era int) {
	for k := 0; k < 14; k++ {
		hsh := mapHash(era*31+k, k*17+era)
		w := 30 + hsh%70
		x := r.X + hsh/7%max(r.W-w, 1)
		y := r.Y + hsh/13%max(r.H-4, 1)
		ui.Fill(dst, ui.Rect{X: x, Y: y, W: w, H: 2}, ui.Tint(ui.Palette.Parchment, 70))
	}
}

func (h *HUD) drawFogText(dst *ebiten.Image, l KMapLayout, f Frame) {
	for i := range l.Eras {
		if l.Empty(i) {
			h.drawUncharted(dst, l, f, i)
		}
	}
}

func (h *HUD) drawUncharted(dst *ebiten.Image, l KMapLayout, f Frame, era int) {
	zone := h.ScreenOf(fogZone(l, era))
	lines := []string{}
	ages := f.State.Road.Ages
	if era < len(ages) && ages[era].Goal != "" {
		lines = h.UnchartedLines(ages[era], era, zone, f.TrueNames)
	}
	block, _ := h.Font.Measure(fogMessage, 2)
	for _, line := range lines {
		lw, _ := h.Font.Measure(line, 1)
		block = max(block, lw)
	}
	cx := stickyCentre(zone, block)
	w, _ := h.Font.Measure(fogMessage, 2)
	h.Font.Draw(dst, fogMessage, cx-w/2, zone.Y+8, 2, ui.Tint(mapInk, 200))
	for i, line := range lines {
		lw, _ := h.Font.Measure(line, 1)
		h.Font.Draw(dst, line, cx-lw/2, zone.Y+FogTextTop+i*ui.LineHeight, 1, ui.Tint(mapInk, 220))
	}
}

func (h *HUD) UnchartedLines(a view.AgeStep, era int, zone ui.Rect, trueNames bool) []string {
	rows := max((zone.H-FogTextTop)/ui.LineHeight, 1)
	return h.Font.WrapLines(EraText(a, era, trueNames), min(zone.W-8, FogTextW), 1, rows)
}

func EraText(a view.AgeStep, era int, trueNames bool) string {
	if a.Asks != "" {
		return EraAsker(era, trueNames) + FogAsksWord + a.Asks
	}
	return EraAsker(era, trueNames) + fogGoalWord + a.Goal
}

func EraAsker(era int, trueNames bool) string {
	if trueNames {
		return eraNames[era].k8sPlain
	}
	return CapitalFirst(eraNames[era].title)
}

func stickyCentre(zone ui.Rect, width int) int {
	centre := zone.X + zone.W/2
	if seen, ok := clipRect(zone, KnowledgeView); ok {
		centre = seen.X + seen.W/2
	}
	return min(max(centre, zone.X+width/2+4), zone.X+zone.W-width/2-4)
}

func (h *HUD) drawVignetteKMap(dst *ebiten.Image) {
	for i := 0; i < vignetteSteps; i++ {
		alpha := uint8(70 - i*7)
		ui.Stroke(dst, KnowledgeView.Inset(i), ui.Tint(mapInk, alpha))
	}
}

func (h *HUD) drawKMapFrame(dst *ebiten.Image) {
	if !h.Lib.NineSlice(dst, "kmap-frame", KnowledgeBox) {
		ui.Stroke(dst, KnowledgeBox, ui.Palette.Gold)
		ui.Stroke(dst, KnowledgeBox.Inset(3), mapInk)
	}
}

func Upper(s string) string { return strings.ToUpper(s) }

func CapitalFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
