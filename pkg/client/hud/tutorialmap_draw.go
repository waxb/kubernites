package hud

import (
	"fmt"
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
	mapRingR       = 18
	mapWaystoneH   = 12
	mapIconFaded   = 0.45
	mapIconLater   = 0.3
	mapFogAlpha    = 70
	mapHere        = "YOU ARE HERE"
	mapDone        = "[x] DONE"
	mapLocked      = "LOCKED"
	mapLater       = "COMING LATER"
	mapRibbonTitle = "LEARN KUBERNETES"
)

var mapSpriteIcons = map[string]string{"hut": "icon/pod", "ward": "icon/probe"}

var mapDecor = []struct {
	kind string
	x, y float64
}{
	{"compass", 53.3, 22.6}, {"mountains", 54, 11.5}, {"forest", 0.2, 17.5}, {"hills", 0.2, 23.5},
	{"waves", 0.4, 11}, {"waves", 1, 12.6}, {"waves", 0.3, 6.5},
}

func (m *TutorialMap) Draw(dst *ebiten.Image, lib *gfx.Library, hover string) {
	tileMapParchment(dst, lib, KmapInner)
	m.drawDecor(dst, lib)
	for _, t := range m.Roads() {
		x, y := mapOrigin[0]+t.X*KMapTile, mapOrigin[1]+t.Y*KMapTile
		if !lib.Draw(dst, gfx.KMapRoadName(t.State, t.N, t.E, t.S, t.W), float64(x), float64(y), 1, render.SpriteOptions{}) {
			ui.Fill(dst, ui.Rect{X: x + 5, Y: y + 5, W: 6, H: 6}, mapInk)
		}
	}
	m.drawHarbor(dst, lib)
	m.drawFog(dst)
	m.drawWaystones(dst)
	for _, s := range m.Stops() {
		m.drawStop(dst, lib, s)
	}
	for i := 0; i < vignetteSteps; i++ {
		ui.Stroke(dst, KmapInner.Inset(i), ui.Tint(mapInk, uint8(70-i*7)))
	}
	if !lib.NineSlice(dst, "kmap-frame", KnowledgeBox) {
		ui.Stroke(dst, KnowledgeBox, ui.Palette.Gold)
		ui.Stroke(dst, KnowledgeBox.Inset(3), mapInk)
	}
	m.drawRibbon(dst, hover)
	if m.card {
		m.drawCard(dst, lib, hover)
	}
}

func tileMapParchment(dst *ebiten.Image, lib *gfx.Library, area ui.Rect) {
	if !lib.Has(gfx.KMapParchmentName()) {
		ui.Fill(dst, area, ui.Palette.Parchment)
		return
	}
	canvas := clipTo(dst, area)
	for y := area.Y; y < area.Y+area.H; y += kmapParchment {
		for x := area.X; x < area.X+area.W; x += kmapParchment {
			lib.Draw(canvas, gfx.KMapParchmentName(), float64(x), float64(y), 1, render.SpriteOptions{})
		}
	}
}

func (m *TutorialMap) drawDecor(dst *ebiten.Image, lib *gfx.Library) {
	for _, d := range mapDecor {
		x := float64(mapOrigin[0]) + d.x*KMapTile
		y := float64(mapOrigin[1]) + d.y*KMapTile
		lib.Draw(dst, gfx.KMapDecorName(d.kind), x, y, 1, render.SpriteOptions{Alpha: 0.8})
	}
}

var mapHarborWaves = [][2]int{{-22, -14}, {-8, -20}, {-16, -6}}

func (m *TutorialMap) drawHarbor(dst *ebiten.Image, lib *gfx.Library) {
	cx, cy := tileCenter(MapHarborX, mapTopRow)
	for _, w := range mapHarborWaves {
		lib.Draw(dst, gfx.KMapDecorName("waves"), float64(cx+w[0]), float64(cy+w[1]), 1, render.SpriteOptions{})
	}
}

func rowBand(row int) ui.Rect {
	_, cy := tileCenter(0, StopRow(row))
	return ui.Rect{X: mapOrigin[0], Y: cy - 2*KMapTile - 4, W: mapTilesW * KMapTile, H: mapRowGap * KMapTile}
}

func (m *TutorialMap) drawFog(dst *ebiten.Image) {
	stops := m.Stops()
	first := -1
	for i, s := range stops {
		if s.State == StopLater {
			first = i
			break
		}
	}
	if first < 0 {
		return
	}
	row := first / mapStopsPerRow
	cx, _ := tileCenter(stops[first].X, stops[first].Y)
	band := rowBand(row)
	half := mapStopGap * KMapTile / 2
	if row%2 == 0 {
		band.W -= cx - half - band.X
		band.X = cx - half
	} else {
		band.W = cx + half - band.X
	}
	m.fogPatch(dst, band, row)
	for r := row + 1; r*mapStopsPerRow < len(stops); r++ {
		m.fogPatch(dst, rowBand(r), r)
	}
}

