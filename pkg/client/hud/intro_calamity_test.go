package hud

import (
	"testing"

	"github.com/waxb/kubernites/pkg/client/ui"
)

func TestAnIntroCardKeepsClearOfTheCalamityCard(t *testing.T) {
	h := New(nil, (&fakeRunner{}).backend())
	f := settled(h, calamityFrame(t, noDepot))
	cal, ok := h.CalamityRect(f)
	if !ok {
		t.Fatal("no calamity card")
	}
	for _, target := range []struct {
		r  ui.Rect
		ok bool
	}{{ui.Rect{X: cal.X - 40, Y: cal.Y + cal.H + 12, W: 30, H: 14}, true}, {ui.Rect{X: cal.X + cal.W/2, Y: cal.Y + cal.H + 30, W: 20, H: 12}, true}, {ui.Rect{}, false}} {
		card := h.placeGuideCard(f, target.r, target.ok, IntroW, 60)
		if Overlaps(card, cal) {
			t.Errorf("the intro card at %+v covers the calamity card at %+v (target %+v)", card, cal, target.r)
		}
	}
}
