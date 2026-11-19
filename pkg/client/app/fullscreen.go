package app

import (
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/waxb/kubernites/pkg/client/hud"
)

type windowControl interface {
	IsFullscreen() bool
	SetFullscreen(on bool)
}

type EbitenWindow struct{}

func (EbitenWindow) IsFullscreen() bool    { return ebiten.IsFullscreen() }
func (EbitenWindow) SetFullscreen(on bool) { ebiten.SetFullscreen(on) }

type HeldWindow struct {
	On bool `json:"-"`
}

func (w *HeldWindow) IsFullscreen() bool    { return w.On }
func (w *HeldWindow) SetFullscreen(on bool) { w.On = on }

func (o Options) WindowControl() windowControl {
	switch {
	case o.Window != nil:
		return o.Window
	case o.Capture != "":
		return &HeldWindow{}
	}
	return EbitenWindow{}
}

func FullscreenKey(in hud.Input) (hud.Input, bool) {
	f11 := in.Pressed(ebiten.KeyF11)
	altEnter := in.Alt && in.Pressed(ebiten.KeyEnter)
	if !f11 && !altEnter {
		return in, false
	}
	next := in
	next.Keys = slices.DeleteFunc(slices.Clone(in.Keys), func(k ebiten.Key) bool {
		return k == ebiten.KeyF11 || altEnter && k == ebiten.KeyEnter
	})
	return next, true
}

func (g *Game) ApplyFullscreenKey(in hud.Input) hud.Input {
	if g.Sess != nil && g.Sess.Hud.StoneTerminal() {
		return in
	}
	next, flip := FullscreenKey(in)
	if flip {
		g.ToggleFullscreen()
	}
	return next
}

func (g *Game) ToggleFullscreen() {
	on := !g.window.IsFullscreen()
	g.window.SetFullscreen(on)
	g.Settings.Fullscreen = on
	if err := SaveSettings(g.SettingsPath(), g.Settings); err != nil {
		g.Log.Warn("could not save settings", "err", err)
	}
}

func FinalFilter(scale float64) ebiten.Filter {
	switch {
	case scale < 1:
		return ebiten.FilterLinear
	case math.Floor(scale) == scale:
		return ebiten.FilterNearest
	}
	return ebiten.FilterPixelated
}
