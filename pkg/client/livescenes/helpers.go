package livescenes

import (
	app "github.com/waxb/kubernites/pkg/client/app"
	"github.com/waxb/kubernites/pkg/client/hud"
	"github.com/waxb/kubernites/pkg/client/model"
	"github.com/waxb/kubernites/pkg/client/world"
	"github.com/waxb/kubernites/pkg/view"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	maxAcks      = 6
	townDrawStep = 6
)

func PressButton(s *app.Session, id string) {
	f := s.Frame(true)
	for _, b := range s.Hud.Buttons(f) {
		if b.ID == id {
			s.Update(hud.Input{MX: b.Rect.X + b.Rect.W/2, MY: b.Rect.Y + b.Rect.H/2, Click: true}, 0, &app.Pointer{}, true)
			return
		}
	}
}

func SelectStructure(s *app.Session, st view.Structure) {
	s.Selection = app.SelectionOf(world.Hit{Kind: world.HitStructure, ID: st.ID, Structure: st}, s.Pods)
}

func SelectPod(s *app.Session, match func(model.PodView) bool) {
	for _, p := range s.Pods {
		if match(p) {
			s.Selection = hud.Selection{Kind: world.HitHearth, ID: p.Namespace + "/" + p.Name, Pod: p}
			return
		}
	}
}

func DismissIntros(s *app.Session) {
	idle := hud.Input{MX: -10, MY: -10}
	WarmSession(s, 1)
	for range 4 * maxAcks {
		s.Hud.Update(idle, s.Frame(true))
		PressButton(s, "intro:ack")
		WarmSession(s, .1)
	}
}

func WarmSession(s *app.Session, seconds float64) {
	dst := ebiten.NewImage(hud.Width, hud.Height)
	defer dst.Deallocate()
	Warm(s, dst, seconds)
}

func Warm(s *app.Session, dst *ebiten.Image, seconds float64) {
	for i := 0; float64(i)*app.TickSeconds < seconds; i++ {
		s.Advance(app.TickSeconds)
		if i%townDrawStep == 0 {
			s.Renderer.Draw(dst, s.Cam, s.Scene())
		}
	}
}