func (m *TutorialMap) fogPatch(dst *ebiten.Image, r ui.Rect, seed int) {
	ui.Fill(dst, r, ui.Tint(fogColor, mapFogAlpha))
	for k := 0; k < 18; k++ {
		hsh := mapHash(seed*37+k, k*19+seed)
		w := 30 + hsh%80
		x := r.X + hsh/7%max(r.W-w, 1)
		y := r.Y + hsh/13%max(r.H-4, 1)
		ui.Fill(dst, ui.Rect{X: x, Y: y, W: w, H: 2}, ui.Tint(ui.Palette.Parchment, 70))
	}
}

func (m *TutorialMap) drawWaystones(dst *ebiten.Image) {
	stops := m.Stops()
	for i, s := range stops {
		if i > 0 && stops[i-1].Chapter.Part == s.Chapter.Part {
			continue
		}
		dir, off := 1, mapStopGap/2
		if (i/mapStopsPerRow)%2 == 1 {
			dir = -1
		}
		if i%mapStopsPerRow == 0 {
			off = 3
		}
		cx, cy := tileCenter(s.X-dir*off, s.Y)
		post := ui.Rect{X: cx - 1, Y: cy - mapWaystoneH - 10, W: 3, H: mapWaystoneH + 4}
		ui.Fill(dst, post, mapInk)
		label := romanPart(s.Chapter.Part)
		w := Literal.Width(label, 1) + 6
		sign := ui.Rect{X: cx - w/2, Y: post.Y - 10, W: w, H: 10}
		ui.Fill(dst, sign, ui.Palette.ParchmentDark)
		ui.Stroke(dst, sign, mapInk)
		Literal.Draw(dst, label, sign.X+3, sign.Y+1, 1, mapInk)
	}
}

func romanPart(part int) string {
	numerals := []string{"", "I", "II", "III", "IV", "V", "VI"}
	if part < 1 || part >= len(numerals) {
		return ""
	}
	return numerals[part]
}

var medalStates = map[string]string{StopDone: "learned", StopCurrent: "available", StopLocked: "locked", StopLater: "locked"}

func (m *TutorialMap) drawStop(dst *ebiten.Image, lib *gfx.Library, s MapStop) {
	r := MedalRect(s)
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	if s.State == StopCurrent {
		alpha := uint8(110 + 145*pulse(m.now, m.Reduce))
		ring(dst, cx, cy, mapRingR, ui.Tint(ui.Palette.GoldBright, alpha))
		ring(dst, cx, cy, mapRingR+1, ui.Tint(ui.Palette.GoldBright, alpha/2))
	}
	if !lib.Draw(dst, gfx.KMapMedallionName(medalStates[s.State]), float64(r.X), float64(r.Y), 1, render.SpriteOptions{}) {
		ui.Fill(dst, r, ui.Palette.ParchmentDark)
	}
	alpha := float32(1)
	switch s.State {
	case StopLocked:
		alpha = mapIconFaded
	case StopLater:
		alpha = mapIconLater
	}
	drawChapterIcon(dst, lib, s.Chapter.Icon, cx, cy, alpha)
	num := fmt.Sprint(s.Chapter.Number)
	plaque := ui.Rect{X: r.X - 5, Y: r.Y - 4, W: Literal.Width(num, 1) + 5, H: 10}
	ui.Fill(dst, plaque, ui.Palette.ParchmentDark)
	ui.Stroke(dst, plaque, mapInk)
	Literal.Draw(dst, num, plaque.X+3, plaque.Y+1, 1, mapInk)
	ink := mapInk
	if s.State == StopLocked || s.State == StopLater {
		ink = ui.Tint(mapInk, 170)
	}
	y := r.Y + r.H + 3
	for _, line := range StopTitle(s) {
		lw := Literal.Width(line, 1)
		Literal.Draw(dst, line, cx-lw/2, y, 1, ink)
		y += ui.LineHeight
	}
	m.drawMark(dst, s, cx, y)
	if s.Chapter.Number == m.focus {
		ui.Stroke(dst, StopRect(s).Inset(-1), ui.Palette.Gold)
	}
}

func (m *TutorialMap) drawMark(dst *ebiten.Image, s MapStop, cx, y int) {
	mark, clr := mapLater, ui.Tint(mapInk, 150)
	switch s.State {
	case StopDone:
		mark, clr = mapDone, ui.Palette.GreenDark
	case StopCurrent:
		mark, clr = mapHere, mapRedInk
	case StopLocked:
		mark, clr = mapLocked, ui.Tint(mapInk, 150)
	}
	w := Literal.Width(mark, 1)
	x := cx - w/2
	if s.State == StopCurrent {
		ui.Fill(dst, ui.Rect{X: x - 9, Y: y - 1, W: 1, H: 10}, mapInk)
		ui.Fill(dst, ui.Rect{X: x - 8, Y: y - 1, W: 5, H: 4}, mapRedInk)
	}
	Literal.Draw(dst, mark, x, y, 1, clr)
}

