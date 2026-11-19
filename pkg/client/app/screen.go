package app

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/ui"
	"github.com/waxb/kubernites/pkg/view"
)

const FillShare = 0.99

func FitScale(outW, outH int) float64 {
	fit := min(float64(outW)/hud.Width, float64(outH)/hud.Height)
	whole := math.Floor(fit)
	if whole >= 1 && whole/fit >= FillShare {
		return whole
	}
	return fit
}

func FitGeoM(outW, outH int) ebiten.GeoM {
	s := FitScale(outW, outH)
	var m ebiten.GeoM
	m.Scale(s, s)
	m.Translate(math.Round((float64(outW)-hud.Width*s)/2), math.Round((float64(outH)-hud.Height*s)/2))
	return m
}

type FinalState struct {
	Def   ebiten.GeoM `json:"-"`
	Fit   ebiten.GeoM `json:"-"`
	Ok    bool        `json:"-"`
	Small bool        `json:"-"`
}

func (g *Game) DrawFinalScreen(screen ebiten.FinalScreen, offscreen *ebiten.Image, geoM ebiten.GeoM) {
	b := screen.Bounds()
	fit := FitGeoM(b.Dx(), b.Dy())
	g.Final = FinalState{Def: geoM, Fit: fit, Ok: true, Small: FitScale(b.Dx(), b.Dy()) < 1}
	screen.Fill(ui.Palette.ConsoleEdge)
	op := &ebiten.DrawImageOptions{GeoM: fit, Filter: FinalFilter(FitScale(b.Dx(), b.Dy()))}
	screen.DrawImage(offscreen, op)
}

func (g *Game) ToLogical(cx, cy float64) (int, int) {
	if !g.Final.Ok {
		return int(cx), int(cy)
	}
	px, py := g.Final.Def.Apply(cx, cy)
	inv := g.Final.Fit
	inv.Invert()
	lx, ly := inv.Apply(px, py)
	return int(math.Floor(lx)), int(math.Floor(ly))
}

func (g *Game) ensureDemo() {
	if g.Demo != nil {
		return
	}
	r := campaignHook.DemoRealm()
	if r == nil {
		return
	}
	g.Demo = NewSession(r, g.Lib)
	if s, ok := FirstKeep(g.Demo.Site); ok {
		g.Demo.CenterOn(float64(s.X), float64(s.Y))
	}
}

func FirstKeep(site view.Site) (view.Structure, bool) {
	for _, s := range site.Structures {
		if s.Kind == view.StructKeep {
			return s, true
		}
	}
	return view.Structure{}, false
}

func (g *Game) drawPlacementHint(dst *ebiten.Image) {
	s := g.Sess
	if s.Placing == "" || s.Ghost == nil || s.Hud.Overlay() != hud.OverlayNone {
		return
	}
	text := "Click to build. Right click or Esc cancels."
	clr := ui.Palette.Cream
	var font ui.Font
	w, _ := font.Measure(text, 1)
	r := ui.Rect{X: hud.Width/2 - w/2 - 6, Y: hud.ConsoleY - 20, W: w + 12, H: 14}
	ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, 220))
	ui.Stroke(dst, r, ui.Palette.Gold)
	font.Draw(dst, text, r.X+6, r.Y+4, 1, clr)
}

const scaleHint = "Enlarge the window for crisper text"

func (g *Game) ScaleHinted() bool {
	small := g.Final.Small || g.Capture != nil && g.Capture.window.X > 0 && FitScale(g.Capture.window.X, g.Capture.window.Y) < 1
	return small && (g.Scene != SceneRealm || g.Sess == nil || g.Sess.Hud.Overlay() == hud.OverlayNone)
}

func (g *Game) drawScaleHint(dst *ebiten.Image) {
	if !g.ScaleHinted() {
		return
	}
	var font ui.Font
	w, _ := font.Measure(scaleHint, 1)
	r := ui.Rect{X: hud.Width/2 - w/2 - 6, Y: hud.ConsoleY - 38, W: w + 12, H: 15}
	ui.Fill(dst, r, ui.Tint(ui.Palette.ConsoleEdge, 220))
	ui.Stroke(dst, r, ui.Palette.Gold)
	font.Draw(dst, scaleHint, r.X+6, r.Y+4, 1, ui.Palette.Text)
}

func (g *Game) drawFade(dst *ebiten.Image) {
	if g.Fade <= 0 {
		return
	}
	ui.Fill(dst, ui.Rect{W: hud.Width, H: hud.Height}, ui.Tint(ui.Palette.Shadow, uint8(255*g.Fade)))
}
