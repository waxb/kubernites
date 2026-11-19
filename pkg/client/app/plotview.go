package app

import "github.com/waxb/kubernites/pkg/client/hud"

const plotMargin = 48

func (s *Session) showPlot() {
	g := s.Ghost
	if g == nil {
		return
	}
	gx, gy := float64(g.X)+float64(g.W-1)/2, float64(g.Y)+float64(g.H-1)/2
	sx, sy := s.Cam.GridToScreen(gx, gy)
	if sx >= plotMargin && sx <= hud.Width-plotMargin && sy >= hud.TopH+plotMargin && sy <= hud.ConsoleY-plotMargin {
		return
	}
	s.rememberHome()
	s.GlideTo(gx, gy)
}
