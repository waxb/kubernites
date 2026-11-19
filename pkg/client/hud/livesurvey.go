package hud

import (
	"image"
	"image/color"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/waxb/kubernites/pkg/client/gfx"
	"github.com/waxb/kubernites/pkg/client/render"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	surveyFPS      = 60
	surveyKeeps    = 6
	surveyBanners  = 8
	surveyHuts     = 30
	surveyLens     = 22
	surveyReach    = surveyLens + 4
	surveyMagnify  = 1.8
	surveyInkFade  = 18
	surveyRollUp   = 30
	surveyRise     = 150
	surveyZoom     = .5
	surveyHutZoom  = .36
	surveyRim      = 3
	surveyHandle   = 22
	surveySides    = 40
	surveyRollH    = 8
	surveyInsetX   = 24
	surveyInsetY   = 8
	surveyFlagGap  = 50
	surveyInkAlpha = .9
)

var (
	surveyTable = color.RGBA{74, 50, 32, 255}
	surveyBrass = color.RGBA{176, 132, 58, 255}
	surveyWood  = color.RGBA{96, 62, 36, 255}
	surveyGlint = color.RGBA{255, 250, 230, 150}
	surveyRoll  = color.RGBA{196, 168, 116, 255}
	surveyShade = color.RGBA{120, 92, 60, 255}
	surveyInk   = [3]float32{.66, .5, .36}
)

type surveyMark struct {
	kind byte
	i    int
}

type liveSurvey struct {
	frames int
	drawAt int
	inked  map[surveyMark]int
	sheet  *ebiten.Image
}

func (s *liveSurvey) reset() {
	sheet := s.sheet
	*s = liveSurvey{sheet: sheet}
}

func (p *LivePicker) Advance(frames int) {
	for range frames {
		p.survey.tick(p.Progress)
	}
}

func (s *liveSurvey) tick(p LiveProgress) {
	s.frames++
	if p.Step == LiveDrawing && s.drawAt == 0 {
		s.drawAt = s.frames
	}
	if s.drawAt != 0 {
		return
	}
	if s.inked == nil {
		s.inked = map[surveyMark]int{}
	}
	lx, ly := s.lens()
	for _, m := range surveyMarks(p) {
		if _, done := s.inked[m]; done {
			continue
		}
		x, y := m.spot()
		if math.Hypot(x-lx, y-ly) <= surveyReach {
			s.inked[m] = s.frames
		}
	}
}

func (s *liveSurvey) lens() (float64, float64) {
	w, h := surveyPaper()
	t := float64(s.frames) / surveyFPS
	return w/2 + (w/2-36)*math.Sin(.7*t), h/2 + (h/2-26)*math.Sin(1.13*t+.9)
}

func (s *liveSurvey) rolled() float64 {
	if s.drawAt == 0 {
		return 0
	}
	return min(float64(s.frames-s.drawAt)/surveyRollUp, 1)
}

func (s *liveSurvey) risen() float64 {
	if s.drawAt == 0 {
		return 0
	}
	return min(max(float64(s.frames-s.drawAt-surveyRollUp)/surveyRise, 0), 1)
}

func surveyPaper() (float64, float64) {
	sea := liveSea()
	return float64(sea.W - 2*surveyInsetX), float64(sea.H - 2*surveyInsetY)
}

func liveSea() ui.Rect {
	return ui.Rect{X: LiveBox.X + 16, Y: LiveBox.Y + riseSeaTop, W: LiveBox.W - 32, H: riseSeaH}
}

func surveyMarks(p LiveProgress) []surveyMark {
	var out []surveyMark
	for i := range min(p.Nodes, surveyKeeps) {
		out = append(out, surveyMark{'k', i})
	}
	for i := range min(p.Namespaces, surveyBanners) {
		out = append(out, surveyMark{'b', i})
	}
	if p.Nodes > 0 {
		for i := range min(p.Pods, surveyHuts) {
			out = append(out, surveyMark{'h', i})
		}
	}
	return out
}

func keepSpot(k int) (float64, float64) {
	w, h := surveyPaper()
	a := 2*math.Pi*float64(k)/surveyKeeps + .4
	return w/2 + w*.24*math.Cos(a), h/2 + 8 + h*.2*math.Sin(a)
}

func (m surveyMark) spot() (float64, float64) {
	w, _ := surveyPaper()
	switch m.kind {
	case 'k':
		return keepSpot(m.i)
	case 'b':
		return 30 + float64(m.i)*surveyFlagGap + float64(m.i%2)*6, 24
	}
	kx, ky := keepSpot(m.i % 3)
	ring := 30 + float64(m.i/3/6)*13
	a := float64(m.i/3%6)*math.Pi/3 + float64(m.i%3)*.5
	return min(max(kx+ring*math.Cos(a), 14), w-14), ky + .5*ring*math.Sin(a)
}

func (s *liveSurvey) canvas(w, h int) *ebiten.Image {
	if s.sheet == nil || s.sheet.Bounds().Dx() != w || s.sheet.Bounds().Dy() != h {
		s.sheet = ebiten.NewImage(w, h)
	}
	s.sheet.Clear()
	return s.sheet
}

