package app

import (
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"
)

const (
	IntroZoom    = 2.0
	RestZoom     = 1.0
	LensDuration = 0.6
	lensHold     = 1.2
	introBias    = 0.3
	GlideSeconds = 0.5
)

type Lens struct {
	Intro   bool `json:"-"`
	Zooming bool `json:"-"`
	from    float64
	to      float64
	back    float64
	start   float64
	Glide   Glide `json:"-"`
}

type Glide struct {
	On    bool `json:"-"`
	from  world.Vec
	To    world.Vec `json:"-"`
	start float64
}

func Ease(t float64) float64 {
	t = min(max(t, 0), 1)
	return t * t * (3 - 2*t)
}

func ViewCenter() (float64, float64) {
	return hud.Width / 2, float64(hud.ConsoleY+hud.TopH) / 2
}

func (s *Session) BeginIntro() {
	g := world.GatewayOf(s.Site)
	mid := world.Lerp(g.Door, world.Vec{X: float64(s.Site.Plaza.X), Y: float64(s.Site.Plaza.Y)}, introBias)
	if harbor, ok := world.HarborFocus(s.Site); ok {
		mid = world.Lerp(harbor, g.Door, introBias)
	}
	s.CenterOn(mid.X, mid.Y)
	cx, cy := ViewCenter()
	s.Cam = s.Cam.ZoomAt(IntroZoom, cx, cy)
	s.Lens = Lens{Intro: true}
}

func (s *Session) EndIntro() {
	if !s.Lens.Intro {
		return
	}
	s.Lens = Lens{Zooming: true, from: s.Cam.Scale(), to: RestZoom, start: s.Clock, Glide: s.Lens.Glide}
}

func (s *Session) glanceOut() {
	if !s.Lens.Intro {
		return
	}
	s.EndIntro()
	s.Lens.back = s.Lens.from
}

func (s *Session) StepLens() {
	s.stepGlide()
	if !s.Lens.Zooming {
		return
	}
	t := (s.Clock - s.Lens.start) / LensDuration
	z := s.Lens.from + (s.Lens.to-s.Lens.from)*Ease(t)
	cx, cy := ViewCenter()
	s.Cam = s.Cam.ZoomAt(z, cx, cy)
	switch {
	case t < 1:
	case s.Lens.back > 0:
		s.Lens = Lens{Zooming: true, from: s.Lens.to, to: s.Lens.back, start: s.Clock + lensHold, Glide: s.Lens.Glide}
	default:
		s.Lens = Lens{Glide: s.Lens.Glide}
	}
}

func (s *Session) ViewTarget() world.Vec {
	cx, cy := ViewCenter()
	gx, gy := s.Cam.ScreenToGrid(cx, cy)
	return world.Vec{X: gx, Y: gy}
}

func (s *Session) GlideTo(gx, gy float64) {
	if s.Reduce {
		s.CenterOn(gx, gy)
		return
	}
	s.Lens.Glide = Glide{On: true, from: s.ViewTarget(), To: world.Vec{X: gx, Y: gy}, start: s.Clock}
}

func (s *Session) stepGlide() {
	g := s.Lens.Glide
	if !g.On {
		return
	}
	t := (s.Clock - g.start) / GlideSeconds
	p := world.Lerp(g.from, g.To, Ease(t))
	s.CenterOn(p.X, p.Y)
	if t >= 1 {
		s.Lens.Glide = Glide{}
	}
}

func (s *Session) Zoom(in hud.Input) {
	switch {
	case s.Lens.Intro:
		s.EndIntro()
	case !s.Lens.Zooming:
		s.Cam = s.Cam.ZoomStep(Sign(in.Wheel), float64(in.MX), float64(in.MY))
		s.StepCam.MovedAt = s.Clock
	}
}

func PlacedStructure(a view.Action, err error) bool {
	return a.Action == "build" && err == nil
}