func ring(dst *ebiten.Image, cx, cy, r int, clr color.RGBA) {
	steps := int(2 * math.Pi * float64(r))
	for i := 0; i < steps; i++ {
		a := 2 * math.Pi * float64(i) / float64(steps)
		x := cx + int(math.Round(float64(r)*math.Cos(a)))
		y := cy + int(math.Round(float64(r)*math.Sin(a)))
		ui.Fill(dst, ui.Rect{X: x, Y: y, W: 1, H: 1}, clr)
	}
}

func drawChapterIcon(dst *ebiten.Image, lib *gfx.Library, icon string, cx, cy int, alpha float32) {
	kind, name, found := strings.Cut(icon, ":")
	if !found {
		kind, name = "page", icon
	}
	switch kind {
	case "page":
		lib.Draw(dst, gfx.PageIconName(name), float64(cx), float64(cy), 1, render.SpriteOptions{Alpha: alpha})
		return
	case "build":
		name = gfx.BuildIcon(name, 1)
	default:
		name = mapSpriteIcons[name]
	}
	sheet, r, ok := lib.Lookup(name)
	if !ok {
		return
	}
	r.AnchorX, r.AnchorY = 0, 0
	scale := min(1, gfx.PortraitScale(r.W, r.H, gfx.PageIconSize, gfx.PageIconSize))
	w, h := float64(r.W)*scale, float64(r.H)*scale
	render.DrawAt(dst, sheet, r, float64(cx)-w/2, float64(cy)-h/2, scale, render.SpriteOptions{Alpha: alpha})
}

func (m *TutorialMap) drawRibbon(dst *ebiten.Image, hover string) {
	x, y := KmapInner.X+12, KmapInner.Y+6
	Literal.DrawShadowAs(dst, mapRibbonTitle, x, y, 2, ui.Palette.Gold, ui.Tint(mapInk, 160))
	Literal.Draw(dst, view.ChapterMapSubtitle, x, y+24, 1, mapInk)
	done := fmt.Sprintf("%d of %d chapters done", m.Record.DoneCount(), len(view.Content.Chapters()))
	w := Literal.Width(done, 1)
	Literal.Draw(dst, done, mapBack.X-w-12, mapBack.Y+5, 1, mapInk)
	ui.Button(dst, Literal, mapBack, "BACK", ui.ButtonState{Hovered: hover == tutorialPrefix+":"+TutorialBack})
	shut := closeButtonAs(KnowledgeBox, tutorialPrefix+":"+TutorialBack+":close", "")
	ui.Button(dst, Literal, shut.Face, "X", ui.ButtonState{Hovered: hover == shut.ID})
	ui.Fill(dst, ui.Rect{X: KmapInner.X + 8, Y: KmapInner.Y + mapRibbonH - 4, W: KmapInner.W - 16, H: 1}, ui.Tint(mapInk, 120))
}

func (m *TutorialMap) drawCard(dst *ebiten.Image, lib *gfx.Library, hover string) {
	s, ok := m.stop(m.focus)
	if !ok {
		return
	}
	r := m.cardRect()
	ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, 245))
	ui.Stroke(dst, r, ui.Palette.Gold)
	ui.Stroke(dst, r.Inset(2), ui.Tint(ui.Palette.Gold, 90))
	x, y := r.X+MapCardPad, r.Y+MapCardPad
	Literal.DrawShadow(dst, strings.ToUpper(s.Chapter.Label())+"  PART "+romanPart(s.Chapter.Part), x, y, 1, ui.Palette.GoldBright)
	y += ui.LineHeight + 2
	for _, row := range m.cardRows(s) {
		if row.label != "" {
			ui.Fill(dst, ui.Rect{X: x, Y: y - 2, W: r.W - 2*MapCardPad, H: 1}, ui.Tint(ui.Palette.Gold, 60))
			Literal.Draw(dst, row.label, x, y, 1, ui.Palette.Gold)
			y += ui.LineHeight
		}
		clr := ui.Palette.Cream
		if row.ink {
			clr = ui.Palette.Text
		}
		if s.State == StopLocked || s.State == StopLater {
			clr = ui.Palette.TextDim
		}
		for _, line := range row.lines {
			Literal.Draw(dst, line, x, y, 1, clr)
			y += ui.LineHeight
		}
		y += 3
	}
	for _, b := range m.cardButtons() {
		if b.Label == "" {
			continue
		}
		hot := hover == b.ID
		if strings.HasPrefix(b.ID, tutorialPrefix+":"+TutorialStart+":") {
			face := ui.Palette.Gold
			if hot {
				face = ui.Palette.GoldBright
			}
			ui.Fill(dst, b.Rect.Offset(1, 2), ui.Palette.Shadow)
			ui.Fill(dst, b.Rect, face)
			ui.Stroke(dst, b.Rect, ui.Palette.Ink)
			Literal.DrawCentered(dst, b.Label, b.Rect, 1, ui.Palette.Ink)
			continue
		}
		ui.Button(dst, Literal, b.Rect, b.Label, ui.ButtonState{Hovered: hot})
	}
}