var surveyDecor = []struct {
	kind string
	x, y float64
}{{"compass", -46, -46}, {"waves", 10, -16}, {"waves", 28, -10}, {"hills", -96, 38}, {"forest", 12, 40}}

func (p *LivePicker) drawSurvey(dst *ebiten.Image) {
	sea := liveSea()
	paper := ui.Rect{X: sea.X + surveyInsetX, Y: sea.Y + surveyInsetY, W: sea.W - 2*surveyInsetX, H: sea.H - 2*surveyInsetY}
	sheet := p.survey.canvas(paper.W, paper.H)
	p.inkSheet(sheet, paper.W, paper.H)
	shown := int(math.Round(float64(paper.H) * (1 - p.survey.rolled())))
	if shown > 0 {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(paper.X), float64(paper.Y))
		dst.DrawImage(sheet.SubImage(image.Rect(0, 0, paper.W, shown)).(*ebiten.Image), op)
	}
	if p.survey.drawAt != 0 {
		roll := ui.Rect{X: paper.X - 4, Y: paper.Y + shown - surveyRollH/2, W: paper.W + 8, H: surveyRollH}
		ui.Fill(dst, roll, surveyRoll)
		ui.Fill(dst, ui.Rect{X: roll.X, Y: roll.Y + roll.H - 2, W: roll.W, H: 2}, surveyShade)
		return
	}
	p.drawLens(dst, sheet, paper)
}

func (p *LivePicker) inkSheet(sheet *ebiten.Image, w, h int) {
	area := ui.Rect{W: w, H: h}
	if p.Lib == nil {
		ui.Fill(sheet, area, ui.Palette.Parchment)
		ui.Stroke(sheet, area.Inset(3), mapInk)
		return
	}
	tileMapParchment(sheet, p.Lib, area)
	ui.Stroke(sheet, area.Inset(3), mapInk)
	for _, d := range surveyDecor {
		x, y := d.x, d.y
		if x < 0 {
			x += float64(w)
		}
		if y < 0 {
			y += float64(h)
		}
		p.Lib.Draw(sheet, gfx.KMapDecorName(d.kind), x, y, 1, render.SpriteOptions{Alpha: .7})
	}
	keep := gfx.PreviewNames(view.Structure{Kind: view.StructKeep, Variant: "stone", State: view.StateReady}, 0)
	marks := surveyMarks(p.Progress)
	sort.SliceStable(marks, func(a, b int) bool {
		_, ya := marks[a].spot()
		_, yb := marks[b].spot()
		return ya < yb
	})
	for _, m := range marks {
		at, ok := p.survey.inked[m]
		if !ok {
			continue
		}
		o := render.SpriteOptions{Alpha: float32(min(float64(p.survey.frames-at+1)/surveyInkFade, 1) * surveyInkAlpha), Tint: surveyInk}
		x, y := m.spot()
		switch m.kind {
		case 'k':
			p.Lib.DrawFirst(sheet, keep, x, y+6, surveyZoom, o)
		case 'b':
			p.Lib.Draw(sheet, gfx.BorderFlagName(m.i), x, y, 1, o)
		default:
			p.Lib.DrawFirst(sheet, gfx.HearthLookNames(1, "ready", "Burstable", 0, m.i), x, y+3, surveyHutZoom, o)
		}
	}
}

func (p *LivePicker) drawLens(dst, sheet *ebiten.Image, paper ui.Rect) {
	lx, ly := p.survey.lens()
	cx, cy := float32(paper.X)+float32(lx), float32(paper.Y)+float32(ly)
	vs := make([]ebiten.Vertex, 0, surveySides+1)
	is := make([]uint16, 0, surveySides*3)
	vs = append(vs, ebiten.Vertex{DstX: cx, DstY: cy, SrcX: float32(lx), SrcY: float32(ly), ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1})
	for k := range surveySides {
		a := 2 * math.Pi * float64(k) / surveySides
		c, s := float32(math.Cos(a)), float32(math.Sin(a))
		sx := min(max(float32(lx)+c*surveyLens/surveyMagnify, 0), float32(paper.W))
		sy := min(max(float32(ly)+s*surveyLens/surveyMagnify, 0), float32(paper.H))
		vs = append(vs, ebiten.Vertex{DstX: cx + c*surveyLens, DstY: cy + s*surveyLens, SrcX: sx, SrcY: sy, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1})
		is = append(is, 0, uint16(k+1), uint16((k+1)%surveySides+1))
	}
	dst.DrawTriangles(vs, is, sheet, &ebiten.DrawTrianglesOptions{})
	hx, hy := cx+surveyLens*.72, cy+surveyLens*.72
	vector.StrokeLine(dst, hx, hy, hx+surveyHandle*.72, hy+surveyHandle*.72, 6, surveyWood, true)
	vector.StrokeCircle(dst, cx, cy, surveyLens, surveyRim, surveyBrass, true)
	vector.StrokeLine(dst, cx-surveyLens*.45, cy-surveyLens*.6, cx-surveyLens*.1, cy-surveyLens*.75, 2, surveyGlint, true)
}
